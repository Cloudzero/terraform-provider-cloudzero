//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package views

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

var (
	_ datasource.DataSource              = &ViewsDataSource{}
	_ datasource.DataSourceWithConfigure = &ViewsDataSource{}
)

type ViewsDataSource struct {
	client *client.Client
}

type ViewsDataSourceModel struct {
	Views []ViewModel `tfsdk:"views"`
}

type ViewModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	PrincipalDimension types.String `tfsdk:"principal_dimension"`
	LastUpdated        types.String `tfsdk:"last_updated"`
}

func NewViewsDataSource() datasource.DataSource {
	return &ViewsDataSource{}
}

func (d *ViewsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_views"
}

func (d *ViewsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all CloudZero Views.",
		Attributes: map[string]schema.Attribute{
			"views": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                  schema.StringAttribute{Computed: true},
						"name":                schema.StringAttribute{Computed: true},
						"principal_dimension": schema.StringAttribute{Computed: true},
						"last_updated":        schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ViewsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ViewsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	views, err := d.client.ListViews(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing views", err.Error())
		return
	}

	var state ViewsDataSourceModel
	for _, v := range views {
		state.Views = append(state.Views, ViewModel{
			ID:                 types.StringValue(v.ID),
			Name:               types.StringValue(v.Name),
			PrincipalDimension: types.StringValue(v.PrincipalDimension),
			LastUpdated:        types.StringValue(v.LastUpdated),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
