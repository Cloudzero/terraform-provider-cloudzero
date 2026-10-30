//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

const (
	systemOwnerID        = "system"
	anomalyDetectionType = "anomaly"
	channelTypeEmail     = "email"
	channelTypeSlack     = "slack"
)

var filterElementType = types.ListType{ElemType: types.StringType}

// checkManageable rejects monitors this resource does not manage, so they never enter state (import runs Read).
func checkManageable(m *client.Monitor) diag.Diagnostics {
	var diags diag.Diagnostics
	if m.OwnerID == systemOwnerID {
		diags.AddError(
			"System monitor cannot be managed by Terraform",
			fmt.Sprintf("Monitor %s was created by CloudZero. Manage it in the CloudZero UI.", m.ID),
		)
	}
	if m.DetectionType != "" && m.DetectionType != anomalyDetectionType {
		diags.AddError(
			"Unsupported monitor type",
			fmt.Sprintf("Monitor %s is a %q monitor. cloudzero_monitor supports only anomaly monitors.", m.ID, m.DetectionType),
		)
	}
	return diags
}

func expandFilter(ctx context.Context, m types.Map, diags *diag.Diagnostics) map[string][]*string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}

	var raw map[string][]types.String
	diags.Append(m.ElementsAs(ctx, &raw, false)...)

	result := make(map[string][]*string, len(raw))
	for k, values := range raw {
		vs := make([]*string, len(values))
		for i, v := range values {
			vs[i] = v.ValueStringPointer()
		}
		result[k] = vs
	}
	return result
}

func expandGroupBy(ctx context.Context, l types.List, diags *diag.Diagnostics) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}

	var result []string
	diags.Append(l.ElementsAs(ctx, &result, false)...)
	return result
}

func expandThreshold(t *MonitorThresholdModel) *client.MonitorThreshold {
	if t == nil {
		return nil
	}
	return &client.MonitorThreshold{
		MinCostImpact: t.MinCostImpact.ValueFloat64Pointer(),
		Percentage:    t.Percentage.ValueFloat64Pointer(),
		Operator:      t.Operator.ValueStringPointer(),
	}
}

func expandNotifications(ctx context.Context, n *MonitorNotificationsModel, diags *diag.Diagnostics) []client.NotificationChannel {
	channels := []client.NotificationChannel{}
	if n == nil {
		return channels
	}

	if !n.Email.IsNull() && !n.Email.IsUnknown() {
		var addrs []string
		diags.Append(n.Email.ElementsAs(ctx, &addrs, false)...)
		channels = append(channels, client.NotificationChannel{ChannelType: channelTypeEmail, Recipients: addrs})
	}
	if !n.SlackChannelID.IsNull() && !n.SlackChannelID.IsUnknown() {
		channels = append(channels, client.NotificationChannel{
			ChannelType: channelTypeSlack,
			Recipients:  []string{n.SlackChannelID.ValueString()},
		})
	}
	return channels
}

// buildUpdateRequest sends only the fields that changed. Sending filter_set makes the API re-check
// it against cost data, so an unrelated edit like a rename must not include it.
func buildUpdateRequest(ctx context.Context, plan, state MonitorResourceModel, diags *diag.Diagnostics) (client.UpdateMonitorRequest, bool) {
	var req client.UpdateMonitorRequest
	changed := false

	if !plan.Name.Equal(state.Name) {
		req.Name = plan.Name.ValueStringPointer()
		changed = true
	}
	if !plan.Description.Equal(state.Description) {
		description := plan.Description.ValueStringPointer()
		req.Description = &description
		changed = true
	}
	if !plan.CostType.Equal(state.CostType) {
		req.CostType = plan.CostType.ValueStringPointer()
		changed = true
	}
	if !plan.Enabled.Equal(state.Enabled) {
		req.IsEnabled = plan.Enabled.ValueBoolPointer()
		changed = true
	}
	if !plan.Filter.Equal(state.Filter) {
		filter := expandFilter(ctx, plan.Filter, diags)
		if filter == nil {
			filter = map[string][]*string{}
		}
		req.FilterSet = &filter
		changed = true
	}
	if !plan.GroupBy.Equal(state.GroupBy) {
		groupBy := expandGroupBy(ctx, plan.GroupBy, diags)
		if groupBy == nil {
			groupBy = []string{}
		}
		req.GroupBy = &groupBy
		changed = true
	}
	if !thresholdEqual(plan.Threshold, state.Threshold) {
		req.Threshold = expandThreshold(plan.Threshold)
		if req.Threshold == nil {
			req.Threshold = &client.MonitorThreshold{}
		}
		changed = true
	}

	return req, changed
}

