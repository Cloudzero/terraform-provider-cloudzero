//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
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
		host:    host,
		apiKey:  apiKey,
		testKey: testKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
		userAgent:  fmt.Sprintf("terraform-provider-cloudzero/%s", version),
		maxRetries: 3,
		retryBase:  1 * time.Second,
		retryMax:   30 * time.Second,
	}
}

// isIdempotent reports whether an HTTP method is safe to retry on server errors
// without risk of duplicate side effects.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

// retryDelay returns the backoff duration before the next attempt.
// For 429 responses it honors the Retry-After header (delta-seconds) when present.
func retryDelay(resp *http.Response, attempt int, base, max time.Duration) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if v := resp.Header.Get("Retry-After"); v != "" {
			if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
				if d := time.Duration(secs) * time.Second; d < max {
					return d
				}
				return max
			}
		}
	}
	return time.Duration(math.Min(
		float64(base)*math.Pow(2, float64(attempt-1)),
		float64(max),
	))
}

// sanitizeErrBody caps a response body and strips control characters before
// embedding it in an error message, preventing log injection and excessive output.
func sanitizeErrBody(b []byte) string {
	const maxBytes = 512
	if len(b) > maxBytes {
		b = b[:maxBytes]
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			return r
		}
		return -1
	}, string(b))
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
	}

	var lastErr error
	var lastResp *http.Response
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(lastResp, attempt, c.retryBase, c.retryMax)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
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
			lastResp = nil
			continue
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			// Retry 429 for any method; retryDelay will honor Retry-After.
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, sanitizeErrBody(respBody))
			lastResp = resp
			continue
		case resp.StatusCode >= 500 && isIdempotent(method):
			// Retry 5xx only for idempotent methods to avoid duplicate side effects.
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, sanitizeErrBody(respBody))
			lastResp = resp
			continue
		case resp.StatusCode >= 500:
			// Non-idempotent 5xx: return immediately — do not risk a duplicate write.
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, sanitizeErrBody(respBody))
		case resp.StatusCode == http.StatusNotFound:
			return &NotFoundError{Message: sanitizeErrBody(respBody)}
		case resp.StatusCode >= 400:
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, sanitizeErrBody(respBody))
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
