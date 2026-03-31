package commands

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/medzin/unraid-cli/internal/client"
)

type vmDomain = client.GetVmsVmsDomainsVmDomain

type vmMutationFunc func(ctx context.Context, c graphql.Client, id string) error

func newVmCmd(preRun func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Virtual machine management",
	}

	cmd.AddCommand(
		newVmListCmd(),
		newVmStartCmd(),
		newVmStopCmd(),
		newVmForceStopCmd(),
		newVmPauseCmd(),
		newVmResumeCmd(),
		newVmRebootCmd(),
		newVmResetCmd(),
	)
	for _, sub := range cmd.Commands() {
		sub.PreRunE = preRun
	}

	return cmd
}

func newVmListCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List virtual machines",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listVms(cmd.Context(), getClient(cmd.Context()), all)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "show all VMs (default: only running)")
	return cmd
}

func newVmStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vmAction(cmd.Context(), getClient(cmd.Context()), args[0], "Starting", "start", func(ctx context.Context, c graphql.Client, id string) error {
				_, err := client.StartVm(ctx, c, id)
				return err
			})
		},
	}
}

func newVmStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a virtual machine (graceful shutdown)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vmAction(cmd.Context(), getClient(cmd.Context()), args[0], "Stopping", "stop", func(ctx context.Context, c graphql.Client, id string) error {
				_, err := client.StopVm(ctx, c, id)
				return err
			})
		},
	}
}

func newVmForceStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "force-stop <name>",
		Short: "Force stop a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vmAction(cmd.Context(), getClient(cmd.Context()), args[0], "Force stopping", "force stop", func(ctx context.Context, c graphql.Client, id string) error {
				_, err := client.ForceStopVm(ctx, c, id)
				return err
			})
		},
	}
}

func newVmPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause <name>",
		Short: "Pause a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vmAction(cmd.Context(), getClient(cmd.Context()), args[0], "Pausing", "pause", func(ctx context.Context, c graphql.Client, id string) error {
				_, err := client.PauseVm(ctx, c, id)
				return err
			})
		},
	}
}

func newVmResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a paused virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vmAction(cmd.Context(), getClient(cmd.Context()), args[0], "Resuming", "resume", func(ctx context.Context, c graphql.Client, id string) error {
				_, err := client.ResumeVm(ctx, c, id)
				return err
			})
		},
	}
}

func newVmRebootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reboot <name>",
		Short: "Reboot a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vmAction(cmd.Context(), getClient(cmd.Context()), args[0], "Rebooting", "reboot", func(ctx context.Context, c graphql.Client, id string) error {
				_, err := client.RebootVm(ctx, c, id)
				return err
			})
		},
	}
}

func newVmResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <name>",
		Short: "Reset a virtual machine (hard reboot)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vmAction(cmd.Context(), getClient(cmd.Context()), args[0], "Resetting", "reset", func(ctx context.Context, c graphql.Client, id string) error {
				_, err := client.ResetVm(ctx, c, id)
				return err
			})
		},
	}
}

func vmAction(ctx context.Context, c graphql.Client, name, presentVerb, pastVerb string, mutation vmMutationFunc) error {
	id, err := resolveVmID(ctx, c, name)
	if err != nil {
		return err
	}

	log.Printf("%s VM '%s'...", presentVerb, name)

	if err := mutation(ctx, c, id); err != nil {
		return err
	}

	return printAction(ctx, fmt.Sprintf("VM '%s' %s command sent.", name, pastVerb))
}

func resolveVmID(ctx context.Context, c graphql.Client, name string) (string, error) {
	resp, err := client.GetVms(ctx, c)
	if err != nil {
		return "", mapVmsUnavailable(err)
	}
	return findVmID(resp.Vms.Domains, name)
}

func findVmID(domains []vmDomain, name string) (string, error) {
	nameLower := strings.ToLower(name)

	for _, domain := range domains {
		if domain.Name != nil && strings.ToLower(*domain.Name) == nameLower {
			return domain.Id, nil
		}
	}

	return "", fmt.Errorf("VM '%s' not found. Use 'vm list --all' to see available VMs", name)
}

func mapVmsUnavailable(err error) error {
	if strings.Contains(err.Error(), "VMs are not available") {
		return errors.New("VMs are not available on this server. Is the VM service enabled?")
	}
	return err
}

func listVms(ctx context.Context, c graphql.Client, showAll bool) error {
	resp, err := client.GetVms(ctx, c)
	if err != nil {
		return mapVmsUnavailable(err)
	}

	domains := filterVmsByState(resp.Vms.Domains, showAll)

	if domains == nil {
		domains = []vmDomain{}
	}

	return render(ctx, domains, func() error {
		w := getOutputWriter(ctx)
		if len(domains) == 0 {
			if showAll {
				_, err := fmt.Fprintln(w, "No VMs found.")
				return err
			}
			_, err := fmt.Fprintln(w, "No running VMs found. Use --all to show all VMs.")
			return err
		}

		if _, err := fmt.Fprintf(w, "%-30s %-12s\n", "NAME", "STATE"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, strings.Repeat("-", 42)); err != nil {
			return err
		}

		for _, vm := range domains {
			name := "unnamed"
			if vm.Name != nil {
				name = *vm.Name
			}
			if _, err := fmt.Fprintf(w, "%-30s %-12s\n", truncate(name, 29), formatVmState(vm.State)); err != nil {
				return err
			}
		}

		return nil
	})
}

func filterVmsByState(domains []vmDomain, showAll bool) []vmDomain {
	if showAll {
		return domains
	}
	var filtered []vmDomain
	for _, d := range domains {
		if d.State == client.VmStateRunning {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func formatVmState(state client.VmState) string {
	switch state {
	case client.VmStateRunning:
		return "running"
	case client.VmStatePaused:
		return "paused"
	case client.VmStateShutdown:
		return "shutdown"
	case client.VmStateShutoff:
		return "shutoff"
	case client.VmStateIdle:
		return "idle"
	case client.VmStateCrashed:
		return "crashed"
	case client.VmStatePmsuspended:
		return "suspended"
	case client.VmStateNostate:
		return "no state"
	default:
		return "unknown"
	}
}
