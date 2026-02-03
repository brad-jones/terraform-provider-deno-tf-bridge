package dynamic

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func FromDynamic(dynVal types.Dynamic) any {
	if dynVal.IsNull() || dynVal.IsUnderlyingValueNull() {
		return nil
	}

	underlyingValue := dynVal.UnderlyingValue()

	switch v := underlyingValue.(type) {
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Number:
		bigFloat := v.ValueBigFloat()
		if bigFloat != nil {
			f64, _ := bigFloat.Float64()
			return f64
		}
		return nil
	case types.List:
		elements := v.Elements()
		result := make([]any, len(elements))
		for i, elem := range elements {
			result[i] = FromValue(elem)
		}
		return result
	case types.Map:
		elements := v.Elements()
		result := make(map[string]any)
		for k, elem := range elements {
			result[k] = FromValue(elem)
		}
		return result
	case types.Object:
		attrs := v.Attributes()
		result := make(map[string]any)
		for k, attr := range attrs {
			result[k] = FromValue(attr)
		}
		return result
	default:
		return fmt.Sprintf("%+v", v)
	}
}

func FromValue(in attr.Value) any {
	if in.IsNull() {
		return nil
	}

	switch v := in.(type) {
	case types.Dynamic:
		return FromDynamic(v)
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Number:
		bigFloat := v.ValueBigFloat()
		if bigFloat != nil {
			f64, _ := bigFloat.Float64()
			return f64
		}
		return nil
	case types.List:
		elements := v.Elements()
		result := make([]any, len(elements))
		for i, elem := range elements {
			result[i] = FromValue(elem)
		}
		return result
	case types.Map:
		elements := v.Elements()
		result := make(map[string]any)
		for k, elem := range elements {
			result[k] = FromValue(elem)
		}
		return result
	case types.Object:
		attrs := v.Attributes()
		result := make(map[string]any)
		for k, attr := range attrs {
			result[k] = FromValue(attr)
		}
		return result
	default:
		return fmt.Sprintf("%+v", v)
	}
}

func ToDynamic(value any) types.Dynamic {
	if value == nil {
		return types.DynamicNull()
	}

	// Dereference pointers
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return types.DynamicNull()
		}
		rv = rv.Elem()
	}
	value = rv.Interface()

	switch v := value.(type) {
	case string:
		return types.DynamicValue(types.StringValue(v))
	case bool:
		return types.DynamicValue(types.BoolValue(v))
	case float64:
		numVal := types.NumberValue(big.NewFloat(v))
		return types.DynamicValue(numVal)
	case float32:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case int:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case int64:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case int32:
		numVal := types.NumberValue(big.NewFloat(float64(v)))
		return types.DynamicValue(numVal)
	case []any:
		elements := make([]attr.Value, len(v))
		for i, elem := range v {
			elements[i] = ToDynamic(elem)
		}
		listVal, _ := types.ListValue(types.DynamicType, elements)
		return types.DynamicValue(listVal)
	case map[string]any:
		// Convert map to object instead of map to support mixed types
		elements := make(map[string]attr.Value)
		attrTypes := make(map[string]attr.Type)
		for k, elem := range v {
			dynValue := ToDynamic(elem)
			elements[k] = dynValue
			attrTypes[k] = types.DynamicType
		}
		objVal, _ := types.ObjectValue(attrTypes, elements)
		return types.DynamicValue(objVal)
	default:
		// Fallback: convert to string
		return types.DynamicValue(types.StringValue(fmt.Sprintf("%+v", v)))
	}
}
