package channel

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Default rate limits per agent.
const (
	commandRateLimit = 30  // commands per minute
	commandBurst     = 10
	dataRateLimit    = 10  // data stream opens per minute
	dataBurst        = 5
	execRateLimit    = 5   // exec stream opens per minute
	execBurst        = 3

	// DefaultCommandTimeout is the max time to wait for a command response.
	DefaultCommandTimeout = 30 * time.Second
	// BackupCommandTimeout is used for long-running operations.
	BackupCommandTimeout = 300 * time.Second
)

// streamWaiter holds a channel that receives a gRPC stream when the agent connects back.
type streamWaiter[T any] struct {
	ch     chan T
	ctx    context.Context
	cancel context.CancelFunc
}

// AgentChannel represents an active command channel to a connected agent.
// It supports sending commands (multiplexed by request_id), opening data/exec
// streams, heartbeat tracking, and per-agent rate limiting.
type AgentChannel struct {
	ServerID string

	// The persistent bidirectional gRPC stream.
	stream agentpb.AgentCommandChannelService_OpenChannelServer

	// Pending command responses keyed by request_id.
	pending   map[string]chan *agentpb.ChannelCommandResult
	pendingMu sync.Mutex

	// Waiters for data/exec streams opened by the agent.
	dataWaiters   map[string]*streamWaiter[agentpb.AgentDataStreamService_OpenDataStreamServer]
	dataWaitersMu sync.Mutex

	execWaiters   map[string]*streamWaiter[agentpb.AgentExecStreamService_OpenExecStreamServer]
	execWaitersMu sync.Mutex

	// Rate limiters.
	cmdLimiter  *rate.Limiter
	dataLimiter *rate.Limiter
	execLimiter *rate.Limiter

	// Priority queue for outbound commands.
	queue   *PriorityQueue
	queueMu sync.Mutex

	// Heartbeat tracking.
	lastHeartbeat   time.Time
	lastHeartbeatMu sync.RWMutex

	// Send serialization (gRPC streams are not concurrent-safe for sends).
	sendMu sync.Mutex

	closeOnce sync.Once
	done      chan struct{}
}

// NewAgentChannel wraps a command channel stream for the given server.
func NewAgentChannel(serverID string, stream agentpb.AgentCommandChannelService_OpenChannelServer) *AgentChannel {
	return &AgentChannel{
		ServerID:      serverID,
		stream:        stream,
		pending:       make(map[string]chan *agentpb.ChannelCommandResult),
		dataWaiters:   make(map[string]*streamWaiter[agentpb.AgentDataStreamService_OpenDataStreamServer]),
		execWaiters:   make(map[string]*streamWaiter[agentpb.AgentExecStreamService_OpenExecStreamServer]),
		cmdLimiter:    rate.NewLimiter(rate.Every(time.Minute/commandRateLimit), commandBurst),
		dataLimiter:   rate.NewLimiter(rate.Every(time.Minute/dataRateLimit), dataBurst),
		execLimiter:   rate.NewLimiter(rate.Every(time.Minute/execRateLimit), execBurst),
		queue:         NewPriorityQueue(),
		lastHeartbeat: time.Now(),
		done:          make(chan struct{}),
	}
}

// SendCommand dispatches a command to the agent and blocks until a response
// arrives or the context is cancelled. Commands are multiplexed by request_id.
func (ch *AgentChannel) SendCommand(ctx context.Context, cmd *agentpb.ChannelCommand) (*agentpb.ChannelCommandResult, error) {
	if err := ch.cmdLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limited: %w", err)
	}

	respCh := make(chan *agentpb.ChannelCommandResult, 1)

	ch.pendingMu.Lock()
	ch.pending[cmd.RequestId] = respCh
	ch.pendingMu.Unlock()

	defer func() {
		ch.pendingMu.Lock()
		delete(ch.pending, cmd.RequestId)
		ch.pendingMu.Unlock()
	}()

	// Send the command via the stream.
	msg := &agentpb.OpenChannelResponse{
		Payload: &agentpb.OpenChannelResponse_Command{Command: cmd},
	}
	if err := ch.send(msg); err != nil {
		return nil, fmt.Errorf("send command: %w", err)
	}

	// Wait for response.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch.done:
		return nil, fmt.Errorf("channel closed")
	case result := <-respCh:
		return result, nil
	}
}

// SignalOpenDataStream tells the agent to open a DataStream for the given operation.
// Returns a channel that will receive the stream when the agent connects back.
func (ch *AgentChannel) SignalOpenDataStream(ctx context.Context, signal *agentpb.OpenDataStreamSignal) (<-chan agentpb.AgentDataStreamService_OpenDataStreamServer, error) {
	if err := ch.dataLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limited: %w", err)
	}

	waiterCtx, cancel := context.WithCancel(ctx)
	w := &streamWaiter[agentpb.AgentDataStreamService_OpenDataStreamServer]{
		ch:     make(chan agentpb.AgentDataStreamService_OpenDataStreamServer, 1),
		ctx:    waiterCtx,
		cancel: cancel,
	}

	ch.dataWaitersMu.Lock()
	ch.dataWaiters[signal.StreamId] = w
	ch.dataWaitersMu.Unlock()

	// Send signal to agent via command channel.
	msg := &agentpb.OpenChannelResponse{
		Payload: &agentpb.OpenChannelResponse_OpenDataStream{OpenDataStream: signal},
	}
	if err := ch.send(msg); err != nil {
		ch.dataWaitersMu.Lock()
		delete(ch.dataWaiters, signal.StreamId)
		ch.dataWaitersMu.Unlock()
		cancel()
		return nil, fmt.Errorf("send data stream signal: %w", err)
	}

	return w.ch, nil
}

