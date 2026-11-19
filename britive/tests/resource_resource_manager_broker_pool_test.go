package tests

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	britive "github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance tests for broker pools: britive_resource_manager_broker_pool,
// britive_resource_manager_broker_pool_token, britive_resource_manager_broker_pool_labels and
// the britive_resource_manager_broker_pool data source.
//
// Every test needs BRITIVE_TENANT and BRITIVE_TOKEN: besides running Terraform, the tests call
// the API directly to change things "outside Terraform" (drift) and to check what Terraform
// left behind. After every apply step the test framework also runs a plan and fails if it is
// not empty, so each step checks that the plan settles.
//
// Optional: BRITIVE_BROKER_POOL_WITH_BROKERS names an existing pool with at least one connected
// broker, for the data source broker check.

const (
	bpPool       = "britive_resource_manager_broker_pool.pool"
	bpPoolB      = "britive_resource_manager_broker_pool.pool_b"
	bpToken      = "britive_resource_manager_broker_pool_token.token"
	bpLabels     = "britive_resource_manager_broker_pool_labels.labels"
	bpData       = "data.britive_resource_manager_broker_pool.lookup"
	bpPoolType   = "britive_resource_manager_broker_pool"
	bpNotFoundRe = `(?i)(non-existent remote object|not found|Error Reading Broker Pool)`
)

// ---------------------------------------------------------------------------
// A. Broker pool resource
// ---------------------------------------------------------------------------

// TestBritiveBrokerPool covers A1, A2, A4–A10, A22, A24 and E6:
//  1. create with only a name: every optional field gets its default
//  2. set every optional field: updated in place, pool_id unchanged
//  3. rename and change every field again: updated in place, pool_id unchanged
//  4. remove every optional field: each one resets to its default, in state and in Britive
//  5. import by the resource id, and by the bare pool ID
//  6. refresh-only: no changes
//
// The pool resource does not touch the default token; that is covered by the default token
// tests. Destroy is checked by CheckDestroy.
func TestBritiveBrokerPool(t *testing.T) {
	client := testAccBritiveClient(t)
	name := "AT-BP-Lifecycle"
	renamed := "AT-BP-Lifecycle-Renamed"
	var poolID string

	full := func(n, desc string, keepAlive, disconnect int) string {
		return fmt.Sprintf(`
		resource "britive_resource_manager_broker_pool" "pool" {
			name               = %q
			description        = %q
			keep_alive_seconds = %d
			disconnect_seconds = %d
		}`, n, desc, keepAlive, disconnect)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolMinimal("pool", name),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttrSet(bpPool, "id"),
					resource.TestCheckResourceAttr(bpPool, "name", name),
					resource.TestCheckResourceAttr(bpPool, "description", ""),
					resource.TestCheckResourceAttr(bpPool, "keep_alive_seconds", "60"),
					resource.TestCheckResourceAttr(bpPool, "disconnect_seconds", "86400"),
					resource.TestCheckResourceAttr(bpPool, "broker_count", "0"),
					// Britive still creates the default token with the pool.
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
				),
			},
			{
				Config:           full(name, "AT broker pool", 120, 3600),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttr(bpPool, "description", "AT broker pool"),
					resource.TestCheckResourceAttr(bpPool, "keep_alive_seconds", "120"),
					resource.TestCheckResourceAttr(bpPool, "disconnect_seconds", "3600"),
					testAccCheckBrokerPoolAPI(client, &poolID, name, "AT broker pool", 120, 3600),
				),
			},
			{
				Config:           full(renamed, "AT broker pool updated", 180, 7200),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttr(bpPool, "name", renamed),
					testAccCheckBrokerPoolAPI(client, &poolID, renamed, "AT broker pool updated", 180, 7200),
				),
			},
			{
				Config:           testAccBrokerPoolMinimal("pool", renamed),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttr(bpPool, "description", ""),
					resource.TestCheckResourceAttr(bpPool, "keep_alive_seconds", "60"),
					resource.TestCheckResourceAttr(bpPool, "disconnect_seconds", "86400"),
					testAccCheckBrokerPoolAPI(client, &poolID, renamed, "", 60, 86400),
				),
			},
			{
				// Import by the resource id: resource-manager/broker-pools/{pool_id}.
				ResourceName:      bpPool,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Import by the bare pool ID.
				ResourceName:      bpPool,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return poolID, nil },
			},
			{
				RefreshState: true,
			},
		},
	})
}

