package agent

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/jkdesenvolvedor/dockeeper/internal/agent/cert"
	"github.com/jkdesenvolvedor/dockeeper/internal/agent/collector"
)

// Version is set at build time via ldflags.
var Version = "dev"

const (
	backoffBase = 1 * time.Second
	backoffMax  = 60 * time.Second
	backoffMul  = 2.0
	jitterFrac  = 0.25

	certCheckInterval = 1 * time.Hour
)

// Agent orchestrates the enrollment, mTLS connection and lifecycle of the
// dockeeper agent process.
type Agent struct {
	cfg    *Config
	certs  *cert.Manager
	client *GRPCClient
	orch   collector.OrchestratorClient
	logger *slog.Logger
	token  string // one-time enrollment token (from CLI)
}

// NewAgent creates an Agent from the given config and enrollment token.
func NewAgent(cfg *Config, token string) (*Agent, error) {
	cm, err := cert.NewManager(cfg.CertsDir())
	if err != nil {
		return nil, fmt.Errorf("init cert manager: %w", err)
	}

	logger := slog.Default()
	if cfg.ServerID != "" {
		logger = logger.With("server_id", cfg.ServerID)
	}

	return &Agent{
		cfg:    cfg,
		certs:  cm,
		logger: logger,
		token:  token,
	}, nil
}

// Run is the main agent loop. It blocks until ctx is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	if err := a.ensureEnrolled(ctx); err != nil {
		return fmt.Errorf("enrollment: %w", err)
	}

	if err := a.connect(); err != nil {
		return fmt.Errorf("grpc connect: %w", err)
	}
	defer func() {
		if err := a.client.Close(); err != nil {
			a.logger.Error("error closing gRPC connection", "error", err)
		}
	}()

	a.logger.Info("agent running", "manager", a.cfg.ManagerURL, "server_id", a.cfg.ServerID)

	// Background cert renewal.
	go a.certRenewalLoop(ctx)

	// Detect container runtime and start discovery.
	orchType, orch, err := DetectOrchestrator(ctx)
	if err != nil {
		a.logger.Warn("no container runtime detected, discovery disabled", "error", err)
	} else {
		a.orch = orch
		a.logger.Info("orchestrator detected", "type", orchType)
		defer func() {
			if closeErr := a.orch.Close(); closeErr != nil {
				a.logger.Error("error closing orchestrator client", "error", closeErr)
			}
		}()

		discovery := collector.NewDiscoveryCollector(orch, a.client.Discovery(), a.cfg.ServerID, a.logger)
		go discovery.Run(ctx)

		metrics := collector.NewMetricsCollector(orch, a.client.Metrics(), a.cfg.ServerID, a.logger)
		go metrics.Run(ctx)
	}

	// Block until shutdown signal.
	<-ctx.Done()
	a.logger.Info("shutting down agent")
	return nil
}

// ensureEnrolled checks for existing certs and performs enrollment if needed.
func (a *Agent) ensureEnrolled(ctx context.Context) error {
	if a.certs.HasCerts() && a.cfg.ServerID != "" {
		a.logger.Info("agent already enrolled", "server_id", a.cfg.ServerID)
		return nil
	}

	if a.token == "" {
		return fmt.Errorf("not enrolled and no --token provided")
	}

	a.logger.Info("starting enrollment", "manager", a.cfg.ManagerURL)

	var attempt int
	for {
		resp, err := Enroll(ctx, a.cfg.ManagerURL, a.token)
		if err == nil {
			if err := a.certs.SaveCerts(resp.GetCaCert(), resp.GetClientCert(), resp.GetClientKey()); err != nil {
				return fmt.Errorf("save certs: %w", err)
			}

			a.cfg.ServerID = resp.GetServerId()
			if err := a.cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			a.logger = a.logger.With("server_id", a.cfg.ServerID)
			a.logger.Info("enrollment complete")
			return nil
		}

		attempt++
		wait := backoff(attempt)
		a.logger.Warn("enrollment failed, retrying", "error", err, "attempt", attempt, "retry_in", wait)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// connect establishes the mTLS gRPC connection to the manager.
func (a *Agent) connect() error {
	tlsCfg, err := a.certs.TLSConfig()
	if err != nil {
		return fmt.Errorf("load TLS config: %w", err)
	}

	client, err := NewGRPCClient(a.cfg.ManagerURL, tlsCfg)
	if err != nil {
		return fmt.Errorf("create gRPC client: %w", err)
	}

	a.client = client
	return nil
}

// certRenewalLoop periodically checks if the client cert needs renewal.
func (a *Agent) certRenewalLoop(ctx context.Context) {
	ticker := time.NewTicker(certCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			needs, err := a.certs.NeedsRenewal()
			if err != nil {
				a.logger.Error("cert renewal check failed", "error", err)
				continue
			}
			if !needs {
				continue
			}

			a.logger.Info("certificate approaching expiry, renewing")
			if err := a.renewCert(ctx); err != nil {
				a.logger.Error("cert renewal failed", "error", err)
			}
		}
	}
}

func (a *Agent) renewCert(ctx context.Context) error {
	tlsCfg, err := a.certs.TLSConfig()
	if err != nil {
		return err
	}

	resp, err := RenewCert(ctx, a.cfg.ManagerURL, a.cfg.ServerID, tlsCfg)
	if err != nil {
		return err
	}

	// Read existing CA (it doesn't change on renewal).
	caCert, caErr := a.readCACert()
	if caErr != nil {
		return fmt.Errorf("read existing CA: %w", caErr)
	}

	return a.certs.SaveCerts(caCert, resp.GetClientCert(), resp.GetClientKey())
}

func (a *Agent) readCACert() ([]byte, error) {
	return os.ReadFile(filepath.Join(a.cfg.CertsDir(), "ca.pem"))
}

// backoff computes the wait duration for the given attempt using exponential
// backoff with jitter.
func backoff(attempt int) time.Duration {
	d := float64(backoffBase)
	for range attempt - 1 {
		d *= backoffMul
		if d > float64(backoffMax) {
			d = float64(backoffMax)
			break
		}
	}
	// Apply jitter: ±25%
	jitter := d * jitterFrac * (2*rand.Float64() - 1)
	d += jitter
	return time.Duration(d)
}
