package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	britive "github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Shared helpers for the policy resource tests (britive_policy, britive_profile_policy,
// britive_resource_manager_profile_policy and britive_resource_manager_resource_policy).

// testAccPolicyAPIClient builds an API client from BRITIVE_TENANT / BRITIVE_TOKEN, the same
// way the provider does, so tests can change policies outside Terraform to simulate drift.
func testAccPolicyAPIClient(t *testing.T) *britive.Client {
	t.Helper()
	tenant := os.Getenv("BRITIVE_TENANT")
	if !strings.Contains(tenant, "://") {
		tenant = "https://" + tenant
	}
	apiBaseURL := fmt.Sprintf("%s/api", strings.TrimSuffix(tenant, "/"))
	client, err := britive.NewClient(apiBaseURL, os.Getenv("BRITIVE_TOKEN"), "acctest", 0)
	if err != nil {
		t.Fatalf("could not create Britive client: %v", err)
	}
	return client
}

// testAccRequireAcc skips the test unless acceptance tests are enabled. Tests that call
// the API while building their configs call it first, since resource.Test only skips once
// it runs.
func testAccRequireAcc(t *testing.T) {
	t.Helper()
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}
}

// testAccNotificationMediumID returns the ID of the notification medium named name, so a
// by-ID policy config can reference it the way the API returns it when read by ID.
func testAccNotificationMediumID(t *testing.T, name string) string {
	t.Helper()
	client := testAccPolicyAPIClient(t)
	req, err := http.NewRequest(http.MethodGet, client.APIBaseURL+"/v1/notification-service/notificationmediums", nil)
	if err != nil {
		t.Fatalf("could not build notification mediums request: %v", err)
	}
	body, err := client.Do(req)
	if err != nil {
		t.Fatalf("could not list notification mediums: %v", err)
	}
	var mediums struct {
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &mediums); err != nil {
		t.Fatalf("could not parse notification mediums: %v", err)
	}
	for _, m := range mediums.Result {
		if m.Name == name {
			return m.ID
		}
	}
	t.Fatalf("notification medium %q not found", name)
	return ""
}

// testAccCaptureAttrs copies attrs of resource n from state into dst, so a later import
// step can compare the imported values against them.
func testAccCaptureAttrs(n string, dst map[string]string, attrs ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errs.NewNotFoundErrorf("%s in state", n)
		}
		for _, attr := range attrs {
			dst[attr] = rs.Primary.Attributes[attr]
		}
		return nil
	}
}

// policyJSONComparators compare the JSON string attributes of policy resources
// semantically, ignoring key order and formatting, using the provider's own helpers.
var policyJSONComparators = map[string]func(old, new string) bool{
	"members":     britive.MembersEqual,
	"condition":   britive.ConditionEqual,
	"permissions": britive.ArrayOfMapsEqual,
	"roles":       britive.ArrayOfMapsEqual,
}

// policyJSONAttrs lists the JSON string attributes ImportStateVerify can't compare
// byte-for-byte; testAccImportStateCheckPolicyJSON compares them semantically instead.
var policyJSONAttrs = []string{"members", "condition", "permissions", "roles"}

// testAccImportStateCheckPolicyJSON checks that each captured JSON attribute matches the
// imported value semantically. Attributes captured as empty are skipped.
func testAccImportStateCheckPolicyJSON(captured map[string]string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected 1 imported instance, got %d", len(states))
		}
		imported := states[0].Attributes
		for attr, want := range captured {
			if want == "" {
				continue
			}
			equal, ok := policyJSONComparators[attr]
			if !ok {
				return fmt.Errorf("no JSON comparator for attribute %q", attr)
			}
			got := imported[attr]
			if got == "" || !equal(got, want) {
				return fmt.Errorf("imported %s = %s, want equivalent of %s", attr, got, want)
			}
		}
		return nil
	}
}

// testAccProfilePolicyImportID returns an import ID for a britive_profile_policy, in the
// form "paps/{profile_id}/policies/{policy_name}" or, with short, "{profile_id}/{policy_name}".
func testAccProfilePolicyImportID(n string, short bool) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", errs.NewNotFoundErrorf("%s in state", n)
		}
		profileID, policyName := rs.Primary.Attributes["profile_id"], rs.Primary.Attributes["policy_name"]
		if short {
			return fmt.Sprintf("%s/%s", profileID, policyName), nil
		}
		return fmt.Sprintf("paps/%s/policies/%s", profileID, policyName), nil
	}
}

