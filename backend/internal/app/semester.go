package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

type SemesterCreateData struct {
	AcademicYear string `json:"academic_year"`
	TermNum      int32  `json:"term_num"`
	Name         string `json:"name"`
	StartsAt     string `json:"starts_at"`
	EndsAt       string `json:"ends_at"`
	IsCurrent    bool   `json:"is_current,omitempty"`
}

type SemesterIDData struct {
	SemesterID int32 `json:"semester_id"`
}

func semesterToMap(semester db.Semester) map[string]any {
	return map[string]any{
		"semester_id":   semester.ID,
		"academic_year": semester.AcademicYear,
		"term_num":      semester.TermNum,
		"name":          semester.Name,
		"starts_at":     formatAPITime(semester.StartsAt),
		"ends_at":       formatAPITime(semester.EndsAt),
		"is_current":    semester.IsCurrent,
		"created_at":    formatAPITime(semester.CreatedAt),
	}
}

func (s *Service) currentSemester(ctx context.Context) (db.Semester, error) {
	semester, found, err := s.store.Semesters.GetCurrent(ctx)
	if err != nil {
		return db.Semester{}, err
	}
	if !found {
		return db.Semester{}, errors.New("current semester not found")
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

	ctx, cancel := s.dbContext()
	defer cancel()

	startsAt, err := parseAPITime(strings.TrimSpace(data.StartsAt))
	if err != nil {
		return Response{OK: false, Error: "starts_at must be RFC3339"}
	}
	endsAt, err := parseAPITime(strings.TrimSpace(data.EndsAt))
	if err != nil {
		return Response{OK: false, Error: "ends_at must be RFC3339"}
	}

	created, err := s.store.Semesters.Create(ctx, db.Semester{
		AcademicYear: strings.TrimSpace(data.AcademicYear),
		TermNum:      data.TermNum,
		Name:         strings.TrimSpace(data.Name),
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		IsCurrent:    data.IsCurrent,
	})
	if err != nil {
		return Response{OK: false, Error: err.Error()}
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

	semester, found, err := s.store.Semesters.Activate(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: "failed to activate semester"}
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
