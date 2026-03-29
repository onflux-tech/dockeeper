package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterAgentPublicRoutes registers public (no auth) endpoints for agent distribution.
// These are authenticated via enrollment token in the query string, not via session.
func RegisterAgentPublicRoutes(se *core.ServeEvent, app core.App) {
	se.Router.GET("/api/dk/agent/download", func(e *core.RequestEvent) error {
		return handleAgentDownload(e)
	})

	se.Router.GET("/api/dk/agent/install/quick", func(e *core.RequestEvent) error {
		return handlePublicQuickInstallScript(e, app)
	})

	se.Router.GET("/api/dk/agent/install/{id}", func(e *core.RequestEvent) error {
		return handlePublicInstallScript(e, app)
	})
}

// RegisterServerRoutes adds server provisioning and management endpoints.
func RegisterServerRoutes(se *core.ServeEvent, app core.App, auditStore *store.AuditStore, cascadeSvc *services.CascadeService, version string, rbac *middleware.RBAC) {
	servers := se.Router.Group("/api/dk/servers")

	servers.GET("", func(e *core.RequestEvent) error {
		return handleListServers(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapServersView))

	servers.POST("/provision", func(e *core.RequestEvent) error {
		return handleProvisionServer(e, app, auditStore)
	}).BindFunc(rbac.RequireCapability(constants.CapServersManage))

	servers.GET("/{id}/install-script", func(e *core.RequestEvent) error {
		return handleInstallScript(e, app, version)
	}).BindFunc(rbac.RequireCapability(constants.CapServersManage))

	servers.PATCH("/{id}", func(e *core.RequestEvent) error {
		return handleUpdateServer(e, app, auditStore)
	}).BindFunc(rbac.RequireCapability(constants.CapServersManage))

	servers.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleDeleteServer(e, cascadeSvc)
	}).BindFunc(rbac.RequireCapability(constants.CapServersManage))

	// Quick install command (returns pre-built curl one-liner with global token).
	servers.GET("/quick-install", func(e *core.RequestEvent) error {
		return handleQuickInstallCommand(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapServersManage))

	// Global enrollment token management.
	se.Router.GET("/api/dk/enrollment-token", func(e *core.RequestEvent) error {
		return handleGetEnrollmentToken(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapServersManage))

	se.Router.POST("/api/dk/enrollment-token/regenerate", func(e *core.RequestEvent) error {
		return handleRegenerateEnrollmentToken(e, app, auditStore)
	}).BindFunc(rbac.RequireCapability(constants.CapServersManage))
}

func handleListServers(e *core.RequestEvent, app core.App) error {
	tag := e.Request.URL.Query().Get("tag")
	filter := ""
	var params []dbx.Params
	if tag != "" {
		filter = "tags ~ {:tag}"
		params = append(params, dbx.Params{"tag": tag})
	}

	var records []*core.Record
	var err error
	if len(params) > 0 {
		records, err = app.FindRecordsByFilter(
			constants.CollServers, filter, "-created", 500, 0, params[0],
		)
	} else {
		records, err = app.FindRecordsByFilter(
			constants.CollServers, "", "-created", 500, 0,
		)
	}
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, records)
}

func handleProvisionServer(e *core.RequestEvent, app core.App, auditStore *store.AuditStore) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := e.BindBody(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	token, err := generateToken(32)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	col, err := app.FindCachedCollectionByNameOrId(constants.CollServers)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	record := core.NewRecord(col)
	record.Set("name", strings.TrimSpace(body.Name))
	record.Set("description", body.Description)
	record.Set("enrollment_token", token)
	record.Set("status", "pending")

	if err := app.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}
	auditStore.Log("server.provision", userID, "server", record.Id, e.RealIP(), map[string]any{
		"name": body.Name,
	})

	return e.JSON(http.StatusCreated, map[string]any{
		"id":               record.Id,
		"name":             body.Name,
		"enrollment_token": token,
		"status":           "pending",
	})
}

