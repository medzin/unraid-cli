package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amedzinski/unraid-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage server configurations",
	}

	cmd.AddCommand(newConfigAddCmd())
	cmd.AddCommand(newConfigRemoveCmd())
	cmd.AddCommand(newConfigDefaultCmd())
	cmd.AddCommand(newConfigListCmd())

	return cmd
}

func newConfigAddCmd() *cobra.Command {
	var (
		url    string
		apiKey string
	)

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new server configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			isFirst := len(cfg.Servers) == 0
			cfg.AddServer(name, url, apiKey)

			if isFirst {
				cfg.Default = name
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Server '%s' added successfully.\n", name)
			if isFirst {
				fmt.Println("Set as default server.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "server URL (e.g., https://192.168.1.100)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("api-key")

	return cmd
}

func newConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a server configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if cfg.RemoveServer(name) {
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Printf("Server '%s' removed successfully.\n", name)
			} else {
				fmt.Printf("Server '%s' not found.\n", name)
			}
			return nil
		},
	}
}

func newConfigDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default <name>",
		Short: "Set the default server",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if err := cfg.SetDefault(name); err != nil {
				return err
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Default server set to '%s'.\n", name)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured servers",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(cfg.Servers) == 0 {
				fmt.Println("No servers configured.")
				fmt.Println("Use 'unraid config add <name> --url <url> --api-key <key>' to add a server.")
				return nil
			}

			fmt.Println("Configured servers:")
			fmt.Println()

			for name, server := range cfg.Servers {
				defaultMarker := ""
				if cfg.Default == name {
					defaultMarker = " (default)"
				}
				fmt.Printf("  %s%s\n", name, defaultMarker)
				fmt.Printf("    URL: %s\n", server.URL)
				fmt.Printf("    API Key: %s\n", config.MaskAPIKey(server.APIKey))
				fmt.Println()
			}
			return nil
		},
	}
}
