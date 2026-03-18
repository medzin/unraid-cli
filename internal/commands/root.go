// Package commands defines the CLI command tree for the Unraid CLI.
package commands

import (
	"github.com/spf13/cobra"

	"github.com/amedzinski/unraid-cli/internal/client"
	"github.com/amedzinski/unraid-cli/internal/config"
)

type contextKey string

const clientKey contextKey = "client"

// NewRootCmd creates the root cobra command with all subcommands.
func NewRootCmd() *cobra.Command {
	var (
		serverFlag string
		urlFlag    string
		apiKeyFlag string
		timeout    uint
	)

	rootCmd := &cobra.Command{
		Use:     "unraid",
		Short:   "CLI client for Unraid API",
		Version: "0.1.0",
	}

	rootCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "server name from config (env: UNRAID_SERVER)")
	rootCmd.PersistentFlags().StringVar(&urlFlag, "url", "", "server URL, overrides config (env: UNRAID_URL)")
	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "API key, overrides config (env: UNRAID_API_KEY)")
	rootCmd.PersistentFlags().UintVar(&timeout, "timeout", 5, "request timeout in seconds (env: UNRAID_TIMEOUT)")

	// resolveClient is a helper that resolves config and creates a client.
	// Used by docker and vm commands in their PreRunE.
	resolveClient := func(cmd *cobra.Command, _ []string) error {
		resolved, err := config.Resolve(serverFlag, urlFlag, apiKeyFlag)
		if err != nil {
			return err
		}
		c := client.New(resolved.URL, resolved.APIKey, timeout)
		cmd.SetContext(withClient(cmd.Context(), c))
		return nil
	}

	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newDockerCmd(resolveClient))
	rootCmd.AddCommand(newVmCmd(resolveClient))

	return rootCmd
}