func handleUpdateServer(e *core.RequestEvent, app core.App, auditStore *store.AuditStore) error {
	serverID := e.Request.PathValue("id")

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	record, err := app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	}

	record.Set("name", body.Name)
	record.Set("description", body.Description)

	if err := app.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}
	auditStore.Log("server.update", userID, "server", record.Id, e.RealIP(), map[string]any{
		"name":        body.Name,
		"description": body.Description,
	})

	return e.JSON(http.StatusOK, map[string]any{
		"id":          record.Id,
		"name":        body.Name,
		"description": body.Description,
	})
}

// agentBinaryCache prevents concurrent downloads of the same agent binary.
var agentBinaryCache sync.Mutex

func handleAgentDownload(e *core.RequestEvent) error {
	arch := e.Request.URL.Query().Get("arch")
	if arch != "amd64" && arch != "arm64" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "arch must be 'amd64' or 'arm64'"})
	}

	filename := fmt.Sprintf("dockeeper-agent-linux-%s", arch)

	// Docker runtime path, then local dev fallback, then cached download.
	cacheDir := "/tmp/dockeeper-agent-cache"
	paths := []string{
		fmt.Sprintf("/app/agents/%s", filename),
		fmt.Sprintf("./bin/%s", filename),
		filepath.Join(cacheDir, filename),
	}

	var filePath string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err == nil && info.Size() > 1024 {
			filePath = p
			break
		}
	}

	// Fallback: download from GitHub Releases and cache locally.
	if filePath == "" {
		cached, err := downloadAgentFromGitHub(arch, cacheDir, filename)
		if err != nil {
			slog.Error("failed to download agent binary from GitHub", "arch", arch, "error", err)
			return e.JSON(http.StatusNotFound, map[string]string{"error": "agent binary not found for arch " + arch})
		}
		filePath = cached
	}

	e.Response.Header().Set("Content-Type", "application/octet-stream")
	e.Response.Header().Set("Content-Disposition", "attachment; filename=dockeeper-agent")
	http.ServeFile(e.Response, e.Request, filePath)
	return nil
}

// downloadAgentFromGitHub fetches the latest agent binary from GitHub Releases.
func downloadAgentFromGitHub(arch, cacheDir, filename string) (string, error) {
	agentBinaryCache.Lock()
	defer agentBinaryCache.Unlock()

	// Double-check after acquiring lock (another request may have cached it).
	cachedPath := filepath.Join(cacheDir, filename)
	if info, err := os.Stat(cachedPath); err == nil && info.Size() > 1024 {
		return cachedPath, nil
	}

	// Fetch latest release metadata from GitHub.
	client := &http.Client{Timeout: 30 * 1000000000} // 30s
	apiURL := "https://api.github.com/repos/onflux-tech/dockeeper/releases/latest"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Dockeeper-Manager")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}

	// Find the matching agent archive (tar.gz).
	prefix := "dockeeper-agent_"
	suffix := fmt.Sprintf("_linux_%s", arch)
	var downloadURL string
	for _, a := range release.Assets {
		name := strings.ToLower(a.Name)
		if strings.HasPrefix(name, prefix) && strings.Contains(name, suffix) && !strings.HasSuffix(name, ".sha256") {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("no agent asset found for linux/%s", arch)
	}

	// Download the archive.
	slog.Info("downloading agent binary from GitHub", "url", downloadURL, "arch", arch)
	dlResp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", dlResp.StatusCode)
	}

	// Create cache directory.
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	// Write to temp file first, then extract the binary from the tar.gz.
	tmpFile, err := os.CreateTemp(cacheDir, "agent-download-*")
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, dlResp.Body); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("write download: %w", err)
	}
	tmpFile.Close()

	// Extract the binary from the tar.gz archive.
	extractedPath, err := extractAgentBinary(tmpPath, cacheDir, filename)
	if err != nil {
		return "", fmt.Errorf("extract binary: %w", err)
	}

	slog.Info("agent binary cached", "path", extractedPath, "arch", arch)
	return extractedPath, nil
}

