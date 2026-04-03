package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"
)

// testHarness bundles a mock GraphQL server, output buffer, and captured request
// variables for integration tests. Use newHarness to create one — cleanup is
// registered automatically via t.Cleanup.
type testHarness struct {
	t    *testing.T
	srv  *httptest.Server
	buf  bytes.Buffer
	vars map[string]any
}

func newHarness(t *testing.T, body any) *testHarness {
	t.Helper()
	h := &testHarness{t: t}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock server failed to decode request: %v", err)
		}
		h.vars = req.Variables
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Errorf("mock server failed to encode response: %v", err)
		}
	}))
	t.Cleanup(h.srv.Close)
	return h
}

// preRun returns a cobra PreRunE that injects the GraphQL client and JSON output
// writer into the command context, bypassing server config resolution.
func (h *testHarness) preRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := withClient(cmd.Context(), graphql.NewClient(h.srv.URL, h.srv.Client()))
		ctx = withOutputFormat(ctx, outputJSON)
		ctx = withOutputWriter(ctx, &h.buf)
		cmd.SetContext(ctx)
		return nil
	}
}

// run sets args on cmd and executes it, fataling on error.
func (h *testHarness) run(cmd *cobra.Command, args ...string) {
	h.t.Helper()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		h.t.Fatalf("unexpected error: %v", err)
	}
}

// mustDecodeJSON unmarshals the harness output buffer into T, fataling on error.
func mustDecodeJSON[T any](t *testing.T, h *testHarness) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(h.buf.Bytes(), &v); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, h.buf.String())
	}
	return v
}

func TestIntegrationDockerListJSON(t *testing.T) {
	body := map[string]any{
		"data": map[string]any{
			"docker": map[string]any{
				"containers": []map[string]any{
					{"id": "ctr-1", "names": []string{"/plex"}, "image": "linuxserver/plex:latest", "state": "RUNNING", "status": "Up 2 hours"},
					{"id": "ctr-2", "names": []string{"/sonarr"}, "image": "linuxserver/sonarr:latest", "state": "STOPPED", "status": "Exited (0)"},
				},
			},
		},
	}

	cases := []struct {
		name      string
		args      []string
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "default shows only running",
			args:      []string{"list"},
			wantCount: 1,
			wantIDs:   []string{"ctr-1"},
		},
		{
			name:      "all flag shows all containers",
			args:      []string{"list", "--all"},
			wantCount: 2,
			wantIDs:   []string{"ctr-1", "ctr-2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, body)
			h.run(newDockerCmd(h.preRun()), tc.args...)

			got := mustDecodeJSON[[]map[string]any](t, h)
			if len(got) != tc.wantCount {
				t.Fatalf("expected %d container(s), got %d", tc.wantCount, len(got))
			}
			for i, wantID := range tc.wantIDs {
				if got[i]["id"] != wantID {
					t.Errorf("containers[%d].id = %v, want %v", i, got[i]["id"], wantID)
				}
			}
			for _, field := range []string{"id", "names", "image", "state", "status"} {
				if _, ok := got[0][field]; !ok {
					t.Errorf("missing field %q in container JSON", field)
				}
			}
		})
	}
}

func TestIntegrationArrayStatusJSON(t *testing.T) {
	h := newHarness(t, map[string]any{
		"data": map[string]any{
			"array": map[string]any{
				"id":    "array-1",
				"state": "STARTED",
				"parities": []map[string]any{
					{"id": "p1", "name": "parity1", "device": "sda", "status": "DISK_OK", "size": 8000000, "isSpinning": true},
				},
				"disks": []map[string]any{
					{"id": "d1", "name": "disk1", "device": "sdb", "status": "DISK_OK", "size": 4000000, "fsSize": 3000000, "fsFree": 1000000, "isSpinning": false},
				},
				"caches": []map[string]any{},
			},
		},
	})
	h.run(newArrayCmd(h.preRun()), "status")

	got := mustDecodeJSON[map[string]any](t, h)
	for _, field := range []string{"state", "parities", "disks", "caches"} {
		if _, ok := got[field]; !ok {
			t.Errorf("missing field %q in array status JSON", field)
		}
	}
	if got["state"] != "STARTED" {
		t.Errorf("state = %v, want STARTED", got["state"])
	}
}