// TestBritiveBrokerPoolValidation covers A11 and A12. These run as plan-only steps, so no
// API call is made.
func TestBritiveBrokerPoolValidation(t *testing.T) {
	pool := func(attr string) string {
		return fmt.Sprintf(`
		resource "britive_resource_manager_broker_pool" "pool" {
			name = "AT-BP-Validation"
			%s
		}`, attr)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      pool(`disconnect_seconds = 0`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`value must be between 1 and`),
			},
			{
				Config:      pool(`disconnect_seconds = -5`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`value must be between 1 and`),
			},
			{
				// The default token fields were removed from the pool resource.
				Config:      pool(`default_token_status = "ACTIVE"`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Unsupported argument`),
			},
			{
				// keep_alive_seconds has no provider-side range check.
				Config:             pool(`keep_alive_seconds = 30`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:             pool(`keep_alive_seconds = 5000`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestBritiveBrokerPoolDuplicateName covers A19: creating a second pool with a name that is
// already used fails with "already exists", and only one pool with that name is left.
func TestBritiveBrokerPoolDuplicateName(t *testing.T) {
	client := testAccBritiveClient(t)
	name := "AT-BP-Duplicate-Name"
	pool := testAccBrokerPoolMinimal("pool", name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: pool,
			},
			{
				Config:      pool + testAccBrokerPoolMinimal("pool_b", name),
				ExpectError: regexp.MustCompile(`(?i)already exists`),
			},
			{
				Config: pool,
				Check:  testAccCheckBrokerPoolNameCountAPI(client, name, 1),
			},
		},
	})
}

// TestBritiveBrokerPoolDrift covers A20 and A21:
//  1. create the pool
//  2. change it through the API: the plan updates it back. The default token is also changed
//     through the API; the pool resource ignores it and leaves it as it is
//  3. delete it through the API: the plan creates it again
func TestBritiveBrokerPoolDrift(t *testing.T) {
	client := testAccBritiveClient(t)
	name := "AT-BP-Drift"
	var poolID, firstPoolID string

	config := fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool" "pool" {
		name               = %q
		description        = "AT drift"
		keep_alive_seconds = 120
		disconnect_seconds = 3600
	}`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				PreConfig: func() {
					if _, err := client.UpdateBrokerPool(poolID, britive.RemoteBrokerPoolUpdateRequest{
						Name: name, Description: "changed outside Terraform", KeepAliveSeconds: 300, DisconnectSeconds: 1000,
					}); err != nil {
						t.Fatalf("changing pool outside Terraform: %v", err)
					}
					desc := "changed outside Terraform"
					if _, err := client.UpdateBrokerPoolToken(poolID, "default", britive.RemoteBrokerPoolTokenUpdateRequest{
						Description: &desc,
					}); err != nil {
						t.Fatalf("changing default token outside Terraform: %v", err)
					}
				},
				Config:           config,
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBrokerPoolAPI(client, &poolID, name, "AT drift", 120, 3600),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", "changed outside Terraform"),
				),
			},
			{
				PreConfig: func() {
					firstPoolID = poolID
					if err := client.DeleteBrokerPool(poolID); err != nil {
						t.Fatalf("deleting pool outside Terraform: %v", err)
					}
				},
				Config:           config,
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckAttrNotEquals(bpPool, "pool_id", &firstPoolID),
					testAccCheckBrokerPoolAPI(client, &poolID, name, "AT drift", 120, 3600),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolImportErrors covers A23, B19 and invalid label import IDs.
func TestBritiveBrokerPoolImportErrors(t *testing.T) {
	client := testAccBritiveClient(t)
	name := "AT-BP-Import-Errors"

	base := testAccBrokerPoolMinimal("pool", name)
	withChildren := base + `
	resource "britive_resource_manager_broker_pool" "missing" {
		name = "AT-BP-Import-Errors-Missing"
	}

	resource "britive_resource_manager_broker_pool_token" "token" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		name    = "AT-import-token"
		status  = "INACTIVE"
	}

	resource "britive_resource_manager_broker_pool_labels" "labels" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		resource_labels {
			label_key = "Environment"
			values    = []
		}
	}`

	poolIDThen := func(suffix string) resource.ImportStateIdFunc {
		return func(s *terraform.State) (string, error) {
			rs, ok := s.RootModule().Resources[bpPool]
			if !ok {
				return "", errs.NewNotFoundErrorf("%s in state", bpPool)
			}
			return rs.Primary.Attributes["pool_id"] + suffix, nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: base,
			},
			{
				// A23: pool that does not exist.
				Config:        withChildren,
				ResourceName:  "britive_resource_manager_broker_pool.missing",
				ImportState:   true,
				ImportStateId: "AT-does-not-exist",
				ExpectError:   regexp.MustCompile(bpNotFoundRe),
			},
			{
				// B19: token import ID without a token name.
				Config:            withChildren,
				ResourceName:      bpToken,
				ImportState:       true,
				ImportStateIdFunc: poolIDThen(""),
				ExpectError:       regexp.MustCompile(`Invalid Import ID`),
			},
			{
				// B19: token that does not exist.
				Config:            withChildren,
				ResourceName:      bpToken,
				ImportState:       true,
				ImportStateIdFunc: poolIDThen("/AT-no-such-token"),
				ExpectError:       regexp.MustCompile(bpNotFoundRe + `|Error Reading Broker Pool Token`),
			},
			{
				Config:        withChildren,
				ResourceName:  bpLabels,
				ImportState:   true,
				ImportStateId: "a/b/c",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// B. Broker pool token resource
// ---------------------------------------------------------------------------

// TestBritiveBrokerPoolToken covers B1–B8, B17 and B20:
//  1. create INACTIVE with a description: POST only, no PATCH needed
//  2. set ACTIVE and change the description: updated in place
//  3. set INACTIVE and remove the description: the description is cleared
//  4. rename: replaced
//  5. import by {pool_id}/{name}, and by the resource id
//  6. remove the token resource: the token is deleted, the pool keeps only its default token
func TestBritiveBrokerPoolToken(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Token"
	var poolID string

	token := func(name, attrs string) string {
		return testAccBrokerPoolMinimal("pool", poolName) + fmt.Sprintf(`
		resource "britive_resource_manager_broker_pool_token" "token" {
			pool_id = britive_resource_manager_broker_pool.pool.pool_id
			name    = %q
			%s
		}`, name, attrs)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: token("AT-token-a", `status = "INACTIVE"`+"\n"+`description = "AT token"`),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttrPair(bpToken, "pool_id", bpPool, "pool_id"),
					resource.TestCheckResourceAttr(bpToken, "status", "INACTIVE"),
					resource.TestCheckResourceAttr(bpToken, "description", "AT token"),
					// The token value is never stored in the state.
					resource.TestCheckNoResourceAttr(bpToken, "token_value"),
					resource.TestCheckResourceAttrSet(bpToken, "created"),
					testAccCheckTokenAPI(client, &poolID, "AT-token-a", "INACTIVE", "AT token"),
				),
			},
			{
				Config:           token("AT-token-a", `status = "ACTIVE"`+"\n"+`description = "AT token updated"`),
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpToken, "status", "ACTIVE"),
					resource.TestCheckResourceAttr(bpToken, "description", "AT token updated"),
					testAccCheckTokenAPI(client, &poolID, "AT-token-a", "ACTIVE", "AT token updated"),
				),
			},
			{
				Config:           token("AT-token-a", `status = "INACTIVE"`),
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpToken, "status", "INACTIVE"),
					resource.TestCheckNoResourceAttr(bpToken, "description"),
					testAccCheckTokenAPI(client, &poolID, "AT-token-a", "INACTIVE", ""),
				),
			},
			{
				Config:           token("AT-token-b", `status = "INACTIVE"`),
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionReplace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpToken, "name", "AT-token-b"),
					testAccCheckTokenNamesAPI(client, &poolID, "AT-token-b", "default"),
				),
			},
			{
				ResourceName:      bpToken,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[bpToken]
					if !ok {
						return "", errs.NewNotFoundErrorf("%s in state", bpToken)
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["pool_id"], rs.Primary.Attributes["name"]), nil
				},
			},
			{
				// Import by the resource id: resource-manager/broker-pools/{pool_id}/tokens/{name}.
				ResourceName:      bpToken,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBrokerPoolMinimal("pool", poolName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokenChangePool covers B9: moving a token to another pool replaces it.
func TestBritiveBrokerPoolTokenChangePool(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolA, poolB string

	config := func(target string) string {
		return testAccBrokerPoolMinimal("pool", "AT-BP-Token-Move-A") +
			testAccBrokerPoolMinimal("pool_b", "AT-BP-Token-Move-B") + fmt.Sprintf(`
		resource "britive_resource_manager_broker_pool_token" "token" {
			pool_id = britive_resource_manager_broker_pool.%s.pool_id
			name    = "AT-token-move"
			status  = "INACTIVE"
		}`, target)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config("pool"),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolA),
					testAccCaptureAttr(bpPoolB, "pool_id", &poolB),
					testAccCheckTokenNamesAPI(client, &poolA, "AT-token-move", "default"),
				),
			},
			{
				Config:           config("pool_b"),
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionReplace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(bpToken, "pool_id", bpPoolB, "pool_id"),
					testAccCheckTokenNamesAPI(client, &poolA, "default"),
					testAccCheckTokenNamesAPI(client, &poolB, "AT-token-move", "default"),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokenValidation covers B11, a missing status and an invalid status
// (plan-only, no API call). The name "default" is allowed.
func TestBritiveBrokerPoolTokenValidation(t *testing.T) {
	token := func(name, attrs string) string {
		return fmt.Sprintf(`
		resource "britive_resource_manager_broker_pool_token" "token" {
			pool_id = "unused"
			name    = %q
			%s
		}`, name, attrs)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             token("default", `status = "ACTIVE"`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:      token("AT-token", ""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`The argument "status" is required`),
			},
			{
				Config:      token("my token!", `status = "INACTIVE"`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must contain only letters, numbers, underscore\s+and\s+hyphen`),
			},
			{
				Config:      token("AT-token", `status = "enabled"`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

// TestBritiveBrokerPoolTokenDuplicateName covers B12: two token resources with the same name
// on one pool. The second create fails and nothing extra is left in Britive.
func TestBritiveBrokerPoolTokenDuplicateName(t *testing.T) {
	testAccBrokerPoolTokenLimit(t, "AT-BP-Token-Duplicate", `
	resource "britive_resource_manager_broker_pool_token" "one" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		name    = "AT-token-same"
		status  = "INACTIVE"
	}

	resource "britive_resource_manager_broker_pool_token" "two" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		name    = "AT-token-same"
		status  = "INACTIVE"
	}`, `Error Creating Broker Pool Token`)
}

// TestBritiveBrokerPoolTokenMaxTokens covers B13: a pool allows at most 3 tokens, the default
// token included, so a third extra token fails.
func TestBritiveBrokerPoolTokenMaxTokens(t *testing.T) {
	testAccBrokerPoolTokenLimit(t, "AT-BP-Token-Max", testAccBrokerPoolTokens("INACTIVE", 3),
		`Error Creating Broker Pool Token`)
}

// TestBritiveBrokerPoolTokenMaxActive covers B14: at most 2 tokens can be ACTIVE. The default
// token is ACTIVE, so the second extra ACTIVE token fails, and the provider deletes the token
// it created before failing (no orphan is left in Britive).
func TestBritiveBrokerPoolTokenMaxActive(t *testing.T) {
	testAccBrokerPoolTokenLimit(t, "AT-BP-Token-Max-Active", testAccBrokerPoolTokens("ACTIVE", 2),
		`Error (Setting Broker Pool Token Status|Creating Broker Pool Token)`)
}

// testAccBrokerPoolTokenLimit applies a pool plus tokens that must fail with errPattern, then
// removes the tokens and checks Britive is left with only the default token.
func testAccBrokerPoolTokenLimit(t *testing.T, poolName, tokens, errPattern string) {
	client := testAccBritiveClient(t)
	var poolID string
	pool := testAccBrokerPoolMinimal("pool", poolName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: pool,
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				Config:      pool + tokens,
				ExpectError: regexp.MustCompile(errPattern),
			},
			{
				Config: pool,
				Check:  testAccCheckTokenNamesAPI(client, &poolID, "default"),
			},
		},
	})
}

// TestBritiveBrokerPoolTokenDrift covers B15 and B16:
//  1. create the token
//  2. change its status and description through the API: the plan updates it back
//  3. delete it through the API: the plan creates it again
func TestBritiveBrokerPoolTokenDrift(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID string
	tokenName := "AT-token-drift"

	config := testAccBrokerPoolMinimal("pool", "AT-BP-Token-Drift") + fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool_token" "token" {
		pool_id     = britive_resource_manager_broker_pool.pool.pool_id
		name        = %q
		description = "AT token drift"
		status      = "ACTIVE"
	}`, tokenName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				PreConfig: func() {
					status, desc := "INACTIVE", "changed outside Terraform"
					if _, err := client.UpdateBrokerPoolToken(poolID, tokenName, britive.RemoteBrokerPoolTokenUpdateRequest{
						Status: &status, Description: &desc,
					}); err != nil {
						t.Fatalf("changing token outside Terraform: %v", err)
					}
				},
				Config:           config,
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionUpdate),
				Check:            testAccCheckTokenAPI(client, &poolID, tokenName, "ACTIVE", "AT token drift"),
			},
			{
				PreConfig: func() {
					if err := client.DeleteBrokerPoolToken(poolID, tokenName); err != nil {
						t.Fatalf("deleting token outside Terraform: %v", err)
					}
				},
				Config:           config,
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionCreate),
				Check:            testAccCheckTokenAPI(client, &poolID, tokenName, "ACTIVE", "AT token drift"),
			},
		},
	})
}

