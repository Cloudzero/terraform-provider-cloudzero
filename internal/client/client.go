package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// Client is the CloudZero API client used by all resources and data sources.
type Client struct {
	host       string
	apiKey     string
	testKey    string
	httpClient *http.Client
	userAgent  string

	maxRetries int
	retryBase  time.Duration
	retryMax   time.Duration
}

// New creates a new CloudZero API client.
func New(host, apiKey, testKey, version string) *Client {
	return &Client{
		host:       host,
		apiKey:     apiKey,
		testKey:    testKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		userAgent:  fmt.Sprintf("terraform-provider-cloudzero/%s", version),
		maxRetries: 3,
		retryBase:  1 * time.Second,
		retryMax:   30 * time.Second,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Min(
				float64(c.retryBase)*math.Pow(2, float64(attempt-1)),
				float64(c.retryMax),
			))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			// Reset body reader for retry
			if body != nil {
				b, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(b)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, c.host+path, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		// CloudZero uses bare API key — no "Bearer" prefix
		req.Header.Set("Authorization", c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if c.testKey != "" {
			req.Header.Set("cloudzero-test-key", c.testKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		// Retry on 429 and 5xx
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			return &NotFoundError{Message: string(respBody)}
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("unmarshaling response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
