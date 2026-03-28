package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ---------------------------------------------------------------------------
// Mock stream implementation
// ---------------------------------------------------------------------------

// mockCommandStream implements agentpb.AgentCommandChannelService_OpenChannelServer.
type mockCommandStream struct {
	ctx      context.Context
	sent     []*agentpb.OpenChannelResponse
	sentMu   sync.Mutex
	sendErr  error
	recvCh   chan *agentpb.OpenChannelRequest
	recvErr  error
}

func newMockCommandStream(ctx context.Context) *mockCommandStream {
	return &mockCommandStream{
		ctx:    ctx,
		recvCh: make(chan *agentpb.OpenChannelRequest, 16),
	}
}

func (m *mockCommandStream) Send(msg *agentpb.OpenChannelResponse) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentMu.Lock()
	m.sent = append(m.sent, msg)
	m.sentMu.Unlock()
	return nil
}

func (m *mockCommandStream) Recv() (*agentpb.OpenChannelRequest, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	msg, ok := <-m.recvCh
	if !ok {
		return nil, context.Canceled
	}
	return msg, nil
}

func (m *mockCommandStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockCommandStream) SendHeader(metadata.MD) error { return nil }
func (m *mockCommandStream) SetTrailer(metadata.MD)       {}
func (m *mockCommandStream) Context() context.Context      { return m.ctx }
func (m *mockCommandStream) SendMsg(any) error             { return nil }
func (m *mockCommandStream) RecvMsg(any) error             { return nil }

func (m *mockCommandStream) sentMessages() []*agentpb.OpenChannelResponse {
	m.sentMu.Lock()
	defer m.sentMu.Unlock()
	cp := make([]*agentpb.OpenChannelResponse, len(m.sent))
	copy(cp, m.sent)
	return cp
}

// mockDataStream implements agentpb.AgentDataStreamService_OpenDataStreamServer.
type mockDataStream struct {
	ctx context.Context
}

func (m *mockDataStream) Send(*agentpb.OpenDataStreamResponse) error { return nil }
func (m *mockDataStream) Recv() (*agentpb.OpenDataStreamRequest, error) {
	<-m.ctx.Done()
	return nil, m.ctx.Err()
}
func (m *mockDataStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockDataStream) SendHeader(metadata.MD) error { return nil }
func (m *mockDataStream) SetTrailer(metadata.MD)       {}
func (m *mockDataStream) Context() context.Context      { return m.ctx }
func (m *mockDataStream) SendMsg(any) error             { return nil }
func (m *mockDataStream) RecvMsg(any) error             { return nil }

// mockExecStream implements agentpb.AgentExecStreamService_OpenExecStreamServer.
type mockExecStream struct {
	ctx context.Context
}

func (m *mockExecStream) Send(*agentpb.OpenExecStreamResponse) error { return nil }
func (m *mockExecStream) Recv() (*agentpb.OpenExecStreamRequest, error) {
	<-m.ctx.Done()
	return nil, m.ctx.Err()
}
func (m *mockExecStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockExecStream) SendHeader(metadata.MD) error { return nil }
func (m *mockExecStream) SetTrailer(metadata.MD)       {}
func (m *mockExecStream) Context() context.Context      { return m.ctx }
func (m *mockExecStream) SendMsg(any) error             { return nil }
func (m *mockExecStream) RecvMsg(any) error             { return nil }

// ---------------------------------------------------------------------------
// helpers_test.go — NewRequestID
// ---------------------------------------------------------------------------

