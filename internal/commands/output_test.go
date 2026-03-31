package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func ctxWithJSON() context.Context {
	return withOutputFormat(context.Background(), outputJSON)
}

func ctxWithText() context.Context {
	return withOutputFormat(context.Background(), outputText)
}

func TestIsJSON(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{"no value set", context.Background(), false},
		{"text format", ctxWithText(), false},
		{"json format", ctxWithJSON(), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isJSON(tc.ctx); got != tc.want {
				t.Errorf("isJSON() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	v := struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}{"test", 42}

	if err := printJSON(&buf, v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["name"] != "test" {
		t.Errorf("name = %v, want test", got["name"])
	}
	if got["value"] != float64(42) {
		t.Errorf("value = %v, want 42", got["value"])
	}
}

func TestPrintAction(t *testing.T) {
	t.Run("text mode writes plain line", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := withOutputFormat(context.Background(), outputText)
		ctx = withOutputWriter(ctx, &buf)

		if err := printAction(ctx, "hello world"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(buf.String()); got != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("json mode writes success action result", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := withOutputFormat(context.Background(), outputJSON)
		ctx = withOutputWriter(ctx, &buf)

		if err := printAction(ctx, "hello world"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var got actionResult
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
		}
		if !got.Success {
			t.Error("success = false, want true")
		}
		if got.Message != "hello world" {
			t.Errorf("message = %q, want %q", got.Message, "hello world")
		}
	})
}
