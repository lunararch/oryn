package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/lunararch/oryn/apps/server/internal/updater"
)

var version = "dev"

func main() {
	var (
		repoDir     = flag.String("repo", ".", "git checkout to update")
		remote      = flag.String("remote", "origin", "git remote")
		branch      = flag.String("branch", "main", "branch used when no release tags exist")
		composeFile = flag.String("compose", "deployments/docker/compose.yaml", "Compose file")
		envFile     = flag.String("env", "deployments/docker/.env", "Compose environment file")
		healthURL   = flag.String("health-url", "http://127.0.0.1:8080/api/v1/health", "server health URL")
		stateFile   = flag.String("state", "", "write update state JSON here")
		sshKey      = flag.String("ssh-key", "", "git SSH key for private repositories")
		useSudo     = flag.Bool("sudo", false, "run docker through sudo -n")
		skipBackup  = flag.Bool("skip-backup", false, "skip the pre-update SQLite backup")
		showVersion = flag.Bool("version", false, "print the updater version")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	docker := []string{"docker"}
	if *useSudo {
		docker = []string{"sudo", "-n", "docker"}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	u := updater.New(updater.Config{
		RepoDir:       *repoDir,
		Remote:        *remote,
		Branch:        *branch,
		ComposeFile:   *composeFile,
		EnvFile:       *envFile,
		HealthURL:     *healthURL,
		StateFile:     *stateFile,
		SSHKey:        *sshKey,
		DockerCommand: docker,
		SkipBackup:    *skipBackup,
	}, execRunner{})

	result, err := u.Apply(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "update failed:", err)
		os.Exit(1)
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s %v: %w: %s", name, args, err, truncate(string(output), 4000))
	}
	return string(output), nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}