// TestBritiveBrokerPoolTokenPoolDeletedOutside: the pool is deleted through the API while a
// token resource exists. Reading the token of a deleted pool must count as "not found", so the
// next plan creates the pool and the token again without errors.
func TestBritiveBrokerPoolTokenPoolDeletedOutside(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID, oldPoolID string
	tokenName := "AT-token-pool-deleted"

	config := testAccBrokerPoolMinimal("pool", "AT-BP-Token-Pool-Deleted") + fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool_token" "token" {
		pool_id     = britive_resource_manager_broker_pool.pool.pool_id
		name        = %q
		description = "AT token pool deleted"
		status      = "ACTIVE"
	}`, tokenName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				PreConfig: func() {
					oldPoolID = poolID
					if err := client.DeleteBrokerPool(poolID); err != nil {
						t.Fatalf("deleting pool outside Terraform: %v", err)
					}
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(bpPool, plancheck.ResourceActionCreate),
					plancheck.ExpectResourceAction(bpToken, plancheck.ResourceActionCreate),
				}},
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckAttrNotEquals(bpPool, "pool_id", &oldPoolID),
					resource.TestCheckResourceAttrPair(bpToken, "pool_id", bpPool, "pool_id"),
					testAccCheckTokenNamesAPI(client, &poolID, tokenName, "default"),
					testAccCheckTokenAPI(client, &poolID, tokenName, "ACTIVE", "AT token pool deleted"),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokenActivateOverLimit: a pool allows 2 ACTIVE tokens. With the
// default token and one other token ACTIVE, changing a third token from INACTIVE to ACTIVE is
// rejected: the apply fails, Britive and the state keep INACTIVE, and setting INACTIVE again in
// the configuration gives an empty plan.
func TestBritiveBrokerPoolTokenActivateOverLimit(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID string

	config := func(thirdStatus string) string {
		return testAccBrokerPoolMinimal("pool", "AT-BP-Token-Activate-Limit") + fmt.Sprintf(`
		resource "britive_resource_manager_broker_pool_token" "active" {
			pool_id = britive_resource_manager_broker_pool.pool.pool_id
			name    = "AT-token-active"
			status  = "ACTIVE"
		}

		resource "britive_resource_manager_broker_pool_token" "token" {
			pool_id = britive_resource_manager_broker_pool.pool.pool_id
			name    = "AT-token-third"
			status  = %q
		}`, thirdStatus)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config("INACTIVE"),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckTokenAPI(client, &poolID, "AT-token-third", "INACTIVE", ""),
				),
			},
			{
				Config:           config("ACTIVE"),
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionUpdate),
				ExpectError:      regexp.MustCompile(`Error Updating Broker Pool Token`),
			},
			{
				// The failed update left the state at INACTIVE, so this plan is empty.
				Config:           config("INACTIVE"),
				ConfigPlanChecks: expectAction(bpToken, plancheck.ResourceActionNoop),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpToken, "status", "INACTIVE"),
					testAccCheckTokenAPI(client, &poolID, "AT-token-third", "INACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-token-active", "ACTIVE", ""),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", "default token for pool"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// B2. Default token (britive_resource_manager_broker_pool_token with name = "default")
// ---------------------------------------------------------------------------

const bpDefaultToken = "britive_resource_manager_broker_pool_token.default"

// testAccDefaultTokenConfig builds a pool, an optional extra token and a "default" token
// resource. The default token depends on the extra token, so it is updated after the extra
// token exists and destroyed before it.
func testAccDefaultTokenConfig(poolName, defaultAttrs, otherStatus string) string {
	config := testAccBrokerPoolMinimal("pool", poolName)
	dependsOn := ""
	if otherStatus != "" {
		config += fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool_token" "token" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		name    = "AT-token-other"
		status  = %q
	}
	`, otherStatus)
		dependsOn = "depends_on = [britive_resource_manager_broker_pool_token.token]"
	}
	if defaultAttrs != "" {
		config += fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool_token" "default" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		name    = "default"
		%s
		%s
	}
	`, defaultAttrs, dependsOn)
	}
	return config
}

// TestBritiveBrokerPoolDefaultToken covers taking over and managing the default token:
//  1. take over the default token of a new pool (ACTIVE + description): no second token is
//     created, the existing token is updated
//  2. remove the description: it is cleared in Britive
//  3. import with {pool_id}/default
//  4. add another ACTIVE token and set the default token INACTIVE: updated in place
//  5. remove the default token resource while another token is ACTIVE: the token is deleted
//  6. add the default token resource again: the token is created again with POST
//  7. set the other token INACTIVE, so the final destroy hits the "last active token" rule on
//     the default token: it is removed from the state with a warning, and the pool delete
//     removes it from Britive (checked by CheckDestroy)
func TestBritiveBrokerPoolDefaultToken(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Default-Token"
	var poolID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccDefaultTokenConfig(poolName, `status = "ACTIVE"`+"\n"+`description = "AT default token"`, ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttr(bpDefaultToken, "name", "default"),
					resource.TestCheckResourceAttr(bpDefaultToken, "status", "ACTIVE"),
					resource.TestCheckResourceAttr(bpDefaultToken, "description", "AT default token"),
					resource.TestCheckNoResourceAttr(bpDefaultToken, "token_value"),
					resource.TestCheckResourceAttrSet(bpDefaultToken, "created"),
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", "AT default token"),
				),
			},
			{
				Config:           testAccDefaultTokenConfig(poolName, `status = "ACTIVE"`, ""),
				ConfigPlanChecks: expectAction(bpDefaultToken, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(bpDefaultToken, "description"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
				),
			},
			{
				ResourceName:      bpDefaultToken,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return poolID + "/default", nil
				},
			},
			{
				Config: testAccDefaultTokenConfig(poolName, `status = "INACTIVE"`, "ACTIVE"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(bpDefaultToken, plancheck.ResourceActionUpdate),
					plancheck.ExpectResourceAction(bpToken, plancheck.ResourceActionCreate),
				}},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpDefaultToken, "status", "INACTIVE"),
					testAccCheckDefaultTokenAPI(client, &poolID, "INACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-token-other", "ACTIVE", ""),
				),
			},
			{
				Config: testAccDefaultTokenConfig(poolName, "", "ACTIVE"),
				Check:  testAccCheckTokenNamesAPI(client, &poolID, "AT-token-other"),
			},
			{
				Config:           testAccDefaultTokenConfig(poolName, `status = "ACTIVE"`+"\n"+`description = "AT recreated"`, "ACTIVE"),
				ConfigPlanChecks: expectAction(bpDefaultToken, plancheck.ResourceActionCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenNamesAPI(client, &poolID, "AT-token-other", "default"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", "AT recreated"),
				),
			},
			{
				Config: testAccDefaultTokenConfig(poolName, `status = "ACTIVE"`+"\n"+`description = "AT recreated"`, "INACTIVE"),
				Check:  testAccCheckTokenAPI(client, &poolID, "AT-token-other", "INACTIVE", ""),
			},
		},
	})
}

// TestBritiveBrokerPoolDefaultTokenRejected covers changes to the default token that Britive
// rejects:
//  1. create the pool only
//  2. take over the default token with status INACTIVE. It is the pool's only active token, so
//     Britive rejects it: the apply fails and the resource is not saved
//  3. Britive left the token as it was (ACTIVE, "default token for pool"); the next plan creates
//     the resource again, and with status ACTIVE the takeover succeeds with the same token
//  4. set status INACTIVE: the update is rejected with an error and the state keeps ACTIVE
//  5. remove the default token resource while it is the last active token: Britive rejects the
//     delete, the resource is removed from the state with a warning, and the token stays
func TestBritiveBrokerPoolDefaultTokenRejected(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Default-Token-Rejected"
	var poolID, tokenValue string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccDefaultTokenConfig(poolName, "", ""),
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				Config:      testAccDefaultTokenConfig(poolName, `status = "INACTIVE"`, ""),
				ExpectError: regexp.MustCompile(`Error Updating Broker Pool Default Token`),
			},
			{
				PreConfig: func() {
					// The rejected takeover changed nothing in Britive.
					if err := testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", "default token for pool")(nil); err != nil {
						t.Fatal(err)
					}
					token, err := client.GetBrokerPoolToken(poolID, "default")
					if err != nil {
						t.Fatalf("reading default token: %v", err)
					}
					tokenValue = token.TokenValue
				},
				Config:           testAccDefaultTokenConfig(poolName, `status = "ACTIVE"`, ""),
				ConfigPlanChecks: expectAction(bpDefaultToken, plancheck.ResourceActionCreate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpDefaultToken, "status", "ACTIVE"),
					// Same token as before: it was never deleted and created again.
					testAccCheckTokenValueAPI(client, &poolID, "default", &tokenValue),
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
				),
			},
			{
				Config:           testAccDefaultTokenConfig(poolName, `status = "INACTIVE"`, ""),
				ConfigPlanChecks: expectAction(bpDefaultToken, plancheck.ResourceActionUpdate),
				ExpectError:      regexp.MustCompile(`Error Updating Broker Pool Token`),
			},
			{
				Config: testAccDefaultTokenConfig(poolName, `status = "ACTIVE"`, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpDefaultToken, "status", "ACTIVE"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
				),
			},
			{
				Config: testAccDefaultTokenConfig(poolName, "", ""),
				Check:  testAccCheckTokenNamesAPI(client, &poolID, "default"),
			},
		},
	})
}

// TestBritiveBrokerPoolDefaultTokenPoolDeletedOutside: the pool is deleted through the API
// while a "default" token resource exists. The next plan creates the pool again, and the
// default token resource takes over the new pool's default token: no second token is created,
// and the configured description is applied to it.
func TestBritiveBrokerPoolDefaultTokenPoolDeletedOutside(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID, oldPoolID string
	config := testAccDefaultTokenConfig("AT-BP-Default-Token-Pool-Deleted",
		`status = "ACTIVE"`+"\n"+`description = "AT default token pool deleted"`, "")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", "AT default token pool deleted"),
				),
			},
			{
				PreConfig: func() {
					oldPoolID = poolID
					if err := client.DeleteBrokerPool(poolID); err != nil {
						t.Fatalf("deleting pool outside Terraform: %v", err)
					}
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(bpPool, plancheck.ResourceActionCreate),
					plancheck.ExpectResourceAction(bpDefaultToken, plancheck.ResourceActionCreate),
				}},
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckAttrNotEquals(bpPool, "pool_id", &oldPoolID),
					resource.TestCheckResourceAttrPair(bpDefaultToken, "pool_id", bpPool, "pool_id"),
					resource.TestCheckResourceAttr(bpDefaultToken, "status", "ACTIVE"),
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", "AT default token pool deleted"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// C. Broker pool labels resource
// ---------------------------------------------------------------------------

// TestBritiveBrokerPoolLabels covers C1–C8 and C17–C19:
//  1. create two keys, one with no values
//  2. reorder the blocks: no changes
//  3. add a value
//  4. reorder the values: no changes
//  5. remove a value
//  6. add a key
//  7. remove a key
//  8. import by pool ID, 9. import by full ID
//  10. remove the labels resource: every label is removed, the pool stays
func TestBritiveBrokerPoolLabels(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolLabelsConfig("AT-BP-Labels", `Environment = ["Development"]`, `Team = []`),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttrPair(bpLabels, "pool_id", bpPool, "pool_id"),
					resource.TestCheckResourceAttrSet(bpLabels, "id"),
					resource.TestCheckResourceAttr(bpLabels, "resource_labels.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(bpLabels, "resource_labels.*", map[string]string{"label_key": "Team", "values.#": "0"}),
					testAccCheckLabelsAPI(client, &poolID, map[string][]string{"Environment": {"Development"}, "Team": {}}),
				),
			},
			{
				Config:   testAccBrokerPoolLabelsConfig("AT-BP-Labels", `Team = []`, `Environment = ["Development"]`),
				PlanOnly: true,
			},
			{
				Config:           testAccBrokerPoolLabelsConfig("AT-BP-Labels", `Environment = ["Development", "Staging"]`, `Team = []`),
				ConfigPlanChecks: expectAction(bpLabels, plancheck.ResourceActionUpdate),
				Check:            testAccCheckLabelsAPI(client, &poolID, map[string][]string{"Environment": {"Development", "Staging"}, "Team": {}}),
			},
			{
				Config:   testAccBrokerPoolLabelsConfig("AT-BP-Labels", `Environment = ["Staging", "Development"]`, `Team = []`),
				PlanOnly: true,
			},
			{
				Config:           testAccBrokerPoolLabelsConfig("AT-BP-Labels", `Environment = ["Staging"]`, `Team = []`),
				ConfigPlanChecks: expectAction(bpLabels, plancheck.ResourceActionUpdate),
				Check:            testAccCheckLabelsAPI(client, &poolID, map[string][]string{"Environment": {"Staging"}, "Team": {}}),
			},
			{
				Config:           testAccBrokerPoolLabelsConfig("AT-BP-Labels", `Environment = ["Staging"]`, `Team = []`, `Owner = []`),
				ConfigPlanChecks: expectAction(bpLabels, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpLabels, "resource_labels.#", "3"),
					testAccCheckLabelsAPI(client, &poolID, map[string][]string{"Environment": {"Staging"}, "Team": {}, "Owner": {}}),
				),
			},
			{
				Config:           testAccBrokerPoolLabelsConfig("AT-BP-Labels", `Environment = ["Staging"]`, `Owner = []`),
				ConfigPlanChecks: expectAction(bpLabels, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpLabels, "resource_labels.#", "2"),
					testAccCheckLabelsAPI(client, &poolID, map[string][]string{"Environment": {"Staging"}, "Owner": {}}),
				),
			},
			{
				ResourceName:      bpLabels,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[bpLabels]
					if !ok {
						return "", errs.NewNotFoundErrorf("%s in state", bpLabels)
					}
					return rs.Primary.Attributes["pool_id"], nil
				},
			},
			{
				ResourceName:      bpLabels,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[bpLabels]
					if !ok {
						return "", errs.NewNotFoundErrorf("%s in state", bpLabels)
					}
					return fmt.Sprintf("resource-manager/broker-pools/%s/labels", rs.Primary.Attributes["pool_id"]), nil
				},
			},
			{
				Config: testAccBrokerPoolMinimal("pool", "AT-BP-Labels"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					testAccCheckLabelsAPI(client, &poolID, map[string][]string{}),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolLabelsValidation covers C9–C11 (plan-only, no API call).
func TestBritiveBrokerPoolLabelsValidation(t *testing.T) {
	labels := func(blocks string) string {
		return fmt.Sprintf(`
		resource "britive_resource_manager_broker_pool_labels" "labels" {
			pool_id = "unused"
			%s
		}`, blocks)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: labels(`
				resource_labels {
					label_key = "Environment"
					values    = ["Development"]
				}
				resource_labels {
					label_key = "Environment"
					values    = ["Staging"]
				}`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Duplicate resource_labels Block`),
			},
			{
				Config:      labels(``),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Missing resource_labels Block`),
			},
			{
				Config: labels(`
				resource_labels {
					label_key = "my key!"
					values    = []
				}`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must contain only letters, numbers, underscore\s+and\s+hyphen`),
			},
		},
	})
}

