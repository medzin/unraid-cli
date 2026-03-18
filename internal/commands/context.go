package commands

import (
	"context"

	"github.com/Khan/genqlient/graphql"

	"github.com/medzin/unraid-cli/internal/client"
)

const introspectionClientKey contextKey = "introspection_client"

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