// extractAgentBinary extracts the dockeeper-agent binary from a tar.gz archive.
func extractAgentBinary(archivePath, destDir, destFilename string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Use tar command for simplicity (available on Alpine/Linux).
	destPath := filepath.Join(destDir, destFilename)
	tmpExtractDir, err := os.MkdirTemp(destDir, "extract-*")
	if err != nil {
		return "", fmt.Errorf("create extract dir: %w", err)
	}
	defer os.RemoveAll(tmpExtractDir)

	// Extract using command line tar (handles both gzip and tar).
	cmd := fmt.Sprintf("tar -xzf %s -C %s", archivePath, tmpExtractDir)
	if execErr := execCommand(cmd); execErr != nil {
		return "", fmt.Errorf("tar extract: %w", execErr)
	}

	// Find the binary in the extracted directory.
	entries, err := os.ReadDir(tmpExtractDir)
	if err != nil {
		return "", fmt.Errorf("read extract dir: %w", err)
	}

	var binaryPath string
	for _, entry := range entries {
		if entry.Name() == "dockeeper-agent" {
			binaryPath = filepath.Join(tmpExtractDir, entry.Name())
			break
		}
	}
	if binaryPath == "" {
		return "", fmt.Errorf("dockeeper-agent binary not found in archive")
	}

	// Move to final cache location.
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(destPath, data, 0o755); err != nil {
		return "", err
	}

	return destPath, nil
}

// execCommand runs a shell command and returns any error.
func execCommand(cmd string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Stderr = os.Stderr
	return c.Run()
}

func handleInstallScript(e *core.RequestEvent, app core.App, version string) error {
	serverID := e.Request.PathValue("id")

	record, err := app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	}

	token := record.GetString("enrollment_token")
	if token == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server has no enrollment token"})
	}

	mode := e.Request.URL.Query().Get("mode")
	if mode == "" {
		mode = "docker"
	}

	managerHTTPURL := resolveManagerHTTPURL(e)

	if version == "" {
		version = "latest"
	}

	var command string
	switch mode {
	case "docker":
		host := e.Request.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		grpcAddr := host + ":50051"
		command = generateDockerScript(grpcAddr, token, version)
	case "binary":
		command = fmt.Sprintf("curl -sSL '%s/api/dk/agent/install/%s?token=%s' | sudo bash", managerHTTPURL, serverID, token)
	default:
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "mode must be 'docker' or 'binary'"})
	}

	return e.JSON(http.StatusOK, map[string]string{
		"mode":    mode,
		"command": command,
	})
}

// handlePublicInstallScript serves the install script as text/plain, authenticated via token query param.
func handlePublicInstallScript(e *core.RequestEvent, app core.App) error {
	serverID := e.Request.PathValue("id")
	token := e.Request.URL.Query().Get("token")

	if token == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "token is required"})
	}

	record, err := app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	}

	if record.GetString("enrollment_token") != token {
		return e.JSON(http.StatusForbidden, map[string]string{"error": "invalid token"})
	}

	// Derive manager addresses.
	host := e.Request.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	grpcAddr := host + ":50051"
	managerHTTPURL := resolveManagerHTTPURL(e)

	script := generateBinaryScript(grpcAddr, token, managerHTTPURL)

	e.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, writeErr := e.Response.Write([]byte(script))
	return writeErr
}

// resolveManagerHTTPURL builds the public HTTP(S) URL of the manager from the request.
func resolveManagerHTTPURL(e *core.RequestEvent) string {
	scheme := e.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if e.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + e.Request.Host
}

