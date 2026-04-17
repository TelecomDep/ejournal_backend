package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		 RETURNING id, login, password_hash, role, created_at`,
		login,
		passwordHash,
		role,
	).Scan(&out.ID, &out.Login, &out.PasswordHash, &out.Role, &out.CreatedAt)
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
		`SELECT id, login, password_hash, role, created_at
		 FROM users
		 WHERE login = $1`,
		login,
	).Scan(&out.ID, &out.Login, &out.PasswordHash, &out.Role, &out.CreatedAt)
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
		`SELECT id, login, password_hash, role, created_at
		 FROM users
		 WHERE id = $1`,
		id,
	).Scan(&out.ID, &out.Login, &out.PasswordHash, &out.Role, &out.CreatedAt)
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
