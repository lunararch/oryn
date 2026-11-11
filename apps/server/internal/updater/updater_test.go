package updater

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyIsNoopWhenAlreadyCurrent(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"git status --porcelain":                                         "",
		"git fetch --prune --tags origin":                                "",
		"git tag --merged origin/main --list v* --sort=-version:refname": "",
		"git rev-parse origin/main^{commit}":                             "abc123",
		"git rev-parse HEAD":                                             "abc123",
	}}
	u := newTestUpdater(t, runner)

	result, err := u.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Updated {
		t.Fatalf("Updated = true, result = %+v", result)
	}
	if runner.called("docker") {
		t.Fatal("docker was called for a no-op update")
	}
}

func TestApplyBuildsNewCommitAndChecksHealth(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"git status --porcelain":                                         "",
		"git fetch --prune --tags origin":                                "",
		"git tag --merged origin/main --list v* --sort=-version:refname": "v0.2.0\nv0.1.0\n",
		"git rev-parse v0.2.0^{commit}":                                  "new456",
		"git rev-parse HEAD":                                             "old123",
		"docker compose --env-file .env -f deployments/docker/compose.yaml exec -T oryn /app/oryn-server backup": "/data/backups/oryn-1.db\n",
	}}
	health := &fakeHealth{errors: []error{nil}}
	u := newTestUpdater(t, runner)
	u.Health = health

	result, err := u.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !result.Updated || result.From != "old123" || result.To != "new456" {
		t.Fatalf("result = %+v", result)
	}
	if result.BackupPath != "/data/backups/oryn-1.db" {
		t.Errorf("backup = %q", result.BackupPath)
	}
	for _, want := range [][]string{
		{"git", "checkout", "--force", "new456"},
		{"docker", "compose", "--env-file", ".env", "-f", "deployments/docker/compose.yaml", "build", "--build-arg", "VERSION=v0.2.0"},
		{"docker", "compose", "--env-file", ".env", "-f", "deployments/docker/compose.yaml", "up", "-d"},
	} {
		if !runner.hasCall(want) {
			t.Errorf("missing call %v; calls = %v", want, runner.callStrings())
		}
	}
	if len(health.urls) != 1 || health.urls[0] != "http://127.0.0.1:8082/api/v1/health" {
		t.Errorf("health URLs = %v", health.urls)
	}
}

func TestApplyRollsBackWhenHealthFails(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"git status --porcelain":                                         "",
		"git fetch --prune --tags origin":                                "",
		"git tag --merged origin/main --list v* --sort=-version:refname": "",
		"git rev-parse origin/main^{commit}":                             "new456",
		"git rev-parse HEAD":                                             "old123",
		"docker compose --env-file .env -f deployments/docker/compose.yaml exec -T oryn /app/oryn-server backup": "/data/backups/oryn-1.db\n",
	}}
	health := &fakeHealth{errors: []error{errors.New("unhealthy"), nil}}
	u := newTestUpdater(t, runner)
	u.Health = health

	_, err := u.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply() succeeded despite a failed health check")
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Errorf("error = %v", err)
	}
	if !runner.hasCall([]string{"git", "checkout", "--force", "old123"}) {
		t.Errorf("missing rollback checkout; calls = %v", runner.callStrings())
	}
	if !runner.hasCall([]string{"docker", "compose", "--env-file", ".env", "-f", "deployments/docker/compose.yaml", "up", "-d"}) {
		t.Errorf("missing rollback compose up; calls = %v", runner.callStrings())
	}
}

func TestApplyRefusesDirtyRepository(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"git status --porcelain": " M README.md\n",
	}}
	u := newTestUpdater(t, runner)
	if _, err := u.Apply(context.Background()); !errors.Is(err, ErrDirtyRepository) {
		t.Fatalf("error = %v, want ErrDirtyRepository", err)
	}
	if runner.called("git fetch") {
		t.Error("fetched a dirty repository")
	}
}

func TestApplyAbortsWhenBackupFails(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string]string{
			"git status --porcelain":                                         "",
			"git fetch --prune --tags origin":                                "",
			"git tag --merged origin/main --list v* --sort=-version:refname": "",
			"git rev-parse origin/main^{commit}":                             "new456",
			"git rev-parse HEAD":                                             "old123",
		},
		errors: map[string]error{
			"docker compose --env-file .env -f deployments/docker/compose.yaml exec -T oryn /app/oryn-server backup": errors.New("backup failed"),
		},
	}
	u := newTestUpdater(t, runner)
	if _, err := u.Apply(context.Background()); err == nil {
		t.Fatal("Apply() succeeded without a backup")
	}
	if runner.hasCall([]string{"git", "checkout", "--force", "new456"}) {
		t.Error("checked out a new release after the backup failed")
	}
}

func TestSelectTargetPrefersLatestValidTag(t *testing.T) {
	ref, err := selectTarget("v0.10.0\nv0.9.0\nnot-a-version\nv1.0.0-rc.1\n", "origin/main")
	if err != nil {
		t.Fatalf("selectTarget() error = %v", err)
	}
	if ref != "v0.10.0" {
		t.Errorf("ref = %q, want v0.10.0", ref)
	}
	ref, err = selectTarget("\n", "origin/main")
	if err != nil || ref != "origin/main" {
		t.Errorf("empty tags => %q, %v", ref, err)
	}
}

func newTestUpdater(t *testing.T, runner *fakeRunner) *Updater {
	t.Helper()
	u := New(Config{
		RepoDir:       t.TempDir(),
		Remote:        "origin",
		Branch:        "main",
		ComposeFile:   "deployments/docker/compose.yaml",
		EnvFile:       ".env",
		HealthURL:     "http://127.0.0.1:8082/api/v1/health",
		StateFile:     filepath.Join(t.TempDir(), "update-state.json"),
		DockerCommand: []string{"docker"},
		Now:           func() time.Time { return time.Unix(1700000000, 0) },
	}, runner)
	u.Health = &fakeHealth{}
	return u
}

type fakeRunner struct {
	responses map[string]string
	errors    map[string]error
	calls     [][]string
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ []string, name string, args ...string) (string, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)
	key := strings.Join(call, " ")
	if err := f.errors[key]; err != nil {
		return "", err
	}
	return f.responses[key], nil
}

func (f *fakeRunner) called(prefix string) bool {
	for _, call := range f.calls {
		if len(call) > 0 && call[0] == prefix {
			return true
		}
	}
	return false
}

func (f *fakeRunner) hasCall(want []string) bool {
	for _, call := range f.calls {
		if strings.Join(call, "\x00") == strings.Join(want, "\x00") {
			return true
		}
	}
	return false
}

func (f *fakeRunner) callStrings() []string {
	out := make([]string, 0, len(f.calls))
	for _, call := range f.calls {
		out = append(out, strings.Join(call, " "))
	}
	return out
}

type fakeHealth struct {
	errors []error
	urls   []string
}

func (f *fakeHealth) Check(_ context.Context, url string) error {
	f.urls = append(f.urls, url)
	if len(f.errors) == 0 {
		return nil
	}
	err := f.errors[0]
	f.errors = f.errors[1:]
	return err
}
