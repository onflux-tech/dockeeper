package executor

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
)

// ExecSession wraps a Docker exec instance with attached I/O.
type ExecSession struct {
	cli    *dockerclient.Client
	execID string
	conn   net.Conn
	reader io.Reader
	tty    bool
}

// NewExecSession creates a Docker exec instance and attaches to it.
func NewExecSession(
	ctx context.Context,
	cli *dockerclient.Client,
	containerID string,
	cmd []string,
	tty bool,
) (*ExecSession, error) {
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}

	execCfg := container.ExecOptions{
		Cmd:          cmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: !tty, // when tty, stderr is merged into stdout
		Tty:          tty,
	}

	resp, err := cli.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return nil, fmt.Errorf("exec create: %w", err)
	}

	attach, err := cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{
		Tty: tty,
	})
	if err != nil {
		return nil, fmt.Errorf("exec attach: %w", err)
	}

	return &ExecSession{
		cli:    cli,
		execID: resp.ID,
		conn:   attach.Conn,
		reader: attach.Reader,
		tty:    tty,
	}, nil
}

// Reader returns the output reader for the exec session.
// If tty=false, the output is Docker-multiplexed (8-byte header per frame).
// If tty=true, the output is raw bytes (all stdout).
func (s *ExecSession) Reader() io.Reader {
	return s.reader
}

// Write sends data to the exec session's stdin.
func (s *ExecSession) Write(data []byte) (int, error) {
	return s.conn.Write(data)
}

// Resize changes the terminal dimensions of the exec session.
func (s *ExecSession) Resize(ctx context.Context, rows, cols uint32) error {
	return s.cli.ContainerExecResize(ctx, s.execID, container.ResizeOptions{
		Height: uint(rows),
		Width:  uint(cols),
	})
}

// IsTTY returns whether the session was started with TTY mode.
func (s *ExecSession) IsTTY() bool {
	return s.tty
}

// ExitCode inspects the exec instance and returns the exit code.
// Returns -1 if the process is still running.
func (s *ExecSession) ExitCode(ctx context.Context) (int, error) {
	inspect, err := s.cli.ContainerExecInspect(ctx, s.execID)
	if err != nil {
		return -1, fmt.Errorf("exec inspect: %w", err)
	}
	if inspect.Running {
		return -1, nil
	}
	return inspect.ExitCode, nil
}

// Close terminates the exec session by closing the hijacked connection.
func (s *ExecSession) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
