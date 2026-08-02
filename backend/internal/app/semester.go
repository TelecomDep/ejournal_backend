package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

type SemesterCreateData struct {
	AcademicYear string `json:"academic_year"`
	TermNum      int32  `json:"term_num"`
	Name         string `json:"name"`
	StartsAt     string `json:"starts_at"`
	EndsAt       string `json:"ends_at"`
	Status       string `json:"status,omitempty"`
	IsCurrent    bool   `json:"is_current,omitempty"`
}

type SemesterIDData struct {
	SemesterID int32 `json:"semester_id"`
}

type SemesterSelectionData struct {
	SemesterID *int32 `json:"semester_id,omitempty"`
}

func optionalAPITime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatAPITime(*value)
}

func semesterToMap(semester db.Semester) map[string]any {
	return map[string]any{
		"semester_id":         semester.ID,
		"academic_year":       semester.AcademicYear,
		"term_num":            semester.TermNum,
		"name":                semester.Name,
		"starts_at":           formatAPITime(semester.StartsAt),
		"ends_at":             formatAPITime(semester.EndsAt),
		"status":              semester.Status,
		"is_current":          semester.IsCurrent,
		"created_at":          formatAPITime(semester.CreatedAt),
		"updated_at":          formatAPITime(semester.UpdatedAt),
		"created_by_user_id":  semester.CreatedByUserID,
		"opened_at":           optionalAPITime(semester.OpenedAt),
		"opened_by_user_id":   semester.OpenedByUserID,
		"closed_at":           optionalAPITime(semester.ClosedAt),
		"closed_by_user_id":   semester.ClosedByUserID,
		"archived_at":         optionalAPITime(semester.ArchivedAt),
		"archived_by_user_id": semester.ArchivedByUserID,
	}
}

func (s *Service) currentSemester(ctx context.Context) (db.Semester, error) {
	semester, found, err := s.store.Semesters.GetCurrent(ctx)
	if err != nil {
		return db.Semester{}, err
	}
	if !found {
		return db.Semester{}, errors.New("open semester not found")
	}
	return semester, nil
}

func (s *Service) semesterByID(ctx context.Context, semesterID int32) (db.Semester, error) {
	if semesterID <= 0 {
		return db.Semester{}, fmt.Errorf("semester id is required")
	}
	semester, found, err := s.store.Semesters.GetByID(ctx, semesterID)
	if err != nil {
		return db.Semester{}, err
	}
	if !found {
		return db.Semester{}, errors.New("semester not found")
	}
	return semester, nil
}

func (s *Service) semesterForOptionalID(ctx context.Context, semesterID *int32) (db.Semester, error) {
	if semesterID != nil {
		return s.semesterByID(ctx, *semesterID)
	}
	return s.currentSemester(ctx)
}

func semesterWriteError(semester db.Semester, now time.Time) error {
	if semester.Status != db.SemesterStatusOpen || !semester.IsCurrent {
		return errors.New("semester is not open for changes")
	}
	if now.Before(semester.StartsAt) {
		return errors.New("semester has not started")
	}
	if !now.Before(semester.EndsAt) {
		return errors.New("semester has ended")
	}
	return nil
}

func (s *Service) semesterForWrite(ctx context.Context, semesterID *int32) (db.Semester, error) {
	semester, err := s.semesterForOptionalID(ctx, semesterID)
	if err != nil {
		return db.Semester{}, err
	}
	if err := semesterWriteError(semester, time.Now().UTC()); err != nil {
		return db.Semester{}, err
	}
	return semester, nil
}

func validateAcademicYear(value string) (int, int, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || len(parts[0]) != 4 || len(parts[1]) != 4 {
		return 0, 0, errors.New("academic_year must have format YYYY/YYYY")
	}
	first, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, errors.New("academic_year must have format YYYY/YYYY")
	}
	second, err := strconv.Atoi(parts[1])
	if err != nil || second != first+1 {
		return 0, 0, errors.New("academic_year must contain consecutive years")
	}
	return first, second, nil
}

func validateSemesterCreate(data SemesterCreateData, startsAt, endsAt time.Time) (string, error) {
	academicYear := strings.TrimSpace(data.AcademicYear)
	firstYear, secondYear, err := validateAcademicYear(academicYear)
	if err != nil {
		return "", err
	}
	if data.TermNum != 1 && data.TermNum != 2 {
		return "", errors.New("term_num must be 1 or 2")
	}
	name := strings.TrimSpace(data.Name)
	if name == "" {
		return "", errors.New("name is required")
	}
	if utf8.RuneCountInString(name) > 255 {
		return "", errors.New("name must not exceed 255 characters")
	}
	if !endsAt.After(startsAt) {
		return "", errors.New("ends_at must be after starts_at")
	}
	if data.TermNum == 1 && startsAt.Year() != firstYear {
		return "", errors.New("term 1 must start in the first academic year")
	}
	if data.TermNum == 1 && endsAt.Year() != firstYear && endsAt.Year() != secondYear {
		return "", errors.New("term 1 dates do not match academic_year")
	}
	if data.TermNum == 2 && (startsAt.Year() != secondYear || endsAt.Year() != secondYear) {
		return "", errors.New("term 2 dates do not match academic_year")
	}

	status := strings.ToLower(strings.TrimSpace(data.Status))
	if status == "" {
		status = db.SemesterStatusPlanned
		if data.IsCurrent {
			status = db.SemesterStatusOpen
		}
	}
	if status != db.SemesterStatusPlanned && status != db.SemesterStatusOpen {
		return "", errors.New("new semester status must be planned or open")
	}
	if status == db.SemesterStatusOpen {
		now := time.Now().UTC()
		if now.Before(startsAt) {
			return "", errors.New("semester has not started")
		}
		if !now.Before(endsAt) {
			return "", errors.New("ended semester cannot be opened")
		}
	}
	return status, nil
}

