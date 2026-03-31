package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"
)

// mockGQLServer starts a test HTTP server that always responds with body,
// regardless of which GraphQL operation was sent.
func mockGQLServer(t *testing.T, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Errorf("mock server failed to encode response: %v", err)
		}
	}))
}

// jsonPreRun returns a PreRunE that injects the GraphQL client and JSON output
// writer into the command context, bypassing server config resolution.
func jsonPreRun(c graphql.Client, w io.Writer) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := withClient(cmd.Context(), c)
		ctx = withOutputFormat(ctx, outputJSON)
		ctx = withOutputWriter(ctx, w)
		cmd.SetContext(ctx)
		return nil
	}
}

func TestIntegrationDockerListJSON(t *testing.T) {
	srv := mockGQLServer(t, map[string]any{
		"data": map[string]any{
			"docker": map[string]any{
				"containers": []map[string]any{
					{
						"id":     "ctr-1",
						"names":  []string{"/plex"},
						"image":  "linuxserver/plex:latest",
						"state":  "RUNNING",
						"status": "Up 2 hours",
					},
				},
			},
		},
	})
	defer srv.Close()

	var buf bytes.Buffer
	cmd := newDockerCmd(jsonPreRun(graphql.NewClient(srv.URL, srv.Client()), &buf))
	cmd.SetArgs([]string{"list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 container, got %d", len(got))
	}
	for _, field := range []string{"id", "names", "image", "state", "status"} {
		if _, ok := got[0][field]; !ok {
			t.Errorf("missing field %q in container JSON", field)
		}
	}
	if got[0]["id"] != "ctr-1" {
		t.Errorf("id = %v, want ctr-1", got[0]["id"])
	}
	if got[0]["state"] != "RUNNING" {
		t.Errorf("state = %v, want RUNNING", got[0]["state"])
	}
}

func TestIntegrationArrayStatusJSON(t *testing.T) {
	srv := mockGQLServer(t, map[string]any{
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
	defer srv.Close()

	var buf bytes.Buffer
	cmd := newArrayCmd(jsonPreRun(graphql.NewClient(srv.URL, srv.Client()), &buf))
	cmd.SetArgs([]string{"status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
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
	// The mock must satisfy both GetDockerContainers (resolveContainerID)
	// and GetContainerLogs in a single response shape.
	srv := mockGQLServer(t, map[string]any{
		"data": map[string]any{
			"docker": map[string]any{
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
	})
	defer srv.Close()

	var buf bytes.Buffer
	cmd := newDockerLogsCmd()
	cmd.PreRunE = jsonPreRun(graphql.NewClient(srv.URL, srv.Client()), &buf)
	cmd.SetArgs([]string{"plex"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(got))
	}
	for _, field := range []string{"timestamp", "message"} {
		if _, ok := got[0][field]; !ok {
			t.Errorf("missing field %q in log line JSON", field)
		}
	}
}

func TestIntegrationServerVersionJSON(t *testing.T) {
	srv := mockGQLServer(t, map[string]any{
		"data": map[string]any{
			"info": map[string]any{
				"versions": map[string]any{
					"core": map[string]any{
						"unraid": "6.12.10",
						"api":    "4.5.0",
					},
				},
			},
		},
	})
	defer srv.Close()

	var buf bytes.Buffer
	cmd := newServerVersionCmd(jsonPreRun(graphql.NewClient(srv.URL, srv.Client()), &buf))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["unraid_os"] != "6.12.10" {
		t.Errorf("unraid_os = %v, want 6.12.10", got["unraid_os"])
	}
	if got["api"] != "4.5.0" {
		t.Errorf("api = %v, want 4.5.0", got["api"])
	}
}

func TestIntegrationArrayStartJSON(t *testing.T) {
	srv := mockGQLServer(t, map[string]any{
		"data": map[string]any{
			"array": map[string]any{
				"setState": map[string]any{
					"id":    "array-1",
					"state": "STARTED",
				},
			},
		},
	})
	defer srv.Close()

	var buf bytes.Buffer
	cmd := newArrayCmd(jsonPreRun(graphql.NewClient(srv.URL, srv.Client()), &buf))
	cmd.SetArgs([]string{"start"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got actionResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if !got.Success {
		t.Errorf("success = false, want true")
	}
	if got.Message != "Array is now started." {
		t.Errorf("message = %q, want %q", got.Message, "Array is now started.")
	}
}
