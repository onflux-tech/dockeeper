package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFileName = "agent.yaml"

// Config holds the agent's persistent configuration.
type Config struct {
	ManagerURL string `yaml:"manager_url"`
	DataDir    string `yaml:"data_dir"`
	ServerID   string `yaml:"server_id,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DataDir: "./dk-agent-data",
	}
}

// ConfigPath returns the full path to the config file.
func (c *Config) ConfigPath() string {
	return filepath.Join(c.DataDir, configFileName)
}

// CertsDir returns the directory where TLS certificates are stored.
func (c *Config) CertsDir() string {
	return filepath.Join(c.DataDir, "certs")
}

// LoadConfig reads the config from the YAML file inside dataDir.
// Returns DefaultConfig if the file does not exist.
func LoadConfig(dataDir string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.DataDir = dataDir

	data, err := os.ReadFile(cfg.ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Always honour the dataDir that was passed in.
	cfg.DataDir = dataDir
	return cfg, nil
}

// Save writes the current config to disk.
func (c *Config) Save() error {
	if err := os.MkdirAll(c.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(c.ConfigPath(), data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
