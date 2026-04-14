//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package view_test

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

func TestAccViewResource_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	updatedName := rName + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccViewConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudzero_view.test", "id"),
					resource.TestCheckResourceAttr("cloudzero_view.test", "name", rName),
					resource.TestCheckResourceAttr("cloudzero_view.test", "principal_dimension", "CZ:Account"),
					resource.TestCheckResourceAttrSet("cloudzero_view.test", "last_updated"),
				),
			},
			// Import
			{
				ResourceName:      "cloudzero_view.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name
			{
				Config: testAccViewConfig(updatedName),
				Check: resource.TestCheckResourceAttr("cloudzero_view.test", "name", updatedName),
			},
			// Destroy is implicit
		},
	})
}

func testAccViewConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudzero_view" "test" {
  name                = %q
  principal_dimension = "CZ:Account"

  filter = {}

  connections {
    email {
      include_all_organizers = false
    }
  }
}
`, name)
}
