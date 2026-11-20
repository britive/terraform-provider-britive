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
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Acceptance tests for broker pools: britive_resource_manager_broker_pool (with its tokens
// map), britive_resource_manager_broker_pool_labels and the britive_resource_manager_broker_pool
// data source. The token ephemeral resource is tested in
// ephemeral_resource_manager_broker_pool_token_test.go.
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
// The configuration keeps only the default token; tokens are covered by the B tests. Destroy
// is checked by CheckDestroy.
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
			tokens = { default = { status = "ACTIVE" } }
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
					// The default token Britive creates with the pool is taken over.
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
					resource.TestCheckResourceAttr(bpPool, "tokens.default.status", "ACTIVE"),
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
			tokens = { default = { status = "ACTIVE" } }
			%s
		}`, attr)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// tokens is required.
				Config: `
				resource "britive_resource_manager_broker_pool" "pool" {
					name = "AT-BP-Validation"
				}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?i)Missing required argument`),
			},
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
//  2. change it through the API: the plan updates it back, including the default token's
//     description, which is also changed through the API
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
		tokens = { default = { status = "ACTIVE" } }
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
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
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

// TestBritiveBrokerPoolImportErrors covers A23 and invalid label import IDs.
func TestBritiveBrokerPoolImportErrors(t *testing.T) {
	client := testAccBritiveClient(t)
	name := "AT-BP-Import-Errors"

	base := testAccBrokerPoolMinimal("pool", name)
	withChildren := base + `
	resource "britive_resource_manager_broker_pool" "missing" {
		name = "AT-BP-Import-Errors-Missing"
		tokens = { default = { status = "ACTIVE" } }
	}

	resource "britive_resource_manager_broker_pool_labels" "labels" {
		pool_id = britive_resource_manager_broker_pool.pool.pool_id
		resource_labels {
			label_key = "Environment"
			values    = []
		}
	}`

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
// B. Tokens in the broker pool resource
// ---------------------------------------------------------------------------

// testAccBrokerPoolWithTokens builds the pool "pool" with the given tokens map entries, each
// written as `name = { status = "...", ... }`.
func testAccBrokerPoolWithTokens(poolName string, tokens ...string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool" "pool" {
		name = %q
		tokens = {
			%s
		}
	}
	`, poolName, strings.Join(tokens, "\n\t\t\t"))
}

