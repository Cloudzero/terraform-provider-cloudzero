//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var costTypes = []string{
	"REAL_COST",
	"AMORTIZED_COST",
	"BILLED_COST",
	"CUSTOM_COST",
	"DISCOUNTED_AMORTIZED_COST",
	"DISCOUNTED_COST",
	"INVOICED_AMORTIZED_COST",
	"ON_DEMAND_COST",
}

var thresholdOperators = []string{"AND", "OR"}

// Write-side bounds the API enforces on threshold settings; outside them create and update return 422.
const (
	minPercentage       = 0.01
	maxThresholdSetting = 1e12
)

// Same pattern the API enforces: a Slack conversation ID, never a channel name.
var slackChannelIDPattern = regexp.MustCompile(`^[CG][A-Z0-9]{8,}$`)

// validateMonitor checks at plan time what the API would otherwise reject or silently rewrite on apply.
func validateMonitor(ctx context.Context, m MonitorResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if known(m.Name) {
		if n := utf8.RuneCountInString(m.Name.ValueString()); n < 1 || n > 255 {
			diags.AddAttributeError(path.Root("name"), "Invalid name", "Must be 1-255 characters.")
		}
	}
	if known(m.Description) && utf8.RuneCountInString(m.Description.ValueString()) > 1000 {
		diags.AddAttributeError(path.Root("description"), "Invalid description", "Must be at most 1000 characters.")
	}
	if known(m.CostType) && !slices.Contains(costTypes, m.CostType.ValueString()) {
		diags.AddAttributeError(path.Root("cost_type"), "Invalid cost type",
			fmt.Sprintf("Expected one of: %s, got: %s", strings.Join(costTypes, ", "), m.CostType.ValueString()))
	}
	if known(m.GroupBy) && len(m.GroupBy.Elements()) > 1 {
		diags.AddAttributeError(path.Root("group_by"), "Too many group_by dimensions", "A monitor can group by at most one dimension.")
	}

	if t := m.Threshold; t != nil {
		diags.Append(validateThreshold(t)...)
	}
	if n := m.Notifications; n != nil {
		diags.Append(validateNotifications(ctx, n)...)
	}

	return diags
}

func validateThreshold(t *MonitorThresholdModel) diag.Diagnostics {
	var diags diag.Diagnostics
	p := path.Root("threshold")

	if t.MinCostImpact.IsUnknown() || t.Percentage.IsUnknown() || t.Operator.IsUnknown() {
		return diags
	}

	hasMin := !t.MinCostImpact.IsNull()
	hasPct := !t.Percentage.IsNull()
	hasOp := !t.Operator.IsNull()

	switch {
	case !hasMin && !hasPct:
		diags.AddAttributeError(p, "Empty threshold",
			"Set min_cost_impact, percentage, or both. To raise an incident for every anomaly, remove the threshold block.")
	case hasMin && hasPct && !hasOp:
		diags.AddAttributeError(p.AtName("operator"), "Missing operator",
			"operator (AND or OR) is required when both min_cost_impact and percentage are set.")
	case (!hasMin || !hasPct) && hasOp:
		diags.AddAttributeError(p.AtName("operator"), "Unexpected operator",
			"operator is only allowed when both min_cost_impact and percentage are set.")
	}

	if hasMin {
		if v := t.MinCostImpact.ValueFloat64(); v < 0 || v > maxThresholdSetting {
			diags.AddAttributeError(p.AtName("min_cost_impact"), "Invalid min_cost_impact",
				fmt.Sprintf("Must be between 0 and %.0f.", maxThresholdSetting))
		}
	}
	if hasPct {
		if v := t.Percentage.ValueFloat64(); v < minPercentage || v > maxThresholdSetting {
			diags.AddAttributeError(p.AtName("percentage"), "Invalid percentage",
				fmt.Sprintf("Must be between %g and %.0f.", minPercentage, maxThresholdSetting))
		}
	}
	if hasOp && !slices.Contains(thresholdOperators, t.Operator.ValueString()) {
		diags.AddAttributeError(p.AtName("operator"), "Invalid operator",
			fmt.Sprintf("Expected one of: %s, got: %s", strings.Join(thresholdOperators, ", "), t.Operator.ValueString()))
	}

	return diags
}

func validateNotifications(ctx context.Context, n *MonitorNotificationsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	p := path.Root("notifications")

	if n.Email.IsUnknown() || n.SlackChannelID.IsUnknown() {
		return diags
	}
	if n.Email.IsNull() && n.SlackChannelID.IsNull() {
		diags.AddAttributeError(p, "Empty notifications",
			"Set email, slack_channel_id, or both. To send no notifications, remove the notifications block.")
		return diags
	}

	if !n.Email.IsNull() {
		var addrs []types.String
		diags.Append(n.Email.ElementsAs(ctx, &addrs, false)...)
		if len(addrs) == 0 {
			diags.AddAttributeError(p.AtName("email"), "Empty email list",
				"Provide at least one address, or remove the email attribute.")
		}
		// The API trims and lowercases addresses, so any other spelling would read back as a change.
		for _, a := range addrs {
			if !known(a) {
				continue
			}
			addr := a.ValueString()
			if addr != strings.ToLower(strings.TrimSpace(addr)) {
				diags.AddAttributeError(p.AtName("email"), "Email address must be lowercase",
					fmt.Sprintf("Write %q as %q.", addr, strings.ToLower(strings.TrimSpace(addr))))
			}
		}
	}

	if known(n.SlackChannelID) && !slackChannelIDPattern.MatchString(n.SlackChannelID.ValueString()) {
		diags.AddAttributeError(p.AtName("slack_channel_id"), "Invalid Slack channel ID",
			fmt.Sprintf("Expected a Slack channel ID such as \"C0123ABCD9\", not a channel name. Got: %q", n.SlackChannelID.ValueString()))
	}

	return diags
}

func known(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}