// handlePublicQuickInstallScript serves the install script authenticated via global enrollment token.
func handlePublicQuickInstallScript(e *core.RequestEvent, app core.App) error {
	token := e.Request.URL.Query().Get("token")
	if token == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "token is required"})
	}

	// Validate against the global enrollment token.
	records, err := app.FindRecordsByFilter(
		constants.CollSettings,
		"key = 'enrollment_token'", "", 1, 0,
	)
	if err != nil || len(records) == 0 {
		return e.JSON(http.StatusForbidden, map[string]string{"error": "invalid token"})
	}

	storedToken := strings.Trim(records[0].GetString("value"), "\"")
	if storedToken != token {
		return e.JSON(http.StatusForbidden, map[string]string{"error": "invalid token"})
	}

	// Derive manager addresses.
	host := e.Request.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	grpcAddr := host + ":50051"
	managerHTTPURL := resolveManagerHTTPURL(e)

	script := generateBinaryScript(grpcAddr, token, managerHTTPURL)

	e.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, writeErr := e.Response.Write([]byte(script))
	return writeErr
}

// handleQuickInstallCommand returns the pre-built curl one-liner using the global enrollment token.
func handleQuickInstallCommand(e *core.RequestEvent, app core.App) error {
	// Get or auto-generate the global token.
	records, err := app.FindRecordsByFilter(
		constants.CollSettings,
		"key = 'enrollment_token'", "", 1, 0,
	)

	var token string
	if err != nil || len(records) == 0 {
		// Auto-generate a token.
		generated, genErr := generateToken(32)
		if genErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		}
		col, colErr := app.FindCachedCollectionByNameOrId(constants.CollSettings)
		if colErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": colErr.Error()})
		}
		record := core.NewRecord(col)
		record.Set("key", "enrollment_token")
		record.Set("value", generated)
		if saveErr := app.Save(record); saveErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": saveErr.Error()})
		}
		token = generated
	} else {
		token = strings.Trim(records[0].GetString("value"), "\"")
	}

	managerHTTPURL := resolveManagerHTTPURL(e)
	command := fmt.Sprintf("curl -sSL '%s/api/dk/agent/install/quick?token=%s' | sudo bash", managerHTTPURL, token)

	return e.JSON(http.StatusOK, map[string]string{
		"command": command,
	})
}

func handleDeleteServer(e *core.RequestEvent, cascadeSvc *services.CascadeService) error {
	serverID := e.Request.PathValue("id")

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	_, err := cascadeSvc.ExecuteServerCascade(serverID, userID, e.RealIP())
	if err != nil {
		if strings.Contains(err.Error(), "find server") {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
		}
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// handleGetEnrollmentToken returns the global enrollment token, auto-generating one if none exists.
func handleGetEnrollmentToken(e *core.RequestEvent, app core.App) error {
	records, err := app.FindRecordsByFilter(
		constants.CollSettings,
		"key = 'enrollment_token'",
		"", 1, 0,
	)
	if err != nil || len(records) == 0 {
		// Auto-generate a token.
		token, genErr := generateToken(32)
		if genErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		}
		col, colErr := app.FindCachedCollectionByNameOrId(constants.CollSettings)
		if colErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": colErr.Error()})
		}
		record := core.NewRecord(col)
		record.Set("key", "enrollment_token")
		record.Set("value", token)
		if saveErr := app.Save(record); saveErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": saveErr.Error()})
		}
		return e.JSON(http.StatusOK, map[string]string{"token": token})
	}

	return e.JSON(http.StatusOK, map[string]string{"token": strings.Trim(records[0].GetString("value"), "\"")})
}

// handleRegenerateEnrollmentToken creates a new global enrollment token, replacing the old one.
func handleRegenerateEnrollmentToken(e *core.RequestEvent, app core.App, auditStore *store.AuditStore) error {
	token, err := generateToken(32)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	records, findErr := app.FindRecordsByFilter(
		constants.CollSettings,
		"key = 'enrollment_token'",
		"", 1, 0,
	)
	if findErr != nil || len(records) == 0 {
		// Create new.
		col, colErr := app.FindCachedCollectionByNameOrId(constants.CollSettings)
		if colErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": colErr.Error()})
		}
		record := core.NewRecord(col)
		record.Set("key", "enrollment_token")
		record.Set("value", token)
		if saveErr := app.Save(record); saveErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": saveErr.Error()})
		}
	} else {
		records[0].Set("value", token)
		if saveErr := app.Save(records[0]); saveErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": saveErr.Error()})
		}
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}
	auditStore.Log("enrollment_token.regenerate", userID, "settings", "", e.RealIP(), nil)

	return e.JSON(http.StatusOK, map[string]string{"token": token})
}