// TestBritiveBrokerPoolTokens covers the tokens map:
//  1. tokens = { default }: the existing default token is taken over (not created again), and
//     its description is set to "" (no description configured)
//  2. add an INACTIVE token with a description: created, default unchanged
//  3. set it ACTIVE and change its description: updated in place, same value
//  4. remove default: the other token is ACTIVE, so default is deleted
//  5. rotate in one apply: replace the token with a new one; the new token is created and
//     activated before the old one is deleted
//  6. import by pool ID: the tokens are read into the state, without values
//  7. refresh-only: no changes
func TestBritiveBrokerPoolTokens(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens"
	var poolID, defaultValue, ciValue string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolWithTokens(poolName, `default = { status = "ACTIVE" }`),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					resource.TestCheckResourceAttr(bpPool, "tokens.%", "1"),
					resource.TestCheckResourceAttr(bpPool, "tokens.default.status", "ACTIVE"),
					resource.TestCheckResourceAttr(bpPool, "tokens.default.description", ""),
					resource.TestCheckResourceAttrSet(bpPool, "tokens.default.created"),
					testAccCheckTokenIDAttr(bpPool, "default", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
					testAccCaptureTokenValueAPI(client, &poolID, "default", &defaultValue),
				),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName,
					`default = { status = "ACTIVE" }`,
					`"AT-ci" = { status = "INACTIVE", description = "AT CI token" }`),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpPool, "tokens.%", "2"),
					resource.TestCheckResourceAttr(bpPool, "tokens.AT-ci.status", "INACTIVE"),
					testAccCheckTokenIDAttr(bpPool, "AT-ci", &poolID),
					testAccCheckTokenAPI(client, &poolID, "AT-ci", "INACTIVE", "AT CI token"),
					testAccCheckTokenValueAPI(client, &poolID, "default", &defaultValue),
					testAccCaptureTokenValueAPI(client, &poolID, "AT-ci", &ciValue),
				),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName,
					`default = { status = "ACTIVE" }`,
					`"AT-ci" = { status = "ACTIVE", description = "AT CI token updated" }`),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenAPI(client, &poolID, "AT-ci", "ACTIVE", "AT CI token updated"),
					testAccCheckTokenValueAPI(client, &poolID, "AT-ci", &ciValue),
				),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName,
					`"AT-ci" = { status = "ACTIVE", description = "AT CI token updated" }`),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpPool, "tokens.%", "1"),
					testAccCheckTokenNamesAPI(client, &poolID, "AT-ci"),
				),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName, `"AT-ops" = { status = "ACTIVE" }`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "AT-ops"),
					testAccCheckTokenAPI(client, &poolID, "AT-ops", "ACTIVE", ""),
				),
			},
			{
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

// TestBritiveBrokerPoolTokensWithoutDefault: the default token is required only when the
// pool is created:
//  1. create without default: fails in the plan, nothing is created
//  2. create with default and two more tokens
//  3. remove default: the pool already exists, so default is deleted like any other token
func TestBritiveBrokerPoolTokensWithoutDefault(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-No-Default"
	var poolID string
	ci := `"AT-ci" = { status = "ACTIVE", description = "AT CI token" }`
	spare := `"AT-spare" = { status = "INACTIVE" }`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config:      testAccBrokerPoolWithTokens(poolName, ci, spare),
				ExpectError: regexp.MustCompile(`Missing Default Broker Pool Token`),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName, `default = { status = "ACTIVE" }`, ci, spare),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "default", "AT-ci", "AT-spare"),
				),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, ci, spare),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpPool, "tokens.%", "2"),
					testAccCheckTokenNamesAPI(client, &poolID, "AT-ci", "AT-spare"),
					testAccCheckTokenAPI(client, &poolID, "AT-ci", "ACTIVE", "AT CI token"),
					testAccCheckTokenAPI(client, &poolID, "AT-spare", "INACTIVE", ""),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensNoActive: tokens without an ACTIVE token pass the plan, and
// Britive rejects them with its own error:
//  1. create with default ACTIVE and AT-ci INACTIVE
//  2. set default INACTIVE: Britive rejects it ("at least 1 remaining active token"); the state
//     keeps the tokens Britive really holds
//  3. back to the first configuration: nothing to change, default is still ACTIVE in Britive
func TestBritiveBrokerPoolTokensNoActive(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-No-Active"
	var poolID string
	ci := `"AT-ci" = { status = "INACTIVE" }`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolWithTokens(poolName, `default = { status = "ACTIVE" }`, ci),
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				Config:      testAccBrokerPoolWithTokens(poolName, `default = { status = "INACTIVE" }`, ci),
				ExpectError: regexp.MustCompile(`(?i)(E1003|remaining active token)`),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, `default = { status = "ACTIVE" }`, ci),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionNoop),
				Check:            testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensOverLimit: more than 2 ACTIVE tokens or more than 3 tokens pass
