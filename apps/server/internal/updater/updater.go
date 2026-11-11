package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrDirtyRepository means the checkout has local changes and cannot be
// updated safely.
var ErrDirtyRepository = errors.New("repository has local changes")

// Runner executes an external command without a shell.
type Runner interface {
	Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error)
}

// HealthChecker verifies that the running server is healthy.
type HealthChecker interface {
	Check(ctx context.Context, url string) error
}

type Config struct {
	RepoDir       string
	Remote        string
	Branch        string
	ComposeFile   string
	EnvFile       string
	HealthURL     string
	StateFile     string
	SSHKey        string
	DockerCommand []string
	SkipBackup    bool
	Now           func() time.Time
}

type Result struct {
	Updated    bool
	From       string
	To         string
	Target     string
	BackupPath string
}

type State struct {
	UpdatedAt  string `json:"updatedAt"`
	From       string `json:"from"`
	To         string `json:"to"`
	Target     string `json:"target"`
	BackupPath string `json:"backupPath"`
}

type Updater struct {
	Config  Config
	Runner  Runner
	Health  HealthChecker
	version string
}

func New(cfg Config, runner Runner) *Updater {
	if cfg.Remote == "" {
		cfg.Remote = "origin"
	}
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if cfg.ComposeFile == "" {
		cfg.ComposeFile = "deployments/docker/compose.yaml"
	}
	if cfg.EnvFile == "" {
		cfg.EnvFile = "deployments/docker/.env"
	}
	if cfg.HealthURL == "" {
		cfg.HealthURL = "http://127.0.0.1:8080/api/v1/health"
	}
	if len(cfg.DockerCommand) == 0 {
		cfg.DockerCommand = []string{"docker"}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Updater{
		Config: cfg,
		Runner: runner,
		Health: &HTTPHealth{Client: &http.Client{Timeout: 5 * time.Second}},
	}
}

func (u *Updater) Apply(ctx context.Context) (Result, error) {
	if u.Runner == nil {
		return Result{}, fmt.Errorf("command runner is required")
	}
	if u.Health == nil {
		return Result{}, fmt.Errorf("health checker is required")
	}

	status, err := u.git(ctx, "status", "--porcelain")
	if err != nil {
		return Result{}, fmt.Errorf("check repository status: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return Result{}, ErrDirtyRepository
	}

	if _, err := u.git(ctx, "fetch", "--prune", "--tags", u.Config.Remote); err != nil {
		return Result{}, fmt.Errorf("fetch repository: %w", err)
	}
	branchRef := u.Config.Remote + "/" + u.Config.Branch
	tags, err := u.git(ctx, "tag", "--merged", branchRef, "--list", "v*", "--sort=-version:refname")
	if err != nil {
		return Result{}, fmt.Errorf("list release tags: %w", err)
	}
	target, err := selectTarget(tags, branchRef)
	if err != nil {
		return Result{}, err
	}
	targetCommit, err := u.git(ctx, "rev-parse", target+"^{commit}")
	if err != nil {
		return Result{}, fmt.Errorf("resolve target %s: %w", target, err)
	}
	targetCommit = strings.TrimSpace(targetCommit)
	current, err := u.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return Result{}, fmt.Errorf("resolve current commit: %w", err)
	}
	current = strings.TrimSpace(current)
	u.version = target
	if !strings.HasPrefix(target, "v") {
		u.version = "main-" + shortCommit(targetCommit)
	}
	result := Result{From: current, To: targetCommit, Target: target}
	if current == targetCommit {
		return result, nil
	}

	if !u.Config.SkipBackup {
		backup, err := u.runDocker(ctx, append(u.composeArgs(), "exec", "-T", "oryn", "/app/oryn-server", "backup")...)
		if err != nil {
			return result, fmt.Errorf("pre-update backup failed: %w", err)
		}
		result.BackupPath = strings.TrimSpace(backup)
	}

	if _, err := u.git(ctx, "checkout", "--force", targetCommit); err != nil {
		return result, fmt.Errorf("checkout target: %w", err)
	}
	if err := u.rebuild(ctx); err != nil {
		if rollbackErr := u.rollback(ctx, current); rollbackErr != nil {
			return result, fmt.Errorf("update failed (%v) and rollback failed: %w", err, rollbackErr)
		}
		return result, fmt.Errorf("update failed and rolled back: %w", err)
	}
	if err := u.Health.Check(ctx, u.Config.HealthURL); err != nil {
		if rollbackErr := u.rollback(ctx, current); rollbackErr != nil {
			return result, fmt.Errorf("health check failed (%v) and rollback failed: %w", err, rollbackErr)
		}
		return result, fmt.Errorf("health check failed and rolled back: %w", err)
	}

	result.Updated = true
	if err := u.writeState(result); err != nil {
		return result, fmt.Errorf("update succeeded but state file failed: %w", err)
	}
	return result, nil
}

func (u *Updater) rebuild(ctx context.Context) error {
	buildArgs := append(u.composeArgs(), "build")
	if u.version != "" {
		buildArgs = append(buildArgs, "--build-arg", "VERSION="+u.version)
	}
	if _, err := u.runDocker(ctx, buildArgs...); err != nil {
		return fmt.Errorf("build image: %w", err)
	}
	if _, err := u.runDocker(ctx, append(u.composeArgs(), "up", "-d")...); err != nil {
		return fmt.Errorf("start image: %w", err)
	}
	return nil
}

func (u *Updater) rollback(ctx context.Context, commit string) error {
	u.version = "rollback-" + shortCommit(commit)
	if _, err := u.git(ctx, "checkout", "--force", commit); err != nil {
		return fmt.Errorf("checkout previous commit: %w", err)
	}
	if err := u.rebuild(ctx); err != nil {
		return err
	}
	return u.Health.Check(ctx, u.Config.HealthURL)
}

func (u *Updater) composeArgs() []string {
	return []string{"compose", "--env-file", u.Config.EnvFile, "-f", u.Config.ComposeFile}
}

func (u *Updater) runDocker(ctx context.Context, args ...string) (string, error) {
	command := append([]string{}, u.Config.DockerCommand...)
	name := command[0]
	command = append(command, args...)
	return u.Runner.Run(ctx, u.Config.RepoDir, nil, name, command[1:]...)
}

func shortCommit(commit string) string {
	if len(commit) <= 12 {
		return commit
	}
	return commit[:12]
}

func (u *Updater) git(ctx context.Context, args ...string) (string, error) {
	env := []string{"GIT_TERMINAL_PROMPT=0"}
	if u.Config.SSHKey != "" {
		env = append(env, "GIT_SSH_COMMAND=ssh -i "+u.Config.SSHKey+" -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes")
	}
	return u.Runner.Run(ctx, u.Config.RepoDir, env, "git", args...)
}

func (u *Updater) writeState(result Result) error {
	if u.Config.StateFile == "" {
		return nil
	}
	state := State{
		UpdatedAt:  u.Config.Now().UTC().Format(time.RFC3339),
		From:       result.From,
		To:         result.To,
		Target:     result.Target,
		BackupPath: result.BackupPath,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(u.Config.StateFile), 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(u.Config.StateFile), ".update-state-*.json")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, u.Config.StateFile)
}

