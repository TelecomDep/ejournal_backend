package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserLoginTaken = errors.New("user login already exists")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
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

	return out, true, nil
}

func (r *UserRepository) CreateResetToken(ctx context.Context, userID int32, token string, expiresAt time.Time) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is required")
	}

	_, _ = r.pool.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1`, userID)

	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID,
		token,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	return nil
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
		token,
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

	_, err := r.pool.Exec(ctx, `DELETE FROM password_reset_tokens WHERE token = $1`, token)
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

	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, userID)
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
		 WHERE ($1 = '' OR role::text = $1)
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
		`SELECT id, login, role::text, email, status,
		        is_2fa_enabled, created_at, updated_at
		 FROM users
		 WHERE ($1 = '' OR role::text = $1)
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
		`SELECT id, login, role::text, email, status,
		        is_2fa_enabled, created_at, updated_at
		 FROM users
		 WHERE id = $1`,
		userID,
	).Scan(
		&item.ID,
		&item.Login,
		&item.Role,
		&item.Email,
		&item.Status,
		&item.TwoFaEnabled,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminUser{}, false, nil
	}
	if err != nil {
		return AdminUser{}, false, fmt.Errorf("get admin user: %w", err)
	}

	return item, true, nil
}
