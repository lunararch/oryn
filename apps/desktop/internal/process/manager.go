package process

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// Spec describes a local process launch. Arguments are passed as an array and
// never interpolated into a shell command.
type Spec struct {
	Path   string
	Args   []string
	Dir    string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer
}

// Result is the outcome of a completed process.
type Result struct {
	ExitCode  int
	Err       error
	StartedAt time.Time
	EndedAt   time.Time
}

// Process is a started child process.
type Process struct {
	cmd  *exec.Cmd
	done chan Result
}

// Start launches a process without a shell.
func Start(spec Spec) (*Process, error) {
	if spec.Path == "" {
		return nil, fmt.Errorf("executable path is required")
	}
	cmd := exec.Command(spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	if spec.Env != nil {
		cmd.Env = spec.Env
	}
	if spec.Stdout != nil {
		cmd.Stdout = spec.Stdout
	}
	if spec.Stderr != nil {
		cmd.Stderr = spec.Stderr
	}

	startedAt := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}
	p := &Process{cmd: cmd, done: make(chan Result, 1)}
	go func() {
		err := cmd.Wait()
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		p.done <- Result{
			ExitCode:  exitCode,
			Err:       err,
			StartedAt: startedAt,
			EndedAt:   time.Now(),
		}
	}()
	return p, nil
}

// Done returns a channel that receives exactly one result.
func (p *Process) Done() <-chan Result {
	return p.done
}

// PID returns the child process ID.
func (p *Process) PID() int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Kill terminates the child process. Callers should only use this for a
// process Oryn started and only after an explicit user action or shutdown.
func (p *Process) Kill() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}
