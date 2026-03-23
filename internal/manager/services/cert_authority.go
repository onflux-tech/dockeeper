package services

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	caDir          = "certs"
	caCertFile     = "ca.pem"
	caKeyFile      = "ca-key.pem"
	serverCertFile = "server.pem"
	serverKeyFile  = "server-key.pem"

	caValidity     = 10 * 365 * 24 * time.Hour // 10 years
	serverValidity = 5 * 365 * 24 * time.Hour  // 5 years
	agentValidity  = 365 * 24 * time.Hour       // 1 year
)

// CertAuthority manages an internal PKI for mTLS between manager and agents.
type CertAuthority struct {
	caCert    *x509.Certificate
	caKey     crypto.PrivateKey
	caCertPEM []byte
	dir       string
}

// NewCertAuthority loads or creates the internal CA inside dataDir/certs/.
func NewCertAuthority(dataDir string) (*CertAuthority, error) {
	dir := filepath.Join(dataDir, caDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create CA dir: %w", err)
	}

	ca := &CertAuthority{dir: dir}
	if err := ca.ensureCA(); err != nil {
		return nil, fmt.Errorf("ensure CA: %w", err)
	}
	if err := ca.ensureServerCert(); err != nil {
		return nil, fmt.Errorf("ensure server cert: %w", err)
	}
	return ca, nil
}

// CACertPEM returns the PEM-encoded CA certificate.
func (ca *CertAuthority) CACertPEM() []byte {
	return ca.caCertPEM
}

// SignAgentCert generates a new ECDSA P-256 key pair and an x509 certificate
// for the agent identified by serverID, signed by the internal CA.
func (ca *CertAuthority) SignAgentCert(serverID string) (certPEM, keyPEM []byte, expiresAt time.Time, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("generate agent key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, nil, time.Time{}, err
	}

	now := time.Now()
	expiresAt = now.Add(agentValidity)

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   serverID,
			Organization: []string{"dockeeper-agent"},
		},
		NotBefore:             now,
		NotAfter:              expiresAt,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.caCert, &key.PublicKey, ca.caKey)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("sign agent cert: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("marshal agent key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, expiresAt, nil
}

// ServerTLSConfig returns a *tls.Config for the gRPC server with mTLS.
// ClientAuth is set to VerifyClientCertIfGiven so that the Enroll RPC can
// be called without a client certificate.
func (ca *CertAuthority) ServerTLSConfig() (*tls.Config, error) {
	serverCert, err := tls.LoadX509KeyPair(
		filepath.Join(ca.dir, serverCertFile),
		filepath.Join(ca.dir, serverKeyFile),
	)
	if err != nil {
		return nil, fmt.Errorf("load server cert: %w", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ca.caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    pool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// ensureCA loads existing CA files or generates a new root CA.
func (ca *CertAuthority) ensureCA() error {
	certPath := filepath.Join(ca.dir, caCertFile)
	keyPath := filepath.Join(ca.dir, caKeyFile)

	certData, certErr := os.ReadFile(certPath)
	keyData, keyErr := os.ReadFile(keyPath)

	if certErr == nil && keyErr == nil {
		return ca.loadCA(certData, keyData)
	}

	if !errors.Is(certErr, os.ErrNotExist) && certErr != nil {
		return fmt.Errorf("read CA cert: %w", certErr)
	}

	return ca.generateCA(certPath, keyPath)
}

func (ca *CertAuthority) loadCA(certData, keyData []byte) error {
	block, _ := pem.Decode(certData)
	if block == nil {
		return errors.New("no PEM block in CA cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return errors.New("no PEM block in CA key")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA key: %w", err)
	}

	ca.caCert = cert
	ca.caKey = key
	ca.caCertPEM = certData
	return nil
}

func (ca *CertAuthority) generateCA(certPath, keyPath string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "dockeeper-ca",
			Organization: []string{"dockeeper"},
		},
		NotBefore:             now,
		NotAfter:              now.Add(caValidity),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("create CA cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return fmt.Errorf("write CA cert: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal CA key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write CA key: %w", err)
	}

	parsedCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("reparse CA cert: %w", err)
	}

	ca.caCert = parsedCert
	ca.caKey = key
	ca.caCertPEM = certPEM
	return nil
}

// ensureServerCert generates a TLS server certificate if one doesn't exist.
func (ca *CertAuthority) ensureServerCert() error {
	certPath := filepath.Join(ca.dir, serverCertFile)
	keyPath := filepath.Join(ca.dir, serverKeyFile)

	if _, err := os.Stat(certPath); err == nil {
		return nil // already exists
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "dockeeper-manager",
			Organization: []string{"dockeeper"},
		},
		NotBefore: now,
		NotAfter:  now.Add(serverValidity),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost", "dockeeper-manager"},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.caCert, &key.PublicKey, ca.caKey)
	if err != nil {
		return fmt.Errorf("sign server cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return fmt.Errorf("write server cert: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal server key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write server key: %w", err)
	}

	return nil
}

func randomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}
	return serial, nil
}
