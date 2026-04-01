// Package probe implements steady-state hypothesis probes for makakito.
package probe

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

// HTTPProbe checks a URL for an expected HTTP status code.
type HTTPProbe struct{}

// Check performs the HTTP probe.
func (p *HTTPProbe) Check(ctx context.Context, params engine.Params) (engine.ProbeResult, error) {
	url := params["url"]
	if url == "" {
		return engine.ProbeResult{}, fmt.Errorf("http probe requires param \"url\"")
	}
	expectedStr := params["expected_status"]
	if expectedStr == "" {
		return engine.ProbeResult{}, fmt.Errorf("http probe requires param \"expected_status\"")
	}
	expectedStatus, err := strconv.Atoi(expectedStr)
	if err != nil {
		return engine.ProbeResult{}, fmt.Errorf("invalid expected_status %q: %w", expectedStr, err)
	}

	timeout := 5 * time.Second
	if t := params["timeout"]; t != "" {
		parsed, err := time.ParseDuration(t)
		if err != nil {
			return engine.ProbeResult{}, fmt.Errorf("invalid timeout %q: %w", t, err)
		}
		timeout = parsed
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return engine.ProbeResult{}, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return engine.ProbeResult{
			Success: false,
			Message: fmt.Sprintf("request failed: %s", err),
		}, nil
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != expectedStatus {
		return engine.ProbeResult{
			Success: false,
			Message: fmt.Sprintf("expected status %d, got %d", expectedStatus, resp.StatusCode),
		}, nil
	}
	return engine.ProbeResult{Success: true, Message: fmt.Sprintf("status %d OK", resp.StatusCode)}, nil
}

// Validate checks probe params.
func (p *HTTPProbe) Validate(params engine.Params) error {
	if params["url"] == "" {
		return fmt.Errorf("http probe requires param \"url\"")
	}
	if params["expected_status"] == "" {
		return fmt.Errorf("http probe requires param \"expected_status\"")
	}
	return nil
}
