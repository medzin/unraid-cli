package commands

import (
	"testing"

	"github.com/medzin/unraid-cli/internal/client"
)

func fullCaps() *client.SchemaCapabilities {
	return &client.SchemaCapabilities{
		QueryFields:     map[string]bool{"docker": true, "vms": true},
		DockerMutations: map[string]bool{"start": true, "stop": true, "pause": true, "unpause": true, "updateContainer": true},
		VmMutations:     map[string]bool{"start": true, "stop": true, "forceStop": true, "pause": true, "resume": true, "reboot": true, "reset": true},
	}
}

func TestIsCommandSupported(t *testing.T) {
	cases := []struct {
		name string
		caps *client.SchemaCapabilities
		cmd  string
		want bool
	}{
		// All supported on a fully-capable server
		{"docker list - full caps", fullCaps(), "docker list", true},
		{"docker start - full caps", fullCaps(), "docker start", true},
		{"docker stop - full caps", fullCaps(), "docker stop", true},
		{"docker restart - full caps", fullCaps(), "docker restart", true},
		{"docker pause - full caps", fullCaps(), "docker pause", true},
		{"docker unpause - full caps", fullCaps(), "docker unpause", true},
		{"docker update - full caps", fullCaps(), "docker update", true},
		{"vm list - full caps", fullCaps(), "vm list", true},
		{"vm start - full caps", fullCaps(), "vm start", true},
		{"vm stop - full caps", fullCaps(), "vm stop", true},
		{"vm force-stop - full caps", fullCaps(), "vm force-stop", true},
		{"vm pause - full caps", fullCaps(), "vm pause", true},
		{"vm resume - full caps", fullCaps(), "vm resume", true},
		{"vm reboot - full caps", fullCaps(), "vm reboot", true},
		{"vm reset - full caps", fullCaps(), "vm reset", true},

		// VmMutations type absent
		{"vm list - vms query absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true},
			DockerMutations: fullCaps().DockerMutations,
			VmMutations:     nil,
		}, "vm list", false},
		{"vm start - VmMutations nil", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true, "vms": true},
			DockerMutations: fullCaps().DockerMutations,
			VmMutations:     nil,
		}, "vm start", false},

		// DockerMutations type absent
		{"docker list - docker query absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"vms": true},
			DockerMutations: nil,
			VmMutations:     fullCaps().VmMutations,
		}, "docker list", false},
		{"docker start - DockerMutations nil", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true, "vms": true},
			DockerMutations: nil,
			VmMutations:     fullCaps().VmMutations,
		}, "docker start", false},

		// Missing specific mutation field
		{"docker update - updateContainer absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true},
			DockerMutations: map[string]bool{"start": true, "stop": true},
			VmMutations:     fullCaps().VmMutations,
		}, "docker update", false},
		{"vm force-stop - forceStop absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true, "vms": true},
			DockerMutations: fullCaps().DockerMutations,
			VmMutations:     map[string]bool{"start": true, "stop": true},
		}, "vm force-stop", false},

		// pause / unpause absent
		{"docker pause - pause absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true},
			DockerMutations: map[string]bool{"start": true, "stop": true},
			VmMutations:     fullCaps().VmMutations,
		}, "docker pause", false},
		{"docker unpause - unpause absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true},
			DockerMutations: map[string]bool{"start": true, "stop": true},
			VmMutations:     fullCaps().VmMutations,
		}, "docker unpause", false},

		// docker restart requires both start and stop
		{"docker restart - stop absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true},
			DockerMutations: map[string]bool{"start": true},
			VmMutations:     fullCaps().VmMutations,
		}, "docker restart", false},
		{"docker restart - start absent", &client.SchemaCapabilities{
			QueryFields:     map[string]bool{"docker": true},
			DockerMutations: map[string]bool{"stop": true},
			VmMutations:     fullCaps().VmMutations,
		}, "docker restart", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := findCommandCheck(tc.cmd)
			if check == nil {
				t.Fatalf("command %q not found in commandCapabilities", tc.cmd)
				return
			}
			got := isCommandSupported(tc.caps, *check)
			if got != tc.want {
				t.Errorf("isCommandSupported(%q) = %v, want %v", tc.cmd, got, tc.want)
			}
		})
	}
}

func findCommandCheck(name string) *capabilityCheck {
	for i := range commandCapabilities {
		if commandCapabilities[i].command == name {
			return &commandCapabilities[i]
		}
	}
	return nil
}
