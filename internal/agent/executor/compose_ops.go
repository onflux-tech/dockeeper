package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// ComposeOutputLine represents a single line of output from docker compose.
type ComposeOutputLine struct {
	Line   string
	Stream string // "stdout" or "stderr"
}

// ComposeResult holds the outcome of a docker compose operation.
type ComposeResult struct {
	ExitCode int
	Error    string
}

// ComposeRunner executes docker compose CLI commands and streams output.
type ComposeRunner struct{}

// NewComposeRunner creates a new ComposeRunner.
func NewComposeRunner() *ComposeRunner {
	return &ComposeRunner{}
}

// buildArgs constructs the common docker compose CLI arguments.
func buildArgs(projectName, workingDir string, files []string, subcommand string, extra ...string) []string {
	args := []string{"compose"}

	if projectName != "" {
		args = append(args, "-p", projectName)
	}

	for _, f := range files {
		args = append(args, "-f", f)
	}

	args = append(args, subcommand)
	args = append(args, extra...)

	return args
}

// RunCompose executes a docker compose subcommand and streams output lines
// through the returned channel. The channel is closed when the command finishes.
// The ComposeResult is sent as context on completion.
func (r *ComposeRunner) RunCompose(
	ctx context.Context,
	projectName, workingDir string,
	files []string,
	subcommand string,
	extra ...string,
) (<-chan ComposeOutputLine, <-chan ComposeResult) {
	outCh := make(chan ComposeOutputLine, 64)
	doneCh := make(chan ComposeResult, 1)

	go func() {
		defer close(outCh)
		defer close(doneCh)

		args := buildArgs(projectName, workingDir, files, subcommand, extra...)
		cmd := exec.CommandContext(ctx, "docker", args...)

		if workingDir != "" {
			cmd.Dir = workingDir
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			doneCh <- ComposeResult{ExitCode: -1, Error: fmt.Sprintf("stdout pipe: %s", err.Error())}
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			doneCh <- ComposeResult{ExitCode: -1, Error: fmt.Sprintf("stderr pipe: %s", err.Error())}
			return
		}

		if err := cmd.Start(); err != nil {
			doneCh <- ComposeResult{ExitCode: -1, Error: fmt.Sprintf("start docker compose: %s", err.Error())}
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)

		streamLines := func(reader io.Reader, stream string) {
			defer wg.Done()
			scanner := bufio.NewScanner(reader)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				outCh <- ComposeOutputLine{Line: scanner.Text(), Stream: stream}
			}
		}

		go streamLines(stdout, "stdout")
		go streamLines(stderr, "stderr")

		wg.Wait()

		waitErr := cmd.Wait()
		exitCode := 0
		var errMsg string

		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
				errMsg = waitErr.Error()
			}
		}

		doneCh <- ComposeResult{ExitCode: exitCode, Error: errMsg}
	}()

	return outCh, doneCh
}

// Up runs `docker compose up -d`.
func (r *ComposeRunner) Up(ctx context.Context, projectName, workingDir string, files []string) (<-chan ComposeOutputLine, <-chan ComposeResult) {
	return r.RunCompose(ctx, projectName, workingDir, files, "up", "-d", "--remove-orphans")
}

// Down runs `docker compose down`.
func (r *ComposeRunner) Down(ctx context.Context, projectName, workingDir string, files []string) (<-chan ComposeOutputLine, <-chan ComposeResult) {
	return r.RunCompose(ctx, projectName, workingDir, files, "down", "--remove-orphans")
}

// Pull runs `docker compose pull`.
func (r *ComposeRunner) Pull(ctx context.Context, projectName, workingDir string, files []string) (<-chan ComposeOutputLine, <-chan ComposeResult) {
	return r.RunCompose(ctx, projectName, workingDir, files, "pull")
}
