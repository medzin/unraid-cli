package commands

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/medzin/unraid-cli/internal/client"
)

type container = client.GetDockerContainersDockerContainersDockerContainer

func newDockerCmd(preRun func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker container management",
	}

	cmd.AddCommand(
		newDockerListCmd(),
		newDockerLogsCmd(),
		newDockerStartCmd(),
		newDockerStopCmd(),
		newDockerRestartCmd(),
		newDockerPauseCmd(),
		newDockerUnpauseCmd(),
		newDockerUpdateCmd(),
	)
	for _, sub := range cmd.Commands() {
		sub.PreRunE = preRun
	}

	return cmd
}

func newDockerListCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Docker containers",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listContainers(cmd.Context(), getClient(cmd.Context()), all)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "show all containers (default: only running)")
	return cmd
}

func newDockerStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a Docker container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return startContainer(cmd.Context(), getClient(cmd.Context()), args[0])
		},
	}
}

func newDockerStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a Docker container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopContainer(cmd.Context(), getClient(cmd.Context()), args[0])
		},
	}
}

func newDockerRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a Docker container (stop then start)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return restartContainer(cmd.Context(), getClient(cmd.Context()), args[0])
		},
	}
}

func newDockerPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause <name>",
		Short: "Pause a Docker container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pauseContainer(cmd.Context(), getClient(cmd.Context()), args[0])
		},
	}
}

func newDockerUnpauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unpause <name>",
		Short: "Unpause a Docker container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return unpauseContainer(cmd.Context(), getClient(cmd.Context()), args[0])
		},
	}
}

func newDockerUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <name>",
		Short: "Update a Docker container to the latest image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateContainer(cmd.Context(), getClient(cmd.Context()), args[0])
		},
	}
}

func newDockerLogsCmd() *cobra.Command {
	var tail int

	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Fetch container logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return containerLogs(cmd.Context(), getClient(cmd.Context()), args[0], tail)
		},
	}

	cmd.Flags().IntVarP(&tail, "lines", "n", 100, "number of lines to show from the end")
	return cmd
}

func containerLogs(ctx context.Context, c graphql.Client, name string, tail int) error {
	id, err := resolveContainerID(ctx, c, name)
	if err != nil {
		return err
	}
	resp, err := client.GetContainerLogs(ctx, c, id, &tail)
	if err != nil {
		return err
	}
	lines := resp.Docker.Logs.Lines
	return render(ctx, lines, func() error {
		w := getOutputWriter(ctx)
		for _, line := range lines {
			if _, err := fmt.Fprintf(w, "%s  %s\n", line.Timestamp, line.Message); err != nil {
				return err
			}
		}
		return nil
	})
}

type containerMutationFunc func(ctx context.Context, c graphql.Client, id string) (string, error)

func containerAction(ctx context.Context, c graphql.Client, name, presentVerb string, mutation containerMutationFunc) error {
	id, err := resolveContainerID(ctx, c, name)
	if err != nil {
		return err
	}
	log.Printf("%s container '%s'...", presentVerb, name)
	state, err := mutation(ctx, c, id)
	if err != nil {
		return err
	}
	return printAction(ctx, fmt.Sprintf("Container '%s' is now %s.", name, state))
}

func resolveContainerID(ctx context.Context, c graphql.Client, name string) (string, error) {
	resp, err := client.GetDockerContainers(ctx, c)
	if err != nil {
		return "", err
	}
	return findContainerID(resp.Docker.Containers, name)
}

func findContainerID(containers []container, name string) (string, error) {
	nameLower := strings.ToLower(name)

	for _, ct := range containers {
		for _, cName := range ct.Names {
			clean := strings.ToLower(strings.TrimPrefix(cName, "/"))
			if clean == nameLower {
				return ct.Id, nil
			}
		}
	}

	return "", fmt.Errorf("container '%s' not found. Use 'docker list --all' to see available containers", name)
}