// testAccPolicyImportID returns an import ID for a britive_policy, in the form
// "policies/{name}" or, with short, "{name}".
func testAccPolicyImportID(n string, short bool) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", errs.NewNotFoundErrorf("%s in state", n)
		}
		name := rs.Primary.Attributes["name"]
		if short {
			return name, nil
		}
		return fmt.Sprintf("policies/%s", name), nil
	}
}

// testAccResourceManagerProfilePolicyImportID returns an import ID for a
// britive_resource_manager_profile_policy, in the form
// "resource-manager/profiles/{profile_id}/policies/{policy_name}" or, with short,
// "{profile_id}/{policy_name}". The profile ID is taken from the resource ID
// ("resource-manager/profiles/{profile_id}/policies/{policy_id}").
func testAccResourceManagerProfilePolicyImportID(n string, short bool) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", errs.NewNotFoundErrorf("%s in state", n)
		}
		parts := strings.Split(rs.Primary.ID, "/")
		if len(parts) != 5 {
			return "", fmt.Errorf("unexpected resource manager profile policy ID %q", rs.Primary.ID)
		}
		profileID, policyName := parts[2], rs.Primary.Attributes["policy_name"]
		if short {
			return fmt.Sprintf("%s/%s", profileID, policyName), nil
		}
		return fmt.Sprintf("resource-manager/profiles/%s/policies/%s", profileID, policyName), nil
	}
}

// testAccResourceManagerResourcePolicyImportID returns an import ID for a
// britive_resource_manager_resource_policy, in the form
// "resource-manager/policies/{policy_name}" or, with short, "{policy_name}".
func testAccResourceManagerResourcePolicyImportID(n string, short bool) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", errs.NewNotFoundErrorf("%s in state", n)
		}
		name := rs.Primary.Attributes["policy_name"]
		if short {
			return name, nil
		}
		return fmt.Sprintf("resource-manager/policies/%s", name), nil
	}
}

// testAccImportSteps returns two import steps for resource n, one per import ID format,
// each verifying plain attributes with ImportStateVerify and the JSON attributes
// semantically against captured. ignore lists further attributes that import can't
// reproduce exactly.
func testAccImportSteps(n string, idFunc func(n string, short bool) resource.ImportStateIdFunc, captured map[string]string, ignore ...string) []resource.TestStep {
	ignored := append(append([]string{}, policyJSONAttrs...), ignore...)
	steps := make([]resource.TestStep, 0, 2)
	for _, short := range []bool{false, true} {
		steps = append(steps, resource.TestStep{
			ResourceName:            n,
			ImportState:             true,
			ImportStateIdFunc:       idFunc(n, short),
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: ignored,
			ImportStateCheck:        testAccImportStateCheckPolicyJSON(captured),
		})
	}
	return steps
}

// testAccPolicyTagMembers returns a members JSON fragment listing the given tags by name.
func testAccPolicyTagMembers(tags ...string) string {
	entries := make([]string, 0, len(tags))
	for _, tag := range tags {
		entries = append(entries, fmt.Sprintf("{ name = %q }", tag))
	}
	return "[" + strings.Join(entries, ", ") + "]"
}

// Tags and user that exist in the acceptance test tenant, used as policy members.
const (
	testAccPolicyTag  = "britiveProviderAcceptanceTestTag"
	testAccPolicyTag1 = "britiveProviderAcceptanceTestTag1"
	testAccPolicyUser = "britiveprovideracceptancetest"
)

// testAccPolicyEditConfig renders a policy config with the fields the edit, import and
// drift tests change: description, is_active, member tags (by name) and an approval or
// IP condition value.
type testAccPolicyEditConfig func(description string, isActive bool, tags []string, conditionValue int) string

// testAccPolicyEditFieldsSteps creates a policy, then updates description, is_active,
// members and condition in place, checking each apply leaves an empty plan.
func testAccPolicyEditFieldsSteps(n string, config testAccPolicyEditConfig) []resource.TestStep {
	return []resource.TestStep{
		{
			Config: config("Initial policy description", true, []string{testAccPolicyTag}, 30),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(n, "description", "Initial policy description"),
				resource.TestCheckResourceAttr(n, "is_active", "true"),
			),
		},
		{
			Config: config("Updated policy description", false, []string{testAccPolicyTag, testAccPolicyTag1}, 60),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(n, plancheck.ResourceActionUpdate),
				},
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(n, "description", "Updated policy description"),
				resource.TestCheckResourceAttr(n, "is_active", "false"),
				resource.TestCheckResourceAttrWith(n, "members", func(v string) error {
					if !strings.Contains(v, testAccPolicyTag1) {
						return fmt.Errorf("members %q does not contain %q", v, testAccPolicyTag1)
					}
					return nil
				}),
			),
		},
	}
}

