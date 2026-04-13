//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package insight_test

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

func TestAccInsightResource_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	updatedName := rName + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccInsightConfig(rName, "Test category", "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudzero_insight.test", "id"),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "title", rName),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "category", "Test category"),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "description", "Test description"),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "status", "new"),
					resource.TestCheckResourceAttrSet("cloudzero_insight.test", "created"),
					resource.TestCheckResourceAttrSet("cloudzero_insight.test", "last_updated"),
				),
			},
			// Import
			{
				ResourceName:      "cloudzero_insight.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update title and status
			{
				Config: testAccInsightConfigFull(updatedName, "Test category", "Updated description", "in_progress", "medium", "500.00"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudzero_insight.test", "title", updatedName),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "status", "in_progress"),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "effort", "medium"),
					resource.TestCheckResourceAttr("cloudzero_insight.test", "cost_impact", "500.00"),
				),
			},
			// Destroy is implicit
		},
	})
}

func testAccInsightConfig(title, category, description string) string {
	return fmt.Sprintf(`
resource "cloudzero_insight" "test" {
  title       = %q
  category    = %q
  description = %q
}
`, title, category, description)
}

func testAccInsightConfigFull(title, category, description, status, effort, costImpact string) string {
	return fmt.Sprintf(`
resource "cloudzero_insight" "test" {
  title       = %q
  category    = %q
  description = %q
  status      = %q
  effort      = %q
  cost_impact = %q
}
`, title, category, description, status, effort, costImpact)
}
