package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type AdminUsersListData struct {
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Search   string `json:"search"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type AdminUserCreateData struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	GroupID   int32  `json:"group_id"`
	LecternID int32  `json:"lectern_id"`
	FacultyID int32  `json:"faculty_id"`
	JobTitle  string `json:"job_title"`
}

type AdminUserUpdateData struct {
	UserID    int32   `json:"user_id"`
	Login     *string `json:"login,omitempty"`
	Password  *string `json:"password,omitempty"`
	Role      *string `json:"role,omitempty"`
	Email     *string `json:"email,omitempty"`
	Status    *string `json:"status,omitempty"`
	StudentID int32   `json:"student_id,omitempty"`
	TeacherID int32   `json:"teacher_id,omitempty"`
	LecternID int32   `json:"lectern_id,omitempty"`
	FacultyID int32   `json:"faculty_id,omitempty"`
}

type AdminUserIDData struct {
	UserID int32 `json:"user_id"`
}

func valid_admin_role(role string) bool {
	return IsValidRole(role)
}

func valid_user_status(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "blocked", "archived":
		return true
	default:
		return false
	}
}

func validate_admin_user_create(data AdminUserCreateData) error {
	data.Login = strings.TrimSpace(data.Login)
	data.Password = strings.TrimSpace(data.Password)
	data.Role = strings.ToLower(strings.TrimSpace(data.Role))
	data.FullName = strings.TrimSpace(data.FullName)

	if data.Login == "" {
		return errors.New("login is required")
	}
	if len(data.Login) > 255 {
		return errors.New("login is too long")
	}
	if len(data.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if !valid_admin_role(data.Role) {
		return errors.New("invalid user role")
	}
	if email := strings.ToLower(strings.TrimSpace(data.Email)); email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || !strings.EqualFold(parsed.Address, email) || len(email) > 254 {
			return errors.New("invalid email")
		}
	}

	switch data.Role {
	case RoleStudent, RoleTeacher:
		if data.FullName == "" {
			return errors.New("full_name is required for student or teacher")
		}
	case RoleHead, RoleSecretary, RoleProgramCreator:
		if data.LecternID <= 0 {
			return errors.New("lectern_id is required for head, secretary or program_creator")
		}
	case RoleDean, RoleDirector:
		if data.FacultyID <= 0 {
			return errors.New("faculty_id is required for dean or director")
		}
	}

	return nil
}

func admin_user_db_error(err error) string {
	var pg_err *pgconn.PgError
	if !errors.As(err, &pg_err) {
		return "failed to save user"
	}

	switch pg_err.Code {
	case "23505":
		return "login, email or profile is already used"
	case "23503":
		return "group, lectern, faculty or profile not found"
	case "23514", "22P02":
		return "invalid user data"
	default:
		return "failed to save user"
	}
}

func (s *Service) require_admin(token string) (User, Response) {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return User{}, Response{OK: false, Error: err.Error()}
	}
	if user.Role != RoleAdmin {
		return User{}, Response{OK: false, Error: "forbidden: admin role required"}
	}
	return user, Response{OK: true}
}

func (s *Service) admin_users_list(token string, data AdminUsersListData) Response {
	if _, auth := s.require_admin(token); !auth.OK {
		return auth
	}

	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 20
	}
	if data.PageSize > 100 {
		data.PageSize = 100
	}

	data.Role = strings.ToLower(strings.TrimSpace(data.Role))
	data.Status = strings.ToLower(strings.TrimSpace(data.Status))
	if data.Role != "" && !valid_admin_role(data.Role) {
		return Response{OK: false, Error: "invalid user role"}
	}
	if data.Status != "" && !valid_user_status(data.Status) {
		return Response{OK: false, Error: "invalid user status"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	items, total, err := s.store.Users.AdminList(ctx, db.AdminUserFilter{
		Page:     data.Page,
		PageSize: data.PageSize,
		Search:   data.Search,
		Role:     data.Role,
		Status:   data.Status,
	})
	if err != nil {
		return Response{OK: false, Error: "failed to load users"}
	}

	pages := int32(0)
	if total > 0 {
		pages = (total + data.PageSize - 1) / data.PageSize
	}

	return Response{OK: true, Result: map[string]any{
		"items": items,
		"pagination": map[string]any{
			"page":      data.Page,
			"page_size": data.PageSize,
			"total":     total,
			"pages":     pages,
		},
	}}
}

func (s *Service) admin_user_get(token string, user_id int32) Response {
	if _, auth := s.require_admin(token); !auth.OK {
		return auth
	}
	if user_id <= 0 {
		return Response{OK: false, Error: "user_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	item, found, err := s.store.Users.AdminGetByID(ctx, user_id)
	if err != nil {
		return Response{OK: false, Error: "failed to load user"}
	}
	if !found {
		return Response{OK: false, Error: "user not found"}
	}

	return Response{OK: true, Result: item}
}

func nullable_id(value int32) any {
	if value <= 0 {
		return nil
	}
	return value
}

func create_admin_role_data(ctx context.Context, tx pgx.Tx, user_id int32, data AdminUserCreateData) error {
	switch data.Role {
	case RoleStudent:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO students (role, user_id, student_name, group_id)
			 VALUES ('student', $1, $2, $3)`,
			user_id,
			strings.TrimSpace(data.FullName),
			nullable_id(data.GroupID),
		)
		return err
	case RoleTeacher:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO teachers (role, user_id, name, lectern_id, job_title)
			 VALUES ('teacher', $1, $2, $3, NULLIF($4, ''))`,
			user_id,
			strings.TrimSpace(data.FullName),
			nullable_id(data.LecternID),
			strings.TrimSpace(data.JobTitle),
		)
		return err
	case RoleHead, RoleSecretary, RoleProgramCreator:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO org_scopes (user_id, lectern_id)
			 VALUES ($1, $2)`,
			user_id,
			data.LecternID,
		)
		return err
	case RoleDean, RoleDirector:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO org_scopes (user_id, faculty_id)
			 VALUES ($1, $2)`,
			user_id,
			data.FacultyID,
		)
		return err
	default:
		return nil
	}
}

func (s *Service) admin_user_create(token string, data AdminUserCreateData) Response {
	actor, auth := s.require_admin(token)
	if !auth.OK {
		return auth
	}
	if err := validate_admin_user_create(data); err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	data.Login = strings.TrimSpace(data.Login)
	data.Email = strings.TrimSpace(data.Email)
	data.Role = strings.ToLower(strings.TrimSpace(data.Role))
	hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(data.Password)), bcrypt.DefaultCost)
	if err != nil {
		return Response{OK: false, Error: "failed to hash password"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	tx, err := s.store.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Response{OK: false, Error: "failed to start transaction"}
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var user_id int32
	err = tx.QueryRow(
		ctx,
		`INSERT INTO users (login, password_hash, role, email)
		 VALUES ($1, $2, $3, NULLIF($4, ''))
		 RETURNING id`,
		data.Login,
		string(hash),
		data.Role,
		data.Email,
	).Scan(&user_id)
	if err != nil {
		return Response{OK: false, Error: admin_user_db_error(err)}
	}

	if err := create_admin_role_data(ctx, tx, user_id, data); err != nil {
		return Response{OK: false, Error: admin_user_db_error(err)}
	}
	if err := tx.Commit(ctx); err != nil {
		return Response{OK: false, Error: "failed to create user"}
	}

	item, found, err := s.store.Users.AdminGetByID(ctx, user_id)
	if err != nil || !found {
		return Response{OK: false, Error: "user created but failed to load result"}
	}
	if err := s.RecordAuditLog(ctx, actor.ID, actor.Login, actor.Role, "user_created", "user", strconv.Itoa(int(user_id)), map[string]any{
		"login": data.Login, "role": data.Role,
	}, ""); err != nil {
		log.Printf("failed to audit admin user creation: %v", err)
	}
	return Response{OK: true, Result: item}
}

func check_admin_role_target(data AdminUserUpdateData, role string) error {
	switch role {
	case RoleStudent:
		if data.StudentID <= 0 {
			return errors.New("student_id is required when role changes to student")
		}
	case RoleTeacher:
		if data.TeacherID <= 0 {
			return errors.New("teacher_id is required when role changes to teacher")
		}
	case RoleHead, RoleSecretary, RoleProgramCreator:
		if data.LecternID <= 0 {
			return errors.New("lectern_id is required for lectern-scoped role")
		}
	case RoleDean, RoleDirector:
		if data.FacultyID <= 0 {
			return errors.New("faculty_id is required for faculty-scoped role")
		}
	}
	return nil
}

func bind_admin_role(ctx context.Context, tx pgx.Tx, user_id int32, role string, data AdminUserUpdateData) error {
	switch role {
	case RoleStudent:
		cmd, err := tx.Exec(
			ctx,
			`UPDATE students
			 SET user_id = $1
			 WHERE student_id = $2
			   AND (user_id IS NULL OR user_id = $1)`,
			user_id,
			data.StudentID,
		)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return errors.New("student profile not found or already used")
		}
	case RoleTeacher:
		cmd, err := tx.Exec(
			ctx,
			`UPDATE teachers
			 SET user_id = $1
			 WHERE teacher_id = $2
			   AND (user_id IS NULL OR user_id = $1)`,
			user_id,
			data.TeacherID,
		)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return errors.New("teacher profile not found or already used")
		}
	case RoleHead, RoleSecretary, RoleProgramCreator:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO org_scopes (user_id, lectern_id)
			 VALUES ($1, $2)`,
			user_id,
			data.LecternID,
		)
		return err
	case RoleDean, RoleDirector:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO org_scopes (user_id, faculty_id)
			 VALUES ($1, $2)`,
			user_id,
			data.FacultyID,
		)
		return err
	}
	return nil
}

func (s *Service) admin_user_update(token string, data AdminUserUpdateData) Response {
	actor, auth := s.require_admin(token)
	if !auth.OK {
		return auth
	}
	if data.UserID <= 0 {
		return Response{OK: false, Error: "user_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	tx, err := s.store.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Response{OK: false, Error: "failed to start transaction"}
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var current db.User
	err = tx.QueryRow(
		ctx,
		`SELECT id, login, password_hash, role::text, email, status,
		        is_2fa_enabled, created_at, updated_at
		 FROM users
		 WHERE id = $1
		 FOR UPDATE`,
		data.UserID,
	).Scan(
		&current.ID,
		&current.Login,
		&current.PasswordHash,
		&current.Role,
		&current.Email,
		&current.Status,
		&current.TwoFaEnabled,
		&current.CreatedAt,
		&current.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{OK: false, Error: "user not found"}
	}
	if err != nil {
		return Response{OK: false, Error: "failed to load user"}
	}

	login := current.Login
	if data.Login != nil {
		login = strings.TrimSpace(*data.Login)
		if login == "" {
			return Response{OK: false, Error: "login is required"}
		}
	}

	email := ""
	if current.Email != nil {
		email = *current.Email
	}
	if data.Email != nil {
		email = strings.ToLower(strings.TrimSpace(*data.Email))
		if email != "" {
			parsed, parseErr := mail.ParseAddress(email)
			if parseErr != nil || !strings.EqualFold(parsed.Address, email) || len(email) > 254 {
				return Response{OK: false, Error: "invalid email"}
			}
		}
	}

	role := current.Role
	if data.Role != nil {
		role = strings.ToLower(strings.TrimSpace(*data.Role))
		if !valid_admin_role(role) {
			return Response{OK: false, Error: "invalid user role"}
		}
	}

	status := current.Status
	if data.Status != nil {
		status = strings.ToLower(strings.TrimSpace(*data.Status))
		if !valid_user_status(status) {
			return Response{OK: false, Error: "invalid user status"}
		}
	}

	if actor.ID == current.ID && (role != current.Role || status != current.Status) {
		return Response{OK: false, Error: "admin cannot change own role or status"}
	}

	if current.Role == RoleAdmin && current.Status == "active" && (role != RoleAdmin || status != "active") {
		var active_admins int32
		if err := tx.QueryRow(
			ctx,
			`SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'`,
		).Scan(&active_admins); err != nil {
			return Response{OK: false, Error: "failed to check active admins"}
		}
		if active_admins <= 1 {
			return Response{OK: false, Error: "cannot disable the last active admin"}
		}
	}

	password_hash := current.PasswordHash
	if data.Password != nil {
		password := strings.TrimSpace(*data.Password)
		if len(password) < 8 {
			return Response{OK: false, Error: "password must be at least 8 characters long"}
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return Response{OK: false, Error: "failed to hash password"}
		}
		password_hash = string(hash)
	}

	role_changed := role != current.Role
	rebind_role := role_changed ||
		(role == RoleStudent && data.StudentID > 0) ||
		(role == RoleTeacher && data.TeacherID > 0) ||
		((role == RoleHead || role == RoleSecretary || role == RoleProgramCreator) && data.LecternID > 0) ||
		((role == RoleDean || role == RoleDirector) && data.FacultyID > 0)
	if role_changed {
		if err := check_admin_role_target(data, role); err != nil {
			return Response{OK: false, Error: err.Error()}
		}
	}
	if rebind_role {
		if _, err := tx.Exec(ctx, `UPDATE students SET user_id = NULL WHERE user_id = $1`, current.ID); err != nil {
			return Response{OK: false, Error: "failed to detach old student profile"}
		}
		if _, err := tx.Exec(ctx, `UPDATE teachers SET user_id = NULL WHERE user_id = $1`, current.ID); err != nil {
			return Response{OK: false, Error: "failed to detach old teacher profile"}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM org_scopes WHERE user_id = $1`, current.ID); err != nil {
			return Response{OK: false, Error: "failed to clear old user scope"}
		}
	}

	_, err = tx.Exec(
		ctx,
		`UPDATE users
		 SET login = $2,
		     password_hash = $3,
		     role = $4,
		     email = NULLIF($5, ''),
		     status = $6,
		     token_version = token_version + 1
		 WHERE id = $1`,
		current.ID,
		login,
		password_hash,
		role,
		email,
		status,
	)
	if err != nil {
		return Response{OK: false, Error: admin_user_db_error(err)}
	}

	if rebind_role {
		if err := bind_admin_role(ctx, tx, current.ID, role, data); err != nil {
			if strings.Contains(err.Error(), "profile not found") {
				return Response{OK: false, Error: err.Error()}
			}
			return Response{OK: false, Error: admin_user_db_error(err)}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Response{OK: false, Error: "failed to update user"}
	}

	item, found, err := s.store.Users.AdminGetByID(ctx, current.ID)
	if err != nil || !found {
		return Response{OK: false, Error: "user updated but failed to load result"}
	}
	if err := s.RecordAuditLog(ctx, actor.ID, actor.Login, actor.Role, "user_updated", "user", strconv.Itoa(int(current.ID)), map[string]any{
		"old_role": current.Role, "new_role": role, "old_status": current.Status, "new_status": status,
	}, ""); err != nil {
		log.Printf("failed to audit admin user update: %v", err)
	}
	return Response{OK: true, Result: item}
}

func (s *Service) admin_user_delete(token string, user_id int32) Response {
	actor, auth := s.require_admin(token)
	if !auth.OK {
		return auth
	}
	if user_id <= 0 {
		return Response{OK: false, Error: "user_id is required"}
	}
	if actor.ID == user_id {
		return Response{OK: false, Error: "admin cannot archive own account"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	tx, err := s.store.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Response{OK: false, Error: "failed to start transaction"}
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var role string
	var status string
	err = tx.QueryRow(
		ctx,
		`SELECT role::text, status FROM users WHERE id = $1 FOR UPDATE`,
		user_id,
	).Scan(&role, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{OK: false, Error: "user not found"}
	}
	if err != nil {
		return Response{OK: false, Error: "failed to load user"}
	}
	if status == "archived" {
		return Response{OK: false, Error: "user is already archived"}
	}

	if role == RoleAdmin && status == "active" {
		var active_admins int32
		if err := tx.QueryRow(
			ctx,
			`SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'`,
		).Scan(&active_admins); err != nil {
			return Response{OK: false, Error: "failed to check active admins"}
		}
		if active_admins <= 1 {
			return Response{OK: false, Error: "cannot disable the last active admin"}
		}
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE users SET status = 'archived', token_version = token_version + 1 WHERE id = $1`,
		user_id,
	); err != nil {
		return Response{OK: false, Error: "failed to archive user"}
	}
	if err := tx.Commit(ctx); err != nil {
		return Response{OK: false, Error: "failed to archive user"}
	}
	if err := s.RecordAuditLog(ctx, actor.ID, actor.Login, actor.Role, "user_archived", "user", strconv.Itoa(int(user_id)), map[string]any{
		"previous_role": role, "previous_status": status,
	}, ""); err != nil {
		log.Printf("failed to audit admin user archive: %v", err)
	}

	return Response{OK: true, Result: map[string]any{
		"user_id": user_id,
		"status":  "archived",
	}}
}

type AdminOrgFaculty struct {
	FacultyID int32             `json:"faculty_id"`
	Name      string            `json:"name"`
	Lecterns  []AdminOrgLectern `json:"lecterns"`
}

type AdminOrgLectern struct {
	LecternID int32           `json:"lectern_id"`
	FacultyID int32           `json:"faculty_id"`
	Name      string          `json:"name"`
	Groups    []AdminOrgGroup `json:"groups"`
}

type AdminOrgGroup struct {
	GroupID   int32  `json:"group_id"`
	LecternID int32  `json:"lectern_id"`
	Name      string `json:"name"`
}

func (s *Service) admin_system_stats(token string) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister {
		return Response{OK: false, Error: "forbidden: admin or minister role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	roleCounts := make(map[string]int32)
	rows, err := s.store.Pool().Query(ctx, `SELECT role::text, COUNT(*)::INTEGER FROM users GROUP BY role`)
	if err == nil {
		for rows.Next() {
			var r string
			var cnt int32
			if scanErr := rows.Scan(&r, &cnt); scanErr == nil {
				roleCounts[r] = cnt
			}
		}
		rows.Close()
	}

	var activeSemester string
	_ = s.store.Pool().QueryRow(ctx, `SELECT name FROM semesters WHERE status = 'open' LIMIT 1`).Scan(&activeSemester)
	if activeSemester == "" {
		activeSemester = "None"
	}

	var totalGroups int32
	_ = s.store.Pool().QueryRow(ctx, `SELECT COUNT(*)::INTEGER FROM groups`).Scan(&totalGroups)

	runtimeStats := s.RuntimeStats()

	return Response{
		OK: true,
		Result: map[string]any{
			"user_counts":     roleCounts,
			"active_semester": activeSemester,
			"total_groups":    totalGroups,
			"runtime_stats":   runtimeStats,
		},
	}
}

func (s *Service) admin_org_structure(token string) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister && actor.Role != RoleDean && actor.Role != RoleDirector {
		return Response{OK: false, Error: "forbidden: supervisory role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()
	scope, err := s.scopeForUser(ctx, actor)
	if err != nil {
		return Response{OK: false, Error: "failed to resolve scope"}
	}
	scopeWhere := "TRUE"
	args := make([]any, 0)
	if !scope.All {
		scopeWhere = "lectern_id = ANY($1)"
		args = append(args, nonNil(scope.LecternIDs))
	}

	facultiesMap := make(map[int32]*AdminOrgFaculty)

	fRows, err := s.store.Pool().Query(ctx, `SELECT DISTINCT f.faculty_id, f.name
		FROM faculties f JOIN lecterns l ON l.faculty_id = f.faculty_id
		WHERE `+strings.ReplaceAll(scopeWhere, "lectern_id", "l.lectern_id")+` ORDER BY f.name`, args...)
	if err != nil {
		return Response{OK: false, Error: "failed to load faculties"}
	}
	for fRows.Next() {
		var f AdminOrgFaculty
		if scanErr := fRows.Scan(&f.FacultyID, &f.Name); scanErr != nil {
			fRows.Close()
			return Response{OK: false, Error: "failed to scan faculties"}
		}
		f.Lecterns = make([]AdminOrgLectern, 0)
		facultiesMap[f.FacultyID] = &f
	}
	fRows.Close()

	lecternsMap := make(map[int32]*AdminOrgLectern)
	lRows, err := s.store.Pool().Query(ctx, `SELECT lectern_id, faculty_id, name FROM lecterns WHERE `+scopeWhere+` ORDER BY name`, args...)
	if err == nil {
		for lRows.Next() {
			var l AdminOrgLectern
			if scanErr := lRows.Scan(&l.LecternID, &l.FacultyID, &l.Name); scanErr == nil {
				l.Groups = make([]AdminOrgGroup, 0)
				lecternsMap[l.LecternID] = &l
			}
		}
		lRows.Close()
	}

	gRows, err := s.store.Pool().Query(ctx, `SELECT group_id, COALESCE(lectern_id, 0), group_name FROM groups WHERE `+scopeWhere+` ORDER BY group_name`, args...)
	if err == nil {
		for gRows.Next() {
			var g AdminOrgGroup
			if scanErr := gRows.Scan(&g.GroupID, &g.LecternID, &g.Name); scanErr == nil {
				if lPtr, ok := lecternsMap[g.LecternID]; ok {
					lPtr.Groups = append(lPtr.Groups, g)
				}
			}
		}
		gRows.Close()
	}
	for _, lPtr := range lecternsMap {
		if fPtr, ok := facultiesMap[lPtr.FacultyID]; ok {
			fPtr.Lecterns = append(fPtr.Lecterns, *lPtr)
		}
	}

	facultiesList := make([]AdminOrgFaculty, 0, len(facultiesMap))
	for _, fPtr := range facultiesMap {
		facultiesList = append(facultiesList, *fPtr)
	}

	return Response{
		OK:     true,
		Result: facultiesList,
	}
}

func (s *Service) admin_roles_list(token string) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}
	return Response{
		OK:     true,
		Result: GetAllRolePermissions(),
	}
}

func (s *Service) admin_role_update(token string, role string, payload RolePermissions) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}

	return Response{OK: false, Error: "role permissions are code-defined and cannot be changed at runtime"}
}

type AdminGenerateTeacherInviteData struct {
	TeacherID  int32  `json:"teacher_id"`
	FullName   string `json:"full_name"`
	LecternID  int32  `json:"lectern_id"`
	JobTitle   string `json:"job_title"`
	CustomCode string `json:"custom_code"`
}

type AdminInvitesListData struct {
	Role   string `json:"role"`
	Status string `json:"status"` // "pending", "used", ""
}

type AdminRevokeInviteData struct {
	InviteID int32 `json:"invite_id"`
}

type AdminInviteItem struct {
	InviteID     int32      `json:"invite_id"`
	InviteCode   string     `json:"invite_code"`
	Role         string     `json:"role"`
	TeacherID    *int32     `json:"teacher_id,omitempty"`
	TeacherName  string     `json:"teacher_name,omitempty"`
	LecternID    *int32     `json:"lectern_id,omitempty"`
	LecternName  string     `json:"lectern_name,omitempty"`
	StudentID    *int32     `json:"student_id,omitempty"`
	StudentName  string     `json:"student_name,omitempty"`
	UsedAt       *time.Time `json:"used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	RegisteredAs string     `json:"registered_as,omitempty"`
}

