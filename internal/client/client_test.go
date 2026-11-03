//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// droppingServer accepts each request and closes the connection without responding,
// which the client sees as a transport error.
func droppingServer(t *testing.T, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testClient(host string) *Client {
	c := New(host, "key", "", "test")
	c.retryBase = time.Millisecond
	c.retryMax = time.Millisecond
	return c
}

func TestDo_TransportErrorRetriesOnlyIdempotentMethods(t *testing.T) {
	cases := []struct {
		method string
		want   int32
	}{
		{http.MethodGet, 4},
		{http.MethodDelete, 4},
		{http.MethodPost, 1},
		{http.MethodPatch, 1},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			var hits atomic.Int32
			c := testClient(droppingServer(t, &hits).URL)

			if err := c.do(context.Background(), tc.method, "/v2/monitors", nil, nil); err == nil {
				t.Fatal("expected an error")
			}
			if got := hits.Load(); got != tc.want {
				t.Errorf("requests sent = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDo_DialErrorRetriesEveryMethod(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			var dials atomic.Int32
			c := testClient("http://" + addr)
			c.httpClient.Transport = &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					dials.Add(1)
					return (&net.Dialer{}).DialContext(ctx, network, address)
				},
			}

			if err := c.do(context.Background(), method, "/v2/monitors", nil, nil); err == nil {
				t.Fatal("expected an error")
			}
			if got := dials.Load(); got != 4 {
				t.Errorf("connection attempts = %d, want 4", got)
			}
		})
	}
}

func TestIsNotFound_MatchesWrappedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	err := testClient(srv.URL).DeleteMonitor(context.Background(), "abc")
	if !IsNotFound(err) {
		t.Fatalf("IsNotFound(%v) = false", err)
	}
	if !IsNotFound(fmt.Errorf("deleting monitor: %w", err)) {
		t.Fatal("IsNotFound did not match a wrapped NotFoundError")
	}
}

func TestUpdateMonitorRequest_JSON(t *testing.T) {
	name := "renamed"
	var noDescription *string
	emptyFilter := map[string][]*string{}
	emptyGroupBy := []string{}

	cases := []struct {
		name string
		req  UpdateMonitorRequest
		want string
	}{
		{"empty", UpdateMonitorRequest{}, `{}`},
		{"name only", UpdateMonitorRequest{Name: &name}, `{"monitor_name":"renamed"}`},
		{"clear description", UpdateMonitorRequest{Description: &noDescription}, `{"description":null}`},
		{
			"clear filter, group_by, threshold",
			UpdateMonitorRequest{FilterSet: &emptyFilter, GroupBy: &emptyGroupBy, Threshold: &MonitorThreshold{}},
			`{"filter_set":{},"group_by":[],"threshold":{}}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