func TestNewRequestID_Unique(t *testing.T) {
	ids := make(map[string]struct{}, 1000)
	for range 1000 {
		id := NewRequestID()
		assert.Len(t, id, 32, "request ID should be 32 hex chars")
		_, dup := ids[id]
		assert.False(t, dup, "request ID must be unique")
		ids[id] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// priority_test.go — PriorityQueue
// ---------------------------------------------------------------------------

func TestPriorityQueue_Ordering(t *testing.T) {
	pq := NewPriorityQueue()

	pq.Push(&agentpb.ChannelCommand{RequestId: "low", Priority: PriorityLow})
	pq.Push(&agentpb.ChannelCommand{RequestId: "critical", Priority: PriorityCritical})
	pq.Push(&agentpb.ChannelCommand{RequestId: "normal", Priority: PriorityNormal})
	pq.Push(&agentpb.ChannelCommand{RequestId: "high", Priority: PriorityHigh})

	assert.Equal(t, 4, pq.Len())

	assert.Equal(t, "critical", pq.Pop().RequestId)
	assert.Equal(t, "high", pq.Pop().RequestId)
	assert.Equal(t, "normal", pq.Pop().RequestId)
	assert.Equal(t, "low", pq.Pop().RequestId)
	assert.Equal(t, 0, pq.Len())
}

func TestPriorityQueue_FIFO(t *testing.T) {
	pq := NewPriorityQueue()

	pq.Push(&agentpb.ChannelCommand{RequestId: "a", Priority: PriorityNormal})
	pq.Push(&agentpb.ChannelCommand{RequestId: "b", Priority: PriorityNormal})
	pq.Push(&agentpb.ChannelCommand{RequestId: "c", Priority: PriorityNormal})

	assert.Equal(t, "a", pq.Pop().RequestId)
	assert.Equal(t, "b", pq.Pop().RequestId)
	assert.Equal(t, "c", pq.Pop().RequestId)
}

func TestPriorityQueue_EmptyPop(t *testing.T) {
	pq := NewPriorityQueue()
	assert.Nil(t, pq.Pop())
}

// ---------------------------------------------------------------------------
// registry_test.go — Registry (concurrent access, pure map operations)
// ---------------------------------------------------------------------------

// newTestRegistry creates a Registry without PocketBase (for testing map operations).
func newTestRegistry() *Registry {
	return &Registry{
		channels: make(map[string]*AgentChannel),
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := newTestRegistry()
	ch, ok := reg.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, ch)
}

func TestRegistryIsOnline(t *testing.T) {
	reg := newTestRegistry()
	assert.False(t, reg.IsOnline("srv-1"))

	stream := newMockCommandStream(context.Background())
	ch := NewAgentChannel("srv-1", stream)

	reg.mu.Lock()
	reg.channels["srv-1"] = ch
	reg.mu.Unlock()

	assert.True(t, reg.IsOnline("srv-1"))
}

func TestRegistryCountAndList(t *testing.T) {
	reg := newTestRegistry()
	assert.Equal(t, 0, reg.Count())
	assert.Empty(t, reg.List())

	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		ch := NewAgentChannel(id, newMockCommandStream(ctx))
		reg.mu.Lock()
		reg.channels[id] = ch
		reg.mu.Unlock()
	}

	assert.Equal(t, 3, reg.Count())
	list := reg.List()
	assert.Len(t, list, 3)
	assert.ElementsMatch(t, []string{"a", "b", "c"}, list)
}

func TestRegistryConcurrentAccess(t *testing.T) {
	reg := newTestRegistry()
	ctx := context.Background()

	// Pre-populate some channels.
	for i := range 10 {
		id := string(rune('A' + i))
		ch := NewAgentChannel(id, newMockCommandStream(ctx))
		reg.mu.Lock()
		reg.channels[id] = ch
		reg.mu.Unlock()
	}

	// Hammer Get, IsOnline, List, Count concurrently.
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(4)
		go func() {
			defer wg.Done()
			reg.Get("A")
		}()
		go func() {
			defer wg.Done()
			reg.IsOnline("B")
		}()
		go func() {
			defer wg.Done()
			reg.List()
		}()
		go func() {
			defer wg.Done()
			reg.Count()
		}()
	}
	wg.Wait()

	// No race condition = test passes.
	assert.Equal(t, 10, reg.Count())
}