// the plan, and Britive rejects them during apply. The provider makes the changes Britive
// accepts first, then saves what Britive holds:
//  1. create with default ACTIVE
//  2. 3 ACTIVE tokens: Britive rejects the third activation
//  3. 4 tokens: Britive rejects the fourth token
//  4. back to default only: the extra tokens are deleted
func TestBritiveBrokerPoolTokensOverLimit(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-Over-Limit"
	var poolID string
	def := `default = { status = "ACTIVE" }`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolWithTokens(poolName, def),
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName, def,
					`"AT-a" = { status = "ACTIVE" }`, `"AT-b" = { status = "ACTIVE" }`),
				ExpectError: regexp.MustCompile(`Error Updating Broker Pool Tokens`),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName, def,
					`"AT-a" = { status = "INACTIVE" }`, `"AT-b" = { status = "INACTIVE" }`, `"AT-c" = { status = "INACTIVE" }`),
				ExpectError: regexp.MustCompile(`Error Updating Broker Pool Tokens`),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName, def),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpPool, "tokens.%", "1"),
					testAccCheckTokenNamesAPI(client, &poolID, "default"),
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensNoActiveOnCreate: a new pool with default INACTIVE as its only
// token. The plan passes; Britive rejects the change after the pool is created, so the apply
// fails and the pool is saved as tainted. The test's destroy deletes it (checked by
// CheckDestroy).
func TestBritiveBrokerPoolTokensNoActiveOnCreate(t *testing.T) {
	client := testAccBritiveClient(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config:      testAccBrokerPoolWithTokens("AT-BP-Tokens-No-Active-Create", `default = { status = "INACTIVE" }`),
				ExpectError: regexp.MustCompile(`(?i)(E1003|remaining active token)`),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensFullPool covers the one case that needs an extra step: the pool
// has 3 tokens, its only ACTIVE token must be deleted, and the other two stay INACTIVE. The
// provider activates one of the two for a short time, and the apply succeeds.
func TestBritiveBrokerPoolTokensFullPool(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-Full"
	var poolID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolWithTokens(poolName,
					`default = { status = "ACTIVE" }`,
					`"AT-b" = { status = "INACTIVE" }`,
					`"AT-c" = { status = "INACTIVE" }`),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "default", "AT-b", "AT-c"),
				),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName,
					`"AT-b" = { status = "INACTIVE" }`,
					`"AT-c" = { status = "INACTIVE" }`,
					`"AT-d" = { status = "ACTIVE" }`),
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenNamesAPI(client, &poolID, "AT-b", "AT-c", "AT-d"),
					testAccCheckTokenAPI(client, &poolID, "AT-b", "INACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-c", "INACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-d", "ACTIVE", ""),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensValidation: these configurations fail in the plan, before any API
