package client

import (
	"context"
	"fmt"
	"net/http"
)

// View represents a CloudZero View.
type View struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	PrincipalDimension string          `json:"principal_dimension"`
	Filter             map[string][]string `json:"filter"`
	Connections        ViewConnections `json:"connections"`
	Anomalies          *ViewAnomalies  `json:"anomalies,omitempty"`
	LastUpdated        string          `json:"last_updated,omitempty"`
	LastEdited         string          `json:"last_edited,omitempty"`
}

type ViewConnections struct {
	Email *ViewConnectionsEmail  `json:"email,omitempty"`
	Slack []ViewConnectionsSlack `json:"slack,omitempty"`
}

type ViewConnectionsEmail struct {
	Addresses          []string `json:"addresses,omitempty"`
	IncludeAllOrganizers bool   `json:"include_all_organizers,omitempty"`
}

type ViewConnectionsSlack struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ViewAnomalies struct {
	Enabled        *bool    `json:"enabled,omitempty"`
	ThresholdType  string   `json:"threshold_type,omitempty"`
	ThresholdValue *int     `json:"threshold_value,omitempty"`
	MinCostImpact  *float64 `json:"min_cost_impact,omitempty"`
}

type CreateViewRequest struct {
	Name               string              `json:"name"`
	PrincipalDimension string              `json:"principal_dimension"`
	Filter             map[string][]string `json:"filter"`
	Connections        ViewConnections     `json:"connections"`
	Anomalies          *ViewAnomalies      `json:"anomalies,omitempty"`
}

type UpdateViewRequest struct {
	Name               *string             `json:"name,omitempty"`
	PrincipalDimension *string             `json:"principal_dimension,omitempty"`
	Filter             map[string][]string `json:"filter,omitempty"`
	Connections        *ViewConnections    `json:"connections,omitempty"`
	Anomalies          *ViewAnomalies      `json:"anomalies,omitempty"`
}

func (c *Client) CreateView(ctx context.Context, req CreateViewRequest) (*View, error) {
	var view View
	err := c.do(ctx, http.MethodPost, "/v2/views", req, &view)
	return &view, err
}

func (c *Client) GetView(ctx context.Context, id string) (*View, error) {
	var resp struct {
		View View `json:"view"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v2/views/%s", id), nil, &resp)
	return &resp.View, err
}

func (c *Client) UpdateView(ctx context.Context, id string, req UpdateViewRequest) (*View, error) {
	var view View
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/v2/views/%s", id), req, &view)
	return &view, err
}

func (c *Client) DeleteView(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v2/views/%s", id), nil, nil)
}

func (c *Client) ListViews(ctx context.Context) ([]View, error) {
	return Paginate(ctx, func(ctx context.Context, cursor string) ([]View, string, error) {
		p := "/v2/views"
		if cursor != "" {
			p += "?cursor=" + cursor
		}
		var resp struct {
			Views      []View `json:"views"`
			Pagination struct {
				Cursor struct {
					NextCursor string `json:"next_cursor"`
				} `json:"cursor"`
			} `json:"pagination"`
		}
		err := c.do(ctx, http.MethodGet, p, nil, &resp)
		return resp.Views, resp.Pagination.Cursor.NextCursor, err
	})
}
