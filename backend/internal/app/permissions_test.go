package app

import (
	"testing"
)

func TestRolePermissionsHierarchy(t *testing.T) {
	roles := []string{
		RoleStudent,
		RoleTeacher,
		RoleSecretary,
		RoleHead,
		RoleProgramCreator,
		RoleDirector,
		RoleDean,
		RoleMinister,
		RoleAdmin,
	}

	for _, role := range roles {
		if !IsValidRole(role) {
			t.Errorf("expected role %q to be valid", role)
		}
		perm, found := GetRolePermissions(role)
		if !found {
			t.Errorf("expected permissions for role %q", role)
		}
		if perm.Role != role {
			t.Errorf("expected role %q, got %q", role, perm.Role)
		}
	}
}

func TestUpdateRolePermissions(t *testing.T) {
	role := RoleTeacher
	original, _ := GetRolePermissions(role)

	updatedPayload := original
	updatedPayload.CanViewGlobalStats = true

	updated, err := UpdateRolePermissions(role, updatedPayload)
	if err != nil {
		t.Fatalf("unexpected error updating role permissions: %v", err)
	}
	if !updated.CanViewGlobalStats {
		t.Errorf("expected CanViewGlobalStats to be true after update")
	}

	// Restore original
	_, _ = UpdateRolePermissions(role, original)
}