func generateDockerScript(grpcAddr, token, version string) string {
	return fmt.Sprintf(`docker run -d \
  --name dockeeper-agent \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v dockeeper-agent-data:/data \
  ghcr.io/onflux-tech/dockeeper-agent:%s \
  --manager-url=%s \
  --token=%s \
  --data-dir=/data`, version, grpcAddr, token)
}

func generateBinaryScript(grpcAddr, token, managerHTTPURL string) string {
	return fmt.Sprintf(`#!/bin/bash
set -euo pipefail

MANAGER_URL="%s"
TOKEN="%s"
MANAGER_HTTP_URL="%s"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/dockeeper-agent"

echo "==> Installing Dockeeper Agent..."

# Detect architecture
ARCH=$(uname -m)
case "${ARCH}" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# Download agent binary to a temp file first (validate before replacing).
DOWNLOAD_URL="${MANAGER_HTTP_URL}/api/dk/agent/download?arch=${ARCH}"
echo "==> Downloading from ${DOWNLOAD_URL}"
TMPBIN=$(mktemp /tmp/dockeeper-agent-download.XXXXXX)
trap "rm -f ${TMPBIN}" EXIT

HTTP_CODE=$(curl -sSL -o "${TMPBIN}" -w "%%{http_code}" "${DOWNLOAD_URL}")
if [ "${HTTP_CODE}" != "200" ]; then
  echo "ERROR: Download failed with HTTP status ${HTTP_CODE}"
  cat "${TMPBIN}" 2>/dev/null || true
  exit 1
fi

# Validate the downloaded file is a real binary (not a JSON error).
FILE_SIZE=$(stat -c%%s "${TMPBIN}" 2>/dev/null || stat -f%%z "${TMPBIN}" 2>/dev/null || echo "0")
if [ "${FILE_SIZE}" -lt 1048576 ]; then
  echo "ERROR: Downloaded file is too small (${FILE_SIZE} bytes). Expected a binary."
  head -c 200 "${TMPBIN}" 2>/dev/null || true
  exit 1
fi

# Stop existing service if running.
if systemctl is-active --quiet dockeeper-agent 2>/dev/null; then
  echo "==> Stopping existing Dockeeper Agent..."
  systemctl stop dockeeper-agent || true
fi
if systemctl is-enabled --quiet dockeeper-agent 2>/dev/null; then
  systemctl disable dockeeper-agent || true
fi

# Backup existing binary if present and functional.
if [ -f "${INSTALL_DIR}/dockeeper-agent" ]; then
  cp -f "${INSTALL_DIR}/dockeeper-agent" "${INSTALL_DIR}/dockeeper-agent.backup" 2>/dev/null || true
fi

# Install the validated binary.
rm -f /etc/systemd/system/dockeeper-agent.service
rm -f "${INSTALL_DIR}/dockeeper-agent"
systemctl daemon-reload 2>/dev/null || true

mv "${TMPBIN}" "${INSTALL_DIR}/dockeeper-agent"
chmod +x "${INSTALL_DIR}/dockeeper-agent"

# Create data directory
mkdir -p "${DATA_DIR}"

# Create systemd service
cat > /etc/systemd/system/dockeeper-agent.service << EOF
[Unit]
Description=Dockeeper Agent
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/dockeeper-agent --manager-url=${MANAGER_URL} --token=${TOKEN} --data-dir=${DATA_DIR}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
systemctl daemon-reload
systemctl enable --now dockeeper-agent

echo "==> Dockeeper Agent installed and running!"
echo "==> Check status: systemctl status dockeeper-agent"
echo "==> Agent uses outbound-only connections to the manager. No inbound ports required."`, grpcAddr, token, managerHTTPURL)
}
