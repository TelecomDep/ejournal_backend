package app

import (
	"fmt"
	"strings"
	"sync"
)

// Granular Role Constants in increasing hierarchical order
const (
	RoleStudent        = "student"         // Студент
	RoleTeacher        = "teacher"         // Преподаватель
	RoleSecretary      = "secretary"       // Секретарь кафедры/факультета
	RoleHead           = "head"            // Заведующий кафедрой
	RoleProgramCreator = "program_creator" // Создатель образовательной программы
	RoleDirector       = "director"        // Директор института
	RoleDean           = "dean"            // Декан факультета
	RoleMinister       = "minister"        // Министр образования
	RoleAdmin          = "admin"           // Главный системный администратор
)

func isTeachingRole(role string) bool {
	return role == RoleTeacher || role == RoleHead
}

type RolePermissions struct {
	Role                 string `json:"role"`
	Title                string `json:"title"`
	Level                int    `json:"level"`
	ScopeType            string `json:"scope_type"` // self, own_groups, lectern, faculty, global
	CanViewGlobalStats   bool   `json:"can_view_global_stats"`
	CanViewAllStudents   bool   `json:"can_view_all_students"`
	CanViewAllTeachers   bool   `json:"can_view_all_teachers"`
	CanManageSemesters   bool   `json:"can_manage_semesters"`
	CanManageUsers       bool   `json:"can_manage_users"`
	CanExportReports     bool   `json:"can_export_reports"`
	CanViewSensitiveData bool   `json:"can_view_sensitive_data"`
}

type PermissionRegistry struct {
	mu          sync.RWMutex
	permissions map[string]RolePermissions
}

var globalPermissions = newPermissionRegistry()

func newPermissionRegistry() *PermissionRegistry {
	r := &PermissionRegistry{
		permissions: make(map[string]RolePermissions),
	}
	r.initDefaults()
	return r
}

func (r *PermissionRegistry) initDefaults() {
	defaults := map[string]RolePermissions{
		RoleStudent: {
			Role:                 RoleStudent,
			Title:                "Студент",
			Level:                1,
			ScopeType:            "self",
			CanViewGlobalStats:   false,
			CanViewAllStudents:   false,
			CanViewAllTeachers:   false,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     false,
			CanViewSensitiveData: false,
		},
		RoleTeacher: {
			Role:                 RoleTeacher,
			Title:                "Преподаватель",
			Level:                2,
			ScopeType:            "own_groups",
			CanViewGlobalStats:   false,
			CanViewAllStudents:   false,
			CanViewAllTeachers:   false,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     true,
			CanViewSensitiveData: false,
		},
		RoleSecretary: {
			Role:                 RoleSecretary,
			Title:                "Секретарь кафедры/факультета",
			Level:                3,
			ScopeType:            "lectern",
			CanViewGlobalStats:   false,
			CanViewAllStudents:   false,
			CanViewAllTeachers:   true,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     true,
			CanViewSensitiveData: false,
		},
		RoleHead: {
			Role:                 RoleHead,
			Title:                "Заведующий кафедрой",
			Level:                4,
			ScopeType:            "lectern",
			CanViewGlobalStats:   false,
			CanViewAllStudents:   false,
			CanViewAllTeachers:   true,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     true,
			CanViewSensitiveData: false,
		},
		RoleProgramCreator: {
			Role:                 RoleProgramCreator,
			Title:                "Руководитель образовательной программы",
			Level:                5,
			ScopeType:            "lectern",
			CanViewGlobalStats:   false,
			CanViewAllStudents:   false,
			CanViewAllTeachers:   true,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     true,
			CanViewSensitiveData: false,
		},
		RoleDirector: {
			Role:                 RoleDirector,
			Title:                "Директор института",
			Level:                6,
			ScopeType:            "faculty",
			CanViewGlobalStats:   true,
			CanViewAllStudents:   true,
			CanViewAllTeachers:   true,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     true,
			CanViewSensitiveData: true,
		},
		RoleDean: {
			Role:                 RoleDean,
			Title:                "Декан факультета",
			Level:                7,
			ScopeType:            "faculty",
			CanViewGlobalStats:   true,
			CanViewAllStudents:   true,
			CanViewAllTeachers:   true,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     true,
			CanViewSensitiveData: true,
		},
		RoleMinister: {
			Role:                 RoleMinister,
			Title:                "Министр образования",
			Level:                8,
			ScopeType:            "global",
			CanViewGlobalStats:   true,
			CanViewAllStudents:   true,
			CanViewAllTeachers:   true,
			CanManageSemesters:   false,
			CanManageUsers:       false,
			CanExportReports:     true,
			CanViewSensitiveData: true,
		},
		RoleAdmin: {
			Role:                 RoleAdmin,
			Title:                "Главный администратор",
			Level:                9,
			ScopeType:            "global",
			CanViewGlobalStats:   true,
			CanViewAllStudents:   true,
			CanViewAllTeachers:   true,
			CanManageSemesters:   true,
			CanManageUsers:       true,
			CanExportReports:     true,
			CanViewSensitiveData: true,
		},
	}
	r.permissions = defaults
}

func (r *PermissionRegistry) GetAll() []RolePermissions {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []RolePermissions
	for _, p := range r.permissions {
		result = append(result, p)
	}
	return result
}

func (r *PermissionRegistry) Get(role string) (RolePermissions, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.permissions[strings.ToLower(strings.TrimSpace(role))]
	return p, ok
}

func (r *PermissionRegistry) Update(role string, update RolePermissions) (RolePermissions, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	roleKey := strings.ToLower(strings.TrimSpace(role))
	current, ok := r.permissions[roleKey]
	if !ok {
		return RolePermissions{}, fmt.Errorf("role %q not found", role)
	}

	// Preserve identity fields
	update.Role = current.Role
	update.Title = current.Title
	update.Level = current.Level
	update.ScopeType = current.ScopeType

	r.permissions[roleKey] = update
	return update, nil
}

func IsValidRole(role string) bool {
	_, ok := globalPermissions.Get(role)
	return ok
}

func GetRolePermissions(role string) (RolePermissions, bool) {
	return globalPermissions.Get(role)
}

func GetAllRolePermissions() []RolePermissions {
	return globalPermissions.GetAll()
}

func UpdateRolePermissions(role string, update RolePermissions) (RolePermissions, error) {
	return globalPermissions.Update(role, update)
}