// SignalOpenExecStream tells the agent to open an ExecStream.
// Returns a channel that will receive the stream when the agent connects back.
func (ch *AgentChannel) SignalOpenExecStream(ctx context.Context, signal *agentpb.OpenExecStreamSignal) (<-chan agentpb.AgentExecStreamService_OpenExecStreamServer, error) {
	if err := ch.execLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limited: %w", err)
	}

	waiterCtx, cancel := context.WithCancel(ctx)
	w := &streamWaiter[agentpb.AgentExecStreamService_OpenExecStreamServer]{
		ch:     make(chan agentpb.AgentExecStreamService_OpenExecStreamServer, 1),
		ctx:    waiterCtx,
		cancel: cancel,
	}

	ch.execWaitersMu.Lock()
	ch.execWaiters[signal.SessionId] = w
	ch.execWaitersMu.Unlock()

	msg := &agentpb.OpenChannelResponse{
		Payload: &agentpb.OpenChannelResponse_OpenExecStream{OpenExecStream: signal},
	}
	if err := ch.send(msg); err != nil {
		ch.execWaitersMu.Lock()
		delete(ch.execWaiters, signal.SessionId)
		ch.execWaitersMu.Unlock()
		cancel()
		return nil, fmt.Errorf("send exec stream signal: %w", err)
	}

	return w.ch, nil
}

// DeliverCommandResult routes a command result to the pending waiter.
func (ch *AgentChannel) DeliverCommandResult(result *agentpb.ChannelCommandResult) {
	ch.pendingMu.Lock()
	respCh, ok := ch.pending[result.RequestId]
	ch.pendingMu.Unlock()

	if !ok {
		slog.Warn("received result for unknown request_id", "server_id", ch.ServerID, "request_id", result.RequestId)
		return
	}

	select {
	case respCh <- result:
	default:
		slog.Warn("result channel full, dropping", "server_id", ch.ServerID, "request_id", result.RequestId)
	}
}

// DeliverDataStream connects an agent-opened DataStream to a waiting consumer.
func (ch *AgentChannel) DeliverDataStream(streamID string, stream agentpb.AgentDataStreamService_OpenDataStreamServer) bool {
	ch.dataWaitersMu.Lock()
	w, ok := ch.dataWaiters[streamID]
	if ok {
		delete(ch.dataWaiters, streamID)
	}
	ch.dataWaitersMu.Unlock()

	if !ok {
		slog.Warn("received data stream for unknown stream_id", "server_id", ch.ServerID, "stream_id", streamID)
		return false
	}

	select {
	case w.ch <- stream:
		return true
	case <-w.ctx.Done():
		return false
	}
}

// DeliverExecStream connects an agent-opened ExecStream to a waiting consumer.
func (ch *AgentChannel) DeliverExecStream(sessionID string, stream agentpb.AgentExecStreamService_OpenExecStreamServer) bool {
	ch.execWaitersMu.Lock()
	w, ok := ch.execWaiters[sessionID]
	if ok {
		delete(ch.execWaiters, sessionID)
	}
	ch.execWaitersMu.Unlock()

	if !ok {
		slog.Warn("received exec stream for unknown session_id", "server_id", ch.ServerID, "session_id", sessionID)
		return false
	}

	select {
	case w.ch <- stream:
		return true
	case <-w.ctx.Done():
		return false
	}
}

// RecordHeartbeat updates the last heartbeat time.
func (ch *AgentChannel) RecordHeartbeat() {
	ch.lastHeartbeatMu.Lock()
	ch.lastHeartbeat = time.Now()
	ch.lastHeartbeatMu.Unlock()
}

// LastHeartbeat returns the time of the last heartbeat.
func (ch *AgentChannel) LastHeartbeat() time.Time {
	ch.lastHeartbeatMu.RLock()
	defer ch.lastHeartbeatMu.RUnlock()
	return ch.lastHeartbeat
}

// SendHeartbeat sends a heartbeat message to the agent.
func (ch *AgentChannel) SendHeartbeat() error {
	msg := &agentpb.OpenChannelResponse{
		Payload: &agentpb.OpenChannelResponse_Heartbeat{
			Heartbeat: &agentpb.ChannelHeartbeat{
				TimestampMs: time.Now().UnixMilli(),
			},
		},
	}
	return ch.send(msg)
}

// Done returns a channel that is closed when the agent channel is closed.
func (ch *AgentChannel) Done() <-chan struct{} {
	return ch.done
}

// Close shuts down the agent channel and cancels all pending waiters.
func (ch *AgentChannel) Close() {
	ch.closeOnce.Do(func() {
		close(ch.done)

		// Cancel all pending command waiters.
		ch.pendingMu.Lock()
		for id := range ch.pending {
			delete(ch.pending, id)
		}
		ch.pendingMu.Unlock()

		// Cancel all data stream waiters.
		ch.dataWaitersMu.Lock()
		for id, w := range ch.dataWaiters {
			w.cancel()
			delete(ch.dataWaiters, id)
		}
		ch.dataWaitersMu.Unlock()

		// Cancel all exec stream waiters.
		ch.execWaitersMu.Lock()
		for id, w := range ch.execWaiters {
			w.cancel()
			delete(ch.execWaiters, id)
		}
		ch.execWaitersMu.Unlock()
	})
}

// send serializes sends to the gRPC stream (not concurrent-safe).
func (ch *AgentChannel) send(msg *agentpb.OpenChannelResponse) error {
	ch.sendMu.Lock()
	defer ch.sendMu.Unlock()
	return ch.stream.Send(msg)
}
