package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRepository struct {
	pool *pgxpool.Pool
}

func NewGroupRepository(pool *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{pool: pool}
}

func (r *GroupRepository) Create(ctx context.Context, group Group) (Group, error) {
	group.GroupName = strings.TrimSpace(group.GroupName)
	if group.GroupName == "" {
		return Group{}, fmt.Errorf("group name is required")
	}

	var out Group
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO groups (group_name, lectern_id)
		 VALUES ($1, $2)
		 RETURNING group_id, group_name, lectern_id`,
		group.GroupName,
		group.LecternID,
	).Scan(&out.ID, &out.GroupName, &out.LecternID)
	if err != nil {
		return Group{}, fmt.Errorf("insert group: %w", err)
	}

	return out, nil
}

func (r *GroupRepository) GetByID(ctx context.Context, id int32) (Group, bool, error) {
	var out Group
	err := r.pool.QueryRow(
		ctx,
		`SELECT group_id, group_name, lectern_id
		 FROM groups
		 WHERE group_id = $1`,
		id,
	).Scan(&out.ID, &out.GroupName, &out.LecternID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Group{}, false, nil
	}
	if err != nil {
		return Group{}, false, fmt.Errorf("get group by id: %w", err)
	}

	return out, true, nil
}

func (r *GroupRepository) Update(ctx context.Context, group Group) (Group, bool, error) {
	group.GroupName = strings.TrimSpace(group.GroupName)
	if group.GroupName == "" {
		return Group{}, false, fmt.Errorf("group name is required")
	}

	var out Group
	err := r.pool.QueryRow(
		ctx,
		`UPDATE groups
		 SET group_name = $2,
		     lectern_id = $3
		 WHERE group_id = $1
		 RETURNING group_id, group_name, lectern_id`,
		group.ID,
		group.GroupName,
		group.LecternID,
	).Scan(&out.ID, &out.GroupName, &out.LecternID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Group{}, false, nil
	}
	if err != nil {
		return Group{}, false, fmt.Errorf("update group: %w", err)
	}

	return out, true, nil
}

func (r *GroupRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM groups WHERE group_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete group: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *GroupRepository) List(ctx context.Context) ([]Group, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT group_id, group_name, lectern_id
		 FROM groups
		 ORDER BY group_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	result := make([]Group, 0)
	for rows.Next() {
		var item Group
		if err := rows.Scan(&item.ID, &item.GroupName, &item.LecternID); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups rows: %w", err)
	}

	return result, nil
}

func (r *GroupRepository) ListByLecternID(ctx context.Context, lecternID int32) ([]Group, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT group_id, group_name, lectern_id
		 FROM groups
		 WHERE lectern_id = $1
		 ORDER BY group_id`,
		lecternID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups by lectern id: %w", err)
	}
	defer rows.Close()

	result := make([]Group, 0)
	for rows.Next() {
		var item Group
		if err := rows.Scan(&item.ID, &item.GroupName, &item.LecternID); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups rows: %w", err)
	}

	return result, nil
}
