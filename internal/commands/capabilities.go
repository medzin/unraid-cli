package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/medzin/unraid-cli/internal/client"
)

type capabilityCheck struct {
	command    string
	conditions []capabilityCondition
}

type conditionScope int

const (
	scopeQuery conditionScope = iota
	scopeArrayMutations
	scopeDockerMutations
	scopeVmMutations
)

type capabilityCondition struct {
	scope     conditionScope
	fieldName string
}

var commandCapabilities = []capabilityCheck{
	{
		command:    "server version",
		conditions: []capabilityCondition{{scopeQuery, "info"}},
	},
	{
		command:    "server log",
		conditions: []capabilityCondition{{scopeQuery, "logFile"}},
	},
	{
		command:    "array status",
		conditions: []capabilityCondition{{scopeQuery, "array"}},
	},
	{
		command: "array start",
		conditions: []capabilityCondition{
			{scopeQuery, "array"},
			{scopeArrayMutations, "setState"},
		},
	},
	{
		command: "array stop",
		conditions: []capabilityCondition{
			{scopeQuery, "array"},
			{scopeArrayMutations, "setState"},
		},
	},
	{
		command:    "docker list",
		conditions: []capabilityCondition{{scopeQuery, "docker"}},
	},
	{
		command: "docker start",
		conditions: []capabilityCondition{
			{scopeQuery, "docker"},
			{scopeDockerMutations, "start"},
		},
	},
	{
		command: "docker stop",
		conditions: []capabilityCondition{
			{scopeQuery, "docker"},
			{scopeDockerMutations, "stop"},
		},
	},
	{
		command: "docker restart",
		conditions: []capabilityCondition{
			{scopeQuery, "docker"},
			{scopeDockerMutations, "start"},
			{scopeDockerMutations, "stop"},
		},
	},
	{
		command: "docker pause",
		conditions: []capabilityCondition{
			{scopeQuery, "docker"},
			{scopeDockerMutations, "pause"},
		},
	},
	{
		command: "docker unpause",
		conditions: []capabilityCondition{
			{scopeQuery, "docker"},
			{scopeDockerMutations, "unpause"},
		},
	},
	{
		command: "docker update",
		conditions: []capabilityCondition{
			{scopeQuery, "docker"},
			{scopeDockerMutations, "updateContainer"},
		},
	},
	{
		command:    "vm list",
		conditions: []capabilityCondition{{scopeQuery, "vms"}},
	},
	{
		command: "vm start",
		conditions: []capabilityCondition{
			{scopeQuery, "vms"},
			{scopeVmMutations, "start"},
		},
	},
	{
		command: "vm stop",
		conditions: []capabilityCondition{
			{scopeQuery, "vms"},
			{scopeVmMutations, "stop"},
		},
	},
	{
		command: "vm force-stop",
		conditions: []capabilityCondition{
			{scopeQuery, "vms"},
			{scopeVmMutations, "forceStop"},
		},
	},
	{
		command: "vm pause",
		conditions: []capabilityCondition{
			{scopeQuery, "vms"},
			{scopeVmMutations, "pause"},
		},
	},
	{
		command: "vm resume",
		conditions: []capabilityCondition{
			{scopeQuery, "vms"},
			{scopeVmMutations, "resume"},
		},
	},
	{
		command: "vm reboot",
		conditions: []capabilityCondition{
			{scopeQuery, "vms"},
			{scopeVmMutations, "reboot"},
		},
	},
	{
		command: "vm reset",
		conditions: []capabilityCondition{
			{scopeQuery, "vms"},
			{scopeVmMutations, "reset"},
		},
	},
}

func isCommandSupported(caps *client.SchemaCapabilities, check capabilityCheck) bool {
	for _, cond := range check.conditions {
		var fields map[string]bool
		switch cond.scope {
		case scopeQuery:
			fields = caps.QueryFields
		case scopeArrayMutations:
			fields = caps.ArrayMutations
		case scopeDockerMutations:
			fields = caps.DockerMutations
		case scopeVmMutations:
			fields = caps.VmMutations
		}
		if !fields[cond.fieldName] {
			return false
		}
	}
	return true
}

func newCapabilitiesCmd(preRun func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "capabilities",
		Short:   "Show which CLI commands are supported by the connected server",
		Args:    cobra.NoArgs,
		PreRunE: preRun,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ic := getIntrospectionClient(cmd.Context())
			caps, err := ic.FetchCapabilities(cmd.Context())
			if err != nil {
				return err
			}

			type capEntry struct {
				Command   string `json:"command"`
				Supported bool   `json:"supported"`
			}
			entries := make([]capEntry, len(commandCapabilities))
			for i, check := range commandCapabilities {
				entries[i] = capEntry{
					Command:   check.command,
					Supported: isCommandSupported(caps, check),
				}
			}

			return render(cmd.Context(), entries, func() error {
				w := getOutputWriter(cmd.Context())
				if _, err := fmt.Fprintf(w, "Capabilities for %s\n\n", ic.URL); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "%-20s  %s\n", "COMMAND", "STATUS"); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(w, strings.Repeat("-", 40)); err != nil {
					return err
				}
				for _, check := range commandCapabilities {
					status := "not available"
					if isCommandSupported(caps, check) {
						status = "supported"
					}
					if _, err := fmt.Fprintf(w, "%-20s  %s\n", check.command, status); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	return cmd
}
