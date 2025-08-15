package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"llmbench/internal/models"
)

// Config holds the application configuration
type Config struct {
	Benchmark models.BenchmarkConfig `mapstructure:"benchmark"`
}

// Manager handles configuration loading and management
type Manager struct {
	config *Config
	viper  *viper.Viper
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	v := viper.New()
	return &Manager{
		viper: v,
	}
}

// Load loads configuration from file and environment variables
func (m *Manager) Load(configPath string) error {
	// Set default values
	m.setDefaults()

	// Set config file path if provided
	if configPath != "" {
		m.viper.SetConfigFile(configPath)
	} else {
		// Look for config in common locations
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		m.viper.SetConfigName("llmbench")
		m.viper.SetConfigType("yaml")
		m.viper.AddConfigPath(".")
		m.viper.AddConfigPath(filepath.Join(home, ".config", "llmbench"))
		m.viper.AddConfigPath("/etc/llmbench")
	}

	// Environment variables
	m.viper.SetEnvPrefix("LLMBENCH")
	m.viper.AutomaticEnv()

	// Read config file
	if err := m.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Unmarshal into config struct
	m.config = &Config{}
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return m.validate()
}

// setDefaults sets default configuration values
func (m *Manager) setDefaults() {
	m.viper.SetDefault("benchmark.concurrency", 1)
	m.viper.SetDefault("benchmark.requests", 10)
	m.viper.SetDefault("benchmark.timeout", "30s")
	m.viper.SetDefault("benchmark.providers", []models.Provider{})
}

// validate validates the loaded configuration
func (m *Manager) validate() error {
	if len(m.config.Benchmark.Providers) == 0 {
		return fmt.Errorf("at least one provider must be configured")
	}

	for i, provider := range m.config.Benchmark.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider %d: name is required", i)
		}
		if provider.BaseURL == "" {
			return fmt.Errorf("provider %s: base_url is required", provider.Name)
		}
		if provider.APIKey == "" {
			return fmt.Errorf("provider %s: api_key is required", provider.Name)
		}
		if provider.Model == "" {
			return fmt.Errorf("provider %s: model is required", provider.Name)
		}
	}

	if m.config.Benchmark.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}

	if m.config.Benchmark.Requests <= 0 {
		return fmt.Errorf("requests must be greater than 0")
	}

	// Validate timeout format
	if _, err := time.ParseDuration(m.config.Benchmark.Timeout); err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	return nil
}

// GetConfig returns the loaded configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}

// GetProviders returns the configured providers
func (m *Manager) GetProviders() []models.Provider {
	if m.config == nil {
		return []models.Provider{}
	}
	return m.config.Benchmark.Providers
}

// GetBenchmarkConfig returns the benchmark configuration
func (m *Manager) GetBenchmarkConfig() models.BenchmarkConfig {
	if m.config == nil {
		return models.BenchmarkConfig{}
	}
	return m.config.Benchmark
}

// CreateSampleConfig creates a sample configuration file
func (m *Manager) CreateSampleConfig(path string) error {
	// Create YAML content manually to avoid encoding issues
	yamlContent := `benchmark:
  providers:
    - name: openai
      base_url: https://api.openai.com/v1
      api_key: your-openai-api-key
      model: gpt-3.5-turbo
    - name: anthropic
      base_url: https://api.anthropic.com/v1
      api_key: your-anthropic-api-key
      model: claude-3-haiku-20240307
  concurrency: 2
  requests: 50
  timeout: 30s
`

	// Write the YAML content directly to file
	return os.WriteFile(path, []byte(yamlContent), 0644)
}
