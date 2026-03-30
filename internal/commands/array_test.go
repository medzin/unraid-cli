package commands

import (
	"testing"

	"github.com/medzin/unraid-cli/internal/client"
)

func TestFormatArrayState(t *testing.T) {
	cases := []struct {
		state client.ArrayState
		want  string
	}{
		{client.ArrayStateStarted, "started"},
		{client.ArrayStateStopped, "stopped"},
		{client.ArrayStateNewArray, "new array"},
		{client.ArrayStateReconDisk, "reconstructing disk"},
		{client.ArrayStateDisableDisk, "disk disabled"},
		{client.ArrayStateSwapDsbl, "swapping disabled disk"},
		{client.ArrayStateInvalidExpansion, "invalid expansion"},
		{client.ArrayStateParityNotBiggest, "parity not biggest"},
		{client.ArrayStateTooManyMissingDisks, "too many missing disks"},
		{client.ArrayStateNewDiskTooSmall, "new disk too small"},
		{client.ArrayStateNoDataDisks, "no data disks"},
		{client.ArrayState("UNKNOWN_STATE"), "unknown"},
	}

	for _, tc := range cases {
		got := formatArrayState(tc.state)
		if got != tc.want {
			t.Errorf("formatArrayState(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestFormatDiskStatus(t *testing.T) {
	ptr := func(s client.ArrayDiskStatus) *client.ArrayDiskStatus { return &s }

	cases := []struct {
		status *client.ArrayDiskStatus
		want   string
	}{
		{nil, "—"},
		{ptr(client.ArrayDiskStatusDiskOk), "ok"},
		{ptr(client.ArrayDiskStatusDiskNp), "not present"},
		{ptr(client.ArrayDiskStatusDiskNpMissing), "missing"},
		{ptr(client.ArrayDiskStatusDiskInvalid), "invalid"},
		{ptr(client.ArrayDiskStatusDiskWrong), "wrong"},
		{ptr(client.ArrayDiskStatusDiskDsbl), "disabled"},
		{ptr(client.ArrayDiskStatusDiskNpDsbl), "not present (disabled)"},
		{ptr(client.ArrayDiskStatusDiskDsblNew), "disabled (new)"},
		{ptr(client.ArrayDiskStatusDiskNew), "new"},
		{ptr(client.ArrayDiskStatus("UNKNOWN")), "unknown"},
	}

	for _, tc := range cases {
		got := formatDiskStatus(tc.status)
		if got != tc.want {
			t.Errorf("formatDiskStatus(%v) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		kb   int64
		want string
	}{
		{0, "—"},
		{512, "0 MB"},
		{1024, "1 MB"},
		{1023 * 1024, "1023 MB"},
		{1024 * 1024, "1 GB"},
		{1023 * 1024 * 1024, "1023 GB"},
		{1024 * 1024 * 1024, "1 TB"},
		{4 * 1024 * 1024 * 1024, "4 TB"},
	}

	for _, tc := range cases {
		got := formatBytes(tc.kb)
		if got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.kb, got, tc.want)
		}
	}
}
