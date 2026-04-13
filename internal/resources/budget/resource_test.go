//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package budget_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-cloudzero/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudzero": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccBudgetResource_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	updatedName := rName + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccBudgetConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudzero_budget.test", "id"),
					resource.TestCheckResourceAttr("cloudzero_budget.test", "name", rName),
					resource.TestCheckResourceAttrSet("cloudzero_budget.test", "view_id"),
					resource.TestCheckResourceAttr("cloudzero_budget.test", "cost_type", "real_cost"),
					resource.TestCheckResourceAttr("cloudzero_budget.test", "granularity", "monthly"),
					resource.TestCheckResourceAttrSet("cloudzero_budget.test", "created"),
					resource.TestCheckResourceAttrSet("cloudzero_budget.test", "last_updated"),
				),
			},
			// Import
			{
				ResourceName:      "cloudzero_budget.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name
			{
				Config: testAccBudgetConfigUpdated(updatedName),
				Check: resource.TestCheckResourceAttr("cloudzero_budget.test", "name", updatedName),
			},
			// Destroy is implicit
		},
	})
}

func testAccBudgetConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudzero_budget" "test" {
  name    = %q
  view_id = "DEFAULT"

  planned_limits = {
    "2025-01-01T00:00:00+00:00" = "10000"
    "2025-02-01T00:00:00+00:00" = "10000"
  }

  alerts = {
    "80"  = "N"
    "100" = "Y"
  }
}
`, name)
}

func testAccBudgetConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "cloudzero_budget" "test" {
  name    = %q
  view_id = "DEFAULT"

  planned_limits = {
    "2025-01-01T00:00:00+00:00" = "15000"
    "2025-02-01T00:00:00+00:00" = "15000"
  }

  alerts = {
    "80"  = "Y"
    "100" = "Y"
  }
}
`, name)
}
