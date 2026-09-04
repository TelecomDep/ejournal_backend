package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserLoginTaken = errors.New("user login already exists")

func hashResetToken(token string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(digest[:])
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) rolesForUser(ctx context.Context, userID int32, primaryRole string) ([]string, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT role::text
		 FROM user_roles
		 WHERE user_id = $1
		 ORDER BY is_primary DESC, role::text`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("load user roles: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0, 1)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user roles: %w", err)
	}
	if len(roles) == 0 && primaryRole != "" {
		roles = append(roles, primaryRole)
	}
	return roles, nil
}

func (r *UserRepository) Create(ctx context.Context, login, passwordHash, role string) (User, error) {
	login = strings.TrimSpace(login)
	passwordHash = strings.TrimSpace(passwordHash)
	role = strings.ToLower(strings.TrimSpace(role))

	if login == "" {
		return User{}, fmt.Errorf("user login is required")
	}
	if passwordHash == "" {
		return User{}, fmt.Errorf("user password hash is required")
	}
	if role == "" {
		return User{}, fmt.Errorf("user role is required")
	}

	var out User
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO users (login, password_hash, role)
		 VALUES ($1, $2, $3)
		 RETURNING id, login, password_hash, role, email, status,
		           is_2fa_enabled, created_at, updated_at`,
		login,
		passwordHash,
		role,
	).Scan(
		&out.ID,
		&out.Login,
		&out.PasswordHash,
		&out.Role,
		&out.Email,
		&out.Status,
		&out.TwoFaEnabled,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrUserLoginTaken
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	out.Roles, err = r.rolesForUser(ctx, out.ID, out.Role)
	if err != nil {
		return User{}, err
	}

	return out, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (User, bool, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return User{}, false, fmt.Errorf("user login is required")
	}

	var out User
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, login, password_hash, role, email, status,
		        is_2fa_enabled, created_at, updated_at
		 FROM users
		 WHERE login = $1`,
		login,
	).Scan(
		&out.ID,
		&out.Login,
		&out.PasswordHash,
		&out.Role,
		&out.Email,
		&out.Status,
		&out.TwoFaEnabled,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, fmt.Errorf("get user by login: %w", err)
	}
	out.Roles, err = r.rolesForUser(ctx, out.ID, out.Role)
	if err != nil {
		return User{}, false, err
	}

	return out, true, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int32) (User, bool, error) {
	if id <= 0 {
		return User{}, false, fmt.Errorf("user id is required")
	}

	var out User
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, login, password_hash, role, email, status,
		        is_2fa_enabled, created_at, updated_at
		 FROM users
		 WHERE id = $1`,
		id,
	).Scan(
		&out.ID,
		&out.Login,
		&out.PasswordHash,
		&out.Role,
		&out.Email,
		&out.Status,
		&out.TwoFaEnabled,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, fmt.Errorf("get user by id: %w", err)
	}
	out.Roles, err = r.rolesForUser(ctx, out.ID, out.Role)
	if err != nil {
		return User{}, false, err
	}

	return out, true, nil
}

