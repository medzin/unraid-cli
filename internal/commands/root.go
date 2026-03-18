// Package commands defines the CLI command tree for the Unraid CLI.
package commands

import (
	"github.com/spf13/cobra"

	"github.com/medzin/unraid-cli/internal/client"
	"github.com/medzin/unraid-cli/internal/config"
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

	// makePreRun resolves the server config then calls fn to finish setting up
	// the command's context. Shared by all subcommands that need a live server.
	makePreRun := func(fn func(*cobra.Command, *config.ResolvedConfig) error) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, _ []string) error {
			resolved, err := config.Resolve(serverFlag, urlFlag, apiKeyFlag)
			if err != nil {
				return err
			}
			return fn(cmd, resolved)
		}
	}

	resolveClient := makePreRun(func(cmd *cobra.Command, r *config.ResolvedConfig) error {
		cmd.SetContext(withClient(cmd.Context(), client.New(r.URL, r.APIKey, timeout)))
		return nil
	})

	resolveIntrospect := makePreRun(func(cmd *cobra.Command, r *config.ResolvedConfig) error {
		cmd.SetContext(withIntrospectionClient(cmd.Context(), client.NewIntrospection(r.URL, r.APIKey, timeout)))
		return nil
	})

	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newDockerCmd(resolveClient))
	rootCmd.AddCommand(newVmCmd(resolveClient))
	rootCmd.AddCommand(newCapabilitiesCmd(resolveIntrospect))

	return rootCmd
}
