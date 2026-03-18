// Package client provides a GraphQL HTTP client for the Unraid API.
package client

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
)

type apiKeyTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("x-api-key", t.apiKey)
	return t.base.RoundTrip(req)
}

// New creates a genqlient GraphQL client configured for Unraid.
func New(url, apiKey string, timeoutSecs uint) graphql.Client {
	transport := &apiKeyTransport{
		apiKey: apiKey,
		base: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // Unraid uses self-signed certs
			},
		},
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeoutSecs) * time.Second,
	}

	return graphql.NewClient(url, httpClient)
}