// TestBritiveBrokerPoolLabelsDrift covers C13–C15:
//  1. create the labels
//  2. add a label key through the API: the plan removes it
//  3. change a label's values through the API: the plan changes them back
//  4. delete a label through the API: the plan adds it again
func TestBritiveBrokerPoolLabelsDrift(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID string
	config := testAccBrokerPoolLabelsConfig("AT-BP-Labels-Drift", `Environment = ["Development"]`, `Team = []`)
	want := map[string][]string{"Environment": {"Development"}, "Team": {}}

	driftStep := func(change func() error) resource.TestStep {
		return resource.TestStep{
			PreConfig: func() {
				if err := change(); err != nil {
					t.Fatalf("changing labels outside Terraform: %v", err)
				}
			},
			Config:           config,
			ConfigPlanChecks: expectAction(bpLabels, plancheck.ResourceActionUpdate),
			Check:            testAccCheckLabelsAPI(client, &poolID, want),
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			driftStep(func() error {
				return client.CreateBrokerPoolLabel(poolID, britive.RemoteBrokerPoolLabel{Key: "Owner", LabelValues: []string{}})
			}),
			driftStep(func() error {
				return client.UpdateBrokerPoolLabel(poolID, "Environment", []string{"Development", "Staging"})
			}),
			driftStep(func() error {
				return client.DeleteBrokerPoolLabel(poolID, "Team")
			}),
		},
	})
}

