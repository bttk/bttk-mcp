package obsidian

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrAPI = errors.New("API error")

// Client is the main entry point for the Obsidian Local REST API client.
type Client struct {
	baseURL *url.URL
	token   string
	http    *http.Client

	// Services
	ActiveFile *ActiveFileService
	Vault      *VaultService
	Periodic   *PeriodicService
	Search     *SearchService
	Commands   *CommandService
	Open       *OpenService
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// NewClient creates a new Obsidian API client.
func NewClient(baseURL, token string, opts ...Option) (*Client, error) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL: u,
		token:   token,
		http: &http.Client{
			Timeout: 10 * time.Second, //nolint:mnd
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	c.initializeServices()

	return c, nil
}

// WithHTTPClient allows providing a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.http = httpClient
	}
}

// WithInsecureTLS disables TLS certificate verification.
// This is often necessary for the Obsidian Local REST API as it uses self-signed certificates.
func WithInsecureTLS() Option {
	return func(c *Client) {
		if c.http.Transport == nil {
			c.http.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Intentional for local dev
			}
		} else if t, ok := c.http.Transport.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = &tls.Config{} //nolint:gosec // Intentional
			}
			t.TLSClientConfig.InsecureSkipVerify = true
		}
	}
}

// WithCertificate loads a CA certificate from the given path and adds it to the client's root CAs.
func WithCertificate(path string) Option {
	return func(c *Client) {
		caCert, err := os.ReadFile(path)
		if err != nil {
			return // Best effort? Or should we handle it? Functional options usually don't return error.
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		if c.http.Transport == nil {
			c.http.Transport = http.DefaultTransport.(*http.Transport).Clone()
		}
		if t, ok := c.http.Transport.(*http.Transport); ok {
			t.TLSClientConfig.RootCAs = caCertPool
		}
	}
}

func (c *Client) initializeServices() {
	c.ActiveFile = &ActiveFileService{client: c}
	c.Vault = &VaultService{client: c}
	c.Periodic = &PeriodicService{client: c}
	c.Search = &SearchService{client: c}
	c.Commands = &CommandService{client: c}
	c.Open = &OpenService{client: c}
}

func (c *Client) do(req *http.Request, v interface{}) error {
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return &errResp
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status code %d, body: %s", ErrAPI, resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		// specific handling for string response (raw content)
		if strPtr, ok := v.(*string); ok {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			*strPtr = string(bodyBytes)
			return nil
		}

		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}
