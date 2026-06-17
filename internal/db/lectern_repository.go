package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LecternRepository struct {
	pool *pgxpool.Pool
}

func NewLecternRepository(pool *pgxpool.Pool) *LecternRepository {
	return &LecternRepository{pool: pool}
}

func (r *LecternRepository) Create(ctx context.Context, code, name string) (Lectern, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Lectern{}, fmt.Errorf("lectern name is required")
	}

	var out Lectern
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO lecterns (code, name) VALUES ($1, $2) RETURNING lectern_id, code, name`,
		strings.TrimSpace(code),
		name,
	).Scan(&out.ID, &out.Code, &out.Name)
	if err != nil {
		return Lectern{}, fmt.Errorf("insert lectern: %w", err)
	}

	return out, nil
}

func (r *LecternRepository) GetByID(ctx context.Context, id int32) (Lectern, bool, error) {
	var out Lectern
	err := r.pool.QueryRow(ctx, `SELECT lectern_id, code, name FROM lecterns WHERE lectern_id = $1`, id).Scan(&out.ID, &out.Code, &out.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lectern{}, false, nil
	}
	if err != nil {
		return Lectern{}, false, fmt.Errorf("get lectern by id: %w", err)
	}

	return out, true, nil
}

func (r *LecternRepository) Update(ctx context.Context, id int32, code, name string) (Lectern, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Lectern{}, false, fmt.Errorf("lectern name is required")
	}

	var out Lectern
	err := r.pool.QueryRow(
		ctx,
		`UPDATE lecterns
		 SET code = $2, name = $3
		 WHERE lectern_id = $1
		 RETURNING lectern_id, code, name`,
		id,
		strings.TrimSpace(code),
		name,
	).Scan(&out.ID, &out.Code, &out.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lectern{}, false, nil
	}
	if err != nil {
		return Lectern{}, false, fmt.Errorf("update lectern: %w", err)
	}

	return out, true, nil
}

func (r *LecternRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM lecterns WHERE lectern_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete lectern: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *LecternRepository) List(ctx context.Context) ([]Lectern, error) {
	rows, err := r.pool.Query(ctx, `SELECT lectern_id, code, name FROM lecterns ORDER BY lectern_id`)
	if err != nil {
		return nil, fmt.Errorf("list lecterns: %w", err)
	}
	defer rows.Close()

	result := make([]Lectern, 0)
	for rows.Next() {
		var row Lectern
		if err := rows.Scan(&row.ID, &row.Code, &row.Name); err != nil {
			return nil, fmt.Errorf("scan lectern: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lecterns rows: %w", err)
	}

	return result, nil
}
