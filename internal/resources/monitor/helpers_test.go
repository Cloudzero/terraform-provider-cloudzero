//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

// A monitor as the API returns one created with only a name: every optional field at its empty default.
const minimalMonitorJSON = `{
	"monitor_id": "m-1",
	"monitor_name": "minimal",
	"description": null,
	"detection_type": "anomaly",
	"cost_type": "REAL_COST",
	"filter_set": {},
	"group_by": [],
	"is_enabled": true,
	"threshold": {"min_cost_impact": null, "percentage": null, "operator": null},
	"notification_channels": [],
	"owner_id": "Set via API",
	"created_at": "2026-10-30T00:00:00Z",
	"updated_at": "2026-10-30T00:00:00Z"
}`

func decodeMonitor(t *testing.T, s string) *client.Monitor {
	t.Helper()
	var m client.Monitor
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return &m
}

func TestFlattenMonitor_EmptyDefaultsBecomeNull(t *testing.T) {
	var model MonitorResourceModel
	var diags diag.Diagnostics
	flattenMonitor(context.Background(), decodeMonitor(t, minimalMonitorJSON), &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if !model.Description.IsNull() {
		t.Errorf("description = %v, want null", model.Description)
	}
	if !model.Filter.IsNull() {
		t.Errorf("filter = %v, want null", model.Filter)
	}
	if !model.GroupBy.IsNull() {
		t.Errorf("group_by = %v, want null", model.GroupBy)
	}
	if model.Threshold != nil {
		t.Errorf("threshold = %+v, want nil", model.Threshold)
	}
	if model.Notifications != nil {
		t.Errorf("notifications = %+v, want nil", model.Notifications)
	}
}

func TestFlattenMonitor_PopulatedFields(t *testing.T) {
	m := decodeMonitor(t, minimalMonitorJSON)
	m.FilterSet = map[string][]*string{"CZ:Service": {strPtr("AmazonS3"), nil}}
	m.GroupBy = []string{"CZ:Account"}
	m.Threshold = client.MonitorThreshold{Percentage: floatPtr(10)}
	m.NotificationChannels = []client.NotificationChannel{
		{ChannelType: "email", Recipients: []string{"a@example.com"}},
		{ChannelType: "slack", Recipients: []string{"C0123ABCD9"}},
	}

	var model MonitorResourceModel
	var diags diag.Diagnostics
	flattenMonitor(context.Background(), m, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	values := model.Filter.Elements()["CZ:Service"].(types.List).Elements()
	if len(values) != 2 || values[0].(types.String).ValueString() != "AmazonS3" || !values[1].IsNull() {
		t.Errorf("filter values = %v, want [AmazonS3, null]", values)
	}
	if got := len(model.GroupBy.Elements()); got != 1 {
		t.Errorf("group_by has %d elements, want 1", got)
	}
	if model.Threshold == nil || model.Threshold.Percentage.ValueFloat64() != 10 || !model.Threshold.MinCostImpact.IsNull() {
		t.Errorf("threshold = %+v, want percentage 10 only", model.Threshold)
	}
	if model.Notifications == nil || len(model.Notifications.Email.Elements()) != 1 ||
		model.Notifications.SlackChannelID.ValueString() != "C0123ABCD9" {
		t.Errorf("notifications = %+v", model.Notifications)
	}
}

func TestBuildUpdateRequest_SendsOnlyChangedFields(t *testing.T) {
	ctx := context.Background()
	var state MonitorResourceModel
	var diags diag.Diagnostics
	m := decodeMonitor(t, minimalMonitorJSON)
	m.FilterSet = map[string][]*string{"CZ:Service": {strPtr("AmazonS3")}}
	m.GroupBy = []string{"CZ:Account"}
	m.Threshold = client.MonitorThreshold{MinCostImpact: floatPtr(100)}
	m.Description = strPtr("old")
	flattenMonitor(ctx, m, &state, &diags)

	t.Run("no change", func(t *testing.T) {
		_, changed := buildUpdateRequest(ctx, state, state, &diags)
		if changed {
			t.Error("changed = true, want false")
		}
	})

	t.Run("rename leaves filter out", func(t *testing.T) {
		plan := state
		plan.Name = types.StringValue("renamed")
		req, changed := buildUpdateRequest(ctx, plan, state, &diags)
		assertJSON(t, req, changed, `{"monitor_name":"renamed"}`)
	})

	t.Run("removing optional fields clears them", func(t *testing.T) {
		plan := state
		plan.Description = types.StringNull()
		plan.Filter = types.MapNull(filterElementType)
		plan.GroupBy = types.ListNull(types.StringType)
		plan.Threshold = nil
		req, changed := buildUpdateRequest(ctx, plan, state, &diags)
		assertJSON(t, req, changed, `{"description":null,"filter_set":{},"group_by":[],"threshold":{}}`)
	})

	if diags.HasError() {
		t.Fatal(diags)
	}
}

func assertJSON(t *testing.T, req client.UpdateMonitorRequest, changed bool, want string) {
	t.Helper()
	if !changed {
		t.Error("changed = false, want true")
	}
	got, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestCheckManageable(t *testing.T) {
	m := decodeMonitor(t, minimalMonitorJSON)
	if d := checkManageable(m); d.HasError() {
		t.Errorf("user anomaly monitor rejected: %v", d)
	}

	m.OwnerID = "system"
	if d := checkManageable(m); !d.HasError() {
		t.Error("system monitor accepted")
	}

	m = decodeMonitor(t, minimalMonitorJSON)
	m.DetectionType = "trend"
	if d := checkManageable(m); !d.HasError() {
		t.Error("trend monitor accepted")
	}
}

func TestValidateMonitor(t *testing.T) {
	ctx := context.Background()
	valid := func() MonitorResourceModel {
		return MonitorResourceModel{
			Name:        types.StringValue("ok"),
			Description: types.StringNull(),
			CostType:    types.StringValue("REAL_COST"),
			GroupBy:     types.ListNull(types.StringType),
		}
	}
	emails := func(addrs ...string) types.Set {
		v, _ := types.SetValueFrom(ctx, types.StringType, addrs)
		return v
	}
	threshold := func(minImpact, pct *float64, op *string) *MonitorThresholdModel {
		return &MonitorThresholdModel{
			MinCostImpact: types.Float64PointerValue(minImpact),
			Percentage:    types.Float64PointerValue(pct),
			Operator:      types.StringPointerValue(op),
		}
	}

	cases := []struct {
		name    string
		mutate  func(*MonitorResourceModel)
		wantErr string
	}{
		{"valid", func(*MonitorResourceModel) {}, ""},
		{"bad cost type", func(m *MonitorResourceModel) { m.CostType = types.StringValue("NOPE") }, "Invalid cost type"},
		{"two group_by", func(m *MonitorResourceModel) {
			m.GroupBy, _ = types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
		}, "Too many group_by"},
		{"empty threshold", func(m *MonitorResourceModel) { m.Threshold = threshold(nil, nil, nil) }, "Empty threshold"},
		{"both limits without operator", func(m *MonitorResourceModel) {
			m.Threshold = threshold(floatPtr(1), floatPtr(5), nil)
		}, "Missing operator"},
		{"operator with one limit", func(m *MonitorResourceModel) {
			m.Threshold = threshold(floatPtr(1), nil, strPtr("AND"))
		}, "Unexpected operator"},
		{"both limits with operator", func(m *MonitorResourceModel) {
			m.Threshold = threshold(floatPtr(1), floatPtr(5), strPtr("OR"))
		}, ""},
		{"zero percentage", func(m *MonitorResourceModel) { m.Threshold = threshold(nil, floatPtr(0), nil) }, "Invalid percentage"},
		{"empty notifications", func(m *MonitorResourceModel) {
			m.Notifications = &MonitorNotificationsModel{Email: types.SetNull(types.StringType), SlackChannelID: types.StringNull()}
		}, "Empty notifications"},
		{"mixed-case email", func(m *MonitorResourceModel) {
			m.Notifications = &MonitorNotificationsModel{Email: emails("Ops@Example.com"), SlackChannelID: types.StringNull()}
		}, "must be lowercase"},
		{"slack channel name", func(m *MonitorResourceModel) {
			m.Notifications = &MonitorNotificationsModel{Email: types.SetNull(types.StringType), SlackChannelID: types.StringValue("#finops")}
		}, "Invalid Slack channel ID"},
		{"valid notifications", func(m *MonitorResourceModel) {
			m.Notifications = &MonitorNotificationsModel{Email: emails("ops@example.com"), SlackChannelID: types.StringValue("C0123ABCD9")}
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutate(&m)
			diags := validateMonitor(ctx, m)
			if tc.wantErr == "" {
				if diags.HasError() {
					t.Errorf("unexpected errors: %v", diags)
				}
				return
			}
			for _, d := range diags.Errors() {
				if strings.Contains(d.Summary(), tc.wantErr) {
					return
				}
			}
			t.Errorf("want error containing %q, got %v", tc.wantErr, diags)
		})
	}
}

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