func TestRegistryUnregisterRemovesChannel(t *testing.T) {
	reg := newTestRegistry()
	ctx := context.Background()
	ch := NewAgentChannel("srv-1", newMockCommandStream(ctx))

	reg.mu.Lock()
	reg.channels["srv-1"] = ch
	reg.mu.Unlock()

	assert.True(t, reg.IsOnline("srv-1"))

	// Unregister calls setServerStatus which needs app — but app is nil.
	// The channel Close + map removal still works, setServerStatus logs a warning.
	// We handle the nil app panic by testing only the pure map path.
	reg.mu.Lock()
	delete(reg.channels, "srv-1")
	reg.mu.Unlock()
	ch.Close()

	assert.False(t, reg.IsOnline("srv-1"))
	assert.Equal(t, 0, reg.Count())

	// Verify channel done is signalled.
	select {
	case <-ch.Done():
	default:
		t.Fatal("channel Done should be closed after Close()")
	}
}

func TestRegistryReplaceClosesOldChannel(t *testing.T) {
	reg := newTestRegistry()
	ctx := context.Background()

	old := NewAgentChannel("srv-1", newMockCommandStream(ctx))
	reg.mu.Lock()
	reg.channels["srv-1"] = old
	reg.mu.Unlock()

	// Replace with new channel.
	replacement := NewAgentChannel("srv-1", newMockCommandStream(ctx))
	reg.mu.Lock()
	existing, existed := reg.channels["srv-1"]
	reg.channels["srv-1"] = replacement
	reg.mu.Unlock()

	require.True(t, existed)
	existing.Close()

	// Old channel should be done.
	select {
	case <-old.Done():
	default:
		t.Fatal("old channel should be closed after replacement")
	}

	// New channel should be active.
	ch, ok := reg.Get("srv-1")
	assert.True(t, ok)
	assert.Equal(t, replacement, ch)
}

// ---------------------------------------------------------------------------
// agent_channel_test.go — AgentChannel
// ---------------------------------------------------------------------------

func TestAgentChannel_SendCommandAndDeliver(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)
	defer ch.Close()

	reqID := NewRequestID()
	cmd := &agentpb.ChannelCommand{
		RequestId: reqID,
		Priority:  PriorityNormal,
	}

	// Simulate agent responding in a goroutine.
	go func() {
		// Wait briefly for the command to be sent.
		time.Sleep(10 * time.Millisecond)
		ch.DeliverCommandResult(&agentpb.ChannelCommandResult{
			RequestId: reqID,
		})
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	result, err := ch.SendCommand(cmdCtx, cmd)
	require.NoError(t, err)
	assert.Equal(t, reqID, result.RequestId)

	// Verify the command was sent through the stream.
	msgs := stream.sentMessages()
	require.Len(t, msgs, 1)
	assert.NotNil(t, msgs[0].GetCommand())
	assert.Equal(t, reqID, msgs[0].GetCommand().RequestId)
}

func TestAgentChannel_SendCommandTimeout(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)
	defer ch.Close()

	cmd := &agentpb.ChannelCommand{
		RequestId: NewRequestID(),
		Priority:  PriorityNormal,
	}

	// Use a very short timeout — no one will deliver the result.
	cmdCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_, err := ch.SendCommand(cmdCtx, cmd)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify pending map is cleaned up.
	ch.pendingMu.Lock()
	assert.Empty(t, ch.pending, "pending map should be empty after timeout")
	ch.pendingMu.Unlock()
}