// TestBritiveBrokerPoolLabelsChangePool covers C16: moving the labels resource to another pool
// replaces it; the labels are removed from the old pool and added to the new one.
func TestBritiveBrokerPoolLabelsChangePool(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolA, poolB string

	config := func(target string) string {
		return testAccBrokerPoolMinimal("pool_b", "AT-BP-Labels-Move-B") +
			testAccBrokerPoolLabelsConfigFor("pool", "AT-BP-Labels-Move-A", target, `Environment = ["Development"]`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config("pool"),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolA),
					testAccCaptureAttr(bpPoolB, "pool_id", &poolB),
					testAccCheckLabelsAPI(client, &poolA, map[string][]string{"Environment": {"Development"}}),
				),
			},
			{
				Config:           config("pool_b"),
				ConfigPlanChecks: expectAction(bpLabels, plancheck.ResourceActionReplace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(bpLabels, "pool_id", bpPoolB, "pool_id"),
					testAccCheckLabelsAPI(client, &poolA, map[string][]string{}),
					testAccCheckLabelsAPI(client, &poolB, map[string][]string{"Environment": {"Development"}}),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolLabelsPoolDeletedOutside covers C21: the pool is deleted through the
// API; the plan creates the pool and its labels again without errors.
func TestBritiveBrokerPoolLabelsPoolDeletedOutside(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID string
	config := testAccBrokerPoolLabelsConfig("AT-BP-Labels-Pool-Deleted", `Environment = ["Development"]`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				PreConfig: func() {
					if err := client.DeleteBrokerPool(poolID); err != nil {
						t.Fatalf("deleting pool outside Terraform: %v", err)
					}
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(bpPool, plancheck.ResourceActionCreate),
					plancheck.ExpectResourceAction(bpLabels, plancheck.ResourceActionCreate),
				}},
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckLabelsAPI(client, &poolID, map[string][]string{"Environment": {"Development"}}),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolLabelsExistingLabels: the labels resource is added to a pool that
// already has labels (for example added in the UI):
//  1. create the pool only
//  2. add Environment = ["Old"] and Owner = [] through the API, then create the labels
//     resource with Environment = ["Development"]. Create must not fail on the existing key, and
//     Britive must end up with the configured values. The labels resource manages every label
//     on the pool, so the plan after the apply shows Owner to be removed (not empty)
//  3. apply again: Owner is removed, and the plan is empty
func TestBritiveBrokerPoolLabelsExistingLabels(t *testing.T) {
	client := testAccBritiveClient(t)
	var poolID string
	poolName := "AT-BP-Labels-Existing"
	config := testAccBrokerPoolLabelsConfig(poolName, `Environment = ["Development"]`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolMinimal("pool", poolName),
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				PreConfig: func() {
					for _, l := range []britive.RemoteBrokerPoolLabel{
						{Key: "Environment", LabelValues: []string{"Old"}},
						{Key: "Owner", LabelValues: []string{}},
					} {
						if err := client.CreateBrokerPoolLabel(poolID, l); err != nil {
							t.Fatalf("adding label %q outside Terraform: %v", l.Key, err)
						}
					}
				},
				Config:             config,
				ConfigPlanChecks:   expectAction(bpLabels, plancheck.ResourceActionCreate),
				ExpectNonEmptyPlan: true,
				Check: testAccCheckLabelsAPI(client, &poolID, map[string][]string{
					"Environment": {"Development"}, "Owner": {},
				}),
			},
			{
				Config:           config,
				ConfigPlanChecks: expectAction(bpLabels, plancheck.ResourceActionUpdate),
				Check:            testAccCheckLabelsAPI(client, &poolID, map[string][]string{"Environment": {"Development"}}),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// D. Broker pool data source
// ---------------------------------------------------------------------------

// TestBritiveBrokerPoolDataSource covers D1, D2 and D8: looking up a pool created in the same
// configuration, with no brokers and no linked resources.
func TestBritiveBrokerPoolDataSource(t *testing.T) {
	client := testAccBritiveClient(t)
	name := "AT-BP-Data-Source"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "britive_resource_manager_broker_pool" "pool" {
					name               = %q
					description        = "AT broker pool data source"
					keep_alive_seconds = 120
					disconnect_seconds = 3600
				}

				data "britive_resource_manager_broker_pool" "lookup" {
					name       = britive_resource_manager_broker_pool.pool.name
					depends_on = [britive_resource_manager_broker_pool.pool]
				}`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(bpData, "pool_id", bpPool, "pool_id"),
					resource.TestCheckResourceAttr(bpData, "name", name),
					resource.TestCheckResourceAttr(bpData, "description", "AT broker pool data source"),
					resource.TestCheckResourceAttr(bpData, "keep_alive_seconds", "120"),
					resource.TestCheckResourceAttr(bpData, "disconnect_seconds", "3600"),
					resource.TestCheckResourceAttr(bpData, "broker_count", "0"),
					resource.TestCheckResourceAttr(bpData, "brokers.#", "0"),
					resource.TestCheckResourceAttr(bpData, "resources.#", "0"),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolDataSourceNotFound covers D6 and D7: a name that does not exist, a
// prefix of an existing name and a different case all fail with "Broker Pool Not Found".
func TestBritiveBrokerPoolDataSourceNotFound(t *testing.T) {
	client := testAccBritiveClient(t)
	name := "AT-BP-Data-Source-Exact"
	pool := testAccBrokerPoolMinimal("pool", name)

	lookup := func(n string) string {
		return pool + fmt.Sprintf(`
		data "britive_resource_manager_broker_pool" "lookup" {
			name       = %q
			depends_on = [britive_resource_manager_broker_pool.pool]
		}`, n)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: pool,
			},
			{
				Config:      lookup("AT-BP-does-not-exist"),
				ExpectError: regexp.MustCompile(`Broker Pool Not Found`),
			},
			{
				Config:      lookup("AT-BP-Data-Source"),
				ExpectError: regexp.MustCompile(`Broker Pool Not Found`),
			},
			{
				Config:      lookup(strings.ToLower(name)),
				ExpectError: regexp.MustCompile(`Broker Pool Not Found`),
			},
		},
	})
}

// TestBritiveBrokerPoolDataSourceWithBrokers covers D3. It needs an existing pool with at least
// one connected broker, named in BRITIVE_BROKER_POOL_WITH_BROKERS; otherwise it is skipped.
func TestBritiveBrokerPoolDataSourceWithBrokers(t *testing.T) {
	name := os.Getenv("BRITIVE_BROKER_POOL_WITH_BROKERS")
	if name == "" {
		t.Skip("BRITIVE_BROKER_POOL_WITH_BROKERS not set; skipping broker data source test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "britive_resource_manager_broker_pool" "lookup" {
					name = %q
				}`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(bpData, "pool_id"),
					resource.TestCheckResourceAttrWith(bpData, "brokers.#", func(v string) error {
						if v == "0" {
							return fmt.Errorf("expected at least one broker in pool %q", name)
						}
						return nil
					}),
					resource.TestCheckResourceAttrSet(bpData, "brokers.0.broker_id"),
					resource.TestCheckResourceAttrSet(bpData, "brokers.0.broker_name"),
					resource.TestCheckResourceAttrSet(bpData, "brokers.0.status"),
					resource.TestCheckResourceAttrSet(bpData, "brokers.0.status_timestamp"),
					resource.TestCheckResourceAttrSet(bpData, "brokers.0.created"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// E. End to end
// ---------------------------------------------------------------------------

// TestBritiveBrokerPoolEndToEnd covers E1–E4, E6 and D4:
//  1. pool + labels + two tokens, linked to a new resource, read back through the data source
//     (which lists the linked resource)
//  2. destroy everything
//  3. apply the same configuration again
//  4. refresh-only: no changes
func TestBritiveBrokerPoolEndToEnd(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-End-To-End"
	resourceName := "AT-BP-End-To-End-Resource"

	config := fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "resource_type" {
		name        = "AT-BP-End-To-End-Resource-Type"
		description = "AT broker pool end to end"
		parameters {
			param_name   = "testfield1"
			param_type   = "password"
			is_mandatory = true
		}
	}

	resource "britive_resource_manager_resource" "resource" {
		name          = %q
		description   = "AT broker pool end to end"
		resource_type = britive_resource_manager_resource_type.resource_type.name
		parameter_values = {
			"testfield1" = "v1"
		}
	}

	resource "britive_resource_manager_broker_pool" "pool" {
		name               = %q
		description        = "AT broker pool end to end"
		keep_alive_seconds = 120
	}

	resource "britive_resource_manager_broker_pool_labels" "labels" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		resource_labels {
			label_key = "Environment"
			values    = ["Development", "Staging"]
		}
		resource_labels {
			label_key = "Team"
			values    = []
		}
	}

	resource "britive_resource_manager_broker_pool_token" "token" {
		pool_id     = britive_resource_manager_broker_pool.pool.pool_id
		name        = "AT-token-active"
		description = "AT active token"
		status      = "ACTIVE"
	}

	resource "britive_resource_manager_broker_pool_token" "token_inactive" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		name    = "AT-token-inactive"
		status  = "INACTIVE"
	}

	resource "britive_resource_manager_resource_broker_pools" "link" {
		resource_id  = britive_resource_manager_resource.resource.id
		broker_pools = [britive_resource_manager_broker_pool.pool.name]
	}

	data "britive_resource_manager_broker_pool" "lookup" {
		name       = britive_resource_manager_broker_pool.pool.name
		depends_on = [britive_resource_manager_resource_broker_pools.link]
	}`, resourceName, poolName)

	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttrPair(bpData, "pool_id", bpPool, "pool_id"),
		resource.TestCheckResourceAttr(bpLabels, "resource_labels.#", "2"),
		resource.TestCheckResourceAttr(bpToken, "status", "ACTIVE"),
		resource.TestCheckResourceAttr("britive_resource_manager_broker_pool_token.token_inactive", "status", "INACTIVE"),
		resource.TestCheckResourceAttr(bpData, "resources.#", "1"),
		resource.TestCheckResourceAttr(bpData, "resources.0.name", resourceName),
		resource.TestCheckResourceAttrSet(bpData, "resources.0.resource_id"),
		resource.TestCheckResourceAttr(bpData, "resources.0.resource_type_name", "AT-BP-End-To-End-Resource-Type"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  check,
			},
			{
				Config:  config,
				Destroy: true,
			},
			{
				Config: config,
				Check:  check,
			},
			{
				RefreshState: true,
			},
		},
	})
}

// TestBritiveBrokerPoolDeleteLinked covers A25: a pool that is still linked to a resource is
// deleted outside Terraform. Britive deletes it and unlinks it from the resource. The next plan
// creates the pool again and replaces the link, and the data source lists the resource again.
func TestBritiveBrokerPoolDeleteLinked(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Linked-Delete"
	resourceName := "AT-BP-Linked-Delete-Resource"
	link := "britive_resource_manager_resource_broker_pools.link"
	var poolID string

	config := fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "resource_type" {
		name        = "AT-BP-Linked-Delete-Resource-Type"
		description = "AT broker pool linked delete"
		parameters {
			param_name   = "testfield1"
			param_type   = "password"
			is_mandatory = true
		}
	}

	resource "britive_resource_manager_resource" "resource" {
		name          = %q
		description   = "AT broker pool linked delete"
		resource_type = britive_resource_manager_resource_type.resource_type.name
		parameter_values = {
			"testfield1" = "v1"
		}
	}

	resource "britive_resource_manager_broker_pool" "pool" {
		name = %q
	}

	resource "britive_resource_manager_resource_broker_pools" "link" {
		resource_id  = britive_resource_manager_resource.resource.id
		broker_pools = [britive_resource_manager_broker_pool.pool.name]
	}

	data "britive_resource_manager_broker_pool" "lookup" {
		name       = britive_resource_manager_broker_pool.pool.name
		depends_on = [britive_resource_manager_resource_broker_pools.link]
	}`, resourceName, poolName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttr(bpData, "resources.#", "1"),
					testAccCheckResourceLinkedPoolsAPI(client, resourceName, poolName, true),
				),
			},
			{
				PreConfig: func() {
					if err := client.DeleteBrokerPool(poolID); err != nil {
						t.Fatalf("deleting a pool that is linked to a resource: %v", err)
					}
					if _, err := client.GetBrokerPool(poolID); !errors.Is(err, britive.ErrNotFound) {
						t.Fatalf("pool %s should be deleted, got err = %v", poolID, err)
					}
					if err := testAccCheckResourceLinkedPoolsAPI(client, resourceName, poolName, false)(nil); err != nil {
						t.Fatal(err)
					}
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(bpPool, plancheck.ResourceActionCreate),
					plancheck.ExpectResourceAction(link, plancheck.ResourceActionReplace),
				}},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrNotEquals(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttr(bpData, "resources.#", "1"),
					resource.TestCheckResourceAttr(bpData, "resources.0.name", resourceName),
					testAccCheckResourceLinkedPoolsAPI(client, resourceName, poolName, true),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Config helpers
// ---------------------------------------------------------------------------

func testAccBrokerPoolMinimal(resName, name string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool" %q {
		name = %q
	}
	`, resName, name)
}

func testAccBrokerPoolTokens(status string, count int) string {
	var b strings.Builder
	for i := 1; i <= count; i++ {
		fmt.Fprintf(&b, `
	resource "britive_resource_manager_broker_pool_token" "t%d" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		name    = "AT-token-%d"
		status  = %q
	}
	`, i, i, status)
	}
	return b.String()
}

// testAccBrokerPoolLabelsConfig builds a pool named poolName plus a labels resource on it.
// Each label is written as `Key = [values]`.
func testAccBrokerPoolLabelsConfig(poolName string, labels ...string) string {
	return testAccBrokerPoolLabelsConfigFor("pool", poolName, "pool", labels...)
}

func testAccBrokerPoolLabelsConfigFor(poolRes, poolName, target string, labels ...string) string {
	var blocks strings.Builder
	for _, l := range labels {
		key, values, _ := strings.Cut(l, "=")
		fmt.Fprintf(&blocks, `
		resource_labels {
			label_key = %q
			values    = %s
		}`, strings.TrimSpace(key), strings.TrimSpace(values))
	}
	return testAccBrokerPoolMinimal(poolRes, poolName) + fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool_labels" "labels" {
		pool_id = britive_resource_manager_broker_pool.%s.pool_id
		%s
	}
	`, target, blocks.String())
}

func expectAction(name string, action plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, action)},
	}
}

