package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	provider "github.com/britive/terraform-provider-britive/britive"
	britive "github.com/britive/terraform-provider-britive/britive-client-go"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// Acceptance tests for the britive_resource_manager_broker_pool_token ephemeral resource. They
// need Terraform 1.10 or later and are skipped on older versions.
//
// An ephemeral value can't be put in a normal resource argument, so the tests pass it to the
// "echo" test provider from terraform-plugin-testing, whose echo resource copies its provider
// configuration into the state. Only that test resource holds the value; the checks then make
// sure no other resource in the state does.

const bpEcho = "echo.test"

// testAccEphemeralProviderFactories adds the echo test provider to the Britive provider.
var testAccEphemeralProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"britive": providerserver.NewProtocol6WithError(provider.New(testVersion)()),
	"echo":    echoprovider.NewProviderServer(),
}

// bpEphemeralTokenByID is the token_id of the test token taken from the pool resource's tokens
// map. It is unknown until the token exists, so the ephemeral read waits for it.
const bpEphemeralTokenByID = `britive_resource_manager_broker_pool.pool.tokens["AT-token-ephemeral"].id`

// testAccEphemeralTokenConfig builds a pool with its default token and an ACTIVE token, and
// two ephemeral reads: the new token (token_id = tokenID) and the pool's default token
// ({pool_id}/default). Both results are passed to the echo provider.
func testAccEphemeralTokenConfig(poolName, tokenID string) string {
	return testAccEphemeralTokenConfigStatus(poolName, "ACTIVE", tokenID)
}

