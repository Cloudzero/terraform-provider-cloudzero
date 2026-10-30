//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package monitor_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"terraform-provider-cloudzero/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudzero": providerserver.NewProtocol6WithError(provider.New("test")()),
}

const testResource = "cloudzero_monitor.test"

// Not parallel: the API rejects a second monitor with the same detection settings, and the minimal
// configs here differ only by name.
func TestAccMonitorResource_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	noChanges := resource.ConfigPlanChecks{
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:           testAccMonitorMinimalConfig(rName),
				ConfigPlanChecks: noChanges,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testResource, "id"),
					resource.TestCheckResourceAttr(testResource, "name", rName),
					resource.TestCheckResourceAttr(testResource, "cost_type", "ON_DEMAND_COST"),
					resource.TestCheckResourceAttr(testResource, "enabled", "true"),
					resource.TestCheckNoResourceAttr(testResource, "filter.%"),
					resource.TestCheckNoResourceAttr(testResource, "threshold.%"),
					resource.TestCheckNoResourceAttr(testResource, "notifications.%"),
				),
			},
			{
				Config:           testAccMonitorFullConfig(rName),
				ConfigPlanChecks: noChanges,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testResource, "filter.cloud_provider.0", "AWS"),
					resource.TestCheckResourceAttr(testResource, "group_by.0", "services"),
					resource.TestCheckResourceAttr(testResource, "threshold.min_cost_impact", "50"),
					resource.TestCheckResourceAttr(testResource, "threshold.percentage", "10"),
					resource.TestCheckResourceAttr(testResource, "threshold.operator", "AND"),
					resource.TestCheckResourceAttr(testResource, "notifications.email.#", "1"),
					resource.TestCheckResourceAttr(testResource, "enabled", "false"),
				),
			},
			{
				ResourceName:      testResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:           testAccMonitorFullConfig(rName + "-renamed"),
				ConfigPlanChecks: noChanges,
				Check:            resource.TestCheckResourceAttr(testResource, "name", rName+"-renamed"),
			},
			{
				Config:           testAccMonitorMinimalConfig(rName),
				ConfigPlanChecks: noChanges,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(testResource, "filter.%"),
					resource.TestCheckNoResourceAttr(testResource, "group_by.#"),
					resource.TestCheckNoResourceAttr(testResource, "threshold.%"),
					resource.TestCheckNoResourceAttr(testResource, "notifications.%"),
				),
			},
		},
	})
}

func TestAccMonitorResource_invalidConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "cloudzero_monitor" "test" {
  name = "invalid"
  notifications {
    email = ["Ops@Example.com"]
  }
}
`,
				ExpectError: regexp.MustCompile(`must be lowercase`),
			},
			{
				Config: `
resource "cloudzero_monitor" "test" {
  name = "invalid"
  threshold {
    min_cost_impact = 50
    percentage      = 10
  }
}
`,
				ExpectError: regexp.MustCompile(`Missing operator`),
			},
		},
	})
}

func testAccMonitorMinimalConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudzero_monitor" "test" {
  name      = %q
  cost_type = "ON_DEMAND_COST"
}
`, name)
}

func testAccMonitorFullConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudzero_monitor" "test" {
  name        = %q
  description = "Managed by the Terraform acceptance tests"
  cost_type   = "ON_DEMAND_COST"
  enabled     = false

  filter = {
    cloud_provider = ["AWS"]
  }
  group_by = ["services"]

  threshold {
    min_cost_impact = 50
    percentage      = 10
    operator        = "AND"
  }

  notifications {
    email = ["tf-acc@example.com"]
  }
}
`, name)
}
