package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeacherRepository struct {
	pool *pgxpool.Pool
}

func nullableInt4ToPtr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	v := value.Int32
	return &v
}

func nullableTextToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
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
	var err error
	if teacher.ID > 0 {
		var userID pgtype.Int4
		var lecternID pgtype.Int4
		var jobTitle pgtype.Text
		err = r.pool.QueryRow(
			ctx,
			`INSERT INTO teachers (teacher_id, role, user_id, name, lectern_id, job_title)
			 VALUES ($1, 'teacher', $2, $3, $4, $5)
			 RETURNING teacher_id, user_id, name, lectern_id, job_title`,
			teacher.ID,
			teacher.UserID,
			teacher.Name,
			teacher.LecternID,
			teacher.JobTitle,
		).Scan(&out.ID, &userID, &out.Name, &lecternID, &jobTitle)
		out.UserID = nullableInt4ToPtr(userID)
		out.LecternID = nullableInt4ToPtr(lecternID)
		out.JobTitle = nullableTextToString(jobTitle)
	} else {
		var userID pgtype.Int4
		var lecternID pgtype.Int4
		var jobTitle pgtype.Text
		err = r.pool.QueryRow(
			ctx,
			`INSERT INTO teachers (role, user_id, name, lectern_id, job_title)
			 VALUES ('teacher', $1, $2, $3, $4)
			 RETURNING teacher_id, user_id, name, lectern_id, job_title`,
			teacher.UserID,
			teacher.Name,
			teacher.LecternID,
			teacher.JobTitle,
		).Scan(&out.ID, &userID, &out.Name, &lecternID, &jobTitle)
		out.UserID = nullableInt4ToPtr(userID)
		out.LecternID = nullableInt4ToPtr(lecternID)
		out.JobTitle = nullableTextToString(jobTitle)
	}
	if err != nil {
		return Teacher{}, fmt.Errorf("insert teacher: %w", err)
	}

	return out, nil
}

func (r *TeacherRepository) GetByID(ctx context.Context, id int32) (Teacher, bool, error) {
	var out Teacher
	var userID pgtype.Int4
	var lecternID pgtype.Int4
	var jobTitle pgtype.Text
	err := r.pool.QueryRow(
		ctx,
		`SELECT teacher_id, user_id, name, lectern_id, job_title
		 FROM teachers
		 WHERE teacher_id = $1`,
		id,
	).Scan(&out.ID, &userID, &out.Name, &lecternID, &jobTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, false, nil
	}
	if err != nil {
		return Teacher{}, false, fmt.Errorf("get teacher by id: %w", err)
	}
	out.UserID = nullableInt4ToPtr(userID)
	out.LecternID = nullableInt4ToPtr(lecternID)
	out.JobTitle = nullableTextToString(jobTitle)

	return out, true, nil
}

func (r *TeacherRepository) GetByUserID(ctx context.Context, userID int32) (Teacher, bool, error) {
	var out Teacher
	var optionalUserID pgtype.Int4
	var lecternID pgtype.Int4
	var jobTitle pgtype.Text
	err := r.pool.QueryRow(
		ctx,
		`SELECT teacher_id, user_id, name, lectern_id, job_title
		 FROM teachers
		 WHERE user_id = $1`,
		userID,
	).Scan(&out.ID, &optionalUserID, &out.Name, &lecternID, &jobTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, false, nil
	}
	if err != nil {
		return Teacher{}, false, fmt.Errorf("get teacher by user id: %w", err)
	}
	out.UserID = nullableInt4ToPtr(optionalUserID)
	out.LecternID = nullableInt4ToPtr(lecternID)
	out.JobTitle = nullableTextToString(jobTitle)

	return out, true, nil
}

func (r *TeacherRepository) Update(ctx context.Context, teacher Teacher) (Teacher, bool, error) {
	teacher.Name = strings.TrimSpace(teacher.Name)
	teacher.JobTitle = strings.TrimSpace(teacher.JobTitle)
	if teacher.Name == "" {
		return Teacher{}, false, fmt.Errorf("teacher name is required")
	}

	var out Teacher
	var userID pgtype.Int4
	var lecternID pgtype.Int4
	var jobTitle pgtype.Text
	err := r.pool.QueryRow(
		ctx,
		`UPDATE teachers
		 SET user_id = $2,
		     name = $3,
		     lectern_id = $4,
		     job_title = $5
		 WHERE teacher_id = $1
		 RETURNING teacher_id, user_id, name, lectern_id, job_title`,
		teacher.ID,
		teacher.UserID,
		teacher.Name,
		teacher.LecternID,
		teacher.JobTitle,
	).Scan(&out.ID, &userID, &out.Name, &lecternID, &jobTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, false, nil
	}
	if err != nil {
		return Teacher{}, false, fmt.Errorf("update teacher: %w", err)
	}
	out.UserID = nullableInt4ToPtr(userID)
	out.LecternID = nullableInt4ToPtr(lecternID)
	out.JobTitle = nullableTextToString(jobTitle)

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
		`SELECT teacher_id, user_id, name, lectern_id, job_title
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
		var userID pgtype.Int4
		var lecternID pgtype.Int4
		var jobTitle pgtype.Text
		if err := rows.Scan(&item.ID, &userID, &item.Name, &lecternID, &jobTitle); err != nil {
			return nil, fmt.Errorf("scan teacher: %w", err)
		}
		item.UserID = nullableInt4ToPtr(userID)
		item.LecternID = nullableInt4ToPtr(lecternID)
		item.JobTitle = nullableTextToString(jobTitle)
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
		`SELECT teacher_id, user_id, name, lectern_id, job_title
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
		var userID pgtype.Int4
		var lecternID pgtype.Int4
		var jobTitle pgtype.Text
		if err := rows.Scan(&item.ID, &userID, &item.Name, &lecternID, &jobTitle); err != nil {
			return nil, fmt.Errorf("scan teacher: %w", err)
		}
		item.UserID = nullableInt4ToPtr(userID)
		item.LecternID = nullableInt4ToPtr(lecternID)
		item.JobTitle = nullableTextToString(jobTitle)
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teachers rows: %w", err)
	}

	return result, nil
}
