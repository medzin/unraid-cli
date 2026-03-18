package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// IntrospectionClient makes raw GraphQL introspection requests.
type IntrospectionClient struct {
	URL        string
	httpClient *http.Client
}

// NewIntrospection creates a client for schema introspection.
func NewIntrospection(url, apiKey string, timeoutSecs uint) *IntrospectionClient {
	return &IntrospectionClient{
		URL:        url,
		httpClient: newHTTPClient(apiKey, timeoutSecs),
	}
}

// FetchCapabilities queries the GraphQL schema via introspection and returns
// which mutation types and query fields are available on this server.
func (c *IntrospectionClient) FetchCapabilities(ctx context.Context) (*SchemaCapabilities, error) {
	body, err := json.Marshal(map[string]string{"query": introspectionQuery})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspection request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result introspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode introspection response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("introspection error: %s", result.Errors[0].Message)
	}

	return parseCapabilities(result), nil
}

func typeToSet(t *introspectTypeResult) map[string]bool {
	if t == nil {
		return nil
	}
	return fieldsToSet(t.Fields)
}

func fieldsToSet(fields []introspectFieldEntry) map[string]bool {
	m := make(map[string]bool, len(fields))
	for _, f := range fields {
		m[f.Name] = true
	}
	return m
}
