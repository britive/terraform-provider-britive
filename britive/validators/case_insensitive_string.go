package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// CaseInsensitiveStringType is a custom attr.Type that treats string values as
// semantically equal when they differ only in case. It is used for attributes
// like "version" where the API normalizes casing but configs may vary.
type CaseInsensitiveStringType struct{}

var _ basetypes.StringTypable = CaseInsensitiveStringType{}

func (t CaseInsensitiveStringType) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (interface{}, error) {
	return nil, fmt.Errorf("cannot apply AttributePathStep %T to %s", step, t.String())
}

func (t CaseInsensitiveStringType) Equal(o attr.Type) bool {
	_, ok := o.(CaseInsensitiveStringType)
	return ok
}

func (t CaseInsensitiveStringType) String() string {
	return "CaseInsensitiveStringType"
}

func (t CaseInsensitiveStringType) TerraformType(_ context.Context) tftypes.Type {
	return tftypes.String
}

func (t CaseInsensitiveStringType) ValueFromString(_ context.Context, v basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return CaseInsensitiveStringValue{StringValue: v}, nil
}

func (t CaseInsensitiveStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	sv := basetypes.StringType{}
	val, err := sv.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	strVal, ok := val.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T from StringType.ValueFromTerraform", val)
	}
	return CaseInsensitiveStringValue{StringValue: strVal}, nil
}

func (t CaseInsensitiveStringType) ValueType(_ context.Context) attr.Value {
	return CaseInsensitiveStringValue{}
}

// CaseInsensitiveStringValue is a string value that considers two values
// semantically equal when they differ only in case (EqualFold). This prevents
// perpetual plan diffs when an API normalizes casing.
type CaseInsensitiveStringValue struct {
	basetypes.StringValue
}

var _ basetypes.StringValuableWithSemanticEquals = CaseInsensitiveStringValue{}

func (v CaseInsensitiveStringValue) Type(_ context.Context) attr.Type {
	return CaseInsensitiveStringType{}
}

func (v CaseInsensitiveStringValue) ToStringValue(_ context.Context) (basetypes.StringValue, diag.Diagnostics) {
	return v.StringValue, nil
}

func (v CaseInsensitiveStringValue) StringSemanticEquals(_ context.Context, other basetypes.StringValuable) (bool, diag.Diagnostics) {
	otherVal, ok := other.(CaseInsensitiveStringValue)
	if !ok {
		return false, nil
	}
	return strings.EqualFold(v.ValueString(), otherVal.ValueString()), nil
}

// NewCaseInsensitiveStringNull returns a null CaseInsensitiveStringValue.
func NewCaseInsensitiveStringNull() CaseInsensitiveStringValue {
	return CaseInsensitiveStringValue{StringValue: basetypes.NewStringNull()}
}

// NewCaseInsensitiveStringUnknown returns an unknown CaseInsensitiveStringValue.
func NewCaseInsensitiveStringUnknown() CaseInsensitiveStringValue {
	return CaseInsensitiveStringValue{StringValue: basetypes.NewStringUnknown()}
}

// NewCaseInsensitiveStringValue returns a known CaseInsensitiveStringValue.
func NewCaseInsensitiveStringValue(value string) CaseInsensitiveStringValue {
	return CaseInsensitiveStringValue{StringValue: basetypes.NewStringValue(value)}
}
