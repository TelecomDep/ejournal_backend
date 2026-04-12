package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeacherRepository struct {
	pool *pgxpool.Pool
}

func NewTeacherRepository(pool *pgxpool.Pool) *TeacherRepository {
	return &TeacherRepository{pool: pool}
}

func (r *TeacherRepository) Create(ctx context.Context, teacher Teacher) (Teacher, error) {
	teacher.Name = strings.TrimSpace(teacher.Name)
	teacher.JobTitle = strings.TrimSpace(teacher.JobTitle)
	if teacher.Name == "" {
		return Teacher{}, fmt.Errorf("teacher name is required")
	}

	var out Teacher
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO teachers (name, lectern_id, job_title)
		 VALUES ($1, $2, $3)
		 RETURNING teacher_id, name, lectern_id, job_title`,
		teacher.Name,
		teacher.LecternID,
		teacher.JobTitle,
	).Scan(&out.ID, &out.Name, &out.LecternID, &out.JobTitle)
	if err != nil {
		return Teacher{}, fmt.Errorf("insert teacher: %w", err)
	}

	return out, nil
}

func (r *TeacherRepository) GetByID(ctx context.Context, id int32) (Teacher, bool, error) {
	var out Teacher
	err := r.pool.QueryRow(
		ctx,
		`SELECT teacher_id, name, lectern_id, job_title
		 FROM teachers
		 WHERE teacher_id = $1`,
		id,
	).Scan(&out.ID, &out.Name, &out.LecternID, &out.JobTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, false, nil
	}
	if err != nil {
		return Teacher{}, false, fmt.Errorf("get teacher by id: %w", err)
	}

	return out, true, nil
}

func (r *TeacherRepository) Update(ctx context.Context, teacher Teacher) (Teacher, bool, error) {
	teacher.Name = strings.TrimSpace(teacher.Name)
	teacher.JobTitle = strings.TrimSpace(teacher.JobTitle)
	if teacher.Name == "" {
		return Teacher{}, false, fmt.Errorf("teacher name is required")
	}

	var out Teacher
	err := r.pool.QueryRow(
		ctx,
		`UPDATE teachers
		 SET name = $2,
		     lectern_id = $3,
		     job_title = $4
		 WHERE teacher_id = $1
		 RETURNING teacher_id, name, lectern_id, job_title`,
		teacher.ID,
		teacher.Name,
		teacher.LecternID,
		teacher.JobTitle,
	).Scan(&out.ID, &out.Name, &out.LecternID, &out.JobTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, false, nil
	}
	if err != nil {
		return Teacher{}, false, fmt.Errorf("update teacher: %w", err)
	}

	return out, true, nil
}

func (r *TeacherRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM teachers WHERE teacher_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete teacher: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *TeacherRepository) List(ctx context.Context) ([]Teacher, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT teacher_id, name, lectern_id, job_title
		 FROM teachers
		 ORDER BY teacher_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list teachers: %w", err)
	}
	defer rows.Close()

	result := make([]Teacher, 0)
	for rows.Next() {
		var item Teacher
		if err := rows.Scan(&item.ID, &item.Name, &item.LecternID, &item.JobTitle); err != nil {
			return nil, fmt.Errorf("scan teacher: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teachers rows: %w", err)
	}

	return result, nil
}

func (r *TeacherRepository) ListByLecternID(ctx context.Context, lecternID int32) ([]Teacher, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT teacher_id, name, lectern_id, job_title
		 FROM teachers
		 WHERE lectern_id = $1
		 ORDER BY teacher_id`,
		lecternID,
	)
	if err != nil {
		return nil, fmt.Errorf("list teachers by lectern id: %w", err)
	}
	defer rows.Close()

	result := make([]Teacher, 0)
	for rows.Next() {
		var item Teacher
		if err := rows.Scan(&item.ID, &item.Name, &item.LecternID, &item.JobTitle); err != nil {
			return nil, fmt.Errorf("scan teacher: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teachers rows: %w", err)
	}

	return result, nil
}
