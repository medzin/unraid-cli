package commands

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"", 5, ""},
		{"hello", 0, ""},
		{"hello", 1, "h"},
		{"hello", 2, "he"},
		{"hello", 3, "hel"},
		{"hello", 4, "h..."},
		{"abcdef", 3, "abc"},
	}

	for _, tc := range cases {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.expected)
		}
	}
}
