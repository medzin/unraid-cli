package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func sampleConfig() *Config {
	return &Config{
		Default: "tower",
		Servers: map[string]ServerConfig{
			"tower":  {URL: "https://192.168.1.100", APIKey: "key-tower"},
			"backup": {URL: "https://192.168.1.101", APIKey: "key-backup"},
		},
	}
}

func TestDefaultConfigHasNoServersAndNoDefault(t *testing.T) {
	cfg := &Config{Servers: make(map[string]ServerConfig)}
	if cfg.Default != "" {
		t.Errorf("expected empty default, got %q", cfg.Default)
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(cfg.Servers))
	}
}

func TestAddServer(t *testing.T) {
	cases := []struct {
		name            string
		setup           func() *Config
		addName         string
		addURL          string
		addKey          string
		addInsecureTLS  bool
		wantCount       int
		wantURL         string
		wantAPIKey      string
		wantInsecureTLS bool
	}{
		{
			name:            "new server secure",
			setup:           func() *Config { return &Config{Servers: make(map[string]ServerConfig)} },
			addName:         "test",
			addURL:          "https://example.com",
			addKey:          "api-key",
			addInsecureTLS:  false,
			wantCount:       1,
			wantURL:         "https://example.com",
			wantAPIKey:      "api-key",
			wantInsecureTLS: false,
		},
		{
			name:            "new server insecure TLS",
			setup:           func() *Config { return &Config{Servers: make(map[string]ServerConfig)} },
			addName:         "test",
			addURL:          "https://192.168.1.100",
			addKey:          "api-key",
			addInsecureTLS:  true,
			wantCount:       1,
			wantURL:         "https://192.168.1.100",
			wantAPIKey:      "api-key",
			wantInsecureTLS: true,
		},
		{
			name:            "overwrites existing",
			setup:           sampleConfig,
			addName:         "tower",
			addURL:          "https://new-url.com",
			addKey:          "new-key",
			addInsecureTLS:  false,
			wantCount:       2,
			wantURL:         "https://new-url.com",
			wantAPIKey:      "new-key",
			wantInsecureTLS: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.setup()
			cfg.AddServer(tc.addName, tc.addURL, tc.addKey, tc.addInsecureTLS)

			if len(cfg.Servers) != tc.wantCount {
				t.Errorf("expected %d servers, got %d", tc.wantCount, len(cfg.Servers))
			}
			srv := cfg.Servers[tc.addName]
			if srv.URL != tc.wantURL {
				t.Errorf("expected URL %s, got %s", tc.wantURL, srv.URL)
			}
			if srv.APIKey != tc.wantAPIKey {
				t.Errorf("expected APIKey %s, got %s", tc.wantAPIKey, srv.APIKey)
			}
			if srv.InsecureTLS != tc.wantInsecureTLS {
				t.Errorf("expected InsecureTLS %v, got %v", tc.wantInsecureTLS, srv.InsecureTLS)
			}
		})
	}
}

func TestGetServer(t *testing.T) {
	cases := []struct {
		name       string
		hasDefault bool
		input      string
		wantURL    string
		wantNil    bool
	}{
		{"by name", true, "backup", "https://192.168.1.101", false},
		{"empty name uses default", true, "", "https://192.168.1.100", false},
		{"empty name no default", false, "", "", true},
		{"unknown name", true, "nonexistent", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := sampleConfig()
			if !tc.hasDefault {
				cfg.Default = ""
			}

			srv := cfg.GetServer(tc.input)
			if tc.wantNil {
				if srv != nil {
					t.Errorf("expected nil, got %+v", srv)
				}
				return
			}
			if srv == nil {
				t.Fatal("expected server, got nil")
				return
			}
			if srv.URL != tc.wantURL {
				t.Errorf("expected URL %s, got %s", tc.wantURL, srv.URL)
			}
		})
	}
}

func TestRemoveServer(t *testing.T) {
	cases := []struct {
		name        string
		removeName  string
		wantRemoved bool
		wantCount   int
		wantDefault string
	}{
		{"removes non-default", "backup", true, 1, "tower"},
		{"removes default and clears it", "tower", true, 1, ""},
		{"unknown returns false", "nonexistent", false, 2, "tower"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := sampleConfig()
			got := cfg.RemoveServer(tc.removeName)

			if got != tc.wantRemoved {
				t.Errorf("RemoveServer() = %v, want %v", got, tc.wantRemoved)
			}
			if len(cfg.Servers) != tc.wantCount {
				t.Errorf("expected %d servers, got %d", tc.wantCount, len(cfg.Servers))
			}
			if cfg.Default != tc.wantDefault {
				t.Errorf("expected default %q, got %q", tc.wantDefault, cfg.Default)
			}
		})
	}
}

func TestSetDefault(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantErr   bool
		wantValue string
	}{
		{"existing server", "backup", false, "backup"},
		{"unknown server", "nonexistent", true, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := sampleConfig()
			err := cfg.SetDefault(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "not found in configuration") {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Default != tc.wantValue {
				t.Errorf("expected default %q, got %q", tc.wantValue, cfg.Default)
			}
		})
	}
}

