package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

type outputFmt string

const (
	outputText outputFmt = "text"
	outputJSON outputFmt = "json"
)

type actionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func isJSON(ctx context.Context) bool {
	f, _ := ctx.Value(outputFmtKey).(outputFmt)
	return f == outputJSON
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// render writes v as JSON or delegates to textFn for human-readable output.
func render(ctx context.Context, v any, textFn func() error) error {
	if isJSON(ctx) {
		return printJSON(getOutputWriter(ctx), v)
	}
	return textFn()
}

func printAction(ctx context.Context, msg string) error {
	w := getOutputWriter(ctx)
	if isJSON(ctx) {
		return printJSON(w, actionResult{Success: true, Message: msg})
	}
	_, err := fmt.Fprintln(w, msg)
	return err
}
