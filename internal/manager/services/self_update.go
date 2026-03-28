package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/events"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Release represents a GitHub release with relevant metadata.
type Release struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	Assets      []ReleaseAsset `json:"assets"`
}

// ReleaseAsset represents a downloadable binary from a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// SelfUpdateService handles checking for and applying updates.
type SelfUpdateService struct {
	currentVersion string
	githubOwner    string
	githubRepo     string
	httpClient     *http.Client
	bus            *events.Bus
	registry       *channel.Registry
	app            core.App

	mu          sync.RWMutex
	cachedCheck *Release
	cachedAt    time.Time
	etag        string
}

// NewSelfUpdateService creates a new SelfUpdateService.
func NewSelfUpdateService(app core.App, bus *events.Bus, registry *channel.Registry, currentVersion string) *SelfUpdateService {
	return &SelfUpdateService{
		currentVersion: currentVersion,
		githubOwner:    "onflux-tech",
		githubRepo:     "dockeeper",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		bus:      bus,
		registry: registry,
		app:      app,
	}
}

// CheckForUpdate queries GitHub Releases API for the latest version.
func (s *SelfUpdateService) CheckForUpdate() (*Release, bool, error) {
	// Return cached result if still valid (5 min cache)
	s.mu.RLock()
	if s.cachedCheck != nil && time.Since(s.cachedAt) < 5*time.Minute {
		release := s.cachedCheck
		s.mu.RUnlock()
		isNewer := isNewerVersion(s.currentVersion, release.TagName)
		return release, isNewer, nil
	}
	s.mu.RUnlock()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", s.githubOwner, s.githubRepo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Dockeeper/"+s.currentVersion)

	s.mu.RLock()
	if s.etag != "" {
		req.Header.Set("If-None-Match", s.etag)
	}
	s.mu.RUnlock()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	// 304 Not Modified — return cached
	if resp.StatusCode == http.StatusNotModified {
		s.mu.RLock()
		release := s.cachedCheck
		s.mu.RUnlock()
		if release != nil {
			return release, isNewerVersion(s.currentVersion, release.TagName), nil
		}
		return nil, false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, fmt.Errorf("decode response: %w", err)
	}

	// Cache result
	s.mu.Lock()
	s.cachedCheck = &release
	s.cachedAt = time.Now()
	s.etag = resp.Header.Get("ETag")
	s.mu.Unlock()

	isNewer := isNewerVersion(s.currentVersion, release.TagName)
	if isNewer && s.bus != nil {
		s.bus.Publish(events.Event{
			Type:     events.SystemUpdateAvailable,
			Severity: events.SeverityInfo,
			Details: map[string]any{
				"current_version": s.currentVersion,
				"latest_version":  release.TagName,
			},
			Timestamp: time.Now(),
		})
	}

	return &release, isNewer, nil
}