// ---------------------------------------------------------------------------
// Check helpers
// ---------------------------------------------------------------------------

// testAccCaptureAttr stores an attribute's value so later steps (PreConfig, API checks) can
// use it.
func testAccCaptureAttr(name, attr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return errs.NewNotFoundErrorf("%s in state", name)
		}
		v := rs.Primary.Attributes[attr]
		if v == "" {
			return fmt.Errorf("%s.%s is empty", name, attr)
		}
		*dst = v
		return nil
	}
}

func testAccCheckAttrEquals(name, attr string, want *string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(name, attr, func(v string) error {
		if v != *want {
			return fmt.Errorf("%s.%s = %q, want %q", name, attr, v, *want)
		}
		return nil
	})
}

func testAccCheckAttrNotEquals(name, attr string, notWant *string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(name, attr, func(v string) error {
		if v == *notWant {
			return fmt.Errorf("%s.%s is still %q", name, attr, v)
		}
		return nil
	})
}

// testAccCheckBrokerPoolAPI reads the pool from Britive and compares its fields.
func testAccCheckBrokerPoolAPI(client *britive.Client, poolID *string, name, description string, keepAlive, disconnect int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		pool, err := client.GetBrokerPool(*poolID)
		if err != nil {
			return fmt.Errorf("reading pool %s: %w", *poolID, err)
		}
		if pool.Name != name {
			return fmt.Errorf("pool name in Britive = %q, want %q", pool.Name, name)
		}
		if pool.Description != description {
			return fmt.Errorf("pool description in Britive = %q, want %q", pool.Description, description)
		}
		if pool.KeepAliveSeconds != keepAlive {
			return fmt.Errorf("keep-alive-seconds in Britive = %d, want %d", pool.KeepAliveSeconds, keepAlive)
		}
		if pool.DisconnectSeconds != disconnect {
			return fmt.Errorf("disconnect-seconds in Britive = %d, want %d", pool.DisconnectSeconds, disconnect)
		}
		return nil
	}
}