func TestIntegrationDockerLogsJSON(t *testing.T) {
	body := map[string]any{
		"data": map[string]any{
			"docker": map[string]any{
				// The mock must satisfy both GetDockerContainers (resolveContainerID)
				// and GetContainerLogs in a single response shape.
				"containers": []map[string]any{
					{"id": "ctr-1", "names": []string{"/plex"}, "image": "linuxserver/plex", "state": "RUNNING", "status": "Up"},
				},
				"logs": map[string]any{
					"lines": []map[string]any{
						{"timestamp": "2024-01-01T00:00:00Z", "message": "Server started"},
						{"timestamp": "2024-01-01T00:00:01Z", "message": "Listening on port 32400"},
					},
				},
			},
		},
	}

	cases := []struct {
		name        string
		args        []string
		wantTailVar float64
	}{
		{name: "default tail=100", args: []string{"plex"}, wantTailVar: 100},
		{name: "explicit tail value", args: []string{"plex", "--lines", "50"}, wantTailVar: 50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, body)
			cmd := newDockerLogsCmd()
			cmd.PreRunE = h.preRun()
			h.run(cmd, tc.args...)

			if h.vars["tail"] != tc.wantTailVar {
				t.Errorf("tail variable = %v, want %v", h.vars["tail"], tc.wantTailVar)
			}

			got := mustDecodeJSON[[]map[string]any](t, h)
			if len(got) != 2 {
				t.Fatalf("expected 2 log lines, got %d", len(got))
			}
			for _, field := range []string{"timestamp", "message"} {
				if _, ok := got[0][field]; !ok {
					t.Errorf("missing field %q in log line JSON", field)
				}
			}
		})
	}
}

func TestIntegrationServerVersionJSON(t *testing.T) {
	h := newHarness(t, map[string]any{
		"data": map[string]any{
			"info": map[string]any{
				"versions": map[string]any{
					"core": map[string]any{"unraid": "6.12.10", "api": "4.5.0"},
				},
			},
		},
	})
	h.run(newServerCmd(h.preRun()), "version")

	got := mustDecodeJSON[map[string]any](t, h)
	if got["unraid_os"] != "6.12.10" {
		t.Errorf("unraid_os = %v, want 6.12.10", got["unraid_os"])
	}
	if got["api"] != "4.5.0" {
		t.Errorf("api = %v, want 4.5.0", got["api"])
	}
}

func TestIntegrationServerLogListJSON(t *testing.T) {
	h := newHarness(t, map[string]any{
		"data": map[string]any{
			"logFiles": []map[string]any{
				{"name": "syslog", "path": "/var/log/syslog"},
				{"name": "unraid", "path": "/var/log/unraid.log"},
			},
		},
	})
	h.run(newServerCmd(h.preRun()), "log", "--list")

	got := mustDecodeJSON[[]map[string]any](t, h)
	if len(got) != 2 {
		t.Fatalf("expected 2 log files, got %d", len(got))
	}
	for _, field := range []string{"name", "path"} {
		if _, ok := got[0][field]; !ok {
			t.Errorf("missing field %q in log file entry", field)
		}
	}
	if got[0]["name"] != "syslog" {
		t.Errorf("name = %v, want syslog", got[0]["name"])
	}
	if got[0]["path"] != "/var/log/syslog" {
		t.Errorf("path = %v, want /var/log/syslog", got[0]["path"])
	}
}

func TestIntegrationServerLogShowJSON(t *testing.T) {
	body := map[string]any{
		"data": map[string]any{
			"logFile": map[string]any{
				"path":       "/var/log/syslog",
				"content":    "line1\nline2\n",
				"totalLines": 2,
			},
		},
	}

	cases := []struct {
		name         string
		args         []string
		wantLinesVar any // expected "lines" in GraphQL variables: float64 or nil
	}{
		{
			name:         "default sends lines=100",
			args:         []string{"log", "/var/log/syslog"},
			wantLinesVar: float64(100),
		},
		{
			name:         "lines=0 sends no lines variable",
			args:         []string{"log", "/var/log/syslog", "--lines", "0"},
			wantLinesVar: nil,
		},
		{
			name:         "explicit lines value is forwarded",
			args:         []string{"log", "/var/log/syslog", "--lines", "50"},
			wantLinesVar: float64(50),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, body)
			h.run(newServerCmd(h.preRun()), tc.args...)

			if h.vars["lines"] != tc.wantLinesVar {
				t.Errorf("lines variable = %v (%T), want %v (%T)",
					h.vars["lines"], h.vars["lines"],
					tc.wantLinesVar, tc.wantLinesVar)
			}

			got := mustDecodeJSON[map[string]any](t, h)
			for _, field := range []string{"path", "total_lines", "content"} {
				if _, ok := got[field]; !ok {
					t.Errorf("missing field %q in output", field)
				}
			}
			if got["path"] != "/var/log/syslog" {
				t.Errorf("path = %v, want /var/log/syslog", got["path"])
			}
			if got["total_lines"] != float64(2) {
				t.Errorf("total_lines = %v, want 2", got["total_lines"])
			}
		})
	}
}

func TestServerLogArgValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing path without --list",
			args:    []string{"log"},
			wantErr: "path required",
		},
		{
			name:    "list and path are mutually exclusive",
			args:    []string{"log", "--list", "/var/log/syslog"},
			wantErr: "cannot specify both --list and a path argument",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newServerCmd(func(*cobra.Command, []string) error { return nil })
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestIntegrationServerNotificationsJSON(t *testing.T) {
	body := map[string]any{
		"data": map[string]any{
			"notifications": map[string]any{
				"list": []map[string]any{
					{
						"id":                 "notif-1",
						"title":              "Docker - Prowlarr",
						"subject":            "Notice [NAS] - Version updated",
						"description":        "Version updated to 1.0.0",
						"importance":         "INFO",
						"type":               "UNREAD",
						"formattedTimestamp": "Thu 02 Apr 2026 12:10:12 AM",
					},
				},
			},
		},
	}

	cases := []struct {
		name          string
		args          []string
		wantNotifType string
		wantLimit     float64
	}{
		{name: "default sends UNREAD", args: []string{"notifications"}, wantNotifType: "UNREAD", wantLimit: 50},
		{name: "archive flag sends ARCHIVE", args: []string{"notifications", "--archive"}, wantNotifType: "ARCHIVE", wantLimit: 50},
		{name: "limit flag is forwarded", args: []string{"notifications", "--limit", "10"}, wantNotifType: "UNREAD", wantLimit: 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, body)
			h.run(newServerCmd(h.preRun()), tc.args...)

			if h.vars["notifType"] != tc.wantNotifType {
				t.Errorf("notifType = %v, want %v", h.vars["notifType"], tc.wantNotifType)
			}
			if h.vars["limit"] != tc.wantLimit {
				t.Errorf("limit = %v, want %v", h.vars["limit"], tc.wantLimit)
			}
		})
	}
}

func TestIntegrationServerNotificationsFieldsJSON(t *testing.T) {
	body := map[string]any{
		"data": map[string]any{
			"notifications": map[string]any{
				"list": []map[string]any{
					{
						"id":                 "notif-1",
						"title":              "Docker - Prowlarr",
						"subject":            "Notice [NAS] - Version updated",
						"description":        "Version updated to 1.0.0",
						"importance":         "INFO",
						"type":               "UNREAD",
						"formattedTimestamp": "Thu 02 Apr 2026 12:10:12 AM",
					},
				},
			},
		},
	}

	h := newHarness(t, body)
	h.run(newServerCmd(h.preRun()), "notifications")

	got := mustDecodeJSON[[]map[string]any](t, h)
	if len(got) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(got))
	}
	n := got[0]

	// importance must be lowercased
	if n["importance"] != "info" {
		t.Errorf("importance = %v, want \"info\"", n["importance"])
	}
	// formattedTimestamp must be mapped to the "timestamp" JSON key
	if n["timestamp"] != "Thu 02 Apr 2026 12:10:12 AM" {
		t.Errorf("timestamp = %v, want \"Thu 02 Apr 2026 12:10:12 AM\"", n["timestamp"])
	}
	if n["title"] != "Docker - Prowlarr" {
		t.Errorf("title = %v, want \"Docker - Prowlarr\"", n["title"])
	}
}

func TestIntegrationArrayStartJSON(t *testing.T) {
	h := newHarness(t, map[string]any{
		"data": map[string]any{
			"array": map[string]any{
				"setState": map[string]any{"id": "array-1", "state": "STARTED"},
			},
		},
	})
	h.run(newArrayCmd(h.preRun()), "start")

	got := mustDecodeJSON[actionResult](t, h)
	if !got.Success {
		t.Errorf("success = false, want true")
	}
	if got.Message != "Array is now started." {
		t.Errorf("message = %q, want %q", got.Message, "Array is now started.")
	}
}
