package deno

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestDenoPermissions_MapToDenoPermissions_Nil tests mapping nil permissions.
func TestDenoPermissions_MapToDenoPermissions_Nil(t *testing.T) {
	var perms *PermissionsTF = nil
	result := perms.MapToDenoPermissions()

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.All {
		t.Error("Expected All to be false for nil permissions")
	}

	if len(result.Allow) != 0 {
		t.Errorf("Expected empty Allow list, got %d items", len(result.Allow))
	}

	if len(result.Deny) != 0 {
		t.Errorf("Expected empty Deny list, got %d items", len(result.Deny))
	}
}

// TestDenoPermissions_MapToDenoPermissions_AllPermissions tests mapping with all permissions.
func TestDenoPermissions_MapToDenoPermissions_AllPermissions(t *testing.T) {
	perms := &PermissionsTF{
		All:   types.BoolValue(true),
		Allow: types.ListNull(types.StringType),
		Deny:  types.ListNull(types.StringType),
	}
	result := perms.MapToDenoPermissions()

	if !result.All {
		t.Error("Expected All to be true")
	}
}

// TestDenoPermissions_MapToDenoPermissions_AllowList tests mapping with allow list.
func TestDenoPermissions_MapToDenoPermissions_AllowList(t *testing.T) {
	allowList, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("net"),
		types.StringValue("read"),
		types.StringValue("write"),
	})

	perms := &PermissionsTF{
		All:   types.BoolValue(false),
		Allow: allowList,
		Deny:  types.ListNull(types.StringType),
	}
	result := perms.MapToDenoPermissions()

	if result.All {
		t.Error("Expected All to be false")
	}

	expectedAllow := []string{"net", "read", "write"}
	if len(result.Allow) != len(expectedAllow) {
		t.Errorf("Expected %d allow items, got %d", len(expectedAllow), len(result.Allow))
	}

	for i, expected := range expectedAllow {
		if result.Allow[i] != expected {
			t.Errorf("Expected allow[%d] to be '%s', got '%s'", i, expected, result.Allow[i])
		}
	}
}

// TestDenoPermissions_MapToDenoPermissions_DenyList tests mapping with deny list.
func TestDenoPermissions_MapToDenoPermissions_DenyList(t *testing.T) {
	denyList, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("write"),
		types.StringValue("env"),
	})

	perms := &PermissionsTF{
		All:   types.BoolValue(false),
		Allow: types.ListNull(types.StringType),
		Deny:  denyList,
	}
	result := perms.MapToDenoPermissions()

	expectedDeny := []string{"write", "env"}
	if len(result.Deny) != len(expectedDeny) {
		t.Errorf("Expected %d deny items, got %d", len(expectedDeny), len(result.Deny))
	}

	for i, expected := range expectedDeny {
		if result.Deny[i] != expected {
			t.Errorf("Expected deny[%d] to be '%s', got '%s'", i, expected, result.Deny[i])
		}
	}
}

// TestDenoPermissions_MapToDenoPermissions_BothLists tests mapping with both allow and deny lists.
func TestDenoPermissions_MapToDenoPermissions_BothLists(t *testing.T) {
	allowList, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("net"),
		types.StringValue("read"),
	})

	denyList, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("write"),
	})

	perms := &PermissionsTF{
		All:   types.BoolValue(false),
		Allow: allowList,
		Deny:  denyList,
	}
	result := perms.MapToDenoPermissions()

	if len(result.Allow) != 2 {
		t.Errorf("Expected 2 allow items, got %d", len(result.Allow))
	}

	if len(result.Deny) != 1 {
		t.Errorf("Expected 1 deny item, got %d", len(result.Deny))
	}
}

// TestDenoPermissions_MapToDenoPermissions_NullLists tests mapping with null lists.
func TestDenoPermissions_MapToDenoPermissions_NullLists(t *testing.T) {
	perms := &PermissionsTF{
		All:   types.BoolValue(false),
		Allow: types.ListNull(types.StringType),
		Deny:  types.ListNull(types.StringType),
	}
	result := perms.MapToDenoPermissions()

	if len(result.Allow) > 0 {
		t.Errorf("Expected empty or nil Allow list for null value, got %d items", len(result.Allow))
	}

	if len(result.Deny) > 0 {
		t.Errorf("Expected empty or nil Deny list for null value, got %d items", len(result.Deny))
	}
}
