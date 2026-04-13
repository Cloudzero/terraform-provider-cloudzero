//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
)

// Insight represents a CloudZero Insight (anomaly alert / recommendation).
type Insight struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Link        string `json:"link,omitempty"`
	Status      string `json:"status,omitempty"`
	Effort      string `json:"effort,omitempty"`
	CostImpact  string `json:"cost_impact,omitempty"`
	Created     string `json:"created,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
}

type CreateInsightRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	Link        *string `json:"link,omitempty"`
	Status      *string `json:"status,omitempty"`
	Effort      *string `json:"effort,omitempty"`
	CostImpact  *string `json:"cost_impact,omitempty"`
}

type UpdateInsightRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	Link        *string `json:"link,omitempty"`
	Status      *string `json:"status,omitempty"`
	Effort      *string `json:"effort,omitempty"`
	CostImpact  *string `json:"cost_impact,omitempty"`
}

func (c *Client) CreateInsight(ctx context.Context, req CreateInsightRequest) (*Insight, error) {
	var resp struct {
		Insight Insight `json:"insight"`
	}
	err := c.do(ctx, http.MethodPost, "/v2/insights", req, &resp)
	return &resp.Insight, err
}

func (c *Client) GetInsight(ctx context.Context, id string) (*Insight, error) {
	var resp struct {
		Insight Insight `json:"insight"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v2/insights/%s", id), nil, &resp)
	return &resp.Insight, err
}

func (c *Client) UpdateInsight(ctx context.Context, id string, req UpdateInsightRequest) (*Insight, error) {
	var resp struct {
		Insight Insight `json:"insight"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/v2/insights/%s", id), req, &resp)
	return &resp.Insight, err
}

func (c *Client) DeleteInsight(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v2/insights/%s", id), nil, nil)
}
