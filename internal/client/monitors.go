//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Monitor represents a CloudZero cost monitor.
type Monitor struct {
	ID                   string                `json:"monitor_id"`
	Name                 string                `json:"monitor_name"`
	Description          *string               `json:"description"`
	DetectionType        string                `json:"detection_type"`
	CostType             string                `json:"cost_type"`
	FilterSet            map[string][]*string  `json:"filter_set"`
	GroupBy              []string              `json:"group_by"`
	IsEnabled            bool                  `json:"is_enabled"`
	Threshold            MonitorThreshold      `json:"threshold"`
	NotificationChannels []NotificationChannel `json:"notification_channels"`
	OwnerID              string                `json:"owner_id"`
	CreatedAt            string                `json:"created_at"`
	UpdatedAt            string                `json:"updated_at"`
}

// MonitorThreshold limits which detected anomalies become incidents. All fields unset means every anomaly does.
type MonitorThreshold struct {
	MinCostImpact *float64 `json:"min_cost_impact,omitempty"`
	Percentage    *float64 `json:"percentage,omitempty"`
	Operator      *string  `json:"operator,omitempty"`
}

func (t MonitorThreshold) IsEmpty() bool {
	return t.MinCostImpact == nil && t.Percentage == nil && t.Operator == nil
}

type NotificationChannel struct {
	ChannelType string   `json:"channel_type"`
	Recipients  []string `json:"recipients"`
}

type CreateMonitorRequest struct {
	Name        string               `json:"monitor_name"`
	Description *string              `json:"description,omitempty"`
	CostType    string               `json:"cost_type,omitempty"`
	FilterSet   map[string][]*string `json:"filter_set,omitempty"`
	GroupBy     []string             `json:"group_by,omitempty"`
	IsEnabled   bool                 `json:"is_enabled"`
	Threshold   *MonitorThreshold    `json:"threshold,omitempty"`
}

// UpdateMonitorRequest is a PATCH body: nil fields are left out and keep their stored values.
// To clear a field, point at its empty value: an empty map, an empty slice, or an empty threshold.
type UpdateMonitorRequest struct {
	Name *string `json:"monitor_name,omitempty"`
	// A non-nil pointer to a nil string sends null, which clears the description.
	Description **string              `json:"description,omitempty"`
	CostType    *string               `json:"cost_type,omitempty"`
	FilterSet   *map[string][]*string `json:"filter_set,omitempty"`
	GroupBy     *[]string             `json:"group_by,omitempty"`
	IsEnabled   *bool                 `json:"is_enabled,omitempty"`
	Threshold   *MonitorThreshold     `json:"threshold,omitempty"`
}

func monitorPath(id string) string {
	return "/v2/monitors/" + url.PathEscape(id)
}

func (c *Client) CreateMonitor(ctx context.Context, req CreateMonitorRequest) (*Monitor, error) {
	var resp Monitor
	err := c.do(ctx, http.MethodPost, "/v2/monitors", req, &resp)
	return &resp, err
}

func (c *Client) GetMonitor(ctx context.Context, id string) (*Monitor, error) {
	var resp Monitor
	err := c.do(ctx, http.MethodGet, monitorPath(id), nil, &resp)
	return &resp, err
}

func (c *Client) UpdateMonitor(ctx context.Context, id string, req UpdateMonitorRequest) (*Monitor, error) {
	var resp Monitor
	err := c.do(ctx, http.MethodPatch, monitorPath(id), req, &resp)
	return &resp, err
}

// UpdateMonitorNotifications replaces every notification channel on the monitor. An empty slice clears them.
func (c *Client) UpdateMonitorNotifications(ctx context.Context, id string, channels []NotificationChannel) (*Monitor, error) {
	if channels == nil {
		channels = []NotificationChannel{}
	}
	body := struct {
		NotificationChannels []NotificationChannel `json:"notification_channels"`
	}{channels}
	var resp Monitor
	err := c.do(ctx, http.MethodPut, fmt.Sprintf("%s/notifications", monitorPath(id)), body, &resp)
	return &resp, err
}

func (c *Client) DeleteMonitor(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, monitorPath(id), nil, nil)
}
