package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	SemesterStatusPlanned  = "planned"
	SemesterStatusOpen     = "open"
	SemesterStatusClosed   = "closed"
	SemesterStatusArchived = "archived"
)

var (
	ErrSemesterAlreadyExists     = errors.New("semester already exists")
	ErrSemesterDateOverlap       = errors.New("semester date range overlaps an existing semester")
	ErrSemesterInvalidTransition = errors.New("invalid semester status transition")
	ErrSemesterHasActiveSessions = errors.New("semester has active attendance sessions")
)

const semesterColumns = `semester_id, academic_year, term_num, name, starts_at, ends_at,
	status, is_current, created_at, updated_at, created_by_user_id,
	opened_at, opened_by_user_id, closed_at, closed_by_user_id,
	archived_at, archived_by_user_id`

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
		&out.Status,
		&out.IsCurrent,
		&out.CreatedAt,
		&out.UpdatedAt,
		&out.CreatedByUserID,
		&out.OpenedAt,
		&out.OpenedByUserID,
		&out.ClosedAt,
		&out.ClosedByUserID,
		&out.ArchivedAt,
		&out.ArchivedByUserID,
	); err != nil {
		return Semester{}, err
	}
	return out, nil
}

func translateSemesterError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch {
	case pgErr.Code == "23P01":
		return ErrSemesterDateOverlap
	case pgErr.Code == "23505" && pgErr.ConstraintName == "semesters_year_term_unique":
		return ErrSemesterAlreadyExists
	case pgErr.Code == "23505" && (pgErr.ConstraintName == "idx_semesters_single_open" || pgErr.ConstraintName == "idx_semesters_current"):
		return ErrSemesterInvalidTransition
	default:
		return err
	}
}

func validSemesterStatus(status string) bool {
	switch status {
	case SemesterStatusPlanned, SemesterStatusOpen, SemesterStatusClosed, SemesterStatusArchived:
		return true
	default:
		return false
	}
}

func hasActiveSessionsInOpenSemester(ctx context.Context, tx pgx.Tx) (bool, error) {
	var active bool
	err := tx.QueryRow(
		ctx,
		`SELECT EXISTS (
		     SELECT 1
		     FROM attendance_sessions session
		     JOIN semesters semester ON semester.semester_id = session.semester_id
		     WHERE semester.status = 'open' AND session.expires_at > now()
		 )`,
	).Scan(&active)
	return active, err
}

