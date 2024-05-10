package util

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func Int64(in int64) basetypes.Int64Value {
	if in == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(in)
}

// StringSet accepts a `[]attr.Value` and returns a `basetypes.SetValue`. The
// return type automatically handles `SetNull` for empty results and coercing
// all element values to a string if there are any elements.
//
// nolint: contextcheck
func StringSet(in []attr.Value) basetypes.SetValue {
	if len(in) == 0 {
		return types.SetNull(types.StringType)
	}
	return types.SetValueMust(types.StringType, in)
}

// Int64Set accepts a `[]attr.Value` and returns a `basetypes.SetValue`. The
// return type automatically handles `SetNull` for empty results and coercing
// all element values to a string if there are any elements.
//
// nolint: contextcheck
func Int64Set(in []attr.Value) basetypes.SetValue {
	if len(in) == 0 {
		return types.SetNull(types.Int64Type)
	}
	return types.SetValueMust(types.Int64Type, in)
}

// String accepts a `string` and returns a `basetypes.StringValue`. The
// return type automatically handles `StringNull` should the string be empty.
//
// Removes the need for the following code when saving to state.
//
//	if response.MyField == "" {
//	    state.MyField = types.StringValue(response.MyField)
//	} else {
//	    state.MyField = types.StringNull()
//	}
//
// Not recommended if you care about returning an empty string for the state.
//
// nolint: contextcheck
func String(in string) basetypes.StringValue {
	if in == "" {
		return types.StringNull()
	}
	return types.StringValue(in)
}

// Bool accepts a `*bool` and returns a `basetypes.BoolValue`. The
// return type automatically handles `BoolNull` should the boolean not be
// initialised.
//
// This flattener saves you repeating code that looks like the following when
// saving to state.
//
//	var enabled *bool
//	if !schema.Enabled.IsNull() {
//	    requestPayload.Enabled = types.BoolValue(enabled)
//	} else {
//	    requestPayload.Enabled = types.BoolNull()
//	}
//
// nolint: contextcheck
//func Bool(in *bool) basetypes.BoolValue {
//	if reflect.ValueOf(in).IsNil() {
//		return types.BoolNull()
//	}
//	return types.BoolValue(cloudflare.Bool(in))
//}

// Int64Set accepts a `types.Set` and returns a slice of int64.
// func Int64Set(ctx context.Context, in types.Set) []int {
// 	results := []int{}
// 	_ = in.ElementsAs(ctx, &results, false)
// 	return results
// }
