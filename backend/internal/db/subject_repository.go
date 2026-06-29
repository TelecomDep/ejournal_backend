package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubjectRepository struct {
	pool *pgxpool.Pool
}

func NewSubjectRepository(pool *pgxpool.Pool) *SubjectRepository {
	return &SubjectRepository{pool: pool}
}

func (r *SubjectRepository) Create(ctx context.Context, subject Subject) (Subject, error) {
	subject.Name = strings.TrimSpace(subject.Name)
	subject.SubjectIndex = strings.TrimSpace(subject.SubjectIndex)
	if subject.Name == "" {
		return Subject{}, fmt.Errorf("subject name is required")
	}

	var out Subject
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO subjects (subject_index, name, in_plan, lectern_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING subject_id, subject_index, name, in_plan, lectern_id`,
		subject.SubjectIndex,
		subject.Name,
		subject.InPlan,
		subject.LecternID,
	).Scan(&out.ID, &out.SubjectIndex, &out.Name, &out.InPlan, &out.LecternID)
	if err != nil {
		return Subject{}, fmt.Errorf("insert subject: %w", err)
	}

	return out, nil
}

func (r *SubjectRepository) GetByID(ctx context.Context, id int32) (Subject, bool, error) {
	var out Subject
	err := r.pool.QueryRow(
		ctx,
		`SELECT subject_id, subject_index, name, in_plan, lectern_id
		 FROM subjects
		 WHERE subject_id = $1`,
		id,
	).Scan(&out.ID, &out.SubjectIndex, &out.Name, &out.InPlan, &out.LecternID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subject{}, false, nil
	}
	if err != nil {
		return Subject{}, false, fmt.Errorf("get subject by id: %w", err)
	}

	return out, true, nil
}

func (r *SubjectRepository) Update(ctx context.Context, subject Subject) (Subject, bool, error) {
	subject.Name = strings.TrimSpace(subject.Name)
	subject.SubjectIndex = strings.TrimSpace(subject.SubjectIndex)
	if subject.Name == "" {
		return Subject{}, false, fmt.Errorf("subject name is required")
	}

	var out Subject
	err := r.pool.QueryRow(
		ctx,
		`UPDATE subjects
		 SET subject_index = $2,
		     name = $3,
		     in_plan = $4,
		     lectern_id = $5
		 WHERE subject_id = $1
		 RETURNING subject_id, subject_index, name, in_plan, lectern_id`,
		subject.ID,
		subject.SubjectIndex,
		subject.Name,
		subject.InPlan,
		subject.LecternID,
	).Scan(&out.ID, &out.SubjectIndex, &out.Name, &out.InPlan, &out.LecternID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subject{}, false, nil
	}
	if err != nil {
		return Subject{}, false, fmt.Errorf("update subject: %w", err)
	}

	return out, true, nil
}

func (r *SubjectRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM subjects WHERE subject_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete subject: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *SubjectRepository) List(ctx context.Context) ([]Subject, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT subject_id, subject_index, name, in_plan, lectern_id
		 FROM subjects
		 ORDER BY subject_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()

	result := make([]Subject, 0)
	for rows.Next() {
		var row Subject
		if err := rows.Scan(&row.ID, &row.SubjectIndex, &row.Name, &row.InPlan, &row.LecternID); err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subjects rows: %w", err)
	}

	return result, nil
}

func (r *SubjectRepository) ListByLecternID(ctx context.Context, lecternID int32) ([]Subject, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT subject_id, subject_index, name, in_plan, lectern_id
		 FROM subjects
		 WHERE lectern_id = $1
		 ORDER BY subject_id`,
		lecternID,
	)
	if err != nil {
		return nil, fmt.Errorf("list subjects by lectern id: %w", err)
	}
	defer rows.Close()

	result := make([]Subject, 0)
	for rows.Next() {
		var row Subject
		if err := rows.Scan(&row.ID, &row.SubjectIndex, &row.Name, &row.InPlan, &row.LecternID); err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subjects rows: %w", err)
	}

	return result, nil
}
