//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package budgets

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

var (
	_ datasource.DataSource              = &BudgetsDataSource{}
	_ datasource.DataSourceWithConfigure = &BudgetsDataSource{}
)

type BudgetsDataSource struct {
	client *client.Client
}

type BudgetsDataSourceModel struct {
	Budgets []BudgetModel `tfsdk:"budgets"`
}

type BudgetModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ViewID      types.String `tfsdk:"view_id"`
	CostType    types.String `tfsdk:"cost_type"`
	Granularity types.String `tfsdk:"granularity"`
	Created     types.String `tfsdk:"created"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func NewBudgetsDataSource() datasource.DataSource {
	return &BudgetsDataSource{}
}

func (d *BudgetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_budgets"
}

func (d *BudgetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all CloudZero Budgets.",
		Attributes: map[string]schema.Attribute{
			"budgets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
						"name":         schema.StringAttribute{Computed: true},
						"view_id":      schema.StringAttribute{Computed: true},
						"cost_type":    schema.StringAttribute{Computed: true},
						"granularity":  schema.StringAttribute{Computed: true},
						"created":      schema.StringAttribute{Computed: true},
						"last_updated": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *BudgetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *BudgetsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	budgets, err := d.client.ListBudgets(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing budgets", err.Error())
		return
	}

	var state BudgetsDataSourceModel
	for _, b := range budgets {
		state.Budgets = append(state.Budgets, BudgetModel{
			ID:          types.StringValue(b.ID),
			Name:        types.StringValue(b.Name),
			ViewID:      types.StringValue(b.View.ID),
			CostType:    types.StringValue(b.CostType),
			Granularity: types.StringValue(b.Granularity),
			Created:     types.StringValue(b.Created),
			LastUpdated: types.StringValue(b.LastUpdated),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
