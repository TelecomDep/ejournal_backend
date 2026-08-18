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
	if len(code1) < 5 || code1[:5] != "TCHR-" {
		t.Fatalf("unexpected code prefix or format: %s", code1)
	}

	code2 := generate_invite_code("TCHR")
	if code1 == code2 {
		t.Fatalf("generated identical invite codes: %s", code1)
	}
}