func (r *UserRepository) DeleteByID(ctx context.Context, id int32) error {
	if id <= 0 {
		return fmt.Errorf("user id is required")
	}

	if _, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete user by id: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateEmail(ctx context.Context, id int32, email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	_, err := r.pool.Exec(ctx, `UPDATE users SET email = $1 WHERE id = $2`, email, id)
	if err != nil {
		return fmt.Errorf("update user email: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (User, bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return User{}, false, fmt.Errorf("email is required")
	}

	var out User
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, login, password_hash, role, email, created_at
		 FROM users
		 WHERE email = $1`,
		email,
	).Scan(&out.ID, &out.Login, &out.PasswordHash, &out.Role, &out.Email, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, fmt.Errorf("get user by email: %w", err)
	}
	out.Roles, err = r.rolesForUser(ctx, out.ID, out.Role)
	if err != nil {
		return User{}, false, err
	}

	return out, true, nil
}

func (r *UserRepository) CreateResetToken(ctx context.Context, userID int32, token string, expiresAt time.Time) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin reset token transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("replace reset token: %w", err)
	}
	_, err = tx.Exec(
		ctx,
		`INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID,
		hashResetToken(token),
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *UserRepository) ResetPasswordWithToken(ctx context.Context, token, passwordHash string) (bool, error) {
	token = strings.TrimSpace(token)
	passwordHash = strings.TrimSpace(passwordHash)
	if token == "" || passwordHash == "" {
		return false, fmt.Errorf("token and password hash are required")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID int32
	err = tx.QueryRow(ctx, `
		SELECT user_id FROM password_reset_tokens
		WHERE token = $1 AND expires_at > NOW()
		FOR UPDATE
	`, hashResetToken(token)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $1, token_version = token_version + 1 WHERE id = $2
	`, passwordHash, userID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM password_reset_tokens WHERE token = $1`, hashResetToken(token)); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (r *UserRepository) GetResetToken(ctx context.Context, token string) (int32, time.Time, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, time.Time{}, fmt.Errorf("token is required")
	}

	var userID int32
	var expiresAt time.Time
	err := r.pool.QueryRow(
		ctx,
		`SELECT user_id, expires_at FROM password_reset_tokens WHERE token = $1`,
		hashResetToken(token),
	).Scan(&userID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, time.Time{}, errors.New("token not found")
	}
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("get reset token: %w", err)
	}

	return userID, expiresAt, nil
}

func (r *UserRepository) DeleteResetToken(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is required")
	}

	_, err := r.pool.Exec(ctx, `DELETE FROM password_reset_tokens WHERE token = $1`, hashResetToken(token))
	if err != nil {
		return fmt.Errorf("delete reset token: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID int32, passwordHash string) error {
	passwordHash = strings.TrimSpace(passwordHash)
	if passwordHash == "" {
		return fmt.Errorf("password hash is required")
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, token_version = token_version + 1
		WHERE id = $2
	`, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	return nil
}

func (r *UserRepository) AdminList(ctx context.Context, filter AdminUserFilter) ([]AdminUser, int32, error) {
	offset := (filter.Page - 1) * filter.PageSize
	search := strings.TrimSpace(filter.Search)

	var total int32
	err := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM users
		 WHERE ($1 = '' OR EXISTS (
		           SELECT 1 FROM user_roles ur
		           WHERE ur.user_id = users.id AND ur.role::text = $1
		       ))
		   AND ($2 = '' OR status = $2)
		   AND (
		       $3 = ''
		       OR login ILIKE '%' || $3 || '%'
		       OR COALESCE(email, '') ILIKE '%' || $3 || '%'
		   )`,
		filter.Role,
		filter.Status,
		search,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin users: %w", err)
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT id, login, role::text,
		        ARRAY(SELECT ur.role::text
		              FROM user_roles ur
		              WHERE ur.user_id = users.id
		              ORDER BY ur.is_primary DESC, ur.role::text),
		        email, status, is_2fa_enabled, created_at, updated_at
		 FROM users
		 WHERE ($1 = '' OR EXISTS (
		           SELECT 1 FROM user_roles ur
		           WHERE ur.user_id = users.id AND ur.role::text = $1
		       ))
		   AND ($2 = '' OR status = $2)
		   AND (
		       $3 = ''
		       OR login ILIKE '%' || $3 || '%'
		       OR COALESCE(email, '') ILIKE '%' || $3 || '%'
		   )
		 ORDER BY id DESC
		 LIMIT $4 OFFSET $5`,
		filter.Role,
		filter.Status,
		search,
		filter.PageSize,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	items := make([]AdminUser, 0)
	for rows.Next() {
		var item AdminUser
		if err := rows.Scan(
			&item.ID,
			&item.Login,
			&item.Role,
			&item.Roles,
			&item.Email,
			&item.Status,
			&item.TwoFaEnabled,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan admin user: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate admin users: %w", err)
	}

	return items, total, nil
}

func (r *UserRepository) AdminGetByID(ctx context.Context, userID int32) (AdminUser, bool, error) {
	if userID <= 0 {
		return AdminUser{}, false, fmt.Errorf("user id is required")
	}

	var item AdminUser
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, login, role::text,
		        ARRAY(SELECT ur.role::text
		              FROM user_roles ur
		              WHERE ur.user_id = users.id
		              ORDER BY ur.is_primary DESC, ur.role::text),
		        email, status, is_2fa_enabled, created_at, updated_at,
		        (SELECT s.student_id FROM students s WHERE s.user_id = users.id LIMIT 1),
		        (SELECT t.teacher_id FROM teachers t WHERE t.user_id = users.id LIMIT 1),
		        (SELECT os.lectern_id FROM org_scopes os
		         WHERE os.user_id = users.id AND os.lectern_id IS NOT NULL LIMIT 1),
		        (SELECT os.faculty_id FROM org_scopes os
		         WHERE os.user_id = users.id AND os.faculty_id IS NOT NULL LIMIT 1)
		 FROM users
		 WHERE id = $1`,
		userID,
	).Scan(
		&item.ID,
		&item.Login,
		&item.Role,
		&item.Roles,
		&item.Email,
		&item.Status,
		&item.TwoFaEnabled,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.StudentID,
		&item.TeacherID,
		&item.LecternID,
		&item.FacultyID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminUser{}, false, nil
	}
	if err != nil {
		return AdminUser{}, false, fmt.Errorf("get admin user: %w", err)
	}

	return item, true, nil
}

type UserAvatar struct {
	UserID      int32
	ImageData   []byte
	ContentType string
	Hash        string
	UpdatedAt   time.Time
}

func (r *UserRepository) SaveAvatar(ctx context.Context, userID int32, imageData []byte, contentType, hash string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_avatars (user_id, image_data, content_type, hash, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			image_data = EXCLUDED.image_data,
			content_type = EXCLUDED.content_type,
			hash = EXCLUDED.hash,
			updated_at = NOW()
	`, userID, imageData, contentType, hash)
	if err != nil {
		return fmt.Errorf("save user avatar: %w", err)
	}
	return nil
}

func (r *UserRepository) GetAvatar(ctx context.Context, userID int32) (UserAvatar, bool, error) {
	var avatar UserAvatar
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, image_data, content_type, hash, updated_at
		FROM user_avatars
		WHERE user_id = $1
	`, userID).Scan(&avatar.UserID, &avatar.ImageData, &avatar.ContentType, &avatar.Hash, &avatar.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserAvatar{}, false, nil
	}
	if err != nil {
		return UserAvatar{}, false, fmt.Errorf("get user avatar: %w", err)
	}
	return avatar, true, nil
}

type Attachment struct {
	ID          int64     `json:"id"`
	OwnerID     int32     `json:"owner_id"`
	Filename    string    `json:"filename"`
	FileSize    int64     `json:"file_size"`
	MimeType    string    `json:"mime_type"`
	StorageType string    `json:"storage_type"`
	Data        []byte    `json:"-"`
	StoragePath string    `json:"storage_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *UserRepository) SaveAttachment(ctx context.Context, att Attachment) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO attachments (owner_id, filename, file_size, mime_type, storage_type, data, storage_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id
	`, att.OwnerID, att.Filename, att.FileSize, att.MimeType, att.StorageType, att.Data, att.StoragePath).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("save attachment: %w", err)
	}
	return id, nil
}

func (r *UserRepository) AttachmentBytesByOwner(ctx context.Context, ownerID int32) (int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(file_size), 0) FROM attachments WHERE owner_id = $1`, ownerID).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum attachment bytes: %w", err)
	}
	return total, nil
}

func (r *UserRepository) GetAttachmentByIDForOwner(ctx context.Context, id int64, ownerID int32) (Attachment, bool, error) {
	var att Attachment
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, filename, file_size, mime_type, storage_type, data, COALESCE(storage_path, ''), created_at
		FROM attachments
		WHERE id = $1 AND owner_id = $2
	`, id, ownerID).Scan(&att.ID, &att.OwnerID, &att.Filename, &att.FileSize, &att.MimeType, &att.StorageType, &att.Data, &att.StoragePath, &att.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, false, nil
	}
	if err != nil {
		return Attachment{}, false, fmt.Errorf("get attachment: %w", err)
	}
	return att, true, nil
}

func (r *UserRepository) DeleteEmail(ctx context.Context, userID int32) error {
	if userID <= 0 {
		return fmt.Errorf("invalid userID")
	}

	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email = NULL WHERE id = $1`, userID)

	if err != nil {
		return fmt.Errorf("delete email: %w", err)
	}

	return nil
}
