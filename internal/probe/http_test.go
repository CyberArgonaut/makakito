package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CyberArgonaut/makakito/internal/engine"
)

func TestHTTPProbeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := &HTTPProbe{}
	result, err := p.Check(context.Background(), engine.Params{
		"url":             srv.URL,
		"expected_status": "200",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Message)
	}
}

func TestHTTPProbeWrongStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	p := &HTTPProbe{}
	result, err := p.Check(context.Background(), engine.Params{
		"url":             srv.URL,
		"expected_status": "200",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
}

func TestHTTPProbeUnreachable(t *testing.T) {
	p := &HTTPProbe{}
	result, err := p.Check(context.Background(), engine.Params{
		"url":             "http://127.0.0.1:19999",
		"expected_status": "200",
		"timeout":         "500ms",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for unreachable host")
	}
}

func TestHTTPProbeCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &HTTPProbe{}
	result, err := p.Check(ctx, engine.Params{
		"url":             srv.URL,
		"expected_status": "200",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for cancelled context")
	}
}

func TestHTTPProbeMissingURL(t *testing.T) {
	p := &HTTPProbe{}
	_, err := p.Check(context.Background(), engine.Params{
		"expected_status": "200",
	})
	if err == nil {
		t.Fatal("expected error for missing url")
	}
}

func TestHTTPProbeMissingExpectedStatus(t *testing.T) {
	p := &HTTPProbe{}
	_, err := p.Check(context.Background(), engine.Params{
		"url": "http://localhost",
	})
	if err == nil {
		t.Fatal("expected error for missing expected_status")
	}
}
