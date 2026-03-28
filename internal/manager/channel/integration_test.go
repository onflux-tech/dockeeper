package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ---------------------------------------------------------------------------
// Task 6.3: Integration — command channel flow
// Simulates: agent connects → heartbeat exchange → manager sends command →
// agent responds → result delivered.
// ---------------------------------------------------------------------------

func TestCommandChannelFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-integration", stream)
	defer ch.Close()

	// Register in a test registry.
	reg := newTestRegistry()
	reg.mu.Lock()
	reg.channels["srv-integration"] = ch
	reg.mu.Unlock()

	// 1. Simulate agent heartbeat.
	beforeHB := ch.LastHeartbeat()
	time.Sleep(2 * time.Millisecond)
	ch.RecordHeartbeat()
	assert.True(t, ch.LastHeartbeat().After(beforeHB))

	// 2. Manager sends heartbeat to agent.
	require.NoError(t, ch.SendHeartbeat())

	// 3. Manager sends command, agent responds concurrently.
	reqID := NewRequestID()
	cmd := &agentpb.ChannelCommand{
		RequestId: reqID,
		Priority:  PriorityHigh,
		Command: &agentpb.ChannelCommand_ExecuteCommand{
			ExecuteCommand: &agentpb.ExecuteCommandRequest{
				ContainerId: "ctr-1",
				Type:        agentpb.CommandType_COMMAND_TYPE_START,
			},
		},
	}

	// Agent goroutine: waits for command on the mock stream, delivers result.
	go func() {
		// Give SendCommand time to Send the message.
		time.Sleep(15 * time.Millisecond)

		// Verify command was sent through stream.
		msgs := stream.sentMessages()
		for _, m := range msgs {
			if c := m.GetCommand(); c != nil && c.RequestId == reqID {
				// Agent processes and responds.
				ch.DeliverCommandResult(&agentpb.ChannelCommandResult{
					RequestId: reqID,
					Result: &agentpb.ChannelCommandResult_ExecuteCommand{
						ExecuteCommand: &agentpb.ExecuteCommandResponse{
							Success: true,
						},
					},
				})
				return
			}
		}
	}()

	cmdCtx, cmdCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cmdCancel()

	result, err := ch.SendCommand(cmdCtx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, reqID, result.RequestId)
	assert.True(t, result.GetExecuteCommand().Success)

	// 4. Verify agent is online in registry.
	assert.True(t, reg.IsOnline("srv-integration"))

	// 5. Close channel — simulates agent disconnect.
	ch.Close()
	select {
	case <-ch.Done():
	case <-time.After(time.Second):
		t.Fatal("channel should be done after Close")
	}
}

func TestCommandChannelConcurrentCommands(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-concurrent", stream)
	defer ch.Close()

	// Send 10 commands concurrently, each with a unique response.
	const n = 10
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			reqID := NewRequestID()
			cmd := &agentpb.ChannelCommand{
				RequestId: reqID,
				Priority:  PriorityNormal,
			}

			// Deliver result after a brief delay.
			go func() {
				time.Sleep(10 * time.Millisecond)
				ch.DeliverCommandResult(&agentpb.ChannelCommandResult{
					RequestId: reqID,
				})
			}()

			cmdCtx, cmdCancel := context.WithTimeout(ctx, 2*time.Second)
			defer cmdCancel()

			result, err := ch.SendCommand(cmdCtx, cmd)
			assert.NoError(t, err, "command %d should succeed", idx)
			if result != nil {
				assert.Equal(t, reqID, result.RequestId)
			}
		}(i)
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Task 6.4: Integration — DataStream flow
// Simulates: manager signals agent to open DataStream → agent opens stream →
// consumer receives it → data flows → stream completes.
// ---------------------------------------------------------------------------

func TestDataStreamFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-ds", stream)
	defer ch.Close()

	streamID := NewRequestID()
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   streamID,
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_LOGS,
	}

	// Manager signals agent to open data stream.
	waiterCh, err := ch.SignalOpenDataStream(ctx, signal)
	require.NoError(t, err)

	// Verify signal was sent via command channel.
	msgs := stream.sentMessages()
	require.NotEmpty(t, msgs)
	lastMsg := msgs[len(msgs)-1]
	require.NotNil(t, lastMsg.GetOpenDataStream())
	assert.Equal(t, streamID, lastMsg.GetOpenDataStream().StreamId)

	// Agent opens data stream (simulated).
	agentDS := &mockDataStream{ctx: ctx}
	delivered := ch.DeliverDataStream(streamID, agentDS)
	require.True(t, delivered, "data stream should be delivered to waiter")

	// Consumer receives the stream.
	select {
	case receivedStream := <-waiterCh:
		assert.NotNil(t, receivedStream)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for data stream delivery")
	}
}

