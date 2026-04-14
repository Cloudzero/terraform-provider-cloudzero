//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package view

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

var (
	_ resource.Resource                = &ViewResource{}
	_ resource.ResourceWithConfigure   = &ViewResource{}
	_ resource.ResourceWithImportState = &ViewResource{}
)

type ViewResource struct {
	client *client.Client
}

// ViewResourceModel maps the Terraform schema to Go types.
type ViewResourceModel struct {
	ID                 types.String          `tfsdk:"id"`
	Name               types.String          `tfsdk:"name"`
	PrincipalDimension types.String          `tfsdk:"principal_dimension"`
	Filter             types.Map             `tfsdk:"filter"`
	Connections        *ViewConnectionsModel `tfsdk:"connections"`
	Anomalies          *ViewAnomaliesModel   `tfsdk:"anomalies"`
	LastUpdated        types.String          `tfsdk:"last_updated"`
}

type ViewConnectionsModel struct {
	Email *ViewConnectionsEmailModel  `tfsdk:"email"`
	Slack []ViewConnectionsSlackModel `tfsdk:"slack"`
}

type ViewConnectionsEmailModel struct {
	Addresses            types.List `tfsdk:"addresses"`
	IncludeAllOrganizers types.Bool `tfsdk:"include_all_organizers"`
}

type ViewConnectionsSlackModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type ViewAnomaliesModel struct {
	Enabled        types.Bool    `tfsdk:"enabled"`
	ThresholdType  types.String  `tfsdk:"threshold_type"`
	ThresholdValue types.Int64   `tfsdk:"threshold_value"`
	MinCostImpact  types.Float64 `tfsdk:"min_cost_impact"`
}

func NewViewResource() resource.Resource {
	return &ViewResource{}
}

func (r *ViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view"
}

func (r *ViewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudZero View.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "View identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "View name.",
			},
			"principal_dimension": schema.StringAttribute{
				Required:    true,
				Description: "Primary group-by dimension for the view.",
			},
			"filter": schema.MapAttribute{
				Required:    true,
				Description: "Filter mapping of dimension name to allowed values.",
				ElementType: types.ListType{ElemType: types.StringType},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "UTC epoch timestamp of last update.",
			},
		},
		Blocks: map[string]schema.Block{
			"connections": schema.SingleNestedBlock{
				Description: "Notification recipients for this view.",
				Blocks: map[string]schema.Block{
					"email": schema.SingleNestedBlock{
						Description: "Email notification settings.",
						Attributes: map[string]schema.Attribute{
							"addresses": schema.ListAttribute{
								Optional:    true,
								Description: "Email addresses to notify.",
								ElementType: types.StringType,
							},
							"include_all_organizers": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
								Description: "Whether to notify all organizers in the organization.",
							},
						},
					},
					"slack": schema.ListNestedBlock{
						Description: "Slack channel notifications.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Required:    true,
									Description: "Slack channel ID.",
								},
								"name": schema.StringAttribute{
									Required:    true,
									Description: "Slack channel name.",
								},
							},
						},
					},
				},
			},
			"anomalies": schema.SingleNestedBlock{
				Description: "Anomaly detection configuration.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
						Description: "Whether anomaly detection is enabled.",
					},
					"threshold_type": schema.StringAttribute{
						Optional:    true,
						Description: "Threshold type: Automatic or Percent.",
					},
					"threshold_value": schema.Int64Attribute{
						Optional:    true,
						Description: "Threshold value. Ignored when threshold_type is Automatic.",
					},
					"min_cost_impact": schema.Float64Attribute{
						Optional:    true,
						Description: "Minimum dollar threshold for anomaly detection.",
					},
				},
			},
		},
	}
}

func (r *ViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *ViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.CreateViewRequest{
		Name:               plan.Name.ValueString(),
		PrincipalDimension: plan.PrincipalDimension.ValueString(),
		Filter:             expandFilter(ctx, plan.Filter, &resp.Diagnostics),
		Connections:        expandConnections(ctx, plan.Connections, &resp.Diagnostics),
		Anomalies:          expandAnomalies(plan.Anomalies),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	view, err := r.client.CreateView(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating view", err.Error())
		return
	}

	flattenView(ctx, view, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	view, err := r.client.GetView(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading view", err.Error())
		return
	}

	flattenView(ctx, view, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	pd := plan.PrincipalDimension.ValueString()
	apiReq := client.UpdateViewRequest{
		Name:               &name,
		PrincipalDimension: &pd,
		Filter:             expandFilter(ctx, plan.Filter, &resp.Diagnostics),
	}
	conns := expandConnections(ctx, plan.Connections, &resp.Diagnostics)
	apiReq.Connections = &conns
	apiReq.Anomalies = expandAnomalies(plan.Anomalies)
	if resp.Diagnostics.HasError() {
		return
	}

	view, err := r.client.UpdateView(ctx, state.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating view", err.Error())
		return
	}

	flattenView(ctx, view, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteView(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting view", err.Error())
		return
	}
}

func (r *ViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