func lockOpenSemester(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(
		ctx,
		`SELECT semester_id FROM semesters WHERE status = 'open' FOR UPDATE`,
	)
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

func (r *SemesterRepository) Create(ctx context.Context, semester Semester, actorUserID int32) (Semester, error) {
	semester.AcademicYear = strings.TrimSpace(semester.AcademicYear)
	semester.Name = strings.TrimSpace(semester.Name)
	semester.Status = strings.ToLower(strings.TrimSpace(semester.Status))
	if semester.Status == "" {
		semester.Status = SemesterStatusPlanned
	}
	if semester.AcademicYear == "" {
		return Semester{}, fmt.Errorf("academic year is required")
	}
	if semester.TermNum != 1 && semester.TermNum != 2 {
		return Semester{}, fmt.Errorf("term number must be 1 or 2")
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
	if semester.Status != SemesterStatusPlanned && semester.Status != SemesterStatusOpen {
		return Semester{}, fmt.Errorf("new semester status must be planned or open")
	}
	if actorUserID <= 0 {
		return Semester{}, fmt.Errorf("semester actor user id is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Semester{}, fmt.Errorf("begin create semester transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `LOCK TABLE semesters IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return Semester{}, fmt.Errorf("lock semesters: %w", err)
	}

	if semester.Status == SemesterStatusOpen {
		if err := lockOpenSemester(ctx, tx); err != nil {
			return Semester{}, fmt.Errorf("lock open semester: %w", err)
		}
		hasActiveSessions, err := hasActiveSessionsInOpenSemester(ctx, tx)
		if err != nil {
			return Semester{}, fmt.Errorf("check active attendance sessions: %w", err)
		}
		if hasActiveSessions {
			return Semester{}, ErrSemesterHasActiveSessions
		}
		if _, err := tx.Exec(
			ctx,
			`UPDATE semesters
			 SET status = 'closed',
			     is_current = FALSE,
			     closed_at = now(),
			     closed_by_user_id = $1,
			     updated_at = now()
			 WHERE status = 'open'`,
			actorUserID,
		); err != nil {
			return Semester{}, fmt.Errorf("close current semester: %w", err)
		}
	}

	out, err := scanSemester(tx.QueryRow(
		ctx,
		`INSERT INTO semesters (
		     academic_year, term_num, name, starts_at, ends_at, status, is_current,
		     created_by_user_id, opened_at, opened_by_user_id
		 )
		 VALUES (
		     $1, $2, $3, $4, $5, $6::text, $6::text = 'open',
		     $7::integer, CASE WHEN $6::text = 'open' THEN now() END,
		     CASE WHEN $6::text = 'open' THEN $7::integer END
		 )
		 RETURNING `+semesterColumns,
		semester.AcademicYear,
		semester.TermNum,
		semester.Name,
		semester.StartsAt.UTC(),
		semester.EndsAt.UTC(),
		semester.Status,
		actorUserID,
	))
	if err != nil {
		return Semester{}, fmt.Errorf("insert semester: %w", translateSemesterError(err))
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
		`SELECT `+semesterColumns+`
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
		`SELECT `+semesterColumns+`
		 FROM semesters
		 WHERE status = 'open' AND is_current = TRUE
		 ORDER BY semester_id DESC
		 LIMIT 1`,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Semester{}, false, nil
	}
	if err != nil {
		return Semester{}, false, fmt.Errorf("get current semester: %w", err)
	}
	return out, true, nil
}

func (r *SemesterRepository) List(ctx context.Context) ([]Semester, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT `+semesterColumns+`
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

func (r *SemesterRepository) Activate(ctx context.Context, id, actorUserID int32) (Semester, bool, error) {
	if id <= 0 || actorUserID <= 0 {
		return Semester{}, false, fmt.Errorf("semester id and actor user id are required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Semester{}, false, fmt.Errorf("begin activate semester transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `LOCK TABLE semesters IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return Semester{}, false, fmt.Errorf("lock semesters: %w", err)
	}

	current, err := scanSemester(tx.QueryRow(
		ctx,
		`SELECT `+semesterColumns+` FROM semesters WHERE semester_id = $1 FOR UPDATE`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Semester{}, false, nil
	}
	if err != nil {
		return Semester{}, false, fmt.Errorf("load semester for activation: %w", err)
	}
	if current.Status == SemesterStatusOpen {
		return current, true, nil
	}
	if current.Status != SemesterStatusPlanned {
		return Semester{}, true, ErrSemesterInvalidTransition
	}

	if err := lockOpenSemester(ctx, tx); err != nil {
		return Semester{}, false, fmt.Errorf("lock open semester: %w", err)
	}
	hasActiveSessions, err := hasActiveSessionsInOpenSemester(ctx, tx)
	if err != nil {
		return Semester{}, false, fmt.Errorf("check active attendance sessions: %w", err)
	}
	if hasActiveSessions {
		return Semester{}, true, ErrSemesterHasActiveSessions
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE semesters
		 SET status = 'closed',
		     is_current = FALSE,
		     closed_at = now(),
		     closed_by_user_id = $1,
		     updated_at = now()
		 WHERE status = 'open'`,
		actorUserID,
	); err != nil {
		return Semester{}, false, fmt.Errorf("close current semester: %w", err)
	}

	out, err := scanSemester(tx.QueryRow(
		ctx,
		`UPDATE semesters
		 SET status = 'open',
		     is_current = TRUE,
		     opened_at = now(),
		     opened_by_user_id = $2,
		     closed_at = NULL,
		     closed_by_user_id = NULL,
		     updated_at = now()
		 WHERE semester_id = $1
		 RETURNING `+semesterColumns,
		id,
		actorUserID,
	))
	if err != nil {
		return Semester{}, false, fmt.Errorf("activate semester: %w", translateSemesterError(err))
	}
	if err := tx.Commit(ctx); err != nil {
		return Semester{}, false, fmt.Errorf("commit activate semester transaction: %w", err)
	}
	return out, true, nil
}

func (r *SemesterRepository) Close(ctx context.Context, id, actorUserID int32) (Semester, bool, error) {
	return r.transition(ctx, id, actorUserID, SemesterStatusOpen, SemesterStatusClosed)
}

func (r *SemesterRepository) Archive(ctx context.Context, id, actorUserID int32) (Semester, bool, error) {
	return r.transition(ctx, id, actorUserID, SemesterStatusClosed, SemesterStatusArchived)
}

func (r *SemesterRepository) transition(
	ctx context.Context,
	id, actorUserID int32,
	fromStatus, toStatus string,
) (Semester, bool, error) {
	if id <= 0 || actorUserID <= 0 || !validSemesterStatus(fromStatus) || !validSemesterStatus(toStatus) {
		return Semester{}, false, fmt.Errorf("invalid semester transition arguments")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Semester{}, false, fmt.Errorf("begin semester transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := scanSemester(tx.QueryRow(
		ctx,
		`SELECT `+semesterColumns+` FROM semesters WHERE semester_id = $1 FOR UPDATE`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Semester{}, false, nil
	}
	if err != nil {
		return Semester{}, false, fmt.Errorf("load semester for transition: %w", err)
	}
	if current.Status == toStatus {
		return current, true, nil
	}
	if current.Status != fromStatus {
		return Semester{}, true, ErrSemesterInvalidTransition
	}
	if toStatus == SemesterStatusClosed {
		var hasActiveSessions bool
		if err := tx.QueryRow(
			ctx,
			`SELECT EXISTS (
			     SELECT 1
			     FROM attendance_sessions
			     WHERE semester_id = $1 AND expires_at > now()
			 )`,
			id,
		).Scan(&hasActiveSessions); err != nil {
			return Semester{}, false, fmt.Errorf("check active attendance sessions: %w", err)
		}
		if hasActiveSessions {
			return Semester{}, true, ErrSemesterHasActiveSessions
		}
	}

	var query string
	switch toStatus {
	case SemesterStatusClosed:
		query = `UPDATE semesters
			SET status = 'closed', is_current = FALSE, closed_at = now(),
			    closed_by_user_id = $2, updated_at = now()
			WHERE semester_id = $1
			RETURNING ` + semesterColumns
	case SemesterStatusArchived:
		query = `UPDATE semesters
			SET status = 'archived', is_current = FALSE, archived_at = now(),
			    archived_by_user_id = $2, updated_at = now()
			WHERE semester_id = $1
			RETURNING ` + semesterColumns
	default:
		return Semester{}, true, ErrSemesterInvalidTransition
	}

	out, err := scanSemester(tx.QueryRow(ctx, query, id, actorUserID))
	if err != nil {
		return Semester{}, false, fmt.Errorf("transition semester: %w", translateSemesterError(err))
	}
	if err := tx.Commit(ctx); err != nil {
		return Semester{}, false, fmt.Errorf("commit semester transition: %w", err)
	}
	return out, true, nil
}

func (r *SemesterRepository) Delete(ctx context.Context, id int32) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("semester id is required")
	}
	res, err := r.pool.Exec(ctx, `DELETE FROM semesters WHERE semester_id = $1 AND status != 'open'`, id)
	if err != nil {
		return false, fmt.Errorf("delete semester: %w", err)
	}
	return res.RowsAffected() > 0, nil
}
