package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SemesterRepository struct {
	pool *pgxpool.Pool
}

func NewSemesterRepository(pool *pgxpool.Pool) *SemesterRepository {
	return &SemesterRepository{pool: pool}
}

func scanSemester(row pgx.Row) (Semester, error) {
	var out Semester
	if err := row.Scan(
		&out.ID,
		&out.AcademicYear,
		&out.TermNum,
		&out.Name,
		&out.StartsAt,
		&out.EndsAt,
		&out.IsCurrent,
		&out.CreatedAt,
	); err != nil {
		return Semester{}, err
	}
	return out, nil
}

func (r *SemesterRepository) Create(ctx context.Context, semester Semester) (Semester, error) {
	semester.AcademicYear = strings.TrimSpace(semester.AcademicYear)
	semester.Name = strings.TrimSpace(semester.Name)
	if semester.AcademicYear == "" {
		return Semester{}, fmt.Errorf("academic year is required")
	}
	if semester.TermNum <= 0 {
		return Semester{}, fmt.Errorf("term number is required")
	}
	if semester.Name == "" {
		return Semester{}, fmt.Errorf("semester name is required")
	}
	if semester.StartsAt.IsZero() || semester.EndsAt.IsZero() {
		return Semester{}, fmt.Errorf("semester date range is required")
	}
	if !semester.EndsAt.After(semester.StartsAt) {
		return Semester{}, fmt.Errorf("semester ends_at must be after starts_at")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Semester{}, fmt.Errorf("begin create semester transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if semester.IsCurrent {
		if _, err := tx.Exec(ctx, `UPDATE semesters SET is_current = FALSE WHERE is_current = TRUE`); err != nil {
			return Semester{}, fmt.Errorf("clear current semester: %w", err)
		}
	}

	out, err := scanSemester(tx.QueryRow(
		ctx,
		`INSERT INTO semesters (academic_year, term_num, name, starts_at, ends_at, is_current)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING semester_id, academic_year, term_num, name, starts_at, ends_at, is_current, created_at`,
		semester.AcademicYear,
		semester.TermNum,
		semester.Name,
		semester.StartsAt.UTC(),
		semester.EndsAt.UTC(),
		semester.IsCurrent,
	))
	if err != nil {
		return Semester{}, fmt.Errorf("insert semester: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Semester{}, fmt.Errorf("commit create semester transaction: %w", err)
	}
	return out, nil
}

func (r *SemesterRepository) GetByID(ctx context.Context, id int32) (Semester, bool, error) {
	if id <= 0 {
		return Semester{}, false, fmt.Errorf("semester id is required")
	}
	out, err := scanSemester(r.pool.QueryRow(
		ctx,
		`SELECT semester_id, academic_year, term_num, name, starts_at, ends_at, is_current, created_at
		 FROM semesters
		 WHERE semester_id = $1`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Semester{}, false, nil
	}
	if err != nil {
		return Semester{}, false, fmt.Errorf("get semester by id: %w", err)
	}
	return out, true, nil
}

func (r *SemesterRepository) GetCurrent(ctx context.Context) (Semester, bool, error) {
	out, err := scanSemester(r.pool.QueryRow(
		ctx,
		`SELECT semester_id, academic_year, term_num, name, starts_at, ends_at, is_current, created_at
		 FROM semesters
		 WHERE is_current = TRUE
		 ORDER BY semester_id DESC
		 LIMIT 1`,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		out, err = scanSemester(r.pool.QueryRow(
			ctx,
			`SELECT semester_id, academic_year, term_num, name, starts_at, ends_at, is_current, created_at
			 FROM semesters
			 WHERE starts_at <= now() AND ends_at > now()
			 ORDER BY starts_at DESC, semester_id DESC
			 LIMIT 1`,
		))
		if errors.Is(err, pgx.ErrNoRows) {
			return Semester{}, false, nil
		}
	}
	if err != nil {
		return Semester{}, false, fmt.Errorf("get current semester: %w", err)
	}
	return out, true, nil
}

func (r *SemesterRepository) List(ctx context.Context) ([]Semester, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT semester_id, academic_year, term_num, name, starts_at, ends_at, is_current, created_at
		 FROM semesters
		 ORDER BY starts_at DESC, semester_id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list semesters: %w", err)
	}
	defer rows.Close()

	result := make([]Semester, 0)
	for rows.Next() {
		item, scanErr := scanSemester(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan semester: %w", scanErr)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semester rows: %w", err)
	}
	return result, nil
}

func (r *SemesterRepository) Activate(ctx context.Context, id int32) (Semester, bool, error) {
	if id <= 0 {
		return Semester{}, false, fmt.Errorf("semester id is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Semester{}, false, fmt.Errorf("begin activate semester transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM semesters WHERE semester_id = $1)`, id).Scan(&exists); err != nil {
		return Semester{}, false, fmt.Errorf("check semester exists: %w", err)
	}
	if !exists {
		return Semester{}, false, nil
	}

	if _, err := tx.Exec(ctx, `UPDATE semesters SET is_current = FALSE WHERE is_current = TRUE`); err != nil {
		return Semester{}, false, fmt.Errorf("clear current semester: %w", err)
	}

	out, err := scanSemester(tx.QueryRow(
		ctx,
		`UPDATE semesters
		 SET is_current = TRUE
		 WHERE semester_id = $1
		 RETURNING semester_id, academic_year, term_num, name, starts_at, ends_at, is_current, created_at`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Semester{}, false, nil
	}
	if err != nil {
		return Semester{}, false, fmt.Errorf("activate semester: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Semester{}, false, fmt.Errorf("commit activate semester transaction: %w", err)
	}
	return out, true, nil
}
