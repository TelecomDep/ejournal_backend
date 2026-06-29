package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ControlTypeRepository struct {
	pool *pgxpool.Pool
}

func NewControlTypeRepository(pool *pgxpool.Pool) *ControlTypeRepository {
	return &ControlTypeRepository{pool: pool}
}

func (r *ControlTypeRepository) Create(ctx context.Context, typeName string) (ControlType, error) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return ControlType{}, fmt.Errorf("control type name is required")
	}

	var out ControlType
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO control_types (type_name)
		 VALUES ($1)
		 ON CONFLICT (type_name) DO UPDATE SET type_name = EXCLUDED.type_name
		 RETURNING type_id, type_name`,
		typeName,
	).Scan(&out.ID, &out.Name)
	if err != nil {
		return ControlType{}, fmt.Errorf("upsert control type: %w", err)
	}

	return out, nil
}

func (r *ControlTypeRepository) GetByID(ctx context.Context, id int32) (ControlType, bool, error) {
	var out ControlType
	err := r.pool.QueryRow(ctx, `SELECT type_id, type_name FROM control_types WHERE type_id = $1`, id).Scan(&out.ID, &out.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return ControlType{}, false, nil
	}
	if err != nil {
		return ControlType{}, false, fmt.Errorf("get control type by id: %w", err)
	}

	return out, true, nil
}

func (r *ControlTypeRepository) Update(ctx context.Context, id int32, typeName string) (ControlType, bool, error) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return ControlType{}, false, fmt.Errorf("control type name is required")
	}

	var out ControlType
	err := r.pool.QueryRow(
		ctx,
		`UPDATE control_types
		 SET type_name = $2
		 WHERE type_id = $1
		 RETURNING type_id, type_name`,
		id,
		typeName,
	).Scan(&out.ID, &out.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return ControlType{}, false, nil
	}
	if err != nil {
		return ControlType{}, false, fmt.Errorf("update control type: %w", err)
	}

	return out, true, nil
}

func (r *ControlTypeRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM control_types WHERE type_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete control type: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *ControlTypeRepository) List(ctx context.Context) ([]ControlType, error) {
	rows, err := r.pool.Query(ctx, `SELECT type_id, type_name FROM control_types ORDER BY type_id`)
	if err != nil {
		return nil, fmt.Errorf("list control types: %w", err)
	}
	defer rows.Close()

	result := make([]ControlType, 0)
	for rows.Next() {
		var row ControlType
		if err := rows.Scan(&row.ID, &row.Name); err != nil {
			return nil, fmt.Errorf("scan control type: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate control types rows: %w", err)
	}

	return result, nil
}
