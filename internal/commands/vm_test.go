package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/medzin/unraid-cli/internal/client"
)

func strPtr(s string) *string { return &s }

func sampleVM(id, name string, state client.VmState) vmDomain {
	return vmDomain{Id: id, Name: strPtr(name), State: state}
}

func sampleVMs() []vmDomain {
	return []vmDomain{
		sampleVM("vm-1", "Windows 11", client.VmStateRunning),
		sampleVM("vm-2", "Ubuntu Server", client.VmStateRunning),
		sampleVM("vm-3", "macOS", client.VmStateShutoff),
		sampleVM("vm-4", "Debian", client.VmStatePaused),
	}
}

func TestFindVmID(t *testing.T) {
	cases := []struct {
		name    string
		vms     []vmDomain
		input   string
		wantID  string
		wantErr string
	}{
		{"exact match", sampleVMs(), "Windows 11", "vm-1", ""},
		{"case insensitive lower", sampleVMs(), "windows 11", "vm-1", ""},
		{"case insensitive upper", sampleVMs(), "UBUNTU SERVER", "vm-2", ""},
		{"unknown name", sampleVMs(), "nonexistent", "", "not found"},
		{"empty list", nil, "anything", "", "not found"},
		{"skips nil name", []vmDomain{{Id: "vm-noname", Name: nil, State: client.VmStateRunning}}, "something", "", "not found"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := findVmID(tc.vms, tc.input)
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
			if id != tc.wantID {
				t.Errorf("expected %s, got %s", tc.wantID, id)
			}
		})
	}
}

func TestFilterVmsByState(t *testing.T) {
	cases := []struct {
		name      string
		vms       []vmDomain
		showAll   bool
		wantCount int
	}{
		{"all VMs when showAll", sampleVMs(), true, 4},
		{"only running when not showAll", sampleVMs(), false, 2},
		{"empty when no running", []vmDomain{
			sampleVM("vm-1", "a", client.VmStateShutoff),
			sampleVM("vm-2", "b", client.VmStatePaused),
		}, false, 0},
		{"empty input", nil, false, 0},
		{"empty input showAll", nil, true, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := filterVmsByState(tc.vms, tc.showAll)
			if len(filtered) != tc.wantCount {
				t.Errorf("expected %d VMs, got %d", tc.wantCount, len(filtered))
			}
		})
	}
}

func TestFormatVmState(t *testing.T) {
	cases := []struct {
		state    client.VmState
		expected string
	}{
		{client.VmStateRunning, "running"},
		{client.VmStatePaused, "paused"},
		{client.VmStateShutdown, "shutdown"},
		{client.VmStateShutoff, "shutoff"},
		{client.VmStateIdle, "idle"},
		{client.VmStateCrashed, "crashed"},
		{client.VmStatePmsuspended, "suspended"},
		{client.VmStateNostate, "no state"},
		{client.VmState("CUSTOM"), "unknown"},
	}

	for _, tc := range cases {
		got := formatVmState(tc.state)
		if got != tc.expected {
			t.Errorf("formatVmState(%q) = %q, want %q", tc.state, got, tc.expected)
		}
	}
}

func TestMapVmsUnavailable(t *testing.T) {
	cases := []struct {
		input    string
		contains string
	}{
		{"GraphQL errors: Failed to retrieve VM domains: VMs are not available", "VM service enabled"},
		{"connection refused", "connection refused"},
	}

	for _, tc := range cases {
		err := mapVmsUnavailable(fmt.Errorf("%s", tc.input))
		if !strings.Contains(err.Error(), tc.contains) {
			t.Errorf("mapVmsUnavailable(%q) should contain %q, got: %v", tc.input, tc.contains, err)
		}
	}
}