var stableVersion = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

func selectTarget(tags string, branchRef string) (string, error) {
	type version struct {
		tag                 string
		major, minor, patch int
	}
	var versions []version
	for _, tag := range strings.Split(tags, "\n") {
		tag = strings.TrimSpace(tag)
		match := stableVersion.FindStringSubmatch(tag)
		if match == nil {
			continue
		}
		major, _ := strconv.Atoi(match[1])
		minor, _ := strconv.Atoi(match[2])
		patch, _ := strconv.Atoi(match[3])
		versions = append(versions, version{tag: tag, major: major, minor: minor, patch: patch})
	}
	if len(versions) == 0 {
		return branchRef, nil
	}
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].major != versions[j].major {
			return versions[i].major > versions[j].major
		}
		if versions[i].minor != versions[j].minor {
			return versions[i].minor > versions[j].minor
		}
		return versions[i].patch > versions[j].patch
	})
	return versions[0].tag, nil
}

type HTTPHealth struct {
	Client *http.Client
}

func (h *HTTPHealth) Check(ctx context.Context, url string) error {
	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(req)
		if err == nil {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			_ = response.Body.Close()
			if readErr == nil && response.StatusCode == http.StatusOK {
				var health struct {
					Status   string `json:"status"`
					Database string `json:"database"`
				}
				if json.Unmarshal(body, &health) == nil && health.Status == "ok" && health.Database == "ok" {
					return nil
				}
			}
			if readErr == nil {
				lastErr = fmt.Errorf("health endpoint returned status %d", response.StatusCode)
			} else {
				lastErr = readErr
			}
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("server did not become healthy: %w", lastErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
