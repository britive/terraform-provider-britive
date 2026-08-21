package tests

import (
	"context"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/resources"
	"github.com/britive/terraform-provider-britive/britive/resources/resourcemanager"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Terraform evaluates a resource's configuration during the validate walk
// without instance-expansion context, so any value a for_each instance derives
// from each.value - including a whole dynamic block - arrives at the provider as
// unknown. Providers must tolerate that; decoding such a config into a model
// whose nested blocks are plain Go slices fails with "Received unknown value,
// however the target type cannot handle unknown values" and blocks every
// for_each-driven configuration at `terraform validate`/`plan` time.
//
// These tests exercise ValidateConfig/ModifyPlan directly (no tenant or
// credentials involved) with the collections in question set to unknown.

// unknownConfigResource is the set of behaviours a resource needs for these tests.
type unknownConfigResource interface {
	resource.Resource
}

// rawWithUnknowns builds a raw config/plan value for the resource's schema where
// every top-level attribute and block is null, except the named ones, which are
// unknown.
func rawWithUnknowns(t *testing.T, r unknownConfigResource, unknownNames ...string) (tftypes.Value, tfsdk.Config) {
	t.Helper()

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", schemaResp.Diagnostics)
	}

	objectType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("expected schema to produce a tftypes.Object")
	}

	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	for _, name := range unknownNames {
		attributeType, ok := objectType.AttributeTypes[name]
		if !ok {
			t.Fatalf("attribute %q not found in schema", name)
		}
		values[name] = tftypes.NewValue(attributeType, tftypes.UnknownValue)
	}

	raw := tftypes.NewValue(objectType, values)
	return raw, tfsdk.Config{Raw: raw, Schema: schemaResp.Schema}
}

func validateConfigWithUnknowns(t *testing.T, r unknownConfigResource, unknownNames ...string) {
	t.Helper()

	validator, ok := r.(resource.ResourceWithValidateConfig)
	if !ok {
		t.Fatalf("%T does not implement ValidateConfig", r)
	}

	_, config := rawWithUnknowns(t, r, unknownNames...)
	resp := &resource.ValidateConfigResponse{}
	validator.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ValidateConfig errored on unknown %v: %v", unknownNames, resp.Diagnostics)
	}
}

func modifyPlanWithUnknowns(t *testing.T, r unknownConfigResource, unknownNames ...string) {
	t.Helper()

	planModifier, ok := r.(resource.ResourceWithModifyPlan)
	if !ok {
		t.Fatalf("%T does not implement ModifyPlan", r)
	}

	raw, config := rawWithUnknowns(t, r, unknownNames...)
	plan := tfsdk.Plan{Raw: raw, Schema: config.Schema}
	req := resource.ModifyPlanRequest{
		Config: config,
		Plan:   plan,
		State:  tfsdk.State{Raw: tftypes.NewValue(raw.Type(), nil), Schema: config.Schema},
	}
	resp := &resource.ModifyPlanResponse{Plan: plan}
	planModifier.ModifyPlan(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan errored on unknown %v: %v", unknownNames, resp.Diagnostics)
	}
}

func TestValidateConfigToleratesUnknownCollections(t *testing.T) {
	tests := map[string]struct {
		resource unknownConfigResource
		unknowns []string
	}{
		"profile associations":              {resources.NewProfileResource(), []string{"associations"}},
		"profile tag_associations":          {resources.NewProfileResource(), []string{"tag_associations"}},
		"profile both":                      {resources.NewProfileResource(), []string{"associations", "tag_associations"}},
		"tag_owner user and tag":            {resources.NewTagOwnerResource(), []string{"user", "tag"}},
		"resource_type_permission response": {resourcemanager.NewResourceTypePermissionsResource(), []string{"response_templates", "variables"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			validateConfigWithUnknowns(t, test.resource, test.unknowns...)
		})
	}
}

func TestModifyPlanToleratesUnknownCollections(t *testing.T) {
	tests := map[string]struct {
		resource unknownConfigResource
		unknowns []string
	}{
		"profile associations":                          {resources.NewProfileResource(), []string{"associations", "tag_associations"}},
		"resource_type_permission collections":          {resourcemanager.NewResourceTypePermissionsResource(), []string{"response_templates", "variables"}},
		"resource_manager profile_permission variables": {resourcemanager.NewProfilePermissionResource(), []string{"variables"}},
		"resource_label values":                         {resourcemanager.NewResourceLabelResource(), []string{"values"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			modifyPlanWithUnknowns(t, test.resource, test.unknowns...)
		})
	}
}
