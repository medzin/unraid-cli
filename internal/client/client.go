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

// newHTTPClient builds an *http.Client with the API key transport and TLS config
// shared by both the genqlient and introspection clients.
func newHTTPClient(apiKey string, timeoutSecs uint, insecureTLS bool) *http.Client {
	return &http.Client{
		Transport: &apiKeyTransport{
			apiKey: apiKey,
			base: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecureTLS,
				},
			},
		},
		Timeout: time.Duration(timeoutSecs) * time.Second,
	}
}

// New creates a genqlient GraphQL client configured for Unraid.
func New(url, apiKey string, timeoutSecs uint, insecureTLS bool) graphql.Client {
	return graphql.NewClient(url, newHTTPClient(apiKey, timeoutSecs, insecureTLS))
}