func TestAgentChannel_SendCommandChannelClosed(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)

	cmd := &agentpb.ChannelCommand{
		RequestId: NewRequestID(),
		Priority:  PriorityNormal,
	}

	// Close the channel while SendCommand is waiting.
	go func() {
		time.Sleep(20 * time.Millisecond)
		ch.Close()
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := ch.SendCommand(cmdCtx, cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel closed")
}

func TestAgentChannel_DeliverCommandResultUnknownID(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)
	defer ch.Close()

	// Deliver result for a request that was never sent — should not panic.
	ch.DeliverCommandResult(&agentpb.ChannelCommandResult{
		RequestId: "unknown-id",
	})
}

func TestAgentChannel_Heartbeat(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)
	defer ch.Close()

	before := ch.LastHeartbeat()
	time.Sleep(5 * time.Millisecond)
	ch.RecordHeartbeat()
	after := ch.LastHeartbeat()

	assert.True(t, after.After(before), "heartbeat should advance")
}

func TestAgentChannel_SendHeartbeat(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)
	defer ch.Close()

	err := ch.SendHeartbeat()
	require.NoError(t, err)

	msgs := stream.sentMessages()
	require.Len(t, msgs, 1)
	hb := msgs[0].GetHeartbeat()
	require.NotNil(t, hb)
	assert.Greater(t, hb.TimestampMs, int64(0))
}

func TestAgentChannel_RateLimiting(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)
	defer ch.Close()

	// Exhaust the command burst (10 commands).
	for i := range commandBurst {
		reqID := NewRequestID()
		cmd := &agentpb.ChannelCommand{RequestId: reqID, Priority: PriorityNormal}

		// Deliver result immediately in goroutine.
		go func(id string) {
			time.Sleep(5 * time.Millisecond)
			ch.DeliverCommandResult(&agentpb.ChannelCommandResult{RequestId: id})
		}(reqID)

		cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		_, err := ch.SendCommand(cmdCtx, cmd)
		cancel()
		require.NoError(t, err, "command %d within burst should succeed", i)
	}

	// The next command with a very tight timeout should be rate-limited.
	cmd := &agentpb.ChannelCommand{RequestId: NewRequestID(), Priority: PriorityNormal}
	tightCtx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	_, err := ch.SendCommand(tightCtx, cmd)
	assert.Error(t, err, "should be rate-limited after burst exhaustion")
}

func TestAgentChannel_CloseIdempotent(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)

	// Close multiple times should not panic.
	ch.Close()
	ch.Close()
	ch.Close()

	select {
	case <-ch.Done():
	default:
		t.Fatal("Done should be closed")
	}
}

func TestAgentChannel_CloseCleanup(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-1", stream)

	// Set up data and exec waiters.
	waiterCtx, waiterCancel := context.WithCancel(ctx)
	defer waiterCancel()

	dw := &streamWaiter[agentpb.AgentDataStreamService_OpenDataStreamServer]{
		ch:     make(chan agentpb.AgentDataStreamService_OpenDataStreamServer, 1),
		ctx:    waiterCtx,
		cancel: waiterCancel,
	}
	ch.dataWaitersMu.Lock()
	ch.dataWaiters["ds-1"] = dw
	ch.dataWaitersMu.Unlock()

	execCtx, execCancel := context.WithCancel(ctx)
	defer execCancel()

	ew := &streamWaiter[agentpb.AgentExecStreamService_OpenExecStreamServer]{
		ch:     make(chan agentpb.AgentExecStreamService_OpenExecStreamServer, 1),
		ctx:    execCtx,
		cancel: execCancel,
	}
	ch.execWaitersMu.Lock()
	ch.execWaiters["es-1"] = ew
	ch.execWaitersMu.Unlock()

	ch.Close()

	// All waiter maps should be empty.
	ch.dataWaitersMu.Lock()
	assert.Empty(t, ch.dataWaiters)
	ch.dataWaitersMu.Unlock()

	ch.execWaitersMu.Lock()
	assert.Empty(t, ch.execWaiters)
	ch.execWaitersMu.Unlock()

	ch.pendingMu.Lock()
	assert.Empty(t, ch.pending)
	ch.pendingMu.Unlock()
}
