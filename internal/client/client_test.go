package client

import (
	"net/http"
	"testing"
)

// recordingTransport is a no-op transport that lets us inspect requests.
type recordingTransport struct{}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func newTestRequest() (*http.Request, error) {
	return http.NewRequest(http.MethodPost, "https://example.com/graphql", nil)
}

func TestApiKeyTransportSetsHeader(t *testing.T) {
	transport := &apiKeyTransport{
		apiKey: "my-secret-key",
		base:   &recordingTransport{},
	}

	req, _ := newTestRequest()
	_, _ = transport.RoundTrip(req)

	if got := req.Header.Get("x-api-key"); got != "my-secret-key" {
		t.Errorf("expected x-api-key header 'my-secret-key', got %q", got)
	}
}
