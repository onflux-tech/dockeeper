package tunnel

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	grpchandlers "github.com/onflux-tech/dockeeper/internal/agent/grpc_handlers"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const (
	heartbeatInterval = 15 * time.Second

	backoffBase = 1 * time.Second
	backoffMax  = 60 * time.Second
	backoffMul  = 2.0
	jitterFrac  = 0.25
)

// Client manages the reverse-tunnel connection to the manager.
// It opens a persistent CommandChannel and spawns on-demand
// DataStream/ExecStream sessions when signalled.
type Client struct {
	serverID string
	conn     *grpc.ClientConn

	cc agentpb.AgentCommandChannelServiceClient
	ds agentpb.AgentDataStreamServiceClient
	es agentpb.AgentExecStreamServiceClient

	// Handlers for unary command dispatch.
	cmdHandler     *grpchandlers.CommandsHandler
	imgHandler     *grpchandlers.ImageHandler
	composeHandler *grpchandlers.ComposeHandler
	netHandler     *grpchandlers.NetworkHandler
	volHandler     *grpchandlers.VolumeHandler
	healthHandler  *grpchandlers.HealthHandler
	updateHandler  *grpchandlers.UpdateHandler

	// Dependencies for streaming operations.
	exec executor.Executor
	orch collector.OrchestratorClient

	logger *slog.Logger
}

// NewClient creates a tunnel client wired to the existing handlers.
func NewClient(
	serverID string,
	conn *grpc.ClientConn,
	exec executor.Executor,
	orch collector.OrchestratorClient,
	logger *slog.Logger,
	onShutdown func(),
	dataDir string,
) *Client {
	cmds := grpchandlers.NewCommandsHandler(exec, logger)
	cmds.SetShutdownCallback(onShutdown)

	c := &Client{
		serverID: serverID,
		conn:     conn,
		cc:       agentpb.NewAgentCommandChannelServiceClient(conn),
		ds:       agentpb.NewAgentDataStreamServiceClient(conn),
		es:       agentpb.NewAgentExecStreamServiceClient(conn),

		cmdHandler:     cmds,
		imgHandler:     grpchandlers.NewImageHandler(exec, logger),
		composeHandler: grpchandlers.NewComposeHandler(logger),
		netHandler:     grpchandlers.NewNetworkHandler(exec, logger),
		volHandler:     grpchandlers.NewVolumeHandler(exec, logger),
		healthHandler:  grpchandlers.NewHealthHandler(logger),
		updateHandler:  grpchandlers.NewUpdateHandler(exec, logger, dataDir),

		exec:   exec,
		orch:   orch,
		logger: logger,
	}

	return c
}

// Run maintains the persistent CommandChannel with automatic reconnect.
// It blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) error {
	var attempt int
	for {
		err := c.runOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		attempt++
		wait := reconnectBackoff(attempt)
		c.logger.Warn("command channel disconnected, reconnecting",
			"error", err, "attempt", attempt, "retry_in", wait,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// runOnce opens a single CommandChannel session and processes messages until
// the stream ends or context is cancelled.
func (c *Client) runOnce(ctx context.Context) error {
	stream, err := c.cc.OpenChannel(ctx)
	if err != nil {
		return fmt.Errorf("open command channel: %w", err)
	}

	// Send ChannelInit as first message.
	if err := stream.Send(&agentpb.OpenChannelRequest{
		Payload: &agentpb.OpenChannelRequest_Init{
			Init: &agentpb.ChannelInit{ServerId: c.serverID},
		},
	}); err != nil {
		return fmt.Errorf("send channel init: %w", err)
	}

	c.logger.Info("command channel connected", "server_id", c.serverID)

	// Start heartbeat sender.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.heartbeatLoop(ctx, stream)
	}()

	// Receive loop.
	err = c.receiveLoop(ctx, stream)

	// Wait for heartbeat goroutine to finish.
	wg.Wait()

	return err
}

// heartbeatLoop sends periodic heartbeats over the command channel.
func (c *Client) heartbeatLoop(ctx context.Context, stream agentpb.AgentCommandChannelService_OpenChannelClient) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := stream.Send(&agentpb.OpenChannelRequest{
				Payload: &agentpb.OpenChannelRequest_Heartbeat{
					Heartbeat: &agentpb.ChannelHeartbeat{
						TimestampMs: time.Now().UnixMilli(),
					},
				},
			}); err != nil {
				c.logger.Warn("heartbeat send failed", "error", err)
				return
			}
		}
	}
}

// receiveLoop reads messages from the manager and dispatches commands/signals.
func (c *Client) receiveLoop(ctx context.Context, stream agentpb.AgentCommandChannelService_OpenChannelClient) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("recv: %w", err)
		}

		switch payload := msg.Payload.(type) {
		case *agentpb.OpenChannelResponse_Heartbeat:
			// Manager heartbeat — just acknowledge, nothing to do.

		case *agentpb.OpenChannelResponse_Command:
			go c.handleCommand(ctx, stream, payload.Command)

		case *agentpb.OpenChannelResponse_OpenDataStream:
			go c.handleDataStreamSignal(ctx, payload.OpenDataStream)

		case *agentpb.OpenChannelResponse_OpenExecStream:
			go c.handleExecStreamSignal(ctx, payload.OpenExecStream)

		default:
			c.logger.Warn("unknown command channel message type")
		}
	}
}

