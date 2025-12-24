// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package cognitoidp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	awstypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfcognitoidp "github.com/hashicorp/terraform-provider-aws/internal/service/cognitoidp"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccCognitoIDPManagedLoginTerms_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.TermsType
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cognito_managed_login_terms.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheckIdentityProvider(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CognitoIDPServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckManagedLoginTermsDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccManagedLoginTermsConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManagedLoginTermsExists(ctx, resourceName, &v),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("managed_login_terms_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("enforcement"), knownvalue.StringExact("NONE")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("terms_source"), knownvalue.StringExact("LINK")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("links"), knownvalue.MapSizeExact(1)),
				},
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "managed_login_terms_id",
				ImportStateIdFunc:                    acctest.AttrsImportStateIdFunc(resourceName, ",", names.AttrUserPoolID, "managed_login_terms_id"),
			},
		},
	})
}

func TestAccCognitoIDPManagedLoginTerms_update(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.TermsType
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cognito_managed_login_terms.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheckIdentityProvider(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CognitoIDPServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckManagedLoginTermsDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccManagedLoginTermsConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManagedLoginTermsExists(ctx, resourceName, &v),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
			},
			{
				Config: testAccManagedLoginTermsConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManagedLoginTermsExists(ctx, resourceName, &v),
					resource.TestMatchResourceAttr(resourceName, "terms_name", regexache.MustCompile(`^privacy-policy`)),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("links"), knownvalue.MapSizeExact(2)),
				},
			},
		},
	})
}

func TestAccCognitoIDPManagedLoginTerms_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.TermsType
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cognito_managed_login_terms.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheckIdentityProvider(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CognitoIDPServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckManagedLoginTermsDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccManagedLoginTermsConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManagedLoginTermsExists(ctx, resourceName, &v),
					acctest.CheckFrameworkResourceDisappears(ctx, acctest.Provider, tfcognitoidp.ResourceManagedLoginTerms, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckManagedLoginTermsDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).CognitoIDPClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_cognito_managed_login_terms" {
				continue
			}

			_, err := tfcognitoidp.FindManagedLoginTermsByTwoPartKey(ctx, conn, rs.Primary.Attributes[names.AttrUserPoolID], rs.Primary.Attributes["managed_login_terms_id"])

			if retry.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("Cognito Managed Login Terms %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckManagedLoginTermsExists(ctx context.Context, n string, v *awstypes.TermsType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).CognitoIDPClient(ctx)

		output, err := tfcognitoidp.FindManagedLoginTermsByTwoPartKey(ctx, conn, rs.Primary.Attributes[names.AttrUserPoolID], rs.Primary.Attributes["managed_login_terms_id"])

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccManagedLoginTermsConfig_base(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q
}

resource "aws_cognito_user_pool_client" "test" {
  name                = %[1]q
  user_pool_id        = aws_cognito_user_pool.test.id
  explicit_auth_flows = ["ADMIN_NO_SRP_AUTH"]
}
`, rName)
}

func testAccManagedLoginTermsConfig_basic(rName string) string {
	return acctest.ConfigCompose(testAccManagedLoginTermsConfig_base(rName), fmt.Sprintf(`
resource "aws_cognito_managed_login_terms" "test" {
  client_id    = aws_cognito_user_pool_client.test.id
  user_pool_id = aws_cognito_user_pool.test.id

  enforcement  = "NONE"
	terms_name   = "terms-of-use"
  terms_source = "LINK"

  links = {
    "cognito:default" = "https://example.com/%[1]s/terms"
  }
}
`, rName))
}

func testAccManagedLoginTermsConfig_updated(rName string) string {
	return acctest.ConfigCompose(testAccManagedLoginTermsConfig_base(rName), fmt.Sprintf(`
resource "aws_cognito_managed_login_terms" "test" {
  client_id    = aws_cognito_user_pool_client.test.id
  user_pool_id = aws_cognito_user_pool.test.id

  enforcement  = "NONE"
	terms_name   = "privacy-policy"
  terms_source = "LINK"

  links = {
    "cognito:default" = "https://example.com/%[1]s/privacy"
		"cognito:english"  = "https://example.com/%[1]s/en/privacy"
  }
}
`, rName))
}
