package collector

import (
	"context"
	"io"
	"log/slog"

	"google.golang.org/grpc"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeOrchestrator struct {
	OrchestratorClient
	listContainers int
	listSummaries  int
	listImages     int
	listNetworks   int
	listVolumes    int
	listStats      int
	containers     []*agentpb.ContainerInfo
	summaries      []*agentpb.ContainerInfo
	inspectCalls   []string
	inspectResult  *agentpb.ContainerInfo
	inspectErr     error
	stats          []ContainerMetrics
	statsErr       error
	onStats        func()
}

func (f *fakeOrchestrator) ListContainers(context.Context) ([]*agentpb.ContainerInfo, error) {
	f.listContainers++
	return f.containers, nil
}

func (f *fakeOrchestrator) ListContainerSummaries(context.Context) ([]*agentpb.ContainerInfo, error) {
	f.listSummaries++
	return f.summaries, nil
}

func (f *fakeOrchestrator) InspectContainer(_ context.Context, containerID string) (*agentpb.ContainerInfo, error) {
	f.inspectCalls = append(f.inspectCalls, containerID)
	if f.inspectErr != nil {
		return nil, f.inspectErr
	}
	return f.inspectResult, nil
}

func (f *fakeOrchestrator) ListImages(context.Context) ([]*agentpb.ImageInfo, error) {
	f.listImages++
	return nil, nil
}

func (f *fakeOrchestrator) ListNetworks(context.Context) ([]*agentpb.NetworkInfo, error) {
	f.listNetworks++
	return nil, nil
}

func (f *fakeOrchestrator) ListVolumes(context.Context) ([]*agentpb.VolumeInfo, error) {
	f.listVolumes++
	return nil, nil
}

func (f *fakeOrchestrator) ListContainerStats(context.Context) ([]ContainerMetrics, error) {
	f.listStats++
	if f.onStats != nil {
		f.onStats()
	}
	return f.stats, f.statsErr
}

type fakeContainerMetricsStream struct {
	grpc.ClientStream
	sent     []*agentpb.StreamContainerMetricsRequest
	sendErr  error
	closeErr error
}

func (s *fakeContainerMetricsStream) Send(req *agentpb.StreamContainerMetricsRequest) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent = append(s.sent, req)
	return nil
}

func (s *fakeContainerMetricsStream) CloseAndRecv() (*agentpb.StreamContainerMetricsResponse, error) {
	if s.closeErr != nil {
		return nil, s.closeErr
	}
	return &agentpb.StreamContainerMetricsResponse{ReceivedCount: int32(len(s.sent))}, nil
}

type fakeMetricsClient struct {
	agentpb.AgentMetricsServiceClient
	stream  *fakeContainerMetricsStream
	openErr error
	opens   int
}

func (f *fakeMetricsClient) StreamContainerMetrics(context.Context, ...grpc.CallOption) (grpc.ClientStreamingClient[agentpb.StreamContainerMetricsRequest, agentpb.StreamContainerMetricsResponse], error) {
	f.opens++
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f.stream, nil
}

type fakeDiscoveryClient struct {
	agentpb.AgentDiscoveryServiceClient
	containers int
	images     int
	networks   int
	volumes    int
	err        error
}

func (f *fakeDiscoveryClient) ReportContainers(context.Context, *agentpb.ReportContainersRequest, ...grpc.CallOption) (*agentpb.ReportContainersResponse, error) {
	f.containers++
	if f.err != nil {
		return nil, f.err
	}
	return &agentpb.ReportContainersResponse{}, nil
}

func (f *fakeDiscoveryClient) ReportImages(context.Context, *agentpb.ReportImagesRequest, ...grpc.CallOption) (*agentpb.ReportImagesResponse, error) {
	f.images++
	return &agentpb.ReportImagesResponse{}, nil
}

func (f *fakeDiscoveryClient) ReportNetworks(context.Context, *agentpb.ReportNetworksRequest, ...grpc.CallOption) (*agentpb.ReportNetworksResponse, error) {
	f.networks++
	return &agentpb.ReportNetworksResponse{}, nil
}

func (f *fakeDiscoveryClient) ReportVolumes(context.Context, *agentpb.ReportVolumesRequest, ...grpc.CallOption) (*agentpb.ReportVolumesResponse, error) {
	f.volumes++
	return &agentpb.ReportVolumesResponse{}, nil
}

type fakeTraefikClient struct {
	agentpb.AgentTraefikServiceClient
	reports int
	err     error
}

func (f *fakeTraefikClient) ReportTraefikRoutes(context.Context, *agentpb.ReportTraefikRoutesRequest, ...grpc.CallOption) (*agentpb.ReportTraefikRoutesResponse, error) {
	f.reports++
	if f.err != nil {
		return nil, f.err
	}
	return &agentpb.ReportTraefikRoutesResponse{}, nil
}
