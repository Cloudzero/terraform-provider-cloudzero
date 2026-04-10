package insight

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
)

var (
	_ resource.Resource                = &InsightResource{}
	_ resource.ResourceWithConfigure   = &InsightResource{}
	_ resource.ResourceWithImportState = &InsightResource{}
)

type InsightResource struct {
	client *client.Client
}

type InsightResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Category    types.String `tfsdk:"category"`
	Link        types.String `tfsdk:"link"`
	Status      types.String `tfsdk:"status"`
	Effort      types.String `tfsdk:"effort"`
	CostImpact  types.String `tfsdk:"cost_impact"`
	Created     types.String `tfsdk:"created"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func NewInsightResource() resource.Resource {
	return &InsightResource{}
}

func (r *InsightResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_insight"
}

func (r *InsightResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudZero Insight.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Insight identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "Insight title.",
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Detailed description of the insight.",
			},
			"category": schema.StringAttribute{
				Required:    true,
				Description: "Classification category.",
			},
			"link": schema.StringAttribute{
				Optional:    true,
				Description: "External URL reference.",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("new"),
				Description: "Status: new, in_progress, on_hold, addressed, or ignored.",
				Validators: []validator.String{
					stringOneOf("new", "in_progress", "on_hold", "addressed", "ignored"),
				},
			},
			"effort": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("not_set"),
				Description: "Implementation effort: not_set, low, medium, or high.",
				Validators: []validator.String{
					stringOneOf("not_set", "low", "medium", "high"),
				},
			},
			"cost_impact": schema.StringAttribute{
				Optional:    true,
				Description: "Dollar amount of cost impact (e.g. \"1000.00\").",
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

func (r *InsightResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InsightResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InsightResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desc := plan.Description.ValueString()
	cat := plan.Category.ValueString()
	apiReq := client.CreateInsightRequest{
		Title:       plan.Title.ValueString(),
		Description: &desc,
		Category:    &cat,
	}
	if !plan.Link.IsNull() {
		v := plan.Link.ValueString()
		apiReq.Link = &v
	}
	if !plan.Status.IsNull() {
		v := plan.Status.ValueString()
		apiReq.Status = &v
	}
	if !plan.Effort.IsNull() {
		v := plan.Effort.ValueString()
		apiReq.Effort = &v
	}
	if !plan.CostImpact.IsNull() {
		v := plan.CostImpact.ValueString()
		apiReq.CostImpact = &v
	}

	insight, err := r.client.CreateInsight(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating insight", err.Error())
		return
	}

	flattenInsight(insight, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *InsightResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InsightResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	insight, err := r.client.GetInsight(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading insight", err.Error())
		return
	}

	flattenInsight(insight, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *InsightResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InsightResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state InsightResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	title := plan.Title.ValueString()
	apiReq := client.UpdateInsightRequest{
		Title: &title,
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		apiReq.Description = &v
	}
	if !plan.Category.IsNull() {
		v := plan.Category.ValueString()
		apiReq.Category = &v
	}
	if !plan.Link.IsNull() {
		v := plan.Link.ValueString()
		apiReq.Link = &v
	}
	if !plan.Status.IsNull() {
		v := plan.Status.ValueString()
		apiReq.Status = &v
	}
	if !plan.Effort.IsNull() {
		v := plan.Effort.ValueString()
		apiReq.Effort = &v
	}
	if !plan.CostImpact.IsNull() {
		v := plan.CostImpact.ValueString()
		apiReq.CostImpact = &v
	}

	insight, err := r.client.UpdateInsight(ctx, state.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating insight", err.Error())
		return
	}

	flattenInsight(insight, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *InsightResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InsightResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteInsight(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting insight", err.Error())
		return
	}
}

func (r *InsightResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenInsight(insight *client.Insight, model *InsightResourceModel) {
	model.ID = types.StringValue(insight.ID)
	model.Title = types.StringValue(insight.Title)
	model.Created = types.StringValue(insight.Created)
	model.LastUpdated = types.StringValue(insight.LastUpdated)

	if insight.Description != "" {
		model.Description = types.StringValue(insight.Description)
	}
	if insight.Category != "" {
		model.Category = types.StringValue(insight.Category)
	}
	if insight.Link != "" {
		model.Link = types.StringValue(insight.Link)
	}
	if insight.Status != "" {
		model.Status = types.StringValue(insight.Status)
	}
	if insight.Effort != "" {
		model.Effort = types.StringValue(insight.Effort)
	}
	if insight.CostImpact != "" {
		// Use the plan value if it's numerically equal to avoid drift from
		// trailing zero differences (API returns "500" for "500.00")
		if !model.CostImpact.IsNull() && normalizeCurrency(model.CostImpact.ValueString()) == normalizeCurrency(insight.CostImpact) {
			// Keep the plan value as-is
		} else {
			model.CostImpact = types.StringValue(insight.CostImpact)
		}
	}
}

// normalizeCurrency strips trailing zeros for comparison: "500.00" -> "500"
func normalizeCurrency(s string) string {
	// Parse and re-format to strip trailing zeros
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return fmt.Sprintf("%g", f)
	}
	return s
}