func thresholdEqual(a, b *MonitorThresholdModel) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.MinCostImpact.Equal(b.MinCostImpact) && a.Percentage.Equal(b.Percentage) && a.Operator.Equal(b.Operator)
}

func notificationsEqual(a, b *MonitorNotificationsModel) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Email.Equal(b.Email) && a.SlackChannelID.Equal(b.SlackChannelID)
}

// flattenMonitor maps the API's empty defaults ({}, [], an all-null threshold, no channels) to null,
// matching an omitted attribute or block. Anything else makes Terraform report an inconsistent result.
func flattenMonitor(ctx context.Context, m *client.Monitor, model *MonitorResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(m.ID)
	model.Name = types.StringValue(m.Name)
	model.Description = types.StringPointerValue(m.Description)
	model.CostType = types.StringValue(m.CostType)
	model.Enabled = types.BoolValue(m.IsEnabled)
	model.OwnerID = types.StringValue(m.OwnerID)
	model.CreatedAt = types.StringValue(m.CreatedAt)
	model.UpdatedAt = types.StringValue(m.UpdatedAt)

	if len(m.FilterSet) == 0 {
		model.Filter = types.MapNull(filterElementType)
	} else {
		filter := make(map[string][]types.String, len(m.FilterSet))
		for k, values := range m.FilterSet {
			vs := make([]types.String, len(values))
			for i, v := range values {
				vs[i] = types.StringPointerValue(v)
			}
			filter[k] = vs
		}
		v, d := types.MapValueFrom(ctx, filterElementType, filter)
		diags.Append(d...)
		model.Filter = v
	}

	if len(m.GroupBy) == 0 {
		model.GroupBy = types.ListNull(types.StringType)
	} else {
		v, d := types.ListValueFrom(ctx, types.StringType, m.GroupBy)
		diags.Append(d...)
		model.GroupBy = v
	}

	if m.Threshold.IsEmpty() {
		model.Threshold = nil
	} else {
		model.Threshold = &MonitorThresholdModel{
			MinCostImpact: types.Float64PointerValue(m.Threshold.MinCostImpact),
			Percentage:    types.Float64PointerValue(m.Threshold.Percentage),
			Operator:      types.StringPointerValue(m.Threshold.Operator),
		}
	}

	model.Notifications = flattenNotifications(ctx, m.NotificationChannels, diags)
}

func flattenNotifications(ctx context.Context, channels []client.NotificationChannel, diags *diag.Diagnostics) *MonitorNotificationsModel {
	if len(channels) == 0 {
		return nil
	}

	n := &MonitorNotificationsModel{
		Email:          types.SetNull(types.StringType),
		SlackChannelID: types.StringNull(),
	}
	for _, c := range channels {
		if len(c.Recipients) == 0 {
			continue
		}
		switch c.ChannelType {
		case channelTypeEmail:
			v, d := types.SetValueFrom(ctx, types.StringType, c.Recipients)
			diags.Append(d...)
			n.Email = v
		case channelTypeSlack:
			n.SlackChannelID = types.StringValue(c.Recipients[0])
		}
	}
	return n
}
