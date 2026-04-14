//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-cloudzero/internal/client"
	awsaccountresource "terraform-provider-cloudzero/internal/resources/aws_account"
	budgetresource "terraform-provider-cloudzero/internal/resources/budget"
	insightresource "terraform-provider-cloudzero/internal/resources/insight"
	viewresource "terraform-provider-cloudzero/internal/resources/view"
	budgetsdatasource "terraform-provider-cloudzero/internal/datasources/budgets"
	viewsdatasource "terraform-provider-cloudzero/internal/datasources/views"
)

var _ provider.Provider = &CloudZeroProvider{}

type CloudZeroProvider struct {
	version string
}

type CloudZeroProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	Host    types.String `tfsdk:"host"`
	TestKey types.String `tfsdk:"test_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CloudZeroProvider{
			version: version,
		}
	}
}

func (p *CloudZeroProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cloudzero"
	resp.Version = p.version
}

func (p *CloudZeroProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage CloudZero resources. Requires an API key from " +
			"https://app.cloudzero.com/organization/api-keys.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "CloudZero API key. May also be set via the " +
					"CLOUDZERO_API_KEY environment variable.",
			},
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "CloudZero API host. Defaults to https://api.cloudzero.com. " +
					"May also be set via the CLOUDZERO_HOST environment variable.",
			},
			"test_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "If set, all API mutations are scoped to this test namespace " +
					"and expire after 1 hour. May also be set via the CLOUDZERO_TEST_KEY " +
					"environment variable. Used for acceptance testing.",
			},
		},
	}
}

func (p *CloudZeroProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config CloudZeroProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown CloudZero API Key",
			"The provider cannot create the API client as the api_key is not yet known. "+
				"Set the value statically in the configuration or use the CLOUDZERO_API_KEY "+
				"environment variable.",
		)
		return
	}

	apiKey := os.Getenv("CLOUDZERO_API_KEY")
	host := os.Getenv("CLOUDZERO_HOST")
	testKey := os.Getenv("CLOUDZERO_TEST_KEY")

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}
	if !config.TestKey.IsNull() {
		testKey = config.TestKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing CloudZero API Key",
			"Set api_key in the provider configuration or the CLOUDZERO_API_KEY "+
				"environment variable.",
		)
		return
	}

	if host == "" {
		host = "https://api.cloudzero.com"
	}

	c := client.New(host, apiKey, testKey, p.version)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *CloudZeroProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		awsaccountresource.NewAWSAccountResource,
		viewresource.NewViewResource,
		budgetresource.NewBudgetResource,
		insightresource.NewInsightResource,
	}
}

func (p *CloudZeroProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		viewsdatasource.NewViewsDataSource,
		budgetsdatasource.NewBudgetsDataSource,
	}
}
