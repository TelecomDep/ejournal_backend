package app

import (
	"context"
	"errors"
	"strings"

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
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleStudent, RoleTeacher, RoleHead, RoleDean, RoleAdmin:
		return true
	default:
		return false
	}
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
	if data.Email != "" && !strings.Contains(data.Email, "@") {
		return errors.New("invalid email")
	}

	switch data.Role {
	case RoleStudent, RoleTeacher:
		if data.FullName == "" {
			return errors.New("full_name is required for student or teacher")
		}
	case RoleHead:
		if data.LecternID <= 0 {
			return errors.New("lectern_id is required for head")
		}
	case RoleDean:
		if data.FacultyID <= 0 {
			return errors.New("faculty_id is required for dean")
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
	case RoleHead:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO org_scopes (user_id, lectern_id)
			 VALUES ($1, $2)`,
			user_id,
			data.LecternID,
		)
		return err
	case RoleDean:
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
	if _, auth := s.require_admin(token); !auth.OK {
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
	case RoleHead:
		if data.LecternID <= 0 {
			return errors.New("lectern_id is required for head")
		}
	case RoleDean:
		if data.FacultyID <= 0 {
			return errors.New("faculty_id is required for dean")
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
	case RoleHead:
		_, err := tx.Exec(
			ctx,
			`INSERT INTO org_scopes (user_id, lectern_id)
			 VALUES ($1, $2)`,
			user_id,
			data.LecternID,
		)
		return err
	case RoleDean:
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
		email = strings.TrimSpace(*data.Email)
		if email != "" && !strings.Contains(email, "@") {
			return Response{OK: false, Error: "invalid email"}
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
		(role == RoleHead && data.LecternID > 0) ||
		(role == RoleDean && data.FacultyID > 0)
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
		     status = $6
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
		`UPDATE users SET status = 'archived' WHERE id = $1`,
		user_id,
	); err != nil {
		return Response{OK: false, Error: "failed to archive user"}
	}
	if err := tx.Commit(ctx); err != nil {
		return Response{OK: false, Error: "failed to archive user"}
	}

	return Response{OK: true, Result: map[string]any{
		"user_id": user_id,
		"status":  "archived",
	}}
}
