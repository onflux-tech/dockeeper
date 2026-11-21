package channel

import (
	"container/heap"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Priority levels for command dispatch.
const (
	PriorityCritical = 0 // Exec sessions
	PriorityHigh     = 1 // Container operations
	PriorityNormal   = 2 // Data operations
	PriorityLow      = 3 // Health checks
)

type queueItem struct {
	cmd   *agentpb.ChannelCommand
	index int
	seqNo uint64 // insertion order for FIFO within same priority
}

type priorityHeap []*queueItem

func (h priorityHeap) Len() int { return len(h) }

func (h priorityHeap) Less(i, j int) bool {
	if h[i].cmd.Priority != h[j].cmd.Priority {
		return h[i].cmd.Priority < h[j].cmd.Priority
	}
	return h[i].seqNo < h[j].seqNo // FIFO within same priority
}

func (h priorityHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *priorityHeap) Push(x any) {
	item := x.(*queueItem)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *priorityHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}

// PriorityQueue orders commands by priority (lower value = higher priority)
// with FIFO ordering for same-priority items.
type PriorityQueue struct {
	h     priorityHeap
	seqNo uint64
}

// NewPriorityQueue creates an empty priority queue.
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{}
	heap.Init(&pq.h)
	return pq
}

// Push adds a command to the queue.
func (pq *PriorityQueue) Push(cmd *agentpb.ChannelCommand) {
	pq.seqNo++
	heap.Push(&pq.h, &queueItem{cmd: cmd, seqNo: pq.seqNo})
}

// Pop removes and returns the highest-priority command.
func (pq *PriorityQueue) Pop() *agentpb.ChannelCommand {
	if pq.h.Len() == 0 {
		return nil
	}
	item := heap.Pop(&pq.h).(*queueItem)
	return item.cmd
}

// Len returns the number of queued commands.
func (pq *PriorityQueue) Len() int {
	return pq.h.Len()
}
