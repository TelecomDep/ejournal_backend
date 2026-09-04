package app

import (
	"encoding/base64"
	"encoding/json"
	"strings"
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

func TestIsTeachingRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{role: RoleTeacher, want: true},
		{role: RoleHead, want: true},
		{role: RoleDean, want: false},
		{role: RoleAdmin, want: false},
		{role: RoleStudent, want: false},
	}

	for _, test := range tests {
		if got := isTeachingRole(test.role); got != test.want {
			t.Errorf("isTeachingRole(%q) = %v, want %v", test.role, got, test.want)
		}
	}
}

func TestUserHasRole(t *testing.T) {
	user := User{
		Role:        RoleDean,
		PrimaryRole: RoleDean,
		Roles:       []string{RoleDean, RoleTeacher, RoleAdmin},
	}
	for _, role := range []string{RoleDean, RoleTeacher, RoleAdmin} {
		if !user.HasRole(role) {
			t.Fatalf("expected user to have role %q", role)
		}
	}
	if user.HasRole(RoleStudent) {
		t.Fatal("did not expect user to have student role")
	}
	user.PrimaryRole = RoleStudent
	if user.HasRole(RoleStudent) {
		t.Fatal("non-empty assignment list must not fall back to an inconsistent primary role")
	}
	legacy := User{Role: RoleTeacher}
	if !legacy.HasRole(RoleTeacher) {
		t.Fatal("expected legacy user without role list to retain its role")
	}
}

func TestSessionTokenCarriesActiveRole(t *testing.T) {
	service := &Service{jwtSecret: []byte("test-secret-with-sufficient-length")}
	token, err := service.generateJWTForRole(42, 3, RoleTeacher)
	if err != nil {
		t.Fatalf("generateJWTForRole() error = %v", err)
	}
	userID, version, role, err := service.validateJWT(token)
	if err != nil {
		t.Fatalf("validateJWT() error = %v", err)
	}
	if userID != 42 || version != 3 || role != RoleTeacher {
		t.Fatalf("validated claims = (%d, %d, %q)", userID, version, role)
	}
}

func TestSessionTokenRejectsForgedActiveRole(t *testing.T) {
	service := &Service{jwtSecret: []byte("test-secret-with-sufficient-length")}
	token, err := service.generateJWTForRole(42, 3, RoleTeacher)
	if err != nil {
		t.Fatalf("generateJWTForRole() error = %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected JWT format: %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal JWT payload: %v", err)
	}
	claims["active_role"] = RoleAdmin
	forgedPayload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal forged JWT payload: %v", err)
	}
	forged := parts[0] + "." + base64.RawURLEncoding.EncodeToString(forgedPayload) + "." + parts[2]

	if _, _, _, err := service.validateJWT(forged); err == nil {
		t.Fatal("expected forged active role to invalidate token signature")
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
