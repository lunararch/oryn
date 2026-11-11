package process

import (
	"bytes"
	"runtime"
	"testing"
	"time"
)

func TestStartAndExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executables")
	}
	var stdout bytes.Buffer
	p, err := Start(Spec{Path: "/bin/echo", Args: []string{"hello"}, Stdout: &stdout})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	result := wait(t, p)
	if result.ExitCode != 0 || result.Err != nil {
		t.Fatalf("result = %+v", result)
	}
	if stdout.String() != "hello\n" {
		t.Errorf("stdout = %q", stdout.String())
	}
}

func TestNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executables")
	}
	p, err := Start(Spec{Path: "/bin/false"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	result := wait(t, p)
	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
	}
	if result.Err == nil {
		t.Error("Err = nil for non-zero exit")
	}
}

func TestWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executables")
	}
	var stdout bytes.Buffer
	dir := t.TempDir()
	p, err := Start(Spec{Path: "/bin/pwd", Dir: dir, Stdout: &stdout})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	wait(t, p)
	if got := stdout.String(); got != dir+"\n" {
		t.Errorf("pwd = %q, want %q", got, dir+"\n")
	}
}

func TestStartFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executable")
	}
	if _, err := Start(Spec{Path: "/definitely/not/a/game"}); err == nil {
		t.Fatal("Start() succeeded for a missing executable")
	}
}

func wait(t *testing.T, p *Process) Result {
	t.Helper()
	select {
	case result := <-p.Done():
		return result
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit")
		return Result{}
	}
}