func TestTOMLRoundtrip(t *testing.T) {
	cfg := sampleConfig()
	data, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed Config
	if err := toml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.Default != cfg.Default {
		t.Errorf("default mismatch: got %q, want %q", parsed.Default, cfg.Default)
	}
	if len(parsed.Servers) != len(cfg.Servers) {
		t.Errorf("server count mismatch: got %d, want %d", len(parsed.Servers), len(cfg.Servers))
	}
	if parsed.Servers["tower"].URL != cfg.Servers["tower"].URL {
		t.Errorf("tower URL mismatch")
	}
}

func TestDeserializeTOML(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantDefault string
		wantServers int
	}{
		{"empty", "", "", 0},
		{"only default", "default = \"myserver\"\n", "myserver", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg Config
			if err := toml.Unmarshal([]byte(tc.input), &cfg); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if cfg.Default != tc.wantDefault {
				t.Errorf("expected default %q, got %q", tc.wantDefault, cfg.Default)
			}
			serverCount := len(cfg.Servers)
			if cfg.Servers == nil {
				serverCount = 0
			}
			if serverCount != tc.wantServers {
				t.Errorf("expected %d servers, got %d", tc.wantServers, serverCount)
			}
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"", "***"},
		{"x", "x..."},
		{"abc", "abc..."},
		{"12345678", "12345678..."},
		{"abcdefghijklmnop", "abcdefgh..."},
	}

	for _, tc := range cases {
		got := MaskAPIKey(tc.input)
		if got != tc.expected {
			t.Errorf("MaskAPIKey(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// --- LoadFrom / SaveTo tests ---

func TestLoadFromMissingFile(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(cfg.Servers))
	}
	if cfg.Default != "" {
		t.Errorf("expected empty default, got %q", cfg.Default)
	}
}

func TestLoadFromValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `default = "tower"

[servers.tower]
url = "https://192.168.1.100"
api_key = "key-tower"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Default != "tower" {
		t.Errorf("expected default tower, got %q", cfg.Default)
	}
	if len(cfg.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(cfg.Servers))
	}
	if cfg.Servers["tower"].URL != "https://192.168.1.100" {
		t.Errorf("unexpected tower URL: %s", cfg.Servers["tower"].URL)
	}
}

func TestLoadFromInvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("not valid {{{{ toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSaveToCreatesDirectoriesAndWritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.toml")

	cfg := sampleConfig()
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if loaded.Default != cfg.Default {
		t.Errorf("default mismatch: got %q, want %q", loaded.Default, cfg.Default)
	}
	if len(loaded.Servers) != len(cfg.Servers) {
		t.Errorf("server count mismatch: got %d, want %d", len(loaded.Servers), len(cfg.Servers))
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("unraid", "config.toml")) {
		t.Errorf("expected path ending in unraid/config.toml, got %s", path)
	}
}

// --- Resolve tests ---

func TestResolve(t *testing.T) {
	cases := []struct {
		name            string
		cliServer       string
		cliURL          string
		cliAPIKey       string
		envURL          string
		envAPIKey       string
		envServer       string
		envInsecureTLS  string
		wantURL         string
		wantAPIKey      string
		wantInsecureTLS bool
		wantErr         string
	}{
		{
			name:       "both CLI args provided",
			cliURL:     "https://cli.com",
			cliAPIKey:  "cli-key",
			wantURL:    "https://cli.com",
			wantAPIKey: "cli-key",
		},
		{
			name:       "both env vars provided",
			envURL:     "https://env.com",
			envAPIKey:  "env-key",
			wantURL:    "https://env.com",
			wantAPIKey: "env-key",
		},
		{
			name:            "env vars with insecure TLS",
			envURL:          "https://env.com",
			envAPIKey:       "env-key",
			envInsecureTLS:  "true",
			wantURL:         "https://env.com",
			wantAPIKey:      "env-key",
			wantInsecureTLS: true,
		},
		{
			name:            "env insecure TLS false by default",
			envURL:          "https://env.com",
			envAPIKey:       "env-key",
			envInsecureTLS:  "",
			wantURL:         "https://env.com",
			wantAPIKey:      "env-key",
			wantInsecureTLS: false,
		},
		{
			name:       "CLI URL overrides env URL",
			cliURL:     "https://cli.com",
			envURL:     "https://env.com",
			envAPIKey:  "env-key",
			wantURL:    "https://cli.com",
			wantAPIKey: "env-key",
		},
		{
			name:       "CLI API key overrides env API key",
			cliAPIKey:  "cli-key",
			envURL:     "https://env.com",
			envAPIKey:  "env-key",
			wantURL:    "https://env.com",
			wantAPIKey: "cli-key",
		},
		{
			name:    "no config falls through to error",
			wantErr: "no server configured",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("UNRAID_URL", tc.envURL)
			t.Setenv("UNRAID_API_KEY", tc.envAPIKey)
			t.Setenv("UNRAID_SERVER", tc.envServer)
			t.Setenv("UNRAID_INSECURE_TLS", tc.envInsecureTLS)

			resolved, err := Resolve(tc.cliServer, tc.cliURL, tc.cliAPIKey, false)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.URL != tc.wantURL {
				t.Errorf("URL = %q, want %q", resolved.URL, tc.wantURL)
			}
			if resolved.APIKey != tc.wantAPIKey {
				t.Errorf("APIKey = %q, want %q", resolved.APIKey, tc.wantAPIKey)
			}
			if resolved.InsecureTLS != tc.wantInsecureTLS {
				t.Errorf("InsecureTLS = %v, want %v", resolved.InsecureTLS, tc.wantInsecureTLS)
			}
		})
	}
}