// testAccPolicyImportByNameSteps creates a policy with members given by name, then
// imports it with each import ID format.
func testAccPolicyImportByNameSteps(n string, config testAccPolicyEditConfig, idFunc func(n string, short bool) resource.ImportStateIdFunc, ignore ...string) []resource.TestStep {
	captured := map[string]string{}
	steps := []resource.TestStep{
		{
			Config: config("Import by name policy description", true, []string{testAccPolicyTag, testAccPolicyTag1}, 30),
			Check:  testAccCaptureAttrs(n, captured, policyJSONAttrs...),
		},
	}
	return append(steps, testAccImportSteps(n, idFunc, captured, ignore...)...)
}

// testAccPolicyDriftSteps creates a policy with two member tags, then drift removes one
// of them outside Terraform: the next plan must show a change, and applying restores it.
// capture runs after the first apply to record whatever drift needs (IDs, names).
func testAccPolicyDriftSteps(config testAccPolicyEditConfig, capture resource.TestCheckFunc, drift func()) []resource.TestStep {
	cfg := config("Drift policy description", true, []string{testAccPolicyTag, testAccPolicyTag1}, 30)
	return []resource.TestStep{
		{
			Config: cfg,
			Check:  capture,
		},
		{
			PreConfig:          drift,
			Config:             cfg,
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		},
		{
			Config: cfg,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
	}
}

// testAccDriftMembers returns members with tags reduced to testAccPolicyTag, keeping
// every other member type as the API returned it.
func testAccDriftMembers(t *testing.T, members interface{}) map[string]interface{} {
	t.Helper()
	m, ok := members.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected members type %T", members)
	}
	m["tags"] = []map[string]string{{"name": testAccPolicyTag}}
	return m
}

// testAccCheckAttrContainsAttrOf checks that attr of resource n contains the value of
// otherAttr of resource other, e.g. that members lists a tag's ID.
func testAccCheckAttrContainsAttrOf(n, attr, other, otherAttr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errs.NewNotFoundErrorf("%s in state", n)
		}
		ors, ok := s.RootModule().Resources[other]
		if !ok {
			return errs.NewNotFoundErrorf("%s in state", other)
		}
		want := ors.Primary.Attributes[otherAttr]
		if got := rs.Primary.Attributes[attr]; want == "" || !strings.Contains(got, want) {
			return fmt.Errorf("%s.%s %q does not contain %s.%s %q", n, attr, got, other, otherAttr, want)
		}
		return nil
	}
}

// testAccMemberTagsByID is the members.tags line of the by-ID policy configs.
const testAccMemberTagsByID = `tags  = [{ id = data.britive_tag.policy_tag.tag_id }]`

// testAccWithSecondTagByID adds testAccPolicyTag1, by ID, to the member tags of a by-ID
// policy config.
func testAccWithSecondTagByID(config string) string {
	return strings.Replace(config, testAccMemberTagsByID,
		`tags  = [{ id = data.britive_tag.policy_tag.tag_id }, { id = data.britive_tag.policy_tag1.tag_id }]`, 1) +
		fmt.Sprintf(`

	data "britive_tag" "policy_tag1" {
		name = %q
	}`, testAccPolicyTag1)
}

// testAccMembersByIDUpdateAndImportSteps follows a by-ID create: it adds a second member
// tag by ID in place, checking the apply keeps IDs and leaves an empty plan, then imports
// the policy with each import ID format.
//
// Import only has the policy name, so it reads the policy by name and the imported
// members carry names: the import steps check that both member tags are there by name
// rather than comparing members and condition with the by-ID state.
func testAccMembersByIDUpdateAndImportSteps(n, config string, idFunc func(n string, short bool) resource.ImportStateIdFunc, ignore ...string) []resource.TestStep {
	steps := []resource.TestStep{
		{
			Config: testAccWithSecondTagByID(config),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(n, plancheck.ResourceActionUpdate),
				},
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				testAccCheckAttrContainsAttrOf(n, "members", "data.britive_tag.policy_tag", "tag_id"),
				testAccCheckAttrContainsAttrOf(n, "members", "data.britive_tag.policy_tag1", "tag_id"),
			),
		},
	}
	ignored := append(append([]string{}, policyJSONAttrs...), ignore...)
	for _, short := range []bool{false, true} {
		steps = append(steps, resource.TestStep{
			ResourceName:            n,
			ImportState:             true,
			ImportStateIdFunc:       idFunc(n, short),
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: ignored,
			ImportStateCheck:        testAccImportStateCheckMembersContain(testAccPolicyTag, testAccPolicyTag1),
		})
	}
	return steps
}

