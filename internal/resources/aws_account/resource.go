//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package aws_account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-cloudzero/internal/client"
)

var (
	_ resource.Resource              = &AWSAccountResource{}
	_ resource.ResourceWithConfigure = &AWSAccountResource{}
)

type AWSAccountResource struct {
	client *client.Client
}

type AWSAccountResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	CloudAccountID      types.String `tfsdk:"cloud_account_id"`
	ExternalID          types.String `tfsdk:"external_id"`
	RoleARN             types.String `tfsdk:"role_arn"`
	AccountName         types.String `tfsdk:"account_name"`
	CloudRegion         types.String `tfsdk:"cloud_region"`
	BucketName          types.String `tfsdk:"bucket_name"`
	BucketPath          types.String `tfsdk:"bucket_path"`
	IsMasterPayer       types.Bool   `tfsdk:"is_master_payer"`
	TransactionID       types.String `tfsdk:"transaction_id"`
}

func NewAWSAccountResource() resource.Resource {
	return &AWSAccountResource{}
}

func (r *AWSAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_account"
}

func (r *AWSAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers an AWS account with CloudZero. Creates the connection " +
			"that enables CloudZero to assume into the account and read cost data.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier (cloud_account_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_account_id": schema.StringAttribute{
				Required:    true,
				Description: "AWS account ID (12-digit number).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"external_id": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "CloudZero external ID for cross-account trust. " +
					"Found at https://app.cloudzero.com/organization/connections/new/aws/resource/manual",
			},
			"role_arn": schema.StringAttribute{
				Required:    true,
				Description: "ARN of the cross-account IAM role created for CloudZero.",
			},
			"account_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Friendly name for the account in CloudZero.",
			},
			"cloud_region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("us-east-1"),
				Description: "AWS region. Defaults to us-east-1.",
			},
			"bucket_name": schema.StringAttribute{
				Optional:    true,
				Description: "S3 bucket name containing CUR data. Required for management/payer accounts.",
			},
			"bucket_path": schema.StringAttribute{
				Optional:    true,
				Description: "S3 key prefix for CUR report files within the bucket.",
			},
			"is_master_payer": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether this is a management/payer account.",
			},
			"transaction_id": schema.StringAttribute{
				Computed:    true,
				Description: "CloudZero transaction ID from the registration request.",
			},
		},
	}
}

