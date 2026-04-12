package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{pool: pool}
}

func (r *StudentRepository) Create(ctx context.Context, student Student) (Student, error) {
	student.StudentName = strings.TrimSpace(student.StudentName)
	if student.StudentName == "" {
		return Student{}, fmt.Errorf("student name is required")
	}

	var out Student
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO students (student_name, group_id)
		 VALUES ($1, $2)
		 RETURNING student_id, student_name, group_id`,
		student.StudentName,
		student.GroupID,
	).Scan(&out.ID, &out.StudentName, &out.GroupID)
	if err != nil {
		return Student{}, fmt.Errorf("insert student: %w", err)
	}

	return out, nil
}

func (r *StudentRepository) GetByID(ctx context.Context, id int32) (Student, bool, error) {
	var out Student
	err := r.pool.QueryRow(
		ctx,
		`SELECT student_id, student_name, group_id
		 FROM students
		 WHERE student_id = $1`,
		id,
	).Scan(&out.ID, &out.StudentName, &out.GroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, false, nil
	}
	if err != nil {
		return Student{}, false, fmt.Errorf("get student by id: %w", err)
	}

	return out, true, nil
}

func (r *StudentRepository) Update(ctx context.Context, student Student) (Student, bool, error) {
	student.StudentName = strings.TrimSpace(student.StudentName)
	if student.StudentName == "" {
		return Student{}, false, fmt.Errorf("student name is required")
	}

	var out Student
	err := r.pool.QueryRow(
		ctx,
		`UPDATE students
		 SET student_name = $2,
		     group_id = $3
		 WHERE student_id = $1
		 RETURNING student_id, student_name, group_id`,
		student.ID,
		student.StudentName,
		student.GroupID,
	).Scan(&out.ID, &out.StudentName, &out.GroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, false, nil
	}
	if err != nil {
		return Student{}, false, fmt.Errorf("update student: %w", err)
	}

	return out, true, nil
}

func (r *StudentRepository) Delete(ctx context.Context, id int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM students WHERE student_id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete student: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *StudentRepository) List(ctx context.Context) ([]Student, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT student_id, student_name, group_id
		 FROM students
		 ORDER BY student_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()

	result := make([]Student, 0)
	for rows.Next() {
		var item Student
		if err := rows.Scan(&item.ID, &item.StudentName, &item.GroupID); err != nil {
			return nil, fmt.Errorf("scan student: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate students rows: %w", err)
	}

	return result, nil
}

func (r *StudentRepository) ListByGroupID(ctx context.Context, groupID int32) ([]Student, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT student_id, student_name, group_id
		 FROM students
		 WHERE group_id = $1
		 ORDER BY student_id`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list students by group id: %w", err)
	}
	defer rows.Close()

	result := make([]Student, 0)
	for rows.Next() {
		var item Student
		if err := rows.Scan(&item.ID, &item.StudentName, &item.GroupID); err != nil {
			return nil, fmt.Errorf("scan student: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate students rows: %w", err)
	}

	return result, nil
}
