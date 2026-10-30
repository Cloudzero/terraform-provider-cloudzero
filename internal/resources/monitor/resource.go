//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

var (
	_ resource.Resource                   = &MonitorResource{}
	_ resource.ResourceWithConfigure      = &MonitorResource{}
	_ resource.ResourceWithImportState    = &MonitorResource{}
	_ resource.ResourceWithValidateConfig = &MonitorResource{}
)

type MonitorResource struct {
	client *client.Client
}

type MonitorResourceModel struct {
	ID            types.String               `tfsdk:"id"`
	Name          types.String               `tfsdk:"name"`
	Description   types.String               `tfsdk:"description"`
	Filter        types.Map                  `tfsdk:"filter"`
	GroupBy       types.List                 `tfsdk:"group_by"`
	CostType      types.String               `tfsdk:"cost_type"`
	Enabled       types.Bool                 `tfsdk:"enabled"`
	Threshold     *MonitorThresholdModel     `tfsdk:"threshold"`
	Notifications *MonitorNotificationsModel `tfsdk:"notifications"`
	OwnerID       types.String               `tfsdk:"owner_id"`
	CreatedAt     types.String               `tfsdk:"created_at"`
	UpdatedAt     types.String               `tfsdk:"updated_at"`
}

type MonitorThresholdModel struct {
	MinCostImpact types.Float64 `tfsdk:"min_cost_impact"`
	Percentage    types.Float64 `tfsdk:"percentage"`
	Operator      types.String  `tfsdk:"operator"`
}

type MonitorNotificationsModel struct {
	Email          types.Set    `tfsdk:"email"`
	SlackChannelID types.String `tfsdk:"slack_channel_id"`
}

func NewMonitorResource() resource.Resource {
	return &MonitorResource{}
}

func (r *MonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *MonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudZero anomaly monitor. Monitors created by CloudZero (system monitors) cannot be managed with Terraform.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Monitor identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Monitor name, 1-255 characters.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Monitor description, up to 1000 characters.",
			},
			"filter": schema.MapAttribute{
				Optional:    true,
				Description: "Spend to watch. Map of dimension ID (e.g. \"services\", \"accounts\", or \"costcontext:Team\") to the values to match. Prefix a dimension ID with \"!\" to exclude its values, \"&\" for case-insensitive partial matching, or both. Omit to watch all spend.",
				ElementType: types.ListType{ElemType: types.StringType},
			},
			"group_by": schema.ListAttribute{
				Optional:    true,
				Description: "Dimension ID (e.g. \"accounts\") to group by, detecting anomalies per value of that dimension. At most one dimension.",
				ElementType: types.StringType,
			},
			"cost_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("REAL_COST"),
				Description: "Cost type to monitor: " + strings.Join(costTypes, ", ") + ". Defaults to REAL_COST.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the monitor runs. Defaults to true.",
			},
			"owner_id": schema.StringAttribute{
				Computed:    true,
				Description: "Who created the monitor. Monitors created with an API key show \"Set via API\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 UTC timestamp of creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 UTC timestamp of last update.",
			},
		},
		Blocks: map[string]schema.Block{
			"threshold": schema.SingleNestedBlock{
				Description: "Minimum impact for a detected anomaly to raise an incident. Omit to raise an incident for every anomaly.",
				Attributes: map[string]schema.Attribute{
					"min_cost_impact": schema.Float64Attribute{
						Optional:    true,
						Description: "Minimum daily spend above the expected range, in dollars.",
					},
					"percentage": schema.Float64Attribute{
						Optional:    true,
						Description: "Minimum daily spend above the expected range, as a percentage of its upper end.",
					},
					"operator": schema.StringAttribute{
						Optional:    true,
						Description: "How to combine min_cost_impact and percentage: AND or OR. Required when both are set, and not allowed otherwise.",
					},
				},
			},
			"notifications": schema.SingleNestedBlock{
				Description: "Where incident notifications are sent. Omit to send none.",
				Attributes: map[string]schema.Attribute{
					"email": schema.SetAttribute{
						Optional:    true,
						Description: "Email addresses to notify, in lowercase.",
						ElementType: types.StringType,
					},
					"slack_channel_id": schema.StringAttribute{
						Optional:    true,
						Description: "Slack channel ID to notify (e.g. \"C0123ABCD9\"), not the channel name. The CloudZero Slack integration must be connected.",
					},
				},
			},
		},
	}
}

func (r *MonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MonitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config MonitorResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateMonitor(ctx, config)...)
}

func (r *MonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MonitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.CreateMonitorRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		CostType:    plan.CostType.ValueString(),
		FilterSet:   expandFilter(ctx, plan.Filter, &resp.Diagnostics),
		GroupBy:     expandGroupBy(ctx, plan.GroupBy, &resp.Diagnostics),
		IsEnabled:   plan.Enabled.ValueBool(),
		Threshold:   expandThreshold(plan.Threshold),
	}
	notifications := plan.Notifications
	channels := expandNotifications(ctx, notifications, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor, err := r.client.CreateMonitor(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", err.Error())
		return
	}

	// Save the monitor before setting notifications so a failure there leaves it tracked (and tainted) rather than orphaned.
	flattenMonitor(ctx, monitor, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() || notifications == nil {
		return
	}

	monitor, err = r.client.UpdateMonitorNotifications(ctx, monitor.ID, channels)
	if err != nil {
		resp.Diagnostics.AddError("Error setting monitor notifications", err.Error())
		return
	}

	flattenMonitor(ctx, monitor, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *MonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MonitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor, err := r.client.GetMonitor(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(checkManageable(monitor)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flattenMonitor(ctx, monitor, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *MonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MonitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state MonitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	apiReq, changed := buildUpdateRequest(ctx, plan, state, &resp.Diagnostics)
	notificationsChanged := !notificationsEqual(plan.Notifications, state.Notifications)
	channels := expandNotifications(ctx, plan.Notifications, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var monitor *client.Monitor
	var err error
	if changed {
		monitor, err = r.client.UpdateMonitor(ctx, id, apiReq)
		if err != nil {
			resp.Diagnostics.AddError("Error updating monitor", err.Error())
			return
		}
	}
	if notificationsChanged {
		monitor, err = r.client.UpdateMonitorNotifications(ctx, id, channels)
		if err != nil {
			resp.Diagnostics.AddError("Error updating monitor notifications", err.Error())
			return
		}
	}
	if monitor == nil {
		monitor, err = r.client.GetMonitor(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error reading monitor", err.Error())
			return
		}
	}

	flattenMonitor(ctx, monitor, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *MonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MonitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMonitor(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting monitor", err.Error())
	}
}

func (r *MonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
