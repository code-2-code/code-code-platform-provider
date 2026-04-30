package providerobservability

import (
	"context"
	"net/http"

	"code-code.internal/platform-k8s/internal/platform/outboundhttp"
)

// observabilityHTTPClient creates the plain business HTTP client used by
// observability probes. Network routing and header auth are owned outside
// provider-service.
func observabilityHTTPClient(ctx context.Context) (*http.Client, error) {
	client, err := outboundhttp.NewClientFactory().NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}
