package validators

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// DurationStringType is a custom attr.Type that treats duration strings as
// semantically equal when they represent the same time.Duration. This prevents
// plan diffs caused by format differences like "20m0s" vs "0h20m0s".
type DurationStringType struct{}

var _ basetypes.StringTypable = DurationStringType{}

func (t DurationStringType) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (interface{}, error) {
	return nil, fmt.Errorf("cannot apply AttributePathStep %T to %s", step, t.String())
}

func (t DurationStringType) Equal(o attr.Type) bool {
	_, ok := o.(DurationStringType)
	return ok
}

func (t DurationStringType) String() string {
	return "DurationStringType"
}

func (t DurationStringType) TerraformType(_ context.Context) tftypes.Type {
	return tftypes.String
}

func (t DurationStringType) ValueFromString(_ context.Context, v basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return DurationStringValue{StringValue: v}, nil
}

func (t DurationStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	sv := basetypes.StringType{}
	val, err := sv.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	strVal, ok := val.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T from StringType.ValueFromTerraform", val)
	}
	return DurationStringValue{StringValue: strVal}, nil
}

func (t DurationStringType) ValueType(_ context.Context) attr.Value {
	return DurationStringValue{}
}

// DurationStringValue is a string value that considers two duration strings
// semantically equal when they parse to the same time.Duration. This prevents
// plan diffs between equivalent formats like "20m0s" and "0h20m0s".
type DurationStringValue struct {
	basetypes.StringValue
}

var _ basetypes.StringValuableWithSemanticEquals = DurationStringValue{}

func (v DurationStringValue) Type(_ context.Context) attr.Type {
	return DurationStringType{}
}

func (v DurationStringValue) ToStringValue(_ context.Context) (basetypes.StringValue, diag.Diagnostics) {
	return v.StringValue, nil
}

func (v DurationStringValue) StringSemanticEquals(_ context.Context, other basetypes.StringValuable) (bool, diag.Diagnostics) {
	otherVal, ok := other.(DurationStringValue)
	if !ok {
		return false, nil
	}
	a, err1 := time.ParseDuration(v.ValueString())
	b, err2 := time.ParseDuration(otherVal.ValueString())
	if err1 != nil || err2 != nil {
		return false, nil
	}
	return a == b, nil
}

// NewDurationStringNull returns a null DurationStringValue.
func NewDurationStringNull() DurationStringValue {
	return DurationStringValue{StringValue: basetypes.NewStringNull()}
}

// NewDurationStringUnknown returns an unknown DurationStringValue.
func NewDurationStringUnknown() DurationStringValue {
	return DurationStringValue{StringValue: basetypes.NewStringUnknown()}
}

// NewDurationStringValue returns a known DurationStringValue.
func NewDurationStringValue(value string) DurationStringValue {
	return DurationStringValue{StringValue: basetypes.NewStringValue(value)}
}
