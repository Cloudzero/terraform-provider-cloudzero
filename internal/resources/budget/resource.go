package budget

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

var (
	_ resource.Resource                = &BudgetResource{}
	_ resource.ResourceWithConfigure   = &BudgetResource{}
	_ resource.ResourceWithImportState = &BudgetResource{}
)

type BudgetResource struct {
	client *client.Client
}

type BudgetResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ViewID        types.String `tfsdk:"view_id"`
	PlannedLimits types.Map    `tfsdk:"planned_limits"`
	Alerts        types.Map    `tfsdk:"alerts"`
	CostType      types.String `tfsdk:"cost_type"`
	Granularity   types.String `tfsdk:"granularity"`
	Created       types.String `tfsdk:"created"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

func NewBudgetResource() resource.Resource {
	return &BudgetResource{}
}

func (r *BudgetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_budget"
}

func (r *BudgetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudZero Budget.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Budget identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Budget name, must be unique within a View.",
			},
			"view_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the View this budget is associated with.",
			},
			"planned_limits": schema.MapAttribute{
				Required:    true,
				Description: "Monthly planned limits. Map of ISO date string to dollar amount (e.g. {\"2024-01-01T00:00:00+00:00\" = \"1000.00\"}).",
				ElementType: types.StringType,
			},
			"alerts": schema.MapAttribute{
				Optional:    true,
				Description: "Alert thresholds. Map of percentage (1-100) to Y/N (e.g. {\"80\" = \"N\", \"100\" = \"Y\"}).",
				ElementType: types.StringType,
			},
			"cost_type": schema.StringAttribute{
				Computed:    true,
				Description: "Budget cost type (always real_cost).",
			},
			"granularity": schema.StringAttribute{
				Computed:    true,
				Description: "Budget granularity (always monthly).",
			},
			"created": schema.StringAttribute{
				Computed:    true,
				Description: "UTC epoch timestamp of creation.",
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "UTC epoch timestamp of last update.",
			},
		},
	}
}

func (r *BudgetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BudgetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BudgetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.CreateBudgetRequest{
		Name: plan.Name.ValueString(),
		View: client.BudgetView{ID: plan.ViewID.ValueString()},
		PlannedLimits: expandPlannedLimits(ctx, plan.PlannedLimits, &resp.Diagnostics),
		Alerts:        expandAlerts(ctx, plan.Alerts, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	budget, err := r.client.CreateBudget(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating budget", err.Error())
		return
	}

	flattenBudget(ctx, budget, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *BudgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BudgetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	budget, err := r.client.GetBudget(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading budget", err.Error())
		return
	}

	flattenBudget(ctx, budget, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *BudgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BudgetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state BudgetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	viewID := plan.ViewID.ValueString()
	limits := expandPlannedLimits(ctx, plan.PlannedLimits, &resp.Diagnostics)
	alerts := expandAlerts(ctx, plan.Alerts, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.UpdateBudgetRequest{
		Name:          &name,
		View:          &client.BudgetView{ID: viewID},
		PlannedLimits: &limits,
		Alerts:        alerts,
	}

	budget, err := r.client.UpdateBudget(ctx, state.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating budget", err.Error())
		return
	}

	flattenBudget(ctx, budget, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *BudgetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BudgetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBudget(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting budget", err.Error())
		return
	}
}

func (r *BudgetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
