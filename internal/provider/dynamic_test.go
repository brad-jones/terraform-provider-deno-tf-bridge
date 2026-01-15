package provider

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestFromDynamic_Null tests conversion of null dynamic value.
func TestFromDynamic_Null(t *testing.T) {
	dynVal := types.DynamicNull()
	result := fromDynamic(dynVal)

	if result != nil {
		t.Errorf("Expected nil for null dynamic value, got %v", result)
	}
}

// TestFromDynamic_String tests conversion of string dynamic value.
func TestFromDynamic_String(t *testing.T) {
	stringVal := types.StringValue("test")
	dynVal := types.DynamicValue(stringVal)
	result := fromDynamic(dynVal)

	if result != "test" {
		t.Errorf("Expected 'test', got %v", result)
	}
}

// TestFromDynamic_Bool tests conversion of bool dynamic value.
func TestFromDynamic_Bool(t *testing.T) {
	boolVal := types.BoolValue(true)
	dynVal := types.DynamicValue(boolVal)
	result := fromDynamic(dynVal)

	if result != true {
		t.Errorf("Expected true, got %v", result)
	}
}

// TestFromDynamic_Number tests conversion of number dynamic value.
func TestFromDynamic_Number(t *testing.T) {
	bigFloat := big.NewFloat(42.5)
	numVal := types.NumberValue(bigFloat)
	dynVal := types.DynamicValue(numVal)
	result := fromDynamic(dynVal)

	if floatResult, ok := result.(float64); !ok {
		t.Errorf("Expected float64, got %T", result)
	} else if floatResult != 42.5 {
		t.Errorf("Expected 42.5, got %v", floatResult)
	}
}

// TestFromDynamic_List tests conversion of list dynamic value.
func TestFromDynamic_List(t *testing.T) {
	listVal, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("a"),
		types.StringValue("b"),
		types.StringValue("c"),
	})
	dynVal := types.DynamicValue(listVal)
	result := fromDynamic(dynVal)

	listResult, ok := result.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", result)
	}

	expected := []any{"a", "b", "c"}
	if !reflect.DeepEqual(listResult, expected) {
		t.Errorf("Expected %v, got %v", expected, listResult)
	}
}

// TestFromDynamic_Map tests conversion of map dynamic value.
func TestFromDynamic_Map(t *testing.T) {
	mapVal, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"key1": types.StringValue("value1"),
		"key2": types.StringValue("value2"),
	})
	dynVal := types.DynamicValue(mapVal)
	result := fromDynamic(dynVal)

	mapResult, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", result)
	}

	if mapResult["key1"] != "value1" {
		t.Errorf("Expected 'value1' for key1, got %v", mapResult["key1"])
	}
	if mapResult["key2"] != "value2" {
		t.Errorf("Expected 'value2' for key2, got %v", mapResult["key2"])
	}
}

// TestFromDynamic_Object tests conversion of object dynamic value.
func TestFromDynamic_Object(t *testing.T) {
	objVal, _ := types.ObjectValue(
		map[string]attr.Type{
			"name": types.StringType,
			"age":  types.NumberType,
		},
		map[string]attr.Value{
			"name": types.StringValue("John"),
			"age":  types.NumberValue(big.NewFloat(30)),
		},
	)
	dynVal := types.DynamicValue(objVal)
	result := fromDynamic(dynVal)

	objResult, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", result)
	}

	if objResult["name"] != "John" {
		t.Errorf("Expected 'John' for name, got %v", objResult["name"])
	}
	if age, ok := objResult["age"].(float64); !ok || age != 30 {
		t.Errorf("Expected 30 for age, got %v", objResult["age"])
	}
}

// TestToDynamic_Nil tests conversion of nil to dynamic value.
func TestToDynamic_Nil(t *testing.T) {
	result := toDynamic(nil)

	if !result.IsNull() {
		t.Error("Expected null dynamic value for nil input")
	}
}

// TestToDynamic_String tests conversion of string to dynamic value.
func TestToDynamic_String(t *testing.T) {
	result := toDynamic("test")

	if result.IsNull() {
		t.Error("Expected non-null dynamic value")
	}

	underlying := result.UnderlyingValue()
	if strVal, ok := underlying.(types.String); !ok {
		t.Errorf("Expected types.String, got %T", underlying)
	} else if strVal.ValueString() != "test" {
		t.Errorf("Expected 'test', got '%s'", strVal.ValueString())
	}
}

// TestToDynamic_Bool tests conversion of bool to dynamic value.
func TestToDynamic_Bool(t *testing.T) {
	result := toDynamic(true)

	underlying := result.UnderlyingValue()
	if boolVal, ok := underlying.(types.Bool); !ok {
		t.Errorf("Expected types.Bool, got %T", underlying)
	} else if !boolVal.ValueBool() {
		t.Error("Expected true")
	}
}

// TestToDynamic_Float tests conversion of float to dynamic value.
func TestToDynamic_Float(t *testing.T) {
	result := toDynamic(42.5)

	underlying := result.UnderlyingValue()
	if numVal, ok := underlying.(types.Number); !ok {
		t.Errorf("Expected types.Number, got %T", underlying)
	} else {
		f64, _ := numVal.ValueBigFloat().Float64()
		if f64 != 42.5 {
			t.Errorf("Expected 42.5, got %v", f64)
		}
	}
}

// TestToDynamic_Map tests conversion of map to dynamic value.
func TestToDynamic_Map(t *testing.T) {
	input := map[string]any{
		"key1": "value1",
		"key2": 42,
	}
	result := toDynamic(input)

	if result.IsNull() {
		t.Error("Expected non-null dynamic value")
	}

	underlying := result.UnderlyingValue()
	if _, ok := underlying.(types.Object); !ok {
		t.Errorf("Expected types.Object, got %T", underlying)
	}
}

// TestToDynamic_Slice tests conversion of slice to dynamic value.
func TestToDynamic_Slice(t *testing.T) {
	input := []any{"a", "b", "c"}
	result := toDynamic(input)

	if result.IsNull() {
		t.Error("Expected non-null dynamic value")
	}

	// The implementation returns a List, not a Tuple
	underlying := result.UnderlyingValue()
	if _, ok := underlying.(types.List); !ok {
		t.Errorf("Expected types.List, got %T", underlying)
	}
}
