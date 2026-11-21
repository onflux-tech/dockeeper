package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/onflux-tech/dockeeper/internal/shared/crypto"
)

// ImageRef holds parsed parts of a Docker image reference.
type ImageRef struct {
	Registry   string // e.g., "registry-1.docker.io"
	Repository string // e.g., "library/nginx"
	Tag        string // e.g., "latest"
	Digest     string // if pinned
}

// RegistryCredential holds decrypted credentials for a registry.
type RegistryCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegistryAdapter checks OCI registries for image digest updates.
type RegistryAdapter struct {
	mu          sync.RWMutex
	credentials map[string]RegistryCredential // registry host → creds
	httpClient  *http.Client
	encKey      string
}

// NewRegistryAdapter creates a registry adapter.
func NewRegistryAdapter(encKey string) *RegistryAdapter {
	return &RegistryAdapter{
		credentials: make(map[string]RegistryCredential),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		encKey:      encKey,
	}
}

// SetCredentials sets the cached credentials map.
func (r *RegistryAdapter) SetCredentials(creds map[string]RegistryCredential) {
	r.mu.Lock()
	r.credentials = creds
	r.mu.Unlock()
}

// LoadCredential decrypts a stored credential config.
func (r *RegistryAdapter) LoadCredential(encryptedConfig string) (*RegistryCredential, error) {
	if r.encKey == "" {
		return nil, fmt.Errorf("encryption key not configured")
	}
	data, err := crypto.Decrypt(encryptedConfig, r.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}
	var cred RegistryCredential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, fmt.Errorf("parse credential: %w", err)
	}
	return &cred, nil
}

// GetLatestDigest fetches the latest image digest from the registry.
func (r *RegistryAdapter) GetLatestDigest(ctx context.Context, imageRef string) (string, error) {
	ref := ParseImageRef(imageRef)
	if ref.Digest != "" {
		return ref.Digest, nil // pinned, no update possible
	}

	r.mu.RLock()
	cred, hasCred := r.credentials[ref.Registry]
	r.mu.RUnlock()

	// Get auth token.
	token, err := r.getAuthToken(ctx, ref, hasCred, cred)
	if err != nil {
		return "", fmt.Errorf("auth for %s: %w", ref.Registry, err)
	}

	// HEAD manifest to get digest.
	digest, err := r.headManifest(ctx, ref, token)
	if err != nil {
		return "", fmt.Errorf("manifest for %s/%s:%s: %w", ref.Registry, ref.Repository, ref.Tag, err)
	}

	return digest, nil
}

// ParseImageRef parses a Docker image reference into its components.
func ParseImageRef(imageRef string) ImageRef {
	ref := ImageRef{Tag: "latest"}

	// Check for digest pinning.
	if idx := strings.Index(imageRef, "@"); idx >= 0 {
		ref.Digest = imageRef[idx+1:]
		imageRef = imageRef[:idx]
	}

	// Check for tag.
	if idx := strings.LastIndex(imageRef, ":"); idx >= 0 {
		possibleTag := imageRef[idx+1:]
		// Make sure it's not a port number (contains no /)
		if !strings.Contains(possibleTag, "/") {
			ref.Tag = possibleTag
			imageRef = imageRef[:idx]
		}
	}

	// Split registry / repository.
	parts := strings.SplitN(imageRef, "/", 2)
	if len(parts) == 1 {
		// No slash: official image like "nginx"
		ref.Registry = "registry-1.docker.io"
		ref.Repository = "library/" + parts[0]
	} else if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
		// First part looks like a hostname
		ref.Registry = parts[0]
		ref.Repository = parts[1]
	} else {
		// User image like "user/repo"
		ref.Registry = "registry-1.docker.io"
		ref.Repository = imageRef
	}

	return ref
}

// getAuthToken obtains a bearer token for registry access.
func (r *RegistryAdapter) getAuthToken(ctx context.Context, ref ImageRef, hasCred bool, cred RegistryCredential) (string, error) {
	// Docker Hub uses auth.docker.io/token
	if ref.Registry == "registry-1.docker.io" {
		return r.getDockerHubToken(ctx, ref.Repository, hasCred, cred)
	}

	// GHCR and other registries use /v2/token or WWW-Authenticate challenge
	return r.getChallengeToken(ctx, ref, hasCred, cred)
}

// getDockerHubToken gets a token from Docker Hub's auth service.
func (r *RegistryAdapter) getDockerHubToken(ctx context.Context, repository string, hasCred bool, cred RegistryCredential) (string, error) {
	url := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repository)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	if hasCred && cred.Username != "" {
		req.SetBasicAuth(cred.Username, cred.Password)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth failed: %s", resp.Status)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

// getChallengeToken handles WWW-Authenticate challenge flow for non-Docker Hub registries.
func (r *RegistryAdapter) getChallengeToken(ctx context.Context, ref ImageRef, hasCred bool, cred RegistryCredential) (string, error) {
	// First, try a plain request to get the WWW-Authenticate header
	challengeURL := fmt.Sprintf("https://%s/v2/", ref.Registry)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, challengeURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "", nil // no auth needed
	}

	wwwAuth := resp.Header.Get("WWW-Authenticate")
	if wwwAuth == "" {
		// Try basic auth directly
		if hasCred {
			return "basic", nil
		}
		return "", fmt.Errorf("no authentication method available")
	}

	// Parse WWW-Authenticate: Bearer realm="...",service="...",scope="..."
	realm, service, scope := parseWWWAuth(wwwAuth, ref)
	if realm == "" {
		return "", fmt.Errorf("could not parse WWW-Authenticate header")
	}

	tokenURL := fmt.Sprintf("%s?service=%s&scope=repository:%s:pull", realm, service, scope)
	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}

	if hasCred && cred.Username != "" {
		tokenReq.SetBasicAuth(cred.Username, cred.Password)
	}

	tokenResp, err := r.httpClient.Do(tokenReq)
	if err != nil {
		return "", err
	}
	defer func() { _ = tokenResp.Body.Close() }()

	if tokenResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed: %s", tokenResp.Status)
	}

	var result struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Token != "" {
		return result.Token, nil
	}
	return result.AccessToken, nil
}

// headManifest does a HEAD request to get the image digest.
func (r *RegistryAdapter) headManifest(ctx context.Context, ref ImageRef, token string) (string, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", ref.Registry, ref.Repository, ref.Tag)

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")

	if token != "" && token != "basic" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("image not found")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("manifest HEAD failed: %s", resp.Status)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("no Docker-Content-Digest header")
	}

	return digest, nil
}

// parseWWWAuth parses a WWW-Authenticate header.
func parseWWWAuth(header string, ref ImageRef) (realm, service, scope string) {
	header = strings.TrimPrefix(header, "Bearer ")

	params := make(map[string]string)
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = strings.Trim(kv[1], "\"")
		}
	}

	realm = params["realm"]
	service = params["service"]
	scope = ref.Repository

	if s, ok := params["scope"]; ok {
		// Use the original scope from the header
		_ = s
	}

	slog.Debug("parsed WWW-Authenticate", "realm", realm, "service", service, "scope", scope)
	return realm, service, scope
}
