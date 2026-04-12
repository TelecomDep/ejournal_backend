package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SemesterLoadRepository struct {
	pool *pgxpool.Pool
}

func NewSemesterLoadRepository(pool *pgxpool.Pool) *SemesterLoadRepository {
	return &SemesterLoadRepository{pool: pool}
}

func (r *SemesterLoadRepository) Create(ctx context.Context, load SemesterLoad) (SemesterLoad, error) {
	var out SemesterLoad
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO semester_load (subject_id, semester_num, zet_value)
		 VALUES ($1, $2, $3)
		 RETURNING load_id, subject_id, semester_num, zet_value`,
		load.SubjectID,
		load.SemesterNum,
		load.ZetValue,
	).Scan(&out.ID, &out.SubjectID, &out.SemesterNum, &out.ZetValue)
	if err != nil {
		return SemesterLoad{}, fmt.Errorf("insert semester load: %w", err)
	}

	return out, nil
}

func (r *SemesterLoadRepository) GetByID(ctx context.Context, id int32) (SemesterLoad, bool, error) {
	var out SemesterLoad
	err := r.pool.QueryRow(
		ctx,
		`SELECT load_id, subject_id, semester_num, zet_value
		 FROM semester_load
		 WHERE load_id = $1`,
		id,
	).Scan(&out.ID, &out.SubjectID, &out.SemesterNum, &out.ZetValue)
	if errors.Is(err, pgx.ErrNoRows) {
		return SemesterLoad{}, false, nil
	}
	if err != nil {
		return SemesterLoad{}, false, fmt.Errorf("get semester load by id: %w", err)
	}

	return out, true, nil
}

func (r *SemesterLoadRepository) ListBySubjectID(ctx context.Context, subjectID int32) ([]SemesterLoad, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT load_id, subject_id, semester_num, zet_value
		 FROM semester_load
		 WHERE subject_id = $1
		 ORDER BY semester_num, load_id`,
		subjectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list semester load by subject id: %w", err)
	}
	defer rows.Close()

	result := make([]SemesterLoad, 0)
	for rows.Next() {
		var item SemesterLoad
		if err := rows.Scan(&item.ID, &item.SubjectID, &item.SemesterNum, &item.ZetValue); err != nil {
			return nil, fmt.Errorf("scan semester load: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semester load rows: %w", err)
	}

	return result, nil
}

func (r *SemesterLoadRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM semester_load WHERE load_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete semester load: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}