// testAccEphemeralTokenConfigStatus is testAccEphemeralTokenConfig with the token status set.
func testAccEphemeralTokenConfigStatus(poolName, status, tokenID string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_broker_pool" "pool" {
		name = %q
		tokens = {
			default              = { status = "ACTIVE" }
			"AT-token-ephemeral" = { status = %q }
		}
	}

	ephemeral "britive_resource_manager_broker_pool_token" "token" {
		token_id = %s
	}

	ephemeral "britive_resource_manager_broker_pool_token" "default" {
		token_id = "${britive_resource_manager_broker_pool.pool.pool_id}/default"
	}

	provider "echo" {
		data = {
			token   = ephemeral.britive_resource_manager_broker_pool_token.token
			default = ephemeral.britive_resource_manager_broker_pool_token.default
		}
	}

	# The echo resource keeps its first state and ignores later provider data, so it must be
	# replaced when the token is created again with a new value. replace_triggered_by does not
	# fire when the referenced resource is created (only updated or replaced), so the token id
	# goes through terraform_data: its input is unknown while the token is being created, which
	# plans an update of terraform_data, and that triggers the replace.
	resource "terraform_data" "token_trigger" {
		input = britive_resource_manager_broker_pool.pool.tokens["AT-token-ephemeral"].id
	}

	resource "echo" "test" {
		lifecycle {
			replace_triggered_by = [terraform_data.token_trigger]
		}
	}
	`, poolName, status, tokenID)
}

// TestBritiveBrokerPoolTokenEphemeral covers the ephemeral resource:
//  1. read a token created in the same apply, by its id in the pool's tokens map, and the pool's
//     default token by {pool_id}/default: the values match Britive, and no resource other than
//     the echo test resource has them in the state
//  2. read the same token by {pool_id}/{name}
//  3. a token that does not exist: error
//  4. an invalid token_id: error
//  5. the value in a normal resource argument: Terraform rejects it
//  6. back to a valid configuration (the pool only), so the post-test destroy can run
func TestBritiveBrokerPoolTokenEphemeral(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Token-Ephemeral"
	pool := testAccBrokerPoolMinimal("pool", poolName)

	ephemeralOnly := func(body string) string {
		return pool + fmt.Sprintf(`
		ephemeral "britive_resource_manager_broker_pool_token" "token" {
			token_id = %s
		}
		%s`, `"${britive_resource_manager_broker_pool.pool.pool_id}/AT-no-such-token"`, body)
	}

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccEphemeralProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccEphemeralTokenConfig(poolName, bpEphemeralTokenByID),
				ConfigStateChecks: []statecheck.StateCheck{
					testAccCheckEphemeralTokens(client, "AT-token-ephemeral"),
				},
			},
			{
				Config: testAccEphemeralTokenConfig(poolName,
					`"${britive_resource_manager_broker_pool.pool.pool_id}/AT-token-ephemeral"`),
				ConfigStateChecks: []statecheck.StateCheck{
					testAccCheckEphemeralTokens(client, "AT-token-ephemeral"),
				},
			},
			{
				Config:      ephemeralOnly(""),
				ExpectError: regexp.MustCompile(`Broker Pool Token Not Found`),
			},
			{
				Config: pool + `
				ephemeral "britive_resource_manager_broker_pool_token" "token" {
					token_id = "a/b/c"
				}`,
				ExpectError: regexp.MustCompile(`Invalid Token ID`),
			},
			{
				// Terraform rejects an ephemeral value in an argument that would be stored.
				Config: pool + `
				ephemeral "britive_resource_manager_broker_pool_token" "default" {
					token_id = "${britive_resource_manager_broker_pool.pool.pool_id}/default"
				}

				resource "terraform_data" "leak" {
					input = ephemeral.britive_resource_manager_broker_pool_token.default.token_value
				}`,
				ExpectError: regexp.MustCompile(`(?i)ephemeral`),
			},
			{
				// The post-test destroy uses the last step's configuration, and Terraform rejects
				// the configuration above even when destroying, so end with a valid one.
				Config: pool,
			},
		},
	})
}

// TestBritiveBrokerPoolTokenEphemeralInactiveAndDeleted covers an INACTIVE token and a token
// deleted outside Terraform:
//  1. read an INACTIVE token by its id in the tokens map: the value is returned and status is
//     INACTIVE
//  2. delete the token through the API, with token_id written as "{pool_id}/{name}": the
//     token_id is known at plan time, so Terraform reads the token during the plan, before the
//     pool resource can create it again, and the plan fails with "not found"
//  3. same situation, with token_id = the id in the tokens map: the id is unknown until the
//     token is created again, so the read waits for it and the apply succeeds
func TestBritiveBrokerPoolTokenEphemeralInactiveAndDeleted(t *testing.T) {
	client := testAccBritiveClient(t)
	poolName := "AT-BP-Token-Ephemeral-Inactive"
	tokenName := "AT-token-ephemeral"
	byName := `"${britive_resource_manager_broker_pool.pool.pool_id}/AT-token-ephemeral"`
	byID := bpEphemeralTokenByID
	var poolID string

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccEphemeralProviderFactories,
		CheckDestroy:             testAccCheckBrokerPoolDestroy(client),
		Steps: []resource.TestStep{
			{
				// By id, so the read waits for the token to be created.
				Config: testAccEphemeralTokenConfigStatus(poolName, "INACTIVE", byID),
				Check:  testAccCaptureAttr(bpPool, "pool_id", &poolID),
				ConfigStateChecks: []statecheck.StateCheck{
					testAccCheckEphemeralTokens(client, tokenName),
					testAccCheckEphemeralStatus("token", "INACTIVE"),
				},
			},
			{
				PreConfig: func() {
					if err := client.DeleteBrokerPoolToken(poolID, tokenName); err != nil {
						t.Fatalf("deleting token outside Terraform: %v", err)
					}
				},
				Config:      testAccEphemeralTokenConfigStatus(poolName, "INACTIVE", byName),
				ExpectError: regexp.MustCompile(`Broker Pool Token Not Found`),
			},
			{
				Config: testAccEphemeralTokenConfigStatus(poolName, "INACTIVE", byID),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(bpPool, plancheck.ResourceActionUpdate),
				}},
				ConfigStateChecks: []statecheck.StateCheck{
					testAccCheckEphemeralTokens(client, tokenName),
					testAccCheckEphemeralStatus("token", "INACTIVE"),
				},
			},
		},
	})
}

// testAccCheckEphemeralStatus checks the status the echo resource received for one ephemeral
// read ("token" or "default").
func testAccCheckEphemeralStatus(key, want string) statecheck.StateCheck {
	return testAccStateCheckFunc(func(state *tfjson.State) error {
		if state == nil || state.Values == nil || state.Values.RootModule == nil {
			return fmt.Errorf("state is empty")
		}
		for _, r := range state.Values.RootModule.Resources {
			if r.Address != bpEcho {
				continue
			}
			data, _ := r.AttributeValues["data"].(map[string]interface{})
			got, _ := data[key].(map[string]interface{})
			if got == nil {
				return fmt.Errorf("echo data has no %q", key)
			}
			if got["status"] != want {
				return fmt.Errorf("ephemeral %q status = %v, want %s", key, got["status"], want)
			}
			return nil
		}
		return fmt.Errorf("%s not in state", bpEcho)
	})
}

// testAccStateCheckFunc adapts a function to statecheck.StateCheck.
type testAccStateCheckFunc func(*tfjson.State) error

func (f testAccStateCheckFunc) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	resp.Error = f(req.State)
}

// testAccCheckEphemeralTokens checks what the echo resource received from the two ephemeral
// reads in testAccEphemeralTokenConfig against Britive, and that no other resource in the
// state holds either token value.
func testAccCheckEphemeralTokens(client *britive.Client, tokenName string) statecheck.StateCheck {
	return testAccStateCheckFunc(func(state *tfjson.State) error {
		if state == nil || state.Values == nil || state.Values.RootModule == nil {
			return fmt.Errorf("state is empty")
		}
		resources := map[string]*tfjson.StateResource{}
		for _, r := range state.Values.RootModule.Resources {
			resources[r.Address] = r
		}

		poolRes, ok := resources[bpPool]
		if !ok {
			return fmt.Errorf("%s not in state", bpPool)
		}
		poolID, _ := poolRes.AttributeValues["pool_id"].(string)

		echo, ok := resources[bpEcho]
		if !ok {
			return fmt.Errorf("%s not in state", bpEcho)
		}
		data, _ := echo.AttributeValues["data"].(map[string]interface{})

		var values []string
		for key, name := range map[string]string{"token": tokenName, "default": "default"} {
			got, _ := data[key].(map[string]interface{})
			if got == nil {
				return fmt.Errorf("echo data has no %q", key)
			}
			want, err := client.GetBrokerPoolToken(poolID, name)
			if err != nil {
				return fmt.Errorf("reading token %s of pool %s: %w", name, poolID, err)
			}
			if got["token_value"] != want.TokenValue || want.TokenValue == "" {
				return fmt.Errorf("ephemeral token_value of %q does not match the value in Britive", name)
			}
			if got["pool_id"] != poolID || got["name"] != name || got["status"] != want.Status {
				return fmt.Errorf("ephemeral %q = pool_id %v, name %v, status %v; want %s, %s, %s",
					key, got["pool_id"], got["name"], got["status"], poolID, name, want.Status)
			}
			values = append(values, want.TokenValue)
		}

		// Only the echo test resource may hold the values.
		for address, r := range resources {
			if address == bpEcho {
				continue
			}
			raw, err := json.Marshal(r.AttributeValues)
			if err != nil {
				return err
			}
			for _, v := range values {
				if strings.Contains(string(raw), v) {
					return fmt.Errorf("%s holds a token value in the state", address)
				}
			}
			if _, has := r.AttributeValues["token_value"]; has {
				return fmt.Errorf("%s has a token_value attribute in the state", address)
			}
		}
		return nil
	})
}
