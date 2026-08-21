// Package tfutils holds small helpers shared by the resource implementations for
// working with Terraform Plugin Framework values.
package tfutils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// HasUnknownStructure reports whether any of the named top-level attributes or
// blocks in raw holds an unknown collection or an unknown object.
//
// Resource models that store a nested block as a plain Go slice (rather than a
// types.Set/types.List) cannot represent an unknown collection: decoding one via
// Config.Get/Plan.Get fails with "Received unknown value, however the target type
// cannot handle unknown values". Unknown collections are legitimate — Terraform
// evaluates a for_each resource during the validate walk without instance
// expansion context, so a dynamic block fed from each.value arrives unknown — so
// callers use this to skip the decode rather than surfacing a provider error.
//
// Unknown *primitive* values (a string attribute referencing an unapplied
// resource) are representable by the types.String/types.Bool/... fields of those
// models and are therefore not reported here.
func HasUnknownStructure(raw tftypes.Value, names ...string) bool {
	if raw.IsNull() {
		return false
	}
	if !raw.IsKnown() {
		return true
	}

	var attrs map[string]tftypes.Value
	if err := raw.As(&attrs); err != nil {
		// Not an object — nothing addressable by name, so nothing to skip.
		return false
	}

	for _, name := range names {
		if v, ok := attrs[name]; ok && hasUnknownStructure(v) {
			return true
		}
	}
	return false
}

func hasUnknownStructure(v tftypes.Value) bool {
	ty := v.Type()
	structural := ty.Is(tftypes.List{}) || ty.Is(tftypes.Set{}) || ty.Is(tftypes.Tuple{}) ||
		ty.Is(tftypes.Map{}) || ty.Is(tftypes.Object{})
	if !structural {
		return false
	}
	if !v.IsKnown() {
		return true
	}
	if v.IsNull() {
		return false
	}

	if ty.Is(tftypes.Object{}) || ty.Is(tftypes.Map{}) {
		var attrs map[string]tftypes.Value
		if err := v.As(&attrs); err != nil {
			return true
		}
		for _, a := range attrs {
			if hasUnknownStructure(a) {
				return true
			}
		}
		return false
	}

	var elems []tftypes.Value
	if err := v.As(&elems); err != nil {
		return true
	}
	for _, e := range elems {
		if hasUnknownStructure(e) {
			return true
		}
	}
	return false
}

// ElementsAsSlice decodes a set of nested blocks into a slice of models. A null
// or unknown set decodes to nil, and individual elements that are null or
// unknown are skipped, so it is safe to call on a configuration that has not
// been fully resolved yet.
func ElementsAsSlice[T any](ctx context.Context, set types.Set) ([]T, diag.Diagnostics) {
	var diags diag.Diagnostics

	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}

	elements := set.Elements()
	out := make([]T, 0, len(elements))
	for _, element := range elements {
		object, ok := element.(basetypes.ObjectValuable)
		if !ok {
			continue
		}
		objectValue, objectDiags := object.ToObjectValue(ctx)
		diags.Append(objectDiags...)
		if objectDiags.HasError() {
			continue
		}
		if objectValue.IsNull() || objectValue.IsUnknown() {
			continue
		}

		var target T
		diags.Append(objectValue.As(ctx, &target, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		out = append(out, target)
	}

	return out, diags
}
