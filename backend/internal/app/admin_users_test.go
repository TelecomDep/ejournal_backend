package app

import "testing"

func TestValidateAdminUserCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    AdminUserCreateData
		wantErr bool
	}{
		{
			name: "admin",
			data: AdminUserCreateData{
				Login:    "admin_2",
				Password: "good_password",
				Role:     RoleAdmin,
			},
		},
		{
			name: "student with profile",
			data: AdminUserCreateData{
				Login:    "student_2",
				Password: "good_password",
				Role:     RoleStudent,
				FullName: "Иван Иванов",
			},
		},
		{
			name: "student without name",
			data: AdminUserCreateData{
				Login:    "student_2",
				Password: "good_password",
				Role:     RoleStudent,
			},
			wantErr: true,
		},
		{
			name: "head without lectern",
			data: AdminUserCreateData{
				Login:    "head_2",
				Password: "good_password",
				Role:     RoleHead,
			},
			wantErr: true,
		},
		{
			name: "dean with faculty",
			data: AdminUserCreateData{
				Login:     "dean_2",
				Password:  "good_password",
				Role:      RoleDean,
				FacultyID: 1,
			},
		},
		{
			name: "dean teacher and admin",
			data: AdminUserCreateData{
				Login:       "multi_role_1",
				Password:    "good_password",
				Roles:       []string{RoleDean, RoleTeacher, RoleAdmin},
				PrimaryRole: RoleDean,
				FullName:    "Иван Иванов",
				FacultyID:   1,
			},
		},
		{
			name: "multi role primary is not assigned",
			data: AdminUserCreateData{
				Login:       "multi_role_2",
				Password:    "good_password",
				Roles:       []string{RoleDean, RoleTeacher},
				PrimaryRole: RoleAdmin,
				FullName:    "Иван Иванов",
				FacultyID:   1,
			},
			wantErr: true,
		},
		{
			name: "multi role missing lectern scope",
			data: AdminUserCreateData{
				Login:       "multi_role_3",
				Password:    "good_password",
				Roles:       []string{RoleTeacher, RoleSecretary},
				PrimaryRole: RoleTeacher,
				FullName:    "Иван Иванов",
			},
			wantErr: true,
		},
		{
			name: "short password",
			data: AdminUserCreateData{
				Login:    "admin_2",
				Password: "1234567",
				Role:     RoleAdmin,
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			data: AdminUserCreateData{
				Login:    "admin_2",
				Password: "good_password",
				Role:     "super_admin",
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validate_admin_user_create(test.data)
			if (err != nil) != test.wantErr {
				t.Fatalf("validate_admin_user_create() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestCheckAdminRoleTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    string
		data    AdminUserUpdateData
		wantErr bool
	}{
		{name: "admin needs no target", role: RoleAdmin},
		{name: "student profile", role: RoleStudent, data: AdminUserUpdateData{StudentID: 10}},
		{name: "student without profile", role: RoleStudent, wantErr: true},
		{name: "teacher profile", role: RoleTeacher, data: AdminUserUpdateData{TeacherID: 20}},
		{name: "head lectern", role: RoleHead, data: AdminUserUpdateData{LecternID: 30}},
		{name: "dean faculty", role: RoleDean, data: AdminUserUpdateData{FacultyID: 40}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := check_admin_role_target(test.data, test.role)
			if (err != nil) != test.wantErr {
				t.Fatalf("check_admin_role_target() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestGenerateInviteCode(t *testing.T) {
	t.Parallel()

	code1 := generate_invite_code("TCHR")
	if len(code1) != 16 {
		t.Fatalf("expected 16-char hex code, got %s (len %d)", code1, len(code1))
	}

	code2 := generate_invite_code("TCHR")
	if code1 == code2 {
		t.Fatalf("generated identical invite codes: %s", code1)
	}
}

func TestNormalizeAdminRoles(t *testing.T) {
	roles, err := normalize_admin_roles([]string{" dean ", "teacher", "dean", "ADMIN"})
	if err != nil {
		t.Fatalf("normalize_admin_roles() error = %v", err)
	}
	want := []string{RoleDean, RoleTeacher, RoleAdmin}
	if !same_role_set(roles, want) || len(roles) != len(want) {
		t.Fatalf("normalize_admin_roles() = %#v, want %#v", roles, want)
	}

	if _, err := normalize_admin_roles(nil); err == nil {
		t.Fatal("expected empty role set to be rejected")
	}
	if _, err := normalize_admin_roles([]string{"super_admin"}); err == nil {
		t.Fatal("expected invalid role to be rejected")
	}
}

func TestNormalizeAdminCreateRoles(t *testing.T) {
	roles, primary, err := normalize_admin_create_roles(AdminUserCreateData{
		Roles:       []string{" dean ", "teacher", "dean", "ADMIN"},
		PrimaryRole: "TEACHER",
	})
	if err != nil {
		t.Fatalf("normalize_admin_create_roles() error = %v", err)
	}
	if primary != RoleTeacher || !same_role_set(roles, []string{RoleDean, RoleTeacher, RoleAdmin}) {
		t.Fatalf("normalize_admin_create_roles() = (%#v, %q)", roles, primary)
	}

	legacyRoles, legacyPrimary, err := normalize_admin_create_roles(AdminUserCreateData{Role: RoleStudent})
	if err != nil {
		t.Fatalf("legacy normalize_admin_create_roles() error = %v", err)
	}
	if len(legacyRoles) != 1 || legacyRoles[0] != RoleStudent || legacyPrimary != RoleStudent {
		t.Fatalf("legacy normalize_admin_create_roles() = (%#v, %q)", legacyRoles, legacyPrimary)
	}
}