func testAccCheckDefaultTokenAPI(client *britive.Client, poolID *string, status, description string) resource.TestCheckFunc {
	return testAccCheckTokenAPI(client, poolID, "default", status, description)
}

// testAccCheckTokenAPI reads one token from Britive and compares its status and description.
func testAccCheckTokenAPI(client *britive.Client, poolID *string, tokenName, status, description string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		token, err := client.GetBrokerPoolToken(*poolID, tokenName)
		if err != nil {
			return fmt.Errorf("reading token %s of pool %s: %w", tokenName, *poolID, err)
		}
		if token.Status != status {
			return fmt.Errorf("token %s status in Britive = %q, want %q", tokenName, token.Status, status)
		}
		if token.Description != description {
			return fmt.Errorf("token %s description in Britive = %q, want %q", tokenName, token.Description, description)
		}
		return nil
	}
}

// testAccCheckTokenValueAPI checks the token's value in Britive is still want, which shows the
// token was not deleted and created again.
func testAccCheckTokenValueAPI(client *britive.Client, poolID *string, tokenName string, want *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		token, err := client.GetBrokerPoolToken(*poolID, tokenName)
		if err != nil {
			return fmt.Errorf("reading token %s of pool %s: %w", tokenName, *poolID, err)
		}
		if token.TokenValue != *want {
			return fmt.Errorf("token %s of pool %s has a different value in Britive: it was created again", tokenName, *poolID)
		}
		return nil
	}
}

