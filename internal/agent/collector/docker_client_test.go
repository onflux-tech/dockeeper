package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type statsServer struct {
	mu        sync.Mutex
	ids       []string
	failID    string
	block     chan struct{}
	calls     map[string]int
	oneShot   atomic.Int64
	active    atomic.Int64
	maxActive atomic.Int64
	server    *httptest.Server
}

func newStatsServer(t *testing.T, ids []string) *statsServer {
	t.Helper()

	s := &statsServer{ids: ids, calls: make(map[string]int)}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			s.mu.Lock()
			summaries := make([]container.Summary, 0, len(s.ids))
			for _, id := range s.ids {
				summaries = append(summaries, container.Summary{
					ID:    id,
					Names: []string{"/" + id[:12]},
					Image: "img",
					State: "running",
				})
			}
			s.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(summaries)
		case strings.HasSuffix(r.URL.Path, "/stats"):
			if r.URL.Query().Get("one-shot") == "1" {
				s.oneShot.Add(1)
			}
			id := path.Base(path.Dir(r.URL.Path))

			s.mu.Lock()
			fail := id == s.failID
			block := s.block
			s.mu.Unlock()
			if fail {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}

			active := s.active.Add(1)
			defer s.active.Add(-1)
			for {
				max := s.maxActive.Load()
				if active <= max || s.maxActive.CompareAndSwap(max, active) {
					break
				}
			}
			if block != nil {
				select {
				case <-r.Context().Done():
					return
				case <-block:
				}
			}
			time.Sleep(10 * time.Millisecond)

			s.mu.Lock()
			s.calls[id]++
			n := s.calls[id]
			s.mu.Unlock()

			_, _ = fmt.Fprintf(w, `{"cpu_stats":{"cpu_usage":{"total_usage":%d},"system_cpu_usage":%d,"online_cpus":1},`+
				`"memory_stats":{"usage":512,"limit":1024},"pids_stats":{"current":3},`+
				`"networks":{"eth0":{"rx_bytes":10,"tx_bytes":20}},`+
				`"blkio_stats":{"io_service_bytes_recursive":[{"op":"read","value":100},{"op":"write","value":200}]}}`,
				100*n, 1000+200*(n-1))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *statsServer) setIDs(ids []string) {
	s.mu.Lock()
	s.ids = ids
	s.mu.Unlock()
}

func (s *statsServer) setFail(id string) {
	s.mu.Lock()
	s.failID = id
	s.mu.Unlock()
}

func newTestDockerClient(t *testing.T, srv *statsServer) *DockerClient {
	t.Helper()

	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.WithHost(srv.server.URL),
		dockerclient.WithVersion("1.47"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	return &DockerClient{cli: cli, lastCPU: make(map[string]container.CPUStats)}
}

func testContainerIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("%012x%052x", (i*7+3)%n, i)
	}
	return ids
}

func TestListContainerStatsUsesOneShotWithBoundedConcurrency(t *testing.T) {
	ids := testContainerIDs(16)
	srv := newStatsServer(t, ids)
	d := newTestDockerClient(t, srv)

	first, err := d.ListContainerStats(context.Background())
	require.NoError(t, err)
	require.Len(t, first, len(ids))

	assert.Equal(t, int64(len(ids)), srv.oneShot.Load())
	assert.LessOrEqual(t, srv.maxActive.Load(), int64(8))
	assert.GreaterOrEqual(t, srv.maxActive.Load(), int64(2))

	want := make([]string, len(ids))
	for i, id := range ids {
		want[i] = id[:12]
	}
	slices.Sort(want)
	got := make([]string, len(first))
	for i, m := range first {
		got[i] = m.ContainerID
	}
	assert.Equal(t, want, got)

	for _, m := range first {
		assert.Zero(t, m.CPUPercent)
		assert.Equal(t, uint64(512), m.MemUsed)
		assert.Equal(t, uint64(1024), m.MemLimit)
		assert.Equal(t, uint32(3), m.PIDs)
		assert.Equal(t, uint64(10), m.NetRx)
		assert.Equal(t, uint64(20), m.NetTx)
		assert.Equal(t, uint64(100), m.BlockRead)
		assert.Equal(t, uint64(200), m.BlockWrite)
	}

	second, err := d.ListContainerStats(context.Background())
	require.NoError(t, err)
	require.Len(t, second, len(ids))

	assert.Equal(t, int64(2*len(ids)), srv.oneShot.Load())
	for _, m := range second {
		assert.InDelta(t, 50.0, m.CPUPercent, 0.001)
	}
}

func TestListContainerStatsPrunesStaleCPUHistory(t *testing.T) {
	ids := testContainerIDs(3)
	srv := newStatsServer(t, ids)
	d := newTestDockerClient(t, srv)

	_, err := d.ListContainerStats(context.Background())
	require.NoError(t, err)
	require.Len(t, d.lastCPU, len(ids))

	srv.setIDs(ids[:1])
	_, err = d.ListContainerStats(context.Background())
	require.NoError(t, err)
	assert.Len(t, d.lastCPU, 1)
}

func TestListContainerStatsSkipsFailedContainers(t *testing.T) {
	ids := testContainerIDs(4)
	srv := newStatsServer(t, ids)
	d := newTestDockerClient(t, srv)

	_, err := d.ListContainerStats(context.Background())
	require.NoError(t, err)
	require.Len(t, d.lastCPU, len(ids))

	failed := ids[1]
	srv.setFail(failed)

	got, err := d.ListContainerStats(context.Background())
	require.NoError(t, err)
	require.Len(t, got, len(ids)-1)

	gotIDs := make([]string, len(got))
	for i, m := range got {
		gotIDs[i] = m.ContainerID
	}
	assert.NotContains(t, gotIDs, failed[:12])
	assert.Contains(t, d.lastCPU, failed)
}

func TestListContainerStatsReturnsOnCancel(t *testing.T) {
	ids := testContainerIDs(16)
	srv := newStatsServer(t, ids)
	release := make(chan struct{})
	srv.mu.Lock()
	srv.block = release
	srv.mu.Unlock()
	t.Cleanup(func() { close(release) })

	d := newTestDockerClient(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	var got []ContainerMetrics
	go func() {
		var err error
		got, err = d.ListContainerStats(ctx)
		done <- err
	}()

	deadline := time.Now().Add(2 * time.Second)
	for srv.active.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	require.Greater(t, srv.active.Load(), int64(0))

	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, got)
	case <-time.After(2 * time.Second):
		t.Fatal("ListContainerStats did not return after cancellation")
	}
}
