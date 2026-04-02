package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// containerJSON is the raw JSON shape returned by GET /containers/json.
type containerJSON struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Labels map[string]string `json:"Labels"`
}

// Client is a minimal Docker daemon client that communicates via the Docker HTTP API.
// It supports Unix socket (default) and TCP transports, controlled by DOCKER_HOST.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClientFromEnv creates a Client using DOCKER_HOST (defaults to unix:///var/run/docker.sock).
func NewClientFromEnv() (*Client, error) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	return newClient(host)
}

func newClient(host string) (*Client, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("parsing DOCKER_HOST %q: %w", host, err)
	}

	var transport http.RoundTripper
	var baseURL string

	switch u.Scheme {
	case "unix":
		socketPath := u.Host + u.Path
		if socketPath == "" {
			socketPath = "/var/run/docker.sock"
		}
		transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		}
		baseURL = "http://localhost"
	case "tcp", "http", "https":
		baseURL = strings.Replace(host, u.Scheme+"://", "http://", 1)
		transport = http.DefaultTransport
	default:
		return nil, fmt.Errorf("unsupported DOCKER_HOST scheme %q", u.Scheme)
	}

	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Transport: transport},
	}, nil
}

// Close is a no-op; it exists so Client can satisfy interfaces that expect Close().
func (c *Client) Close() error { return nil }

// ContainerList returns containers from the daemon.
func (c *Client) ContainerList(ctx context.Context, opts ListOptions) ([]ContainerSummary, error) {
	all := "false"
	if opts.All {
		all = "true"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/containers/json?all="+all, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("building container list request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker API error listing containers: %s", resp.Status)
	}
	var raw []containerJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding container list: %w", err)
	}
	out := make([]ContainerSummary, len(raw))
	for i, r := range raw {
		out[i] = ContainerSummary(r)
	}
	return out, nil
}

// Ping checks that the Docker daemon is reachable.
func (c *Client) Ping(ctx context.Context) (Ping, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/_ping", http.NoBody)
	if err != nil {
		return Ping{}, fmt.Errorf("building ping request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Ping{}, fmt.Errorf("pinging docker daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Ping{}, fmt.Errorf("docker daemon ping returned: %s", resp.Status)
	}
	return Ping{}, nil
}

// ContainerStop stops the named container.
func (c *Client) ContainerStop(ctx context.Context, containerID string, _ StopOptions) error {
	return c.post(ctx, "/containers/"+containerID+"/stop")
}

// ContainerStart starts the named container.
func (c *Client) ContainerStart(ctx context.Context, containerID string, _ StartOptions) error {
	return c.post(ctx, "/containers/"+containerID+"/start")
}

// ContainerPause pauses the named container.
func (c *Client) ContainerPause(ctx context.Context, containerID string) error {
	return c.post(ctx, "/containers/"+containerID+"/pause")
}

// ContainerUnpause unpauses the named container.
func (c *Client) ContainerUnpause(ctx context.Context, containerID string) error {
	return c.post(ctx, "/containers/"+containerID+"/unpause")
}

func (c *Client) post(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, http.NoBody)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", path, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker API call %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("docker API %s returned: %s", path, resp.Status)
	}
	return nil
}