// ExecuteManagerUpdate downloads and replaces the manager binary.
func (s *SelfUpdateService) ExecuteManagerUpdate(release *Release) error {
	// Detect Docker environment
	if isDocker() {
		return fmt.Errorf("self-update not supported in Docker containers — update the container image instead")
	}

	if runtime.GOOS == "windows" {
		return fmt.Errorf("self-update not supported on Windows")
	}

	s.bus.Publish(events.Event{
		Type:     events.SystemUpdateStarted,
		Severity: events.SeverityInfo,
		Details:  map[string]any{"version": release.TagName},
		Timestamp: time.Now(),
	})

	// Find the correct asset
	asset := s.findAsset(release, "manager", runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		return fmt.Errorf("no binary asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download binary
	slog.Info("downloading update", "url", asset.BrowserDownloadURL, "size", asset.Size)
	resp, err := s.httpClient.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Write to temp file + compute checksum
	tmpFile, err := os.CreateTemp("", "dockeeper-manager-update-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("download write: %w", err)
	}
	tmpFile.Close()

	checksum := hex.EncodeToString(hasher.Sum(nil))
	slog.Info("download complete", "checksum", checksum)

	// Find and verify checksum from assets
	checksumAsset := s.findChecksumAsset(release)
	if checksumAsset != nil {
		if err := s.verifyChecksum(checksumAsset, asset.Name, checksum); err != nil {
			return err
		}
	}

	// Atomic binary replacement
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	currentBinary, _ = filepath.EvalSymlinks(currentBinary)

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	backupPath := currentBinary + ".backup"
	_ = os.Remove(backupPath)

	if err := os.Rename(currentBinary, backupPath); err != nil {
		return fmt.Errorf("backup current: %w", err)
	}

	if err := os.Rename(tmpPath, currentBinary); err != nil {
		// Restore backup
		_ = os.Rename(backupPath, currentBinary)
		return fmt.Errorf("install new binary: %w", err)
	}

	s.bus.Publish(events.Event{
		Type:     events.SystemUpdateCompleted,
		Severity: events.SeverityInfo,
		Details:  map[string]any{"version": release.TagName},
		Timestamp: time.Now(),
	})

	slog.Info("manager binary replaced, restart required", "version", release.TagName)
	return nil
}

// UpdateAgent sends a new agent binary to an agent via a DataStream.
func (s *SelfUpdateService) UpdateAgent(ctx context.Context, serverID, version string, binaryData []byte, checksum string) error {
	ch, ok := s.registry.Get(serverID)
	if !ok {
		return fmt.Errorf("no active channel for server %s", serverID)
	}

	streamID := channel.NewRequestID()
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   streamID,
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_AGENT_UPDATE,
		Params: &agentpb.OpenDataStreamSignal_UpdateBinary{
			UpdateBinary: &agentpb.DataStreamUpdateBinaryParams{
				Version:  version,
				Checksum: checksum,
			},
		},
	}

	streamCh, err := ch.SignalOpenDataStream(ctx, signal)
	if err != nil {
		return fmt.Errorf("signal agent update: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case stream, ok := <-streamCh:
		if !ok {
			return fmt.Errorf("update stream channel closed")
		}

		// Send binary data in 64KB chunks via the stream.
		const chunkSize = 64 * 1024
		for i := 0; i < len(binaryData); i += chunkSize {
			end := i + chunkSize
			if end > len(binaryData) {
				end = len(binaryData)
			}
			if sendErr := stream.Send(&agentpb.OpenDataStreamResponse{
				StreamId: streamID,
				Payload:  &agentpb.OpenDataStreamResponse_Data{Data: binaryData[i:end]},
			}); sendErr != nil {
				return fmt.Errorf("send binary chunk: %w", sendErr)
			}
		}

		// Wait for completion.
		for {
			msg, recvErr := stream.Recv()
			if recvErr != nil {
				return fmt.Errorf("agent update stream: %w", recvErr)
			}
			if complete := msg.GetComplete(); complete != nil {
				slog.Info("agent binary updated", "server", serverID, "version", version)
				return nil
			}
			if errPayload := msg.GetError(); errPayload != nil {
				return fmt.Errorf("agent update failed: %s", errPayload.GetMessage())
			}
		}
	}
}

// DownloadAsset downloads a release asset and returns its bytes and SHA256 checksum.
func (s *SelfUpdateService) DownloadAsset(release *Release, component, goos, goarch string) ([]byte, string, error) {
	asset := s.findAsset(release, component, goos, goarch)
	if asset == nil {
		return nil, "", fmt.Errorf("no %s asset found for %s/%s", component, goos, goarch)
	}

	// Use a longer timeout for large binary downloads.
	dlClient := &http.Client{Timeout: 5 * time.Minute}

	slog.Info("downloading asset", "name", asset.Name, "size", asset.Size)
	resp, err := dlClient.Get(asset.BrowserDownloadURL)
	if err != nil {
		return nil, "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	hasher := sha256.New()
	reader := io.TeeReader(resp.Body, hasher)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("read body: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Verify checksum if available.
	checksumAsset := s.findChecksumAsset(release)
	if checksumAsset != nil {
		if err := s.verifyChecksum(checksumAsset, asset.Name, checksum); err != nil {
			return nil, "", err
		}
	}

	return data, checksum, nil
}

// CurrentVersion returns the running version.
func (s *SelfUpdateService) CurrentVersion() string {
	return s.currentVersion
}

// PeriodicCheck runs a background update check and publishes events.
func (s *SelfUpdateService) PeriodicCheck() {
	release, isNewer, err := s.CheckForUpdate()
	if err != nil {
		slog.Warn("periodic update check failed", "error", err)
		return
	}
	if isNewer {
		slog.Info("new version available", "current", s.currentVersion, "latest", release.TagName)
	}
}

// findAsset locates the appropriate binary asset for the given OS/arch.
// GoReleaser names archives as: dockeeper-{component}_{version}_{os}_{arch}.tar.gz
// We match by prefix + suffix to skip the version segment in the middle.
func (s *SelfUpdateService) findAsset(release *Release, component, goos, goarch string) *ReleaseAsset {
	prefix := fmt.Sprintf("dockeeper-%s_", component)
	suffix := fmt.Sprintf("_%s_%s", goos, goarch)

	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".sha256") {
			continue
		}
		if strings.HasPrefix(name, prefix) && strings.Contains(name, suffix) {
			a := asset
			return &a
		}
	}
	return nil
}

// findChecksumAsset finds the checksums file from release assets.
func (s *SelfUpdateService) findChecksumAsset(release *Release) *ReleaseAsset {
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), "checksums.txt") || strings.HasSuffix(strings.ToLower(asset.Name), ".sha256") {
			a := asset
			return &a
		}
	}
	return nil
}

// verifyChecksum downloads and verifies the checksum file.
func (s *SelfUpdateService) verifyChecksum(checksumAsset *ReleaseAsset, binaryName, actualChecksum string) error {
	resp, err := s.httpClient.Get(checksumAsset.BrowserDownloadURL)
	if err != nil {
		slog.Warn("could not download checksum file, skipping verification", "error", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil // skip verification on error
	}

	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && strings.Contains(parts[1], binaryName) {
			if parts[0] != actualChecksum {
				return fmt.Errorf("checksum mismatch: expected %s, got %s", parts[0], actualChecksum)
			}
			slog.Info("checksum verified", "file", binaryName)
			return nil
		}
	}

	return nil // no matching entry found, skip verification
}

// isDocker detects if running inside a Docker container.
func isDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// isNewerVersion compares semver strings (vX.Y.Z format).
func isNewerVersion(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	if current == "" || latest == "" || current == "dev" {
		return false
	}

	cParts := strings.Split(current, ".")
	lParts := strings.Split(latest, ".")

	for i := 0; i < 3; i++ {
		var c, l int
		if i < len(cParts) {
			fmt.Sscanf(cParts[i], "%d", &c)
		}
		if i < len(lParts) {
			fmt.Sscanf(lParts[i], "%d", &l)
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}
	return false
}
