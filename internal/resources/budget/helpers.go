package budget

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

// expandPlannedLimits converts the flat Terraform map (date -> amount string)
// into the API's nested structure { monthly: { date: { amount: string } } }.
func expandPlannedLimits(ctx context.Context, m types.Map, diags *diag.Diagnostics) client.BudgetPlannedLimits {
	result := client.BudgetPlannedLimits{
		Monthly: make(map[string]client.BudgetAmount),
	}

	if m.IsNull() || m.IsUnknown() {
		return result
	}

	var raw map[string]string
	diags.Append(m.ElementsAs(ctx, &raw, false)...)
	for date, amount := range raw {
		result.Monthly[date] = client.BudgetAmount{Amount: amount}
	}
	return result
}

// expandAlerts converts the Terraform map to a Go map.
func expandAlerts(ctx context.Context, m types.Map, diags *diag.Diagnostics) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}

	var raw map[string]string
	diags.Append(m.ElementsAs(ctx, &raw, false)...)
	return raw
}

// flattenBudget maps the API response back to the Terraform state model.
func flattenBudget(ctx context.Context, budget *client.Budget, model *BudgetResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(budget.ID)
	model.Name = types.StringValue(budget.Name)
	model.ViewID = types.StringValue(budget.View.ID)
	model.CostType = types.StringValue(budget.CostType)
	model.Granularity = types.StringValue(budget.Granularity)
	model.Created = types.StringValue(budget.Created)
	model.LastUpdated = types.StringValue(budget.LastUpdated)

	// Flatten planned_limits: { monthly: { date: { amount } } } -> flat map { date: amount }
	if budget.PlannedLimits.Monthly != nil {
		flat := make(map[string]string)
		for date, ba := range budget.PlannedLimits.Monthly {
			flat[date] = ba.Amount
		}
		m, d := types.MapValueFrom(ctx, types.StringType, flat)
		diags.Append(d...)
		model.PlannedLimits = m
	}

	// Flatten alerts
	if budget.Alerts != nil {
		m, d := types.MapValueFrom(ctx, types.StringType, budget.Alerts)
		diags.Append(d...)
		model.Alerts = m
	} else {
		model.Alerts = types.MapNull(types.StringType)
	}
}
