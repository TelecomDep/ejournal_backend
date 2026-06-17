package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubjectControlRepository struct {
	pool *pgxpool.Pool
}

func NewSubjectControlRepository(pool *pgxpool.Pool) *SubjectControlRepository {
	return &SubjectControlRepository{pool: pool}
}

func (r *SubjectControlRepository) Create(ctx context.Context, control SubjectControl) (SubjectControl, error) {
	var out SubjectControl
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO subject_controls (subject_id, type_id, semester_num)
		 VALUES ($1, $2, $3)
		 RETURNING control_id, subject_id, type_id, semester_num`,
		control.SubjectID,
		control.TypeID,
		control.SemesterNum,
	).Scan(&out.ID, &out.SubjectID, &out.TypeID, &out.SemesterNum)
	if err != nil {
		return SubjectControl{}, fmt.Errorf("insert subject control: %w", err)
	}

	return out, nil
}

func (r *SubjectControlRepository) GetByID(ctx context.Context, id int32) (SubjectControl, bool, error) {
	var out SubjectControl
	err := r.pool.QueryRow(
		ctx,
		`SELECT control_id, subject_id, type_id, semester_num
		 FROM subject_controls
		 WHERE control_id = $1`,
		id,
	).Scan(&out.ID, &out.SubjectID, &out.TypeID, &out.SemesterNum)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubjectControl{}, false, nil
	}
	if err != nil {
		return SubjectControl{}, false, fmt.Errorf("get subject control by id: %w", err)
	}

	return out, true, nil
}

func (r *SubjectControlRepository) ListBySubjectID(ctx context.Context, subjectID int32) ([]SubjectControl, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT control_id, subject_id, type_id, semester_num
		 FROM subject_controls
		 WHERE subject_id = $1
		 ORDER BY semester_num, control_id`,
		subjectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list subject controls by subject id: %w", err)
	}
	defer rows.Close()

	result := make([]SubjectControl, 0)
	for rows.Next() {
		var item SubjectControl
		if err := rows.Scan(&item.ID, &item.SubjectID, &item.TypeID, &item.SemesterNum); err != nil {
			return nil, fmt.Errorf("scan subject control: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subject controls rows: %w", err)
	}

	return result, nil
}

func (r *SubjectControlRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM subject_controls WHERE control_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete subject control: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}