// call. Britive's limits (no ACTIVE token, more than 3 tokens, more than 2 ACTIVE) are not
// checked here: Britive rejects them (see TestBritiveBrokerPoolTokensNoActive and
// TestBritiveBrokerPoolTokensOverLimit).
func TestBritiveBrokerPoolTokensValidation(t *testing.T) {
	config := func(tokens ...string) string {
		return testAccBrokerPoolWithTokens("AT-BP-Tokens-Validation", tokens...)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A new pool must include the default token.
				Config:      config(`"AT-ci" = { status = "ACTIVE" }`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Missing Default Broker Pool Token`),
			},
			{
				Config:      config(),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`at least one token`),
			},
			{
				Config:      config(`"bad name!" = { status = "ACTIVE" }`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`only letters, numbers,\s+underscore\s+and\s+hyphen`),
			},
			{
				Config:      config(`default = { status = "active" }`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensDrift: tokens changed outside Terraform are changed back:
//  1. create default + AT-ci
//  2. add a token through the API: the plan deletes it
//  3. change AT-ci's status and description through the API: the plan changes them back
//  4. delete default through the API: the plan creates it again (with a new value)
//  5. delete the pool through the API: the plan creates the pool and its tokens again
func TestBritiveBrokerPoolTokensDrift(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-Drift"
	var poolID string
	config := testAccBrokerPoolWithTokens(poolName,
		`default = { status = "ACTIVE" }`,
		`"AT-ci" = { status = "INACTIVE", description = "AT CI token" }`)

	driftStep := func(what string, change func() error) resource.TestStep {
		return resource.TestStep{
			PreConfig: func() {
				if err := change(); err != nil {
					t.Fatalf("%s outside Terraform: %v", what, err)
				}
			},
			Config:           config,
			ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionUpdate),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckTokenNamesAPI(client, &poolID, "default", "AT-ci"),
				testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
				testAccCheckTokenAPI(client, &poolID, "AT-ci", "INACTIVE", "AT CI token"),
			),
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
			driftStep("adding a token", func() error {
				_, err := client.CreateBrokerPoolToken(poolID, britive.RemoteBrokerPoolTokenCreateRequest{TokenName: "AT-extra"})
				return err
			}),
			driftStep("changing a token", func() error {
				status, desc := "ACTIVE", "changed outside Terraform"
				_, err := client.UpdateBrokerPoolToken(poolID, "AT-ci", britive.RemoteBrokerPoolTokenUpdateRequest{Status: &status, Description: &desc})
				return err
			}),
			driftStep("deleting the default token", func() error {
				status := "ACTIVE"
				if _, err := client.UpdateBrokerPoolToken(poolID, "AT-ci", britive.RemoteBrokerPoolTokenUpdateRequest{Status: &status}); err != nil {
					return err
				}
				return client.DeleteBrokerPoolToken(poolID, "default")
			}),
			{
				PreConfig: func() {
					if err := client.DeleteBrokerPool(poolID); err != nil {
						t.Fatalf("deleting pool outside Terraform: %v", err)
					}
				},
				Config:           config,
				ConfigPlanChecks: expectAction(bpPool, plancheck.ResourceActionCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "default", "AT-ci"),
					testAccCheckTokenAPI(client, &poolID, "AT-ci", "INACTIVE", "AT CI token"),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensImport: a pool with 2 tokens, created through the API, is
// imported. The state gets each token's name, status and description, and no token value.
func TestBritiveBrokerPoolTokensImport(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-Import"
	var poolID string

	config := testAccBrokerPoolWithTokens(poolName,
		`default = { status = "ACTIVE", description = "default token for pool" }`,
		`"AT-ci" = { status = "INACTIVE", description = "AT CI token" }`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					pool, err := client.CreateBrokerPool(britive.RemoteBrokerPoolCreateRequest{Name: poolName})
					if err != nil {
						t.Fatalf("creating pool outside Terraform: %v", err)
					}
					poolID = pool.PoolID
					if _, err := client.CreateBrokerPoolToken(poolID, britive.RemoteBrokerPoolTokenCreateRequest{TokenName: "AT-ci", Description: "AT CI token"}); err != nil {
						t.Fatalf("creating token outside Terraform: %v", err)
					}
				},
				Config:             config,
				ResourceName:       bpPool,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc:  func(*terraform.State) (string, error) { return poolID, nil },
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("imported %d resources, want 1", len(states))
					}
					attrs := states[0].Attributes
					want := map[string]string{
						"tokens.%":                 "2",
						"tokens.default.status":    "ACTIVE",
						"tokens.AT-ci.status":      "INACTIVE",
						"tokens.AT-ci.description": "AT CI token",
						"tokens.AT-ci.id":          poolID + "/AT-ci",
					}
					for k, v := range want {
						if attrs[k] != v {
							return fmt.Errorf("%s = %q, want %q", k, attrs[k], v)
						}
					}
					for k := range attrs {
						if strings.Contains(strings.ToLower(k), "value") {
							return fmt.Errorf("imported state has %s", k)
						}
					}
					return nil
				},
			},
			{
				// The configuration matches what was imported, so there is nothing to change.
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// tokenPath is the plan/state path of a field of one token in the tokens map.
func tokenPath(name, field string) tfjsonpath.Path {
	return tfjsonpath.New("tokens").AtMapKey(name).AtMapKey(field)
}

// expectNewToken checks the plan adds the token with id and created unknown ("known after
// apply"), not null. A null there makes Terraform fail the apply with "Provider produced
// inconsistent result".
func expectNewToken(name string) []plancheck.PlanCheck {
	return []plancheck.PlanCheck{
		plancheck.ExpectUnknownValue(bpPool, tokenPath(name, "id")),
		plancheck.ExpectUnknownValue(bpPool, tokenPath(name, "created")),
	}
}

// expectKeptToken checks the plan keeps the token's id and created from the state.
func expectKeptToken(name string) []plancheck.PlanCheck {
	return []plancheck.PlanCheck{
		plancheck.ExpectKnownValue(bpPool, tokenPath(name, "id"), knownvalue.NotNull()),
		plancheck.ExpectKnownValue(bpPool, tokenPath(name, "created"), knownvalue.NotNull()),
	}
}

func planChecks(action plancheck.ResourceActionType, groups ...[]plancheck.PlanCheck) resource.ConfigPlanChecks {
	checks := []plancheck.PlanCheck{plancheck.ExpectResourceAction(bpPool, action)}
	for _, g := range groups {
		checks = append(checks, g...)
	}
	return resource.ConfigPlanChecks{PreApply: checks}
}

// TestBritiveBrokerPoolTokensAddToExistingPool covers adding tokens to a pool that already
// exists. The plan must show id and created of a new token as unknown, and keep them for
// existing tokens:
//  1. create with default only
//  2. add ci-token and remove default in one apply
//  3. add ops: ops is new, ci-token keeps its id and created
//  4. change only ci-token's description: no token gets an unknown id
//  5. delete ops through the API: the plan creates it again with an unknown id
//  6. remove ops
//  7. add ops back: it is a new token with a new value
func TestBritiveBrokerPoolTokensAddToExistingPool(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-Add"
	var poolID, opsValue string
	ci := `"ci-token" = { status = "ACTIVE" }`
	ciDesc := `"ci-token" = { status = "ACTIVE", description = "AT CI token" }`
	ops := `"ops_token-2" = { status = "INACTIVE" }`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolWithTokens(poolName, `default = { status = "ACTIVE", description = "AT default token" }`),
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, ci),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate, expectNewToken("ci-token")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					testAccCheckTokenIDAttr(bpPool, "ci-token", &poolID),
					resource.TestCheckResourceAttrSet(bpPool, "tokens.ci-token.created"),
					testAccCheckTokenNamesAPI(client, &poolID, "ci-token"),
					testAccCheckTokenAPI(client, &poolID, "ci-token", "ACTIVE", ""),
				),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, ci, ops),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate, expectNewToken("ops_token-2"), expectKeptToken("ci-token")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenIDAttr(bpPool, "ops_token-2", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "ci-token", "ops_token-2"),
					testAccCheckTokenAPI(client, &poolID, "ops_token-2", "INACTIVE", ""),
					testAccCaptureTokenValueAPI(client, &poolID, "ops_token-2", &opsValue),
				),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, ciDesc, ops),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate, expectKeptToken("ci-token"), expectKeptToken("ops_token-2")),
				Check:            testAccCheckTokenAPI(client, &poolID, "ci-token", "ACTIVE", "AT CI token"),
			},
			{
				PreConfig: func() {
					if err := client.DeleteBrokerPoolToken(poolID, "ops_token-2"); err != nil {
						t.Fatalf("deleting token outside Terraform: %v", err)
					}
				},
				Config:           testAccBrokerPoolWithTokens(poolName, ciDesc, ops),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate, expectNewToken("ops_token-2"), expectKeptToken("ci-token")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenNamesAPI(client, &poolID, "ci-token", "ops_token-2"),
					testAccCheckTokenValueChangedAPI(client, &poolID, "ops_token-2", &opsValue),
					testAccCaptureTokenValueAPI(client, &poolID, "ops_token-2", &opsValue),
				),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, ciDesc),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate, expectKeptToken("ci-token")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(bpPool, "tokens.%", "1"),
					testAccCheckTokenNamesAPI(client, &poolID, "ci-token"),
				),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, ciDesc, ops),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate, expectNewToken("ops_token-2"), expectKeptToken("ci-token")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenNamesAPI(client, &poolID, "ci-token", "ops_token-2"),
					testAccCheckTokenValueChangedAPI(client, &poolID, "ops_token-2", &opsValue),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensStatusSwap: the ACTIVE and INACTIVE tokens swap in one apply.
// The INACTIVE token must be activated before the ACTIVE one is set INACTIVE; the token values
// do not change.
func TestBritiveBrokerPoolTokensStatusSwap(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-Swap"
	var poolID, defaultValue, ciValue string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolWithTokens(poolName, `default = { status = "ACTIVE" }`, `"AT-ci" = { status = "INACTIVE" }`),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttr(bpPool, "pool_id", &poolID),
					testAccCaptureTokenValueAPI(client, &poolID, "default", &defaultValue),
					testAccCaptureTokenValueAPI(client, &poolID, "AT-ci", &ciValue),
				),
			},
			{
				Config:           testAccBrokerPoolWithTokens(poolName, `default = { status = "INACTIVE" }`, `"AT-ci" = { status = "ACTIVE" }`),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate, expectKeptToken("default"), expectKeptToken("AT-ci")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDefaultTokenAPI(client, &poolID, "INACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-ci", "ACTIVE", ""),
					testAccCheckTokenValueAPI(client, &poolID, "default", &defaultValue),
					testAccCheckTokenValueAPI(client, &poolID, "AT-ci", &ciValue),
				),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName, `default = { status = "ACTIVE" }`, `"AT-ci" = { status = "INACTIVE" }`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDefaultTokenAPI(client, &poolID, "ACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-ci", "INACTIVE", ""),
				),
			},
		},
	})
}

// TestBritiveBrokerPoolTokensReplaceAll: a full pool (3 tokens) is replaced by 3 tokens with
// other names in one apply. Every new token is unknown in the plan, and the pool never drops
// below one ACTIVE token, so Britive accepts every call.
func TestBritiveBrokerPoolTokensReplaceAll(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Tokens-Replace-All"
	var poolID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccBrokerPoolWithTokens(poolName,
					`default = { status = "ACTIVE" }`,
					`"AT-b" = { status = "ACTIVE" }`,
					`"AT-c" = { status = "INACTIVE" }`),
				Check: testAccCaptureAttr(bpPool, "pool_id", &poolID),
			},
			{
				Config: testAccBrokerPoolWithTokens(poolName,
					`"AT-x" = { status = "ACTIVE" }`,
					`"AT-y" = { status = "INACTIVE" }`,
					`"AT-z" = { status = "INACTIVE" }`),
				ConfigPlanChecks: planChecks(plancheck.ResourceActionUpdate,
					expectNewToken("AT-x"), expectNewToken("AT-y"), expectNewToken("AT-z")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrEquals(bpPool, "pool_id", &poolID),
					testAccCheckTokenNamesAPI(client, &poolID, "AT-x", "AT-y", "AT-z"),
					testAccCheckTokenAPI(client, &poolID, "AT-x", "ACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-y", "INACTIVE", ""),
					testAccCheckTokenAPI(client, &poolID, "AT-z", "INACTIVE", ""),
				),
			},
		},
	})
}

// testAccCheckTokenValueChangedAPI checks the token's value in Britive is no longer old, which
// shows the token was created again.
func testAccCheckTokenValueChangedAPI(client *britive.Client, poolID *string, tokenName string, old *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		token, err := client.GetBrokerPoolToken(*poolID, tokenName)
		if err != nil {
			return fmt.Errorf("reading token %s of pool %s: %w", tokenName, *poolID, err)
		}
		if token.TokenValue == *old {
			return fmt.Errorf("token %s of pool %s still has its old value: it was not created again", tokenName, *poolID)
		}
		return nil
	}
}

// testAccCheckTokenIDAttr checks tokens.<name>.id is {pool_id}/{name}.
func testAccCheckTokenIDAttr(res, name string, poolID *string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(res, "tokens."+name+".id", func(v string) error {
		if want := *poolID + "/" + name; v != want {
			return fmt.Errorf("tokens.%s.id = %q, want %q", name, v, want)
		}
		return nil
	})
}

// testAccCaptureTokenValueAPI stores a token's value in Britive, so later steps can check the
// token was not deleted and created again.
func testAccCaptureTokenValueAPI(client *britive.Client, poolID *string, tokenName string, dst *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		token, err := client.GetBrokerPoolToken(*poolID, tokenName)
		if err != nil {
			return fmt.Errorf("reading token %s of pool %s: %w", tokenName, *poolID, err)
		}
		*dst = token.TokenValue
		return nil
	}
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
// configuration, with no brokers and no linked resources. It also checks the tokens map: every
// token with its status, description, id and created, and no token value.
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
					tokens = {
						default              = { status = "ACTIVE" }
						"AT-token-data-source" = { status = "INACTIVE", description = "AT data source token" }
					}
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
					resource.TestCheckResourceAttr(bpData, "tokens.%", "2"),
					resource.TestCheckResourceAttr(bpData, "tokens.default.status", "ACTIVE"),
					resource.TestCheckResourceAttr(bpData, "tokens.default.description", ""),
					resource.TestCheckResourceAttrPair(bpData, "tokens.default.id", bpPool, "tokens.default.id"),
					resource.TestCheckResourceAttrPair(bpData, "tokens.default.created", bpPool, "tokens.default.created"),
					resource.TestCheckResourceAttr(bpData, "tokens.AT-token-data-source.status", "INACTIVE"),
					resource.TestCheckResourceAttr(bpData, "tokens.AT-token-data-source.description", "AT data source token"),
					resource.TestCheckResourceAttrPair(bpData, "tokens.AT-token-data-source.id", bpPool, "tokens.AT-token-data-source.id"),
					resource.TestCheckResourceAttrSet(bpData, "tokens.AT-token-data-source.created"),
					resource.TestCheckNoResourceAttr(bpData, "tokens.default.value"),
					resource.TestCheckNoResourceAttr(bpData, "tokens.AT-token-data-source.value"),
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
//  1. pool with two tokens + labels, linked to a new resource, read back through the data source
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
		tokens = {
			default             = { status = "ACTIVE" }
			"AT-token-active"   = { status = "ACTIVE", description = "AT active token" }
			"AT-token-inactive" = { status = "INACTIVE" }
		}
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
		resource.TestCheckResourceAttr(bpPool, "tokens.AT-token-active.status", "ACTIVE"),
		resource.TestCheckResourceAttr(bpPool, "tokens.AT-token-inactive.status", "INACTIVE"),
		resource.TestCheckResourceAttr(bpData, "resources.#", "1"),
		resource.TestCheckResourceAttr(bpData, "resources.0.name", resourceName),
		resource.TestCheckResourceAttrSet(bpData, "resources.0.resource_id"),
		resource.TestCheckResourceAttr(bpData, "resources.0.resource_type_name", "AT-BP-End-To-End-Resource-Type"),
		resource.TestCheckResourceAttr(bpData, "tokens.%", "3"),
		resource.TestCheckResourceAttr(bpData, "tokens.AT-token-active.status", "ACTIVE"),
		resource.TestCheckResourceAttr(bpData, "tokens.AT-token-inactive.status", "INACTIVE"),
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
		tokens = { default = { status = "ACTIVE" } }
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

// testAccBrokerPoolMinimal builds a pool with only a name and the required tokens, which keep
// the default token ACTIVE.
func testAccBrokerPoolMinimal(resName, name string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool" %q {
		name = %q
		tokens = { default = { status = "ACTIVE" } }
	}
	`, resName, name)
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