// testAccCheckTokenNamesAPI checks the pool has exactly these tokens in Britive.
func testAccCheckTokenNamesAPI(client *britive.Client, poolID *string, want ...string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		tokens, err := client.GetBrokerPoolTokens(*poolID)
		if err != nil {
			return fmt.Errorf("reading tokens of pool %s: %w", *poolID, err)
		}
		got := make([]string, 0, len(tokens))
		for _, tk := range tokens {
			got = append(got, tk.TokenName)
		}
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			return fmt.Errorf("tokens of pool %s in Britive = %v, want %v", *poolID, got, want)
		}
		return nil
	}
}

// testAccCheckLabelsAPI checks the pool has exactly these labels in Britive.
func testAccCheckLabelsAPI(client *britive.Client, poolID *string, want map[string][]string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		labels, err := client.GetBrokerPoolLabels(*poolID)
		if err != nil {
			return fmt.Errorf("reading labels of pool %s: %w", *poolID, err)
		}
		got := make(map[string]string, len(labels))
		for _, l := range labels {
			values := append([]string(nil), l.LabelValues...)
			sort.Strings(values)
			got[l.Key] = strings.Join(values, ",")
		}
		if len(got) != len(want) {
			return fmt.Errorf("labels of pool %s in Britive = %v, want %v", *poolID, got, want)
		}
		for key, values := range want {
			v := append([]string(nil), values...)
			sort.Strings(v)
			if gv, ok := got[key]; !ok || gv != strings.Join(v, ",") {
				return fmt.Errorf("labels of pool %s in Britive = %v, want %v", *poolID, got, want)
			}
		}
		return nil
	}
}

// testAccCheckBrokerPoolDestroy checks every pool that was in state is gone from Britive.
func testAccCheckBrokerPoolDestroy(client *britive.Client) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != bpPoolType {
				continue
			}
			poolID := rs.Primary.Attributes["pool_id"]
			_, err := client.GetBrokerPool(poolID)
			if errors.Is(err, britive.ErrNotFound) {
				continue
			}
			if err != nil {
				return fmt.Errorf("checking pool %s was deleted: %w", poolID, err)
			}
			return fmt.Errorf("broker pool %s still exists", poolID)
		}
		return nil
	}
}

// testAccCheckBrokerPoolNameCountAPI checks how many pools in Britive have exactly this name.
func testAccCheckBrokerPoolNameCountAPI(client *britive.Client, name string, want int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		pools, err := client.ListBrokerPools(name)
		if err != nil {
			return fmt.Errorf("listing pools named %q: %w", name, err)
		}
		got := 0
		for _, p := range pools {
			if p.Name == name {
				got++
			}
		}
		if got != want {
			return fmt.Errorf("%d pools named %q in Britive, want %d", got, name, want)
		}
		return nil
	}
}

// testAccCheckResourceLinkedPoolsAPI checks whether the resource lists the pool among its
// broker pools in Britive.
func testAccCheckResourceLinkedPoolsAPI(client *britive.Client, resourceName, poolName string, linked bool) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		pools, err := client.GetBrokerPoolsResource(resourceName)
		if err != nil && !errors.Is(err, britive.ErrNotFound) {
			return fmt.Errorf("reading broker pools of resource %q: %w", resourceName, err)
		}
		found := false
		if pools != nil {
			for _, p := range *pools {
				if p.Name == poolName {
					found = true
				}
			}
		}
		if found != linked {
			return fmt.Errorf("resource %q linked to pool %q in Britive = %t, want %t", resourceName, poolName, found, linked)
		}
		return nil
	}
}
