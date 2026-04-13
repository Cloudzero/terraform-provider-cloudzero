//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package view

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

func expandFilter(ctx context.Context, m types.Map, diags *diag.Diagnostics) map[string][]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}

	var raw map[string][]string
	diags.Append(m.ElementsAs(ctx, &raw, false)...)
	return raw
}

func expandConnections(ctx context.Context, c *ViewConnectionsModel, diags *diag.Diagnostics) client.ViewConnections {
	if c == nil {
		return client.ViewConnections{}
	}

	result := client.ViewConnections{}

	if c.Email != nil {
		email := &client.ViewConnectionsEmail{
			IncludeAllOrganizers: c.Email.IncludeAllOrganizers.ValueBool(),
		}
		if !c.Email.Addresses.IsNull() && !c.Email.Addresses.IsUnknown() {
			var addrs []string
			diags.Append(c.Email.Addresses.ElementsAs(ctx, &addrs, false)...)
			email.Addresses = addrs
		}
		result.Email = email
	}

	for _, s := range c.Slack {
		result.Slack = append(result.Slack, client.ViewConnectionsSlack{
			ID:   s.ID.ValueString(),
			Name: s.Name.ValueString(),
		})
	}

	return result
}

func expandAnomalies(a *ViewAnomaliesModel) *client.ViewAnomalies {
	if a == nil {
		return nil
	}

	result := &client.ViewAnomalies{}

	if !a.Enabled.IsNull() && !a.Enabled.IsUnknown() {
		v := a.Enabled.ValueBool()
		result.Enabled = &v
	}
	if !a.ThresholdType.IsNull() && !a.ThresholdType.IsUnknown() {
		result.ThresholdType = a.ThresholdType.ValueString()
	}
	if !a.ThresholdValue.IsNull() && !a.ThresholdValue.IsUnknown() {
		v := int(a.ThresholdValue.ValueInt64())
		result.ThresholdValue = &v
	}
	if !a.MinCostImpact.IsNull() && !a.MinCostImpact.IsUnknown() {
		v := a.MinCostImpact.ValueFloat64()
		result.MinCostImpact = &v
	}

	return result
}

func flattenView(ctx context.Context, view *client.View, model *ViewResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(view.ID)
	model.Name = types.StringValue(view.Name)
	model.PrincipalDimension = types.StringValue(view.PrincipalDimension)
	model.LastUpdated = types.StringValue(view.LastUpdated)

	// Flatten filter
	if view.Filter != nil {
		filterMap := make(map[string][]string)
		for k, v := range view.Filter {
			filterMap[k] = v
		}
		m, d := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, filterMap)
		diags.Append(d...)
		model.Filter = m
	}

	// Flatten connections
	if model.Connections == nil {
		model.Connections = &ViewConnectionsModel{}
	}
	if view.Connections.Email != nil {
		if model.Connections.Email == nil {
			model.Connections.Email = &ViewConnectionsEmailModel{
				Addresses: types.ListNull(types.StringType),
			}
		}
		if len(view.Connections.Email.Addresses) > 0 {
			addrs, d := types.ListValueFrom(ctx, types.StringType, view.Connections.Email.Addresses)
			diags.Append(d...)
			model.Connections.Email.Addresses = addrs
		} else if !model.Connections.Email.Addresses.IsNull() {
			model.Connections.Email.Addresses = types.ListNull(types.StringType)
		}
		model.Connections.Email.IncludeAllOrganizers = types.BoolValue(view.Connections.Email.IncludeAllOrganizers)
	}
	model.Connections.Slack = nil
	for _, s := range view.Connections.Slack {
		model.Connections.Slack = append(model.Connections.Slack, ViewConnectionsSlackModel{
			ID:   types.StringValue(s.ID),
			Name: types.StringValue(s.Name),
		})
	}

	// Flatten anomalies — only populate if the user configured the block
	if view.Anomalies != nil && model.Anomalies != nil {
		if view.Anomalies.Enabled != nil {
			model.Anomalies.Enabled = types.BoolValue(*view.Anomalies.Enabled)
		}
		if view.Anomalies.ThresholdType != "" {
			model.Anomalies.ThresholdType = types.StringValue(view.Anomalies.ThresholdType)
		}
		if view.Anomalies.ThresholdValue != nil {
			model.Anomalies.ThresholdValue = types.Int64Value(int64(*view.Anomalies.ThresholdValue))
		}
		if view.Anomalies.MinCostImpact != nil {
			model.Anomalies.MinCostImpact = types.Float64Value(*view.Anomalies.MinCostImpact)
		}
	}
}
