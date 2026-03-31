package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/medzin/unraid-cli/internal/client"
)

type serverVersionOutput struct {
	UnraidOS string `json:"unraid_os"`
	API      string `json:"api"`
}

func newServerVersionCmd(preRun func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:     "server-version",
		Short:   "Show Unraid server and API versions",
		Args:    cobra.NoArgs,
		PreRunE: preRun,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := client.GetServerVersion(cmd.Context(), getClient(cmd.Context()))
			if err != nil {
				return err
			}
			core := resp.Info.Versions.Core
			out := serverVersionOutput{
				UnraidOS: derefStr(core.Unraid, "unknown"),
				API:      derefStr(core.Api, "unknown"),
			}
			return render(cmd.Context(), out, func() error {
				w := getOutputWriter(cmd.Context())
				if _, err := fmt.Fprintf(w, "Unraid OS:  %s\n", out.UnraidOS); err != nil {
					return err
				}
				_, err := fmt.Fprintf(w, "API:        %s\n", out.API)
				return err
			})
		},
	}
}