func (r *AWSAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AWSAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AWSAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleARN := plan.RoleARN.ValueString()
	isMasterPayer := plan.IsMasterPayer.ValueBool()

	// Build the links based on account type
	links := client.AWSAccountLinkLinks{
		Audit:           client.AWSAccountLinkRole{RoleARN: nil},
		CloudTrailOwner: client.AWSAccountLinkCloudTrail{SQSQueueARN: nil, SQSQueuePolicyName: nil},
		MasterPayer:     client.AWSAccountLinkRole{RoleARN: nil},
		ResourceOwner:   client.AWSAccountLinkRole{RoleARN: nil},
		Legacy:          client.AWSAccountLinkRole{RoleARN: nil},
	}

	if isMasterPayer {
		links.MasterPayer.RoleARN = &roleARN
	} else {
		links.ResourceOwner.RoleARN = &roleARN
	}

	// Build discovery
	var bucketName *string
	var bucketPath *string
	if !plan.BucketName.IsNull() && !plan.BucketName.IsUnknown() {
		v := plan.BucketName.ValueString()
		bucketName = &v
	}
	if !plan.BucketPath.IsNull() && !plan.BucketPath.IsUnknown() {
		v := plan.BucketPath.ValueString()
		bucketPath = &v
	}

	apiReq := client.AWSAccountLinkRequest{
		Version:       "1",
		MessageSource: "cli",
		MessageType:   "account-link-provisioned",
		Data: client.AWSAccountLinkData{
			Metadata: client.AWSAccountLinkMetadata{
				CloudRegion:        plan.CloudRegion.ValueString(),
				ExternalID:         plan.ExternalID.ValueString(),
				CloudAccountID:     plan.CloudAccountID.ValueString(),
				CZAccountName:      plan.AccountName.ValueString(),
				ReactorID:          "terraform-provider-cloudzero",
				ReactorCallbackURL: "https://api.cloudzero.com/accounts/v1/link",
			},
			Links: links,
			Discovery: client.AWSAccountLinkDiscovery{
				IsAuditAccount:               false,
				IsCloudTrailOwnerAccount:      false,
				IsMasterPayerAccount:          isMasterPayer,
				IsOrganizationMasterAccount:   false,
				IsResourceOwnerAccount:        !isMasterPayer,
				RemoteCloudTrailBucket:        true,
				MasterPayerBillingBucketName:  bucketName,
				MasterPayerBillingBucketPath:  bucketPath,
			},
		},
	}

	result, err := r.client.RegisterAWSAccount(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error registering AWS account with CloudZero", err.Error())
		return
	}

	plan.ID = plan.CloudAccountID
	if result != nil {
		plan.TransactionID = types.StringValue(result.TransactionID)
	}

	tflog.Info(ctx, "Registered AWS account with CloudZero", map[string]interface{}{
		"cloud_account_id": plan.CloudAccountID.ValueString(),
		"transaction_id":   plan.TransactionID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *AWSAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// No-op: no public API to read account connection status.
	// State is preserved as-is from the last Create or Update.
	var state AWSAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *AWSAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Re-register with updated values (same POST, idempotent on the backend)
	var plan AWSAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AWSAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleARN := plan.RoleARN.ValueString()
	isMasterPayer := plan.IsMasterPayer.ValueBool()

	links := client.AWSAccountLinkLinks{
		Audit:           client.AWSAccountLinkRole{RoleARN: nil},
		CloudTrailOwner: client.AWSAccountLinkCloudTrail{SQSQueueARN: nil, SQSQueuePolicyName: nil},
		MasterPayer:     client.AWSAccountLinkRole{RoleARN: nil},
		ResourceOwner:   client.AWSAccountLinkRole{RoleARN: nil},
		Legacy:          client.AWSAccountLinkRole{RoleARN: nil},
	}

	if isMasterPayer {
		links.MasterPayer.RoleARN = &roleARN
	} else {
		links.ResourceOwner.RoleARN = &roleARN
	}

	var bucketName *string
	var bucketPath *string
	if !plan.BucketName.IsNull() && !plan.BucketName.IsUnknown() {
		v := plan.BucketName.ValueString()
		bucketName = &v
	}
	if !plan.BucketPath.IsNull() && !plan.BucketPath.IsUnknown() {
		v := plan.BucketPath.ValueString()
		bucketPath = &v
	}

	apiReq := client.AWSAccountLinkRequest{
		Version:       "1",
		MessageSource: "cli",
		MessageType:   "account-link-provisioned",
		Data: client.AWSAccountLinkData{
			Metadata: client.AWSAccountLinkMetadata{
				CloudRegion:        plan.CloudRegion.ValueString(),
				ExternalID:         plan.ExternalID.ValueString(),
				CloudAccountID:     plan.CloudAccountID.ValueString(),
				CZAccountName:      plan.AccountName.ValueString(),
				ReactorID:          "terraform-provider-cloudzero",
				ReactorCallbackURL: "https://api.cloudzero.com/accounts/v1/link",
			},
			Links: links,
			Discovery: client.AWSAccountLinkDiscovery{
				IsAuditAccount:               false,
				IsCloudTrailOwnerAccount:      false,
				IsMasterPayerAccount:          isMasterPayer,
				IsOrganizationMasterAccount:   false,
				IsResourceOwnerAccount:        !isMasterPayer,
				RemoteCloudTrailBucket:        true,
				MasterPayerBillingBucketName:  bucketName,
				MasterPayerBillingBucketPath:  bucketPath,
			},
		},
	}

	result, err := r.client.RegisterAWSAccount(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AWS account registration", err.Error())
		return
	}

	plan.ID = plan.CloudAccountID
	if result != nil {
		plan.TransactionID = types.StringValue(result.TransactionID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *AWSAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AWSAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning(
		"Account deregistration not supported",
		"CloudZero does not currently support programmatic account deregistration. "+
			"The account connection for "+state.CloudAccountID.ValueString()+" will remain active in CloudZero. "+
			"Remove it manually at Organization > Connected Accounts in the CloudZero UI if needed.",
	)
}
