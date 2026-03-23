package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	caFile   = "ca.pem"
	certFile = "client.pem"
	keyFile  = "client-key.pem"

	// renewalThreshold is the fraction of the cert's total lifetime after
	// which the agent should request a renewed certificate.
	renewalThreshold = 0.80
)

// Manager handles loading, saving and inspecting TLS certificates
// used for mTLS communication with the manager.
type Manager struct {
	dir string
}

// NewManager creates a Manager that stores certs in dir.
// The directory is created if it does not exist.
func NewManager(dir string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create certs dir: %w", err)
	}
	return &Manager{dir: dir}, nil
}

// HasCerts returns true when all three certificate files are present on disk.
func (m *Manager) HasCerts() bool {
	for _, f := range []string{caFile, certFile, keyFile} {
		if _, err := os.Stat(filepath.Join(m.dir, f)); err != nil {
			return false
		}
	}
	return true
}

// SaveCerts persists the CA, client certificate and client key to disk.
func (m *Manager) SaveCerts(ca, cert, key []byte) error {
	files := map[string][]byte{
		caFile:   ca,
		certFile: cert,
		keyFile:  key,
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(m.dir, name), data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// TLSConfig builds a *tls.Config suitable for mTLS gRPC dialing.
func (m *Manager) TLSConfig() (*tls.Config, error) {
	caCert, err := os.ReadFile(filepath.Join(m.dir, caFile))
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to parse CA certificate")
	}

	clientCert, err := tls.LoadX509KeyPair(
		filepath.Join(m.dir, certFile),
		filepath.Join(m.dir, keyFile),
	)
	if err != nil {
		return nil, fmt.Errorf("load client key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// NeedsRenewal checks whether the client certificate has passed the
// renewal threshold (80% of its total validity period).
func (m *Manager) NeedsRenewal() (bool, error) {
	leaf, err := m.parseLeaf()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	total := leaf.NotAfter.Sub(leaf.NotBefore)
	threshold := leaf.NotBefore.Add(time.Duration(float64(total) * renewalThreshold))
	return time.Now().After(threshold), nil
}

// ExpiresAt returns the expiration time of the client certificate.
func (m *Manager) ExpiresAt() (time.Time, error) {
	leaf, err := m.parseLeaf()
	if err != nil {
		return time.Time{}, err
	}
	return leaf.NotAfter, nil
}

func (m *Manager) parseLeaf() (*x509.Certificate, error) {
	data, err := os.ReadFile(filepath.Join(m.dir, certFile))
	if err != nil {
		return nil, fmt.Errorf("read client cert: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block found in client certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return cert, nil
}
