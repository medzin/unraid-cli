// Package config manages server configurations for the Unraid CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ServerConfig holds connection details for a single Unraid server.
type ServerConfig struct {
	URL    string `toml:"url"`
	APIKey string `toml:"api_key"`
}

// Config holds all server configurations and the default server name.
type Config struct {
	Default string                  `toml:"default,omitempty"`
	Servers map[string]ServerConfig `toml:"servers"`
}

// ResolvedConfig is the final server URL and API key after resolution.
type ResolvedConfig struct {
	URL    string
	APIKey string
}

// ConfigPath returns the platform-specific path to the config file.
func ConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine config directory: %w", err)
	}
	return filepath.Join(configDir, "unraid", "config.toml"), nil
}

// Load reads the config from the default config path.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads the config from the given path. Returns an empty config if the file doesn't exist.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Servers: make(map[string]ServerConfig)}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %s: %w", path, err)
	}
	if cfg.Servers == nil {
		cfg.Servers = make(map[string]ServerConfig)
	}
	return &cfg, nil
}

// Save writes the config to the default config path.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the config to the given path, creating directories as needed.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %s: %w", dir, err)
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %s: %w", path, err)
	}
	return nil
}

// GetServer returns the server config for the given name, or the default if name is empty.
func (c *Config) GetServer(name string) *ServerConfig {
	if name == "" {
		name = c.Default
	}
	if name == "" {
		return nil
	}
	srv, ok := c.Servers[name]
	if !ok {
		return nil
	}
	return &srv
}

// AddServer adds or overwrites a server configuration.
func (c *Config) AddServer(name, url, apiKey string) {
	c.Servers[name] = ServerConfig{URL: url, APIKey: apiKey}
}

// RemoveServer removes a server. Returns true if it existed.
// Clears the default if the removed server was the default.
func (c *Config) RemoveServer(name string) bool {
	_, ok := c.Servers[name]
	if !ok {
		return false
	}
	delete(c.Servers, name)
	if c.Default == name {
		c.Default = ""
	}
	return true
}

// SetDefault sets the default server. Returns an error if the server doesn't exist.
func (c *Config) SetDefault(name string) error {
	if _, ok := c.Servers[name]; !ok {
		return fmt.Errorf("server '%s' not found in configuration", name)
	}
	c.Default = name
	return nil
}

// Resolve determines the final URL and API key from CLI flags, env vars, and config file.
// Priority: CLI args > env vars > config file.
func Resolve(cliServer, cliURL, cliAPIKey string) (*ResolvedConfig, error) {
	// If both URL and API key are provided directly, use them
	if cliURL != "" && cliAPIKey != "" {
		return &ResolvedConfig{URL: cliURL, APIKey: cliAPIKey}, nil
	}

	// Check environment variables
	envURL := os.Getenv("UNRAID_URL")
	envAPIKey := os.Getenv("UNRAID_API_KEY")
	envServer := os.Getenv("UNRAID_SERVER")

	// If both env vars are set, use them (CLI args override individual values)
	if envURL != "" && envAPIKey != "" {
		url := envURL
		if cliURL != "" {
			url = cliURL
		}
		apiKey := envAPIKey
		if cliAPIKey != "" {
			apiKey = cliAPIKey
		}
		return &ResolvedConfig{URL: url, APIKey: apiKey}, nil
	}

	// Load config file
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	// Determine which server to use
	serverName := cliServer
	if serverName == "" {
		serverName = envServer
	}

	if srv := cfg.GetServer(serverName); srv != nil {
		url := srv.URL
		if cliURL != "" {
			url = cliURL
		}
		apiKey := srv.APIKey
		if cliAPIKey != "" {
			apiKey = cliAPIKey
		}
		return &ResolvedConfig{URL: url, APIKey: apiKey}, nil
	}

	return nil, fmt.Errorf(
		"no server configured. Use 'unraid config add <name>' to add a server, " +
			"or set UNRAID_URL and UNRAID_API_KEY environment variables",
	)
}

// MaskAPIKey masks an API key for display, showing at most the first 8 characters.
func MaskAPIKey(key string) string {
	if key == "" {
		return "***"
	}
	visible := 8
	if len(key) < visible {
		visible = len(key)
	}
	return key[:visible] + "..."
}
