package client

import (
	"context"
	"fmt"
	"net/http"
)

// Budget represents a CloudZero Budget.
type Budget struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	View         BudgetView                `json:"view"`
	PlannedLimits BudgetPlannedLimits      `json:"planned_limits"`
	Alerts       map[string]string         `json:"alerts,omitempty"`
	CostType     string                    `json:"cost_type,omitempty"`
	Granularity  string                    `json:"granularity,omitempty"`
	Created      string                    `json:"created,omitempty"`
	LastUpdated  string                    `json:"last_updated,omitempty"`
}

type BudgetView struct {
	ID string `json:"id"`
}

// BudgetPlannedLimits maps period (e.g. "monthly") to date-amount pairs.
type BudgetPlannedLimits struct {
	Monthly map[string]BudgetAmount `json:"monthly"`
}

// BudgetAmount holds a string dollar amount.
type BudgetAmount struct {
	Amount string `json:"amount"`
}

type CreateBudgetRequest struct {
	Name          string              `json:"name"`
	View          BudgetView          `json:"view"`
	PlannedLimits BudgetPlannedLimits `json:"planned_limits"`
	Alerts        map[string]string   `json:"alerts,omitempty"`
}

type UpdateBudgetRequest struct {
	Name          *string              `json:"name,omitempty"`
	View          *BudgetView          `json:"view,omitempty"`
	PlannedLimits *BudgetPlannedLimits `json:"planned_limits,omitempty"`
	Alerts        map[string]string    `json:"alerts,omitempty"`
}

func (c *Client) CreateBudget(ctx context.Context, req CreateBudgetRequest) (*Budget, error) {
	var resp struct {
		Budget Budget `json:"budget"`
	}
	err := c.do(ctx, http.MethodPost, "/v2/budgets", req, &resp)
	return &resp.Budget, err
}

func (c *Client) GetBudget(ctx context.Context, id string) (*Budget, error) {
	var resp struct {
		Budget Budget `json:"budget"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v2/budgets/%s", id), nil, &resp)
	return &resp.Budget, err
}

func (c *Client) UpdateBudget(ctx context.Context, id string, req UpdateBudgetRequest) (*Budget, error) {
	var resp struct {
		Budget Budget `json:"budget"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/v2/budgets/%s", id), req, &resp)
	return &resp.Budget, err
}

func (c *Client) DeleteBudget(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v2/budgets/%s", id), nil, nil)
}

func (c *Client) ListBudgets(ctx context.Context) ([]Budget, error) {
	return Paginate(ctx, func(ctx context.Context, cursor string) ([]Budget, string, error) {
		p := "/v2/budgets"
		if cursor != "" {
			p += "?cursor=" + cursor
		}
		var resp struct {
			Budgets    []Budget `json:"budgets"`
			Pagination struct {
				Cursor struct {
					NextCursor string `json:"next_cursor"`
				} `json:"cursor"`
			} `json:"pagination"`
		}
		err := c.do(ctx, http.MethodGet, p, nil, &resp)
		return resp.Budgets, resp.Pagination.Cursor.NextCursor, err
	})
}