func generate_invite_code(prefix string) string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%s-%X", prefix, b)
}

func (s *Service) admin_generate_teacher_invite(token string, data AdminGenerateTeacherInviteData) Response {
	if _, auth := s.require_admin(token); !auth.OK {
		return auth
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var teacherID int32
	var teacherName string

	if data.TeacherID > 0 {
		var existingUserID *int32
		err := s.store.Pool().QueryRow(
			ctx,
			`SELECT teacher_id, name, user_id FROM teachers WHERE teacher_id = $1`,
			data.TeacherID,
		).Scan(&teacherID, &teacherName, &existingUserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{OK: false, Error: "teacher not found"}
		}
		if err != nil {
			return Response{OK: false, Error: "failed to check teacher"}
		}
		if existingUserID != nil {
			return Response{OK: false, Error: "teacher already has an active user account"}
		}

		var existingInviteCode string
		err = s.store.Pool().QueryRow(
			ctx,
			`SELECT invite_code FROM registration_invites WHERE teacher_id = $1 AND used_at IS NULL LIMIT 1`,
			teacherID,
		).Scan(&existingInviteCode)
		if err == nil && existingInviteCode != "" {
			return Response{
				OK: true,
				Result: map[string]any{
					"invite_code":  existingInviteCode,
					"teacher_id":   teacherID,
					"teacher_name": teacherName,
					"is_existing":  true,
				},
			}
		}
	} else {
		fullName := strings.TrimSpace(data.FullName)
		if fullName == "" {
			return Response{OK: false, Error: "full_name is required when creating a new teacher invite"}
		}

		err := s.store.Pool().QueryRow(
			ctx,
			`INSERT INTO teachers (role, name, lectern_id, job_title)
			 VALUES ('teacher', $1, $2, NULLIF($3, ''))
			 RETURNING teacher_id, name`,
			fullName,
			nullable_id(data.LecternID),
			strings.TrimSpace(data.JobTitle),
		).Scan(&teacherID, &teacherName)
		if err != nil {
			return Response{OK: false, Error: admin_user_db_error(err)}
		}
	}

	inviteCode := strings.TrimSpace(strings.ToUpper(data.CustomCode))
	if inviteCode == "" {
		inviteCode = generate_invite_code("TCHR")
	}

	var inviteID int32
	var createdAt time.Time
	if err := s.store.Pool().QueryRow(
		ctx,
		`INSERT INTO registration_invites (invite_code, role, teacher_id)
		 VALUES ($1, 'teacher', $2)
		 RETURNING invite_id, created_at`,
		inviteCode,
		teacherID,
	).Scan(&inviteID, &createdAt); err != nil {
		return Response{OK: false, Error: "failed to generate invite code (code may already exist)"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"invite_id":    inviteID,
			"invite_code":  inviteCode,
			"teacher_id":   teacherID,
			"teacher_name": teacherName,
			"created_at":   createdAt,
		},
	}
}

func (s *Service) admin_list_invites(token string, data AdminInvitesListData) Response {
	if _, auth := s.require_admin(token); !auth.OK {
		return auth
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	query := `
		SELECT 
			ri.invite_id,
			ri.invite_code,
			ri.role::text,
			ri.teacher_id,
			COALESCE(t.name, ''),
			t.lectern_id,
			COALESCE(l.name, ''),
			ri.student_id,
			COALESCE(st.student_name, ''),
			ri.used_at,
			ri.created_at,
			COALESCE(u.login, '')
		FROM registration_invites ri
		LEFT JOIN teachers t ON ri.teacher_id = t.teacher_id
		LEFT JOIN lecterns l ON t.lectern_id = l.lectern_id
		LEFT JOIN students st ON ri.student_id = st.student_id
		LEFT JOIN users u ON u.id = COALESCE(t.user_id, st.user_id)
		WHERE 1=1
	`
	args := []any{}
	argID := 1

	role := strings.ToLower(strings.TrimSpace(data.Role))
	if role != "" {
		query += fmt.Sprintf(" AND ri.role = $%d::user_role", argID)
		args = append(args, role)
		argID++
	}

	status := strings.ToLower(strings.TrimSpace(data.Status))
	if status == "pending" {
		query += " AND ri.used_at IS NULL"
	} else if status == "used" {
		query += " AND ri.used_at IS NOT NULL"
	}

	query += " ORDER BY ri.created_at DESC"

	rows, err := s.store.Pool().Query(ctx, query, args...)
	if err != nil {
		return Response{OK: false, Error: "failed to list invites"}
	}
	defer rows.Close()

	items := make([]AdminInviteItem, 0)
	for rows.Next() {
		var item AdminInviteItem
		var roleStr string
		var teacherID, lecternID, studentID *int32
		var teacherName, lecternName, studentName, registeredAs string

		err := rows.Scan(
			&item.InviteID,
			&item.InviteCode,
			&roleStr,
			&teacherID,
			&teacherName,
			&lecternID,
			&lecternName,
			&studentID,
			&studentName,
			&item.UsedAt,
			&item.CreatedAt,
			&registeredAs,
		)
		if err != nil {
			continue
		}

		item.Role = roleStr
		item.TeacherID = teacherID
		item.TeacherName = teacherName
		item.LecternID = lecternID
		item.LecternName = lecternName
		item.StudentID = studentID
		item.StudentName = studentName
		item.RegisteredAs = registeredAs

		items = append(items, item)
	}

	return Response{
		OK:     true,
		Result: items,
	}
}

func (s *Service) admin_revoke_invite(token string, data AdminRevokeInviteData) Response {
	if _, auth := s.require_admin(token); !auth.OK {
		return auth
	}
	if data.InviteID <= 0 {
		return Response{OK: false, Error: "invite_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	cmd, err := s.store.Pool().Exec(
		ctx,
		`DELETE FROM registration_invites WHERE invite_id = $1 AND used_at IS NULL`,
		data.InviteID,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to revoke invite"}
	}
	if cmd.RowsAffected() == 0 {
		return Response{OK: false, Error: "invite code not found or already used"}
	}

	return Response{OK: true, Result: map[string]any{"invite_id": data.InviteID, "status": "revoked"}}
}

