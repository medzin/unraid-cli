package commands

import (
	"context"
	"io"
	"os"

	"github.com/Khan/genqlient/graphql"

	"github.com/medzin/unraid-cli/internal/client"
)

const (
	introspectionClientKey contextKey = "introspection_client"
	outputFmtKey           contextKey = "output_fmt"
	outputWriterKey        contextKey = "output_writer"
)

func withOutputFormat(ctx context.Context, f outputFmt) context.Context {
	return context.WithValue(ctx, outputFmtKey, f)
}

func withOutputWriter(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, outputWriterKey, w)
}

func getOutputWriter(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(outputWriterKey).(io.Writer); ok {
		return w
	}
	return os.Stdout
}

func withClient(ctx context.Context, c graphql.Client) context.Context {
	return context.WithValue(ctx, clientKey, c)
}

func getClient(ctx context.Context) graphql.Client {
	return ctx.Value(clientKey).(graphql.Client)
}

func withIntrospectionClient(ctx context.Context, c *client.IntrospectionClient) context.Context {
	return context.WithValue(ctx, introspectionClientKey, c)
}

func getIntrospectionClient(ctx context.Context) *client.IntrospectionClient {
	return ctx.Value(introspectionClientKey).(*client.IntrospectionClient)
}