// handleCommand dispatches a unary command to the appropriate handler and
// sends the result back on the command channel.
func (c *Client) handleCommand(ctx context.Context, stream agentpb.AgentCommandChannelService_OpenChannelClient, cmd *agentpb.ChannelCommand) {
	result := c.dispatchCommand(ctx, cmd)

	if err := stream.Send(&agentpb.OpenChannelRequest{
		Payload: &agentpb.OpenChannelRequest_CommandResult{
			CommandResult: result,
		},
	}); err != nil {
		c.logger.Warn("failed to send command result",
			"request_id", cmd.RequestId, "error", err,
		)
	}
}

// dispatchCommand routes a ChannelCommand to the correct handler method.
func (c *Client) dispatchCommand(ctx context.Context, cmd *agentpb.ChannelCommand) *agentpb.ChannelCommandResult {
	result := &agentpb.ChannelCommandResult{RequestId: cmd.RequestId}

	switch req := cmd.Command.(type) {
	case *agentpb.ChannelCommand_ExecuteCommand:
		resp, err := c.cmdHandler.ExecuteCommand(ctx, req.ExecuteCommand)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_ExecuteCommand{ExecuteCommand: resp}
		}

	case *agentpb.ChannelCommand_RemoveImage:
		resp, err := c.imgHandler.RemoveImage(ctx, req.RemoveImage)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_RemoveImage{RemoveImage: resp}
		}

	case *agentpb.ChannelCommand_PruneImages:
		resp, err := c.imgHandler.PruneImages(ctx, req.PruneImages)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_PruneImages{PruneImages: resp}
		}

	case *agentpb.ChannelCommand_ComposeConfig:
		resp, err := c.composeHandler.ComposeConfig(ctx, req.ComposeConfig)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_ComposeConfig{ComposeConfig: resp}
		}

	case *agentpb.ChannelCommand_CreateNetwork:
		resp, err := c.netHandler.CreateNetwork(ctx, req.CreateNetwork)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_CreateNetwork{CreateNetwork: resp}
		}

	case *agentpb.ChannelCommand_RemoveNetwork:
		resp, err := c.netHandler.RemoveNetwork(ctx, req.RemoveNetwork)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_RemoveNetwork{RemoveNetwork: resp}
		}

	case *agentpb.ChannelCommand_ConnectContainer:
		resp, err := c.netHandler.ConnectContainer(ctx, req.ConnectContainer)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_ConnectContainer{ConnectContainer: resp}
		}

	case *agentpb.ChannelCommand_DisconnectContainer:
		resp, err := c.netHandler.DisconnectContainer(ctx, req.DisconnectContainer)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_DisconnectContainer{DisconnectContainer: resp}
		}

	case *agentpb.ChannelCommand_CreateVolume:
		resp, err := c.volHandler.CreateVolume(ctx, req.CreateVolume)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_CreateVolume{CreateVolume: resp}
		}

	case *agentpb.ChannelCommand_RemoveVolume:
		resp, err := c.volHandler.RemoveVolume(ctx, req.RemoveVolume)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_RemoveVolume{RemoveVolume: resp}
		}

	case *agentpb.ChannelCommand_InspectVolume:
		resp, err := c.volHandler.InspectVolume(ctx, req.InspectVolume)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_InspectVolume{InspectVolume: resp}
		}

	case *agentpb.ChannelCommand_HealthCheck:
		resp, err := c.healthHandler.ExecuteHealthCheck(ctx, req.HealthCheck)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_HealthCheck{HealthCheck: resp}
		}

	case *agentpb.ChannelCommand_CheckImageDigest:
		resp, err := c.updateHandler.CheckImageDigest(ctx, req.CheckImageDigest)
		if err != nil {
			result.Result = errorResult(err)
		} else {
			result.Result = &agentpb.ChannelCommandResult_CheckImageDigest{CheckImageDigest: resp}
		}

	default:
		result.Result = &agentpb.ChannelCommandResult_Error{
			Error: &agentpb.ChannelCommandError{
				Code:    400,
				Message: "unsupported command type",
			},
		}
	}

	return result
}

// errorResult wraps a Go error into a ChannelCommandError result.
func errorResult(err error) *agentpb.ChannelCommandResult_Error {
	return &agentpb.ChannelCommandResult_Error{
		Error: &agentpb.ChannelCommandError{
			Code:    500,
			Message: err.Error(),
		},
	}
}

// reconnectBackoff computes a wait duration for the given attempt using
// exponential backoff with jitter.
func reconnectBackoff(attempt int) time.Duration {
	d := float64(backoffBase)
	for range attempt - 1 {
		d *= backoffMul
		if d > float64(backoffMax) {
			d = float64(backoffMax)
			break
		}
	}
	jitter := d * jitterFrac * (2*rand.Float64() - 1)
	d += jitter
	return time.Duration(d)
}