// testAccImportStateCheckMembersContain checks that the imported members mention each of
// values.
func testAccImportStateCheckMembersContain(values ...string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected 1 imported instance, got %d", len(states))
		}
		members := states[0].Attributes["members"]
		for _, v := range values {
			if !strings.Contains(members, fmt.Sprintf("%q", v)) {
				return fmt.Errorf("imported members %s do not contain %q", members, v)
			}
		}
		return nil
	}
}

// testAccPolicyByIDSteps covers a policy whose members and condition reference tags, users
// and the notification medium by ID - the form the API returns when a policy is read by
// ID. config renders it for a description and policy name. It checks that:
//   - create leaves members and condition as configured, with no "inconsistent result
//     after apply" and an empty plan after refresh,
//   - changing only the description updates in place (the update must address the policy
//     by ID, or the API rejects the ID members with PP-0005),
//   - renaming the policy updates in place,
//   - adding a second tag by ID keeps IDs, and import (by name) finds both tags.
func testAccPolicyByIDSteps(n, policyName string, config func(description, policyName string) string, idFunc func(n string, short bool) resource.ImportStateIdFunc, ignore ...string) []resource.TestStep {
	renamed := policyName + " Renamed"
	steps := []resource.TestStep{
		{
			Config: config("By ID policy description", policyName),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				testAccCheckAttrContainsAttrOf(n, "members", "data.britive_tag.policy_tag", "tag_id"),
				testAccCheckAttrContainsAttrOf(n, "members", "data.britive_user.policy_user", "user_id"),
				testAccCheckAttrContainsAttrOf(n, "condition", "data.britive_tag.policy_tag", "tag_id"),
			),
		},
		{
			Config: config("Updated by ID policy description", policyName),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(n, plancheck.ResourceActionUpdate),
				},
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: resource.TestCheckResourceAttr(n, "description", "Updated by ID policy description"),
		},
		{
			Config: config("Updated by ID policy description", renamed),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(n, plancheck.ResourceActionUpdate),
				},
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
	}
	return append(steps, testAccMembersByIDUpdateAndImportSteps(n, config("Updated by ID policy description", renamed), idFunc, ignore...)...)
}

// testAccPolicySwitchNamesAndIDsSteps creates a policy with members and condition by name,
// switches the config to the same members and condition by ID, then back to names. Each
// switch is an in-place update that must be accepted by the API and leave an empty plan.
func testAccPolicySwitchNamesAndIDsSteps(n string, byName, byID string) []resource.TestStep {
	switchStep := func(config string) resource.TestStep {
		return resource.TestStep{
			Config: config,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(n, plancheck.ResourceActionUpdate),
				},
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		}
	}
	return []resource.TestStep{
		{
			Config: byName,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
		switchStep(byID),
		switchStep(byName),
	}
}

// testAccPolicyMembersByName is the members of the name-based policy configs, with tags
// given by name.
func testAccPolicyMembersByName(tags ...string) string {
	return fmt.Sprintf(`{
			tags  = %s
			users = [{ name = %q }]
		}`, testAccPolicyTagMembers(tags...), testAccPolicyUser)
}

// testAccPolicyMembersByID is the members of the by-ID policy configs.
const testAccPolicyMembersByID = `{
			` + testAccMemberTagsByID + `
			users = [{ id = data.britive_user.policy_user.user_id }]
		}`

// testAccPolicyApprovalCondition is an approval condition approved by testAccPolicyTag and
// notified by the "Email" medium, both given by name or, with byID, by ID.
func testAccPolicyApprovalCondition(byID bool, emailMediumID string, timeToApprove int) string {
	approverTag, medium := fmt.Sprintf("%q", testAccPolicyTag), `"Email"`
	if byID {
		approverTag, medium = "data.britive_tag.policy_tag.tag_id", fmt.Sprintf("%q", emailMediumID)
	}
	return fmt.Sprintf(`{
			approval = {
				approvers          = { tags = [%s] }
				notificationMedium = [%s]
				managerApproval    = { condition = "All", required = true }
				timeToApprove      = %d
				isValidForInDays   = false
				validFor           = 120
			}
		}`, approverTag, medium, timeToApprove)
}

// testAccPolicyByIDDataSources looks up the tag and user the by-ID configs reference.
func testAccPolicyByIDDataSources() string {
	return fmt.Sprintf(`
	data "britive_tag" "policy_tag" {
		name = %q
	}

	data "britive_user" "policy_user" {
		name = %q
	}`, testAccPolicyTag, testAccPolicyUser)
}