func startContainer(ctx context.Context, c graphql.Client, name string) error {
	return containerAction(ctx, c, name, "Starting", func(ctx context.Context, c graphql.Client, id string) (string, error) {
		resp, err := client.StartDockerContainer(ctx, c, id)
		if err != nil {
			return "", err
		}
		return strings.ToLower(string(resp.Docker.Start.State)), nil
	})
}

func stopContainer(ctx context.Context, c graphql.Client, name string) error {
	return containerAction(ctx, c, name, "Stopping", func(ctx context.Context, c graphql.Client, id string) (string, error) {
		resp, err := client.StopDockerContainer(ctx, c, id)
		if err != nil {
			return "", err
		}
		return strings.ToLower(string(resp.Docker.Stop.State)), nil
	})
}

func pauseContainer(ctx context.Context, c graphql.Client, name string) error {
	return containerAction(ctx, c, name, "Pausing", func(ctx context.Context, c graphql.Client, id string) (string, error) {
		resp, err := client.PauseDockerContainer(ctx, c, id)
		if err != nil {
			return "", err
		}
		return strings.ToLower(string(resp.Docker.Pause.State)), nil
	})
}

func unpauseContainer(ctx context.Context, c graphql.Client, name string) error {
	return containerAction(ctx, c, name, "Unpausing", func(ctx context.Context, c graphql.Client, id string) (string, error) {
		resp, err := client.UnpauseDockerContainer(ctx, c, id)
		if err != nil {
			return "", err
		}
		return strings.ToLower(string(resp.Docker.Unpause.State)), nil
	})
}

func restartContainer(ctx context.Context, c graphql.Client, name string) error {
	return containerAction(ctx, c, name, "Restarting", func(ctx context.Context, c graphql.Client, id string) (string, error) {
		if _, err := client.StopDockerContainer(ctx, c, id); err != nil {
			return "", err
		}
		resp, err := client.StartDockerContainer(ctx, c, id)
		if err != nil {
			return "", err
		}
		return strings.ToLower(string(resp.Docker.Start.State)), nil
	})
}

func updateContainer(ctx context.Context, c graphql.Client, name string) error {
	id, err := resolveContainerID(ctx, c, name)
	if err != nil {
		return err
	}

	log.Printf("Updating container '%s'...", name)
	resp, err := client.UpdateDockerContainer(ctx, c, id)
	if err != nil {
		return err
	}

	state := strings.ToLower(string(resp.Docker.UpdateContainer.State))
	return printAction(ctx, fmt.Sprintf("Container '%s' updated successfully (state: %s).", name, state))
}

func listContainers(ctx context.Context, c graphql.Client, showAll bool) error {
	resp, err := client.GetDockerContainers(ctx, c)
	if err != nil {
		return err
	}

	containers := filterContainersByState(resp.Docker.Containers, showAll)

	if containers == nil {
		containers = []container{}
	}

	return render(ctx, containers, func() error {
		w := getOutputWriter(ctx)
		if len(containers) == 0 {
			if showAll {
				_, err := fmt.Fprintln(w, "No containers found.")
				return err
			}
			_, err := fmt.Fprintln(w, "No running containers found. Use --all to show all containers.")
			return err
		}

		if _, err := fmt.Fprintf(w, "%-30s %-40s %-10s %-20s\n", "NAME", "IMAGE", "STATE", "STATUS"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, strings.Repeat("-", 100)); err != nil {
			return err
		}

		for _, ct := range containers {
			name := "unnamed"
			if len(ct.Names) > 0 {
				name = strings.TrimPrefix(ct.Names[0], "/")
			}
			if _, err := fmt.Fprintf(w, "%-30s %-40s %-10s %-20s\n",
				truncate(name, 29),
				truncate(ct.Image, 39),
				strings.ToLower(string(ct.State)),
				truncate(ct.Status, 19),
			); err != nil {
				return err
			}
		}

		return nil
	})
}

func filterContainersByState(containers []container, showAll bool) []container {
	if showAll {
		return containers
	}
	var filtered []container
	for _, ct := range containers {
		if ct.State == client.ContainerStateRunning {
			filtered = append(filtered, ct)
		}
	}
	return filtered
}
