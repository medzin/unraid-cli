package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/medzin/unraid-cli/internal/client"
)

func newServerCmd(preRun func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Query information about the Unraid server",
	}
	cmd.AddCommand(newServerVersionCmd(), newServerLogCmd())
	for _, sub := range cmd.Commands() {
		sub.PreRunE = preRun
	}
	return cmd
}

type serverVersionOutput struct {
	UnraidOS string `json:"unraid_os"`
	API      string `json:"api"`
}

func newServerVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show Unraid OS and API versions",
		Args:  cobra.NoArgs,
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

type serverLogFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type serverLogOutput struct {
	Path       string `json:"path"`
	TotalLines int    `json:"total_lines"`
	Content    string `json:"content"`
}

func newServerLogCmd() *cobra.Command {
	var lines int
	var list bool

	cmd := &cobra.Command{
		Use:   "log [path]",
		Short: "Show a server log file",
		Long: `Show the content of a server log file.
Run with --list to see available log files.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if list && len(args) > 0 {
				return fmt.Errorf("cannot specify both --list and a path argument")
			}
			if list {
				return runServerLogList(cmd)
			}
			if len(args) == 0 {
				return fmt.Errorf("path required; use --list to see available log files")
			}
			return runServerLogShow(cmd, args[0], lines)
		},
	}

	cmd.Flags().IntVarP(&lines, "lines", "n", 100, "number of lines to show (0 for all)")
	cmd.Flags().BoolVar(&list, "list", false, "list available log files")

	return cmd
}

func runServerLogList(cmd *cobra.Command) error {
	resp, err := client.GetLogFiles(cmd.Context(), getClient(cmd.Context()))
	if err != nil {
		return err
	}

	entries := make([]serverLogFile, len(resp.LogFiles))
	for i, f := range resp.LogFiles {
		entries[i] = serverLogFile{Name: f.Name, Path: f.Path}
	}

	return render(cmd.Context(), entries, func() error {
		w := getOutputWriter(cmd.Context())
		if _, err := fmt.Fprintf(w, "%-30s  %s\n", "NAME", "PATH"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, strings.Repeat("-", 60)); err != nil {
			return err
		}
		for _, e := range entries {
			if _, err := fmt.Fprintf(w, "%-30s  %s\n", e.Name, e.Path); err != nil {
				return err
			}
		}
		return nil
	})
}

func runServerLogShow(cmd *cobra.Command, path string, lines int) error {
	var linesPtr *int
	if lines > 0 {
		linesPtr = &lines
	}

	resp, err := client.GetLogFile(cmd.Context(), getClient(cmd.Context()), path, linesPtr)
	if err != nil {
		return err
	}

	out := serverLogOutput{
		Path:       resp.LogFile.Path,
		TotalLines: resp.LogFile.TotalLines,
		Content:    resp.LogFile.Content,
	}

	return render(cmd.Context(), out, func() error {
		_, err := fmt.Fprint(getOutputWriter(cmd.Context()), out.Content)
		return err
	})
}
