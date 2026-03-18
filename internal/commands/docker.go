package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/amedzinski/unraid-cli/internal/client"
)

type container = client.GetDockerContainersDockerContainersDockerContainer

func newDockerCmd(preRun func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker container management",
	}

	listCmd := newDockerListCmd()
	listCmd.PreRunE = preRun

	startCmd := newDockerStartCmd()
	startCmd.PreRunE = preRun

	stopCmd := newDockerStopCmd()
	stopCmd.PreRunE = preRun

	restartCmd := newDockerRestartCmd()
	restartCmd.PreRunE = preRun

	updateCmd := newDockerUpdateCmd()
	updateCmd.PreRunE = preRun

	cmd.AddCommand(listCmd, startCmd, stopCmd, restartCmd, updateCmd)

	return cmd
}

func newDockerListCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "list-containers"},
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
	id, err := resolveContainerID(ctx, c, name)
	if err != nil {
		return err
	}

	fmt.Printf("Starting container '%s'...\n", name)
	resp, err := client.StartDockerContainer(ctx, c, id)
	if err != nil {
		return err
	}

	state := strings.ToLower(string(resp.Docker.Start.State))
	fmt.Printf("Container '%s' is now %s.\n", name, state)
	return nil
}

func stopContainer(ctx context.Context, c graphql.Client, name string) error {
	id, err := resolveContainerID(ctx, c, name)
	if err != nil {
		return err
	}

	fmt.Printf("Stopping container '%s'...\n", name)
	resp, err := client.StopDockerContainer(ctx, c, id)
	if err != nil {
		return err
	}

	state := strings.ToLower(string(resp.Docker.Stop.State))
	fmt.Printf("Container '%s' is now %s.\n", name, state)
	return nil
}

func restartContainer(ctx context.Context, c graphql.Client, name string) error {
	id, err := resolveContainerID(ctx, c, name)
	if err != nil {
		return err
	}

	fmt.Printf("Restarting container '%s'...\n", name)
	_, err = client.StopDockerContainer(ctx, c, id)
	if err != nil {
		return err
	}

	resp, err := client.StartDockerContainer(ctx, c, id)
	if err != nil {
		return err
	}

	state := strings.ToLower(string(resp.Docker.Start.State))
	fmt.Printf("Container '%s' is now %s.\n", name, state)
	return nil
}

func updateContainer(ctx context.Context, c graphql.Client, name string) error {
	id, err := resolveContainerID(ctx, c, name)
	if err != nil {
		return err
	}

	fmt.Printf("Updating container '%s'...\n", name)
	resp, err := client.UpdateDockerContainer(ctx, c, id)
	if err != nil {
		return err
	}

	state := strings.ToLower(string(resp.Docker.UpdateContainer.State))
	fmt.Printf("Container '%s' updated successfully (state: %s).\n", name, state)
	return nil
}

func listContainers(ctx context.Context, c graphql.Client, showAll bool) error {
	resp, err := client.GetDockerContainers(ctx, c)
	if err != nil {
		return err
	}

	containers := filterContainersByState(resp.Docker.Containers, showAll)

	if len(containers) == 0 {
		if showAll {
			fmt.Println("No containers found.")
		} else {
			fmt.Println("No running containers found. Use --all to show all containers.")
		}
		return nil
	}

	fmt.Printf("%-30s %-40s %-10s %-20s\n", "NAME", "IMAGE", "STATE", "STATUS")
	fmt.Println(strings.Repeat("-", 100))

	for _, ct := range containers {
		name := "unnamed"
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}

		state := strings.ToLower(string(ct.State))

		fmt.Printf("%-30s %-40s %-10s %-20s\n",
			truncate(name, 29),
			truncate(ct.Image, 39),
			state,
			truncate(ct.Status, 19),
		)
	}

	return nil
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
