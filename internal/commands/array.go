package commands

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/spf13/cobra"

	"github.com/medzin/unraid-cli/internal/client"
)

func newArrayCmd(preRun func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "array",
		Short: "Array management",
	}

	cmd.AddCommand(newArrayStartCmd(), newArrayStopCmd(), newArrayStatusCmd())
	for _, sub := range cmd.Commands() {
		sub.PreRunE = preRun
	}

	return cmd
}

func newArrayStartCmd() *cobra.Command {
	return newArraySetStateCmd("start", "Start the array", "Starting array...", client.ArrayStateInputStateStart)
}

func newArrayStopCmd() *cobra.Command {
	return newArraySetStateCmd("stop", "Stop the array", "Stopping array...", client.ArrayStateInputStateStop)
}

func newArraySetStateCmd(use, short, msg string, state client.ArrayStateInputState) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			log.Println(msg)
			resp, err := client.SetArrayState(cmd.Context(), getClient(cmd.Context()), state)
			if err != nil {
				return err
			}
			return printAction(cmd.Context(), fmt.Sprintf("Array is now %s.", formatArrayState(resp.Array.SetState.State)))
		},
	}
}

func newArrayStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show array status and disk list",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := client.GetArray(cmd.Context(), getClient(cmd.Context()))
			if err != nil {
				return err
			}
			a := resp.Array
			return render(cmd.Context(), a, func() error {
				w := getOutputWriter(cmd.Context())
				if _, err := fmt.Fprintf(w, "State:  %s\n", formatArrayState(a.State)); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
				if err := printArrayDisks(w, "Parity", diskSlice(ptrSlice(a.Parities))); err != nil {
					return err
				}
				if err := printArrayDisks(w, "Data", diskSlice(ptrSlice(a.Disks))); err != nil {
					return err
				}
				return printArrayDisks(w, "Cache", diskSlice(ptrSlice(a.Caches)))
			})
		},
	}
}

type arrayDiskRow struct {
	name       string
	device     string
	status     string
	size       int64
	isSpinning *bool
}

func diskSlice[T interface {
	GetName() *string
	GetDevice() *string
	GetStatus() *client.ArrayDiskStatus
	GetSize() *int64
	GetIsSpinning() *bool
}](disks []T) []arrayDiskRow {
	rows := make([]arrayDiskRow, len(disks))
	for i, d := range disks {
		rows[i] = arrayDiskRow{
			name:       derefStr(d.GetName(), "—"),
			device:     derefStr(d.GetDevice(), "—"),
			status:     formatDiskStatus(d.GetStatus()),
			size:       derefInt64(d.GetSize()),
			isSpinning: d.GetIsSpinning(),
		}
	}
	return rows
}

func ptrSlice[T any](s []T) []*T {
	out := make([]*T, len(s))
	for i := range s {
		out[i] = &s[i]
	}
	return out
}

func printArrayDisks(w io.Writer, label string, disks []arrayDiskRow) error {
	if len(disks) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "%s disks:\n", label); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %-16s %-10s %-22s %-12s %s\n", "NAME", "DEVICE", "STATUS", "SIZE", "SPINNING"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  "+strings.Repeat("-", 72)); err != nil {
		return err
	}
	for _, d := range disks {
		spinning := "—"
		if d.isSpinning != nil {
			if *d.isSpinning {
				spinning = "yes"
			} else {
				spinning = "no"
			}
		}
		if _, err := fmt.Fprintf(w, "  %-16s %-10s %-22s %-12s %s\n",
			truncate(d.name, 15),
			truncate(d.device, 9),
			d.status,
			formatBytes(d.size),
			spinning,
		); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func formatArrayState(state client.ArrayState) string {
	switch state {
	case client.ArrayStateStarted:
		return "started"
	case client.ArrayStateStopped:
		return "stopped"
	case client.ArrayStateNewArray:
		return "new array"
	case client.ArrayStateReconDisk:
		return "reconstructing disk"
	case client.ArrayStateDisableDisk:
		return "disk disabled"
	case client.ArrayStateSwapDsbl:
		return "swapping disabled disk"
	case client.ArrayStateInvalidExpansion:
		return "invalid expansion"
	case client.ArrayStateParityNotBiggest:
		return "parity not biggest"
	case client.ArrayStateTooManyMissingDisks:
		return "too many missing disks"
	case client.ArrayStateNewDiskTooSmall:
		return "new disk too small"
	case client.ArrayStateNoDataDisks:
		return "no data disks"
	default:
		return "unknown"
	}
}

func formatDiskStatus(s *client.ArrayDiskStatus) string {
	if s == nil {
		return "—"
	}
	switch *s {
	case client.ArrayDiskStatusDiskOk:
		return "ok"
	case client.ArrayDiskStatusDiskNp:
		return "not present"
	case client.ArrayDiskStatusDiskNpMissing:
		return "missing"
	case client.ArrayDiskStatusDiskInvalid:
		return "invalid"
	case client.ArrayDiskStatusDiskWrong:
		return "wrong"
	case client.ArrayDiskStatusDiskDsbl:
		return "disabled"
	case client.ArrayDiskStatusDiskNpDsbl:
		return "not present (disabled)"
	case client.ArrayDiskStatusDiskDsblNew:
		return "disabled (new)"
	case client.ArrayDiskStatusDiskNew:
		return "new"
	default:
		return "unknown"
	}
}

func formatBytes(kb int64) string {
	if kb == 0 {
		return "—"
	}
	const unit = 1024
	mb := kb / unit
	if mb < unit {
		return fmt.Sprintf("%d MB", mb)
	}
	gb := mb / unit
	if gb < unit {
		return fmt.Sprintf("%d GB", gb)
	}
	return fmt.Sprintf("%d TB", gb/unit)
}