func TestDataStreamNoWaiter(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-ds-no", stream)
	defer ch.Close()

	// Deliver a data stream with no matching waiter.
	ds := &mockDataStream{ctx: ctx}
	ok := ch.DeliverDataStream("nonexistent-id", ds)
	assert.False(t, ok, "should return false with no waiter")
}

func TestDataStreamWaiterCancelled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-ds-cancel", stream)
	defer ch.Close()

	// Create a signal with a cancellable context.
	signalCtx, signalCancel := context.WithCancel(ctx)
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   "ds-cancel",
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_LOGS,
	}

	waiterCh, err := ch.SignalOpenDataStream(signalCtx, signal)
	require.NoError(t, err)

	// Cancel the signal context before agent delivers.
	signalCancel()
	time.Sleep(10 * time.Millisecond)

	ds := &mockDataStream{ctx: ctx}
	_ = ch.DeliverDataStream("ds-cancel", ds)

	// The key observable: when the context is cancelled, the consumer
	// should either receive nothing or receive the stream (Go select is
	// non-deterministic when multiple cases are ready). Either way the
	// waiter is cleaned up from the map.
	ch.dataWaitersMu.Lock()
	_, exists := ch.dataWaiters["ds-cancel"]
	ch.dataWaitersMu.Unlock()
	assert.False(t, exists, "waiter should be removed from map after delivery attempt")

	// Drain the channel to avoid goroutine leaks.
	select {
	case <-waiterCh:
	default:
	}
}

// ---------------------------------------------------------------------------
// Task 6.5: Integration — ExecStream detach/reattach on disconnect
// Simulates: manager opens exec → agent delivers → channel closes →
// new channel created → new exec signal → new agent delivers.
// ---------------------------------------------------------------------------

func TestExecStreamDetachReattach(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// --- Phase 1: First connection ---
	stream1 := newMockCommandStream(ctx)
	ch1 := NewAgentChannel("srv-exec", stream1)

	signal1 := &agentpb.OpenExecStreamSignal{
		SessionId:   "sess-1",
		ContainerId: "ctr-1",
		Command:     []string{"/bin/sh"},
		Tty:         true,
	}

	waiterCh1, err := ch1.SignalOpenExecStream(ctx, signal1)
	require.NoError(t, err)

	// Agent delivers exec stream.
	execStream1 := &mockExecStream{ctx: ctx}
	ok := ch1.DeliverExecStream("sess-1", execStream1)
	require.True(t, ok)

	select {
	case s := <-waiterCh1:
		assert.NotNil(t, s)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for exec stream 1")
	}

	// --- Simulate disconnect ---
	ch1.Close()
	select {
	case <-ch1.Done():
	default:
		t.Fatal("channel 1 should be done")
	}

	// --- Phase 2: Reconnection with new channel ---
	stream2 := newMockCommandStream(ctx)
	ch2 := NewAgentChannel("srv-exec", stream2)
	defer ch2.Close()

	// New exec signal on the new channel.
	signal2 := &agentpb.OpenExecStreamSignal{
		SessionId:   "sess-2",
		ContainerId: "ctr-1",
		Command:     []string{"/bin/sh"},
		Tty:         true,
	}

	waiterCh2, err := ch2.SignalOpenExecStream(ctx, signal2)
	require.NoError(t, err)

	// Agent delivers exec stream on new channel.
	execStream2 := &mockExecStream{ctx: ctx}
	ok = ch2.DeliverExecStream("sess-2", execStream2)
	require.True(t, ok)

	select {
	case s := <-waiterCh2:
		assert.NotNil(t, s)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for exec stream 2 on reconnected channel")
	}

	// Verify old session_id on new channel returns false.
	ok = ch2.DeliverExecStream("sess-1", &mockExecStream{ctx: ctx})
	assert.False(t, ok, "old session should not have waiter on new channel")
}

func TestExecStreamCloseWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-exec-close", stream)

	signal := &agentpb.OpenExecStreamSignal{
		SessionId:   "sess-close",
		ContainerId: "ctr-1",
		Command:     []string{"/bin/sh"},
	}

	_, err := ch.SignalOpenExecStream(ctx, signal)
	require.NoError(t, err)

	// Close channel while exec waiter is pending.
	ch.Close()

	// Waiter map should be cleaned up.
	ch.execWaitersMu.Lock()
	assert.Empty(t, ch.execWaiters, "exec waiters should be cleaned up on close")
	ch.execWaitersMu.Unlock()
}

func TestExecStreamNoWaiter(t *testing.T) {
	ctx := context.Background()
	stream := newMockCommandStream(ctx)
	ch := NewAgentChannel("srv-exec-no", stream)
	defer ch.Close()

	es := &mockExecStream{ctx: ctx}
	ok := ch.DeliverExecStream("nonexistent-session", es)
	assert.False(t, ok, "should return false with no waiter")
}
