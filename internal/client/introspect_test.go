package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldsToSet(t *testing.T) {
	cases := []struct {
		name   string
		fields []introspectFieldEntry
		want   map[string]bool
	}{
		{
			name:   "nil slice returns empty map",
			fields: nil,
			want:   map[string]bool{},
		},
		{
			name:   "empty slice returns empty map",
			fields: []introspectFieldEntry{},
			want:   map[string]bool{},
		},
		{
			name:   "single field",
			fields: []introspectFieldEntry{{Name: "start"}},
			want:   map[string]bool{"start": true},
		},
		{
			name:   "multiple fields",
			fields: []introspectFieldEntry{{Name: "start"}, {Name: "stop"}, {Name: "pause"}},
			want:   map[string]bool{"start": true, "stop": true, "pause": true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fieldsToSet(tc.fields)
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d entries, got %d: %v", len(tc.want), len(got), got)
			}
			for k := range tc.want {
				if !got[k] {
					t.Errorf("expected key %q to be present", k)
				}
			}
		})
	}
}

func TestTypeToSet(t *testing.T) {
	t.Run("nil type returns nil map", func(t *testing.T) {
		if typeToSet(nil) != nil {
			t.Error("expected nil for nil type")
		}
	})

	t.Run("non-nil type with fields returns set", func(t *testing.T) {
		got := typeToSet(&introspectTypeResult{
			Fields: []introspectFieldEntry{{Name: "start"}, {Name: "stop"}},
		})
		if !got["start"] || !got["stop"] {
			t.Errorf("unexpected set: %v", got)
		}
	})

	t.Run("non-nil type with no fields returns empty map", func(t *testing.T) {
		got := typeToSet(&introspectTypeResult{Fields: nil})
		if got == nil {
			t.Error("expected non-nil map for non-nil type")
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got: %v", got)
		}
	})
}

func TestFetchCapabilities(t *testing.T) {
	allPresentResponse := map[string]any{
		"data": map[string]any{
			"vmMutations": map[string]any{
				"fields": []map[string]any{
					{"name": "start"},
					{"name": "stop"},
					{"name": "forceStop"},
					{"name": "pause"},
					{"name": "resume"},
					{"name": "reboot"},
					{"name": "reset"},
				},
			},
			"dockerMutations": map[string]any{
				"fields": []map[string]any{
					{"name": "start"}, {"name": "stop"}, {"name": "updateContainer"},
				},
			},
			"schema": map[string]any{
				"queryType": map[string]any{
					"fields": []map[string]any{
						{"name": "docker"}, {"name": "vms"},
					},
				},
			},
		},
	}

	cases := []struct {
		name            string
		responseBody    any
		statusCode      int
		wantErr         bool
		wantErrContains string
		check           func(t *testing.T, caps *SchemaCapabilities)
	}{
		{
			name:         "all types present",
			responseBody: allPresentResponse,
			statusCode:   http.StatusOK,
			check: func(t *testing.T, caps *SchemaCapabilities) {
				if !caps.QueryFields["docker"] || !caps.QueryFields["vms"] {
					t.Error("expected docker and vms in QueryFields")
				}
				if !caps.VmMutations["start"] || !caps.VmMutations["forceStop"] {
					t.Error("expected start and forceStop in VmMutations")
				}
				if !caps.DockerMutations["start"] || !caps.DockerMutations["updateContainer"] {
					t.Error("expected start and updateContainer in DockerMutations")
				}
			},
		},
		{
			name: "VmMutations type absent (null in response)",
			responseBody: map[string]any{
				"data": map[string]any{
					"vmMutations":     nil,
					"dockerMutations": map[string]any{"fields": []map[string]any{{"name": "start"}}},
					"schema":          map[string]any{"queryType": map[string]any{"fields": []map[string]any{{"name": "docker"}}}},
				},
			},
			statusCode: http.StatusOK,
			check: func(t *testing.T, caps *SchemaCapabilities) {
				if caps.VmMutations != nil {
					t.Errorf("expected nil VmMutations, got %v", caps.VmMutations)
				}
				if !caps.DockerMutations["start"] {
					t.Error("expected DockerMutations to have start")
				}
			},
		},
		{
			name: "DockerMutations type absent (null in response)",
			responseBody: map[string]any{
				"data": map[string]any{
					"vmMutations":     map[string]any{"fields": []map[string]any{{"name": "start"}}},
					"dockerMutations": nil,
					"schema":          map[string]any{"queryType": map[string]any{"fields": []map[string]any{{"name": "vms"}}}},
				},
			},
			statusCode: http.StatusOK,
			check: func(t *testing.T, caps *SchemaCapabilities) {
				if caps.DockerMutations != nil {
					t.Errorf("expected nil DockerMutations, got %v", caps.DockerMutations)
				}
				if !caps.VmMutations["start"] {
					t.Error("expected VmMutations to have start")
				}
			},
		},
		{
			name: "GraphQL errors in response",
			responseBody: map[string]any{
				"errors": []map[string]any{{"message": "not authorized"}},
			},
			statusCode:      http.StatusOK,
			wantErr:         true,
			wantErrContains: "not authorized",
		},
		{
			name:            "invalid JSON response",
			responseBody:    "not json",
			statusCode:      http.StatusOK,
			wantErr:         true,
			wantErrContains: "failed to decode",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				_ = json.NewEncoder(w).Encode(tc.responseBody)
			}))
			defer srv.Close()

			ic := &IntrospectionClient{URL: srv.URL, httpClient: srv.Client()}
			caps, err := ic.FetchCapabilities(context.Background())

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("expected error containing %q, got: %v", tc.wantErrContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.check(t, caps)
		})
	}
}

func TestFetchCapabilitiesRequestFormat(t *testing.T) {
	var gotReq struct {
		Query string `json:"query"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"vmMutations": nil, "dockerMutations": nil,
			"schema": map[string]any{"queryType": map[string]any{"fields": nil}},
		}})
	}))
	defer srv.Close()

	ic := &IntrospectionClient{URL: srv.URL, httpClient: srv.Client()}
	_, _ = ic.FetchCapabilities(context.Background())
}
