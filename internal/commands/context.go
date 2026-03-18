package commands

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

func withClient(ctx context.Context, c graphql.Client) context.Context {
	return context.WithValue(ctx, clientKey, c)
}

func getClient(ctx context.Context) graphql.Client {
	return ctx.Value(clientKey).(graphql.Client)
}