func semesterRepositoryError(err error) string {
	switch {
	case errors.Is(err, db.ErrSemesterAlreadyExists):
		return "semester already exists"
	case errors.Is(err, db.ErrSemesterDateOverlap):
		return "semester date range overlaps an existing semester"
	case errors.Is(err, db.ErrSemesterInvalidTransition):
		return "invalid semester status transition"
	case errors.Is(err, db.ErrSemesterHasActiveSessions):
		return "semester has active attendance sessions"
	default:
		return "failed to save semester"
	}
}

func (s *Service) semestersList() Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	semesters, err := s.store.Semesters.List(ctx)
	if err != nil {
		return Response{OK: false, Error: "failed to load semesters"}
	}

	items := make([]map[string]any, 0, len(semesters))
	for _, semester := range semesters {
		items = append(items, semesterToMap(semester))
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"items": items,
		},
	}
}

func (s *Service) currentSemesterInfo() Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	semester, err := s.currentSemester(ctx)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"semester": semesterToMap(semester),
		},
	}
}

func (s *Service) createSemester(sessionToken string, data SemesterCreateData) Response {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if user.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}

	startsAt, err := parseAPITime(strings.TrimSpace(data.StartsAt))
	if err != nil {
		return Response{OK: false, Error: "starts_at must be RFC3339"}
	}
	endsAt, err := parseAPITime(strings.TrimSpace(data.EndsAt))
	if err != nil {
		return Response{OK: false, Error: "ends_at must be RFC3339"}
	}
	status, err := validateSemesterCreate(data, startsAt, endsAt)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	created, err := s.store.Semesters.Create(ctx, db.Semester{
		AcademicYear: strings.TrimSpace(data.AcademicYear),
		TermNum:      data.TermNum,
		Name:         strings.TrimSpace(data.Name),
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		Status:       status,
		IsCurrent:    status == db.SemesterStatusOpen,
	}, user.ID)
	if err != nil {
		return Response{OK: false, Error: semesterRepositoryError(err)}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"semester": semesterToMap(created),
		},
	}
}

func (s *Service) activateSemester(sessionToken string, data SemesterIDData) Response {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if user.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	target, err := s.semesterByID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if target.Status != db.SemesterStatusOpen {
		now := time.Now().UTC()
		if now.Before(target.StartsAt) {
			return Response{OK: false, Error: "semester has not started"}
		}
		if !now.Before(target.EndsAt) {
			return Response{OK: false, Error: "ended semester cannot be opened"}
		}
	}

	semester, found, err := s.store.Semesters.Activate(ctx, data.SemesterID, user.ID)
	if err != nil {
		return Response{OK: false, Error: semesterRepositoryError(err)}
	}
	if !found {
		return Response{OK: false, Error: "semester not found"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"semester": semesterToMap(semester),
		},
	}
}

func (s *Service) closeSemester(sessionToken string, data SemesterIDData) Response {
	return s.transitionSemester(sessionToken, data, db.SemesterStatusClosed)
}

func (s *Service) archiveSemester(sessionToken string, data SemesterIDData) Response {
	return s.transitionSemester(sessionToken, data, db.SemesterStatusArchived)
}

func (s *Service) transitionSemester(sessionToken string, data SemesterIDData, targetStatus string) Response {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if user.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var semester db.Semester
	var found bool
	switch targetStatus {
	case db.SemesterStatusClosed:
		semester, found, err = s.store.Semesters.Close(ctx, data.SemesterID, user.ID)
	case db.SemesterStatusArchived:
		semester, found, err = s.store.Semesters.Archive(ctx, data.SemesterID, user.ID)
	default:
		err = db.ErrSemesterInvalidTransition
	}
	if err != nil {
		return Response{OK: false, Error: semesterRepositoryError(err)}
	}
	if !found {
		return Response{OK: false, Error: "semester not found"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"semester": semesterToMap(semester),
		},
	}
}

func (s *Service) deleteSemester(sessionToken string, data SemesterIDData) Response {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if user.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	found, err := s.store.Semesters.Delete(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: "failed to delete semester"}
	}
	if !found {
		return Response{OK: false, Error: "semester not found or active"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"deleted":     true,
			"semester_id": data.SemesterID,
		},
	}
}
