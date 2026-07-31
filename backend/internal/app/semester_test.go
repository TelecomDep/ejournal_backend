package app

import (
	"testing"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

func TestValidateAcademicYear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: "2026/2027"},
		{name: "invalid format", value: "2026-2027", wantErr: true},
		{name: "non consecutive", value: "2026/2028", wantErr: true},
		{name: "non numeric", value: "year/2027", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := validateAcademicYear(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateAcademicYear(%q) error = %v, wantErr %v", test.value, err, test.wantErr)
			}
		})
	}
}

func TestValidateSemesterCreate(t *testing.T) {
	t.Parallel()

	location := time.FixedZone("Asia/Novosibirsk", 7*60*60)
	fallStart := time.Date(2099, time.September, 1, 0, 0, 0, 0, location)
	fallEnd := time.Date(2100, time.January, 31, 23, 59, 59, 0, location)

	tests := []struct {
		name       string
		data       SemesterCreateData
		startsAt   time.Time
		endsAt     time.Time
		wantStatus string
		wantErr    bool
	}{
		{
			name: "planned fall semester",
			data: SemesterCreateData{
				AcademicYear: "2099/2100",
				TermNum:      1,
				Name:         "Осенний семестр",
			},
			startsAt:   fallStart,
			endsAt:     fallEnd,
			wantStatus: db.SemesterStatusPlanned,
		},
		{
			name: "future semester cannot be opened",
			data: SemesterCreateData{
				AcademicYear: "2099/2100",
				TermNum:      1,
				Name:         "Осенний семестр",
				IsCurrent:    true,
			},
			startsAt: fallStart,
			endsAt:   fallEnd,
			wantErr:  true,
		},
		{
			name: "invalid term",
			data: SemesterCreateData{
				AcademicYear: "2099/2100",
				TermNum:      3,
				Name:         "Семестр",
			},
			startsAt: fallStart,
			endsAt:   fallEnd,
			wantErr:  true,
		},
		{
			name: "term dates do not match academic year",
			data: SemesterCreateData{
				AcademicYear: "2099/2100",
				TermNum:      2,
				Name:         "Весенний семестр",
			},
			startsAt: fallStart,
			endsAt:   fallEnd,
			wantErr:  true,
		},
		{
			name: "overlong name",
			data: SemesterCreateData{
				AcademicYear: "2099/2100",
				TermNum:      1,
				Name:         string(make([]rune, 256)),
			},
			startsAt: fallStart,
			endsAt:   fallEnd,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			status, err := validateSemesterCreate(test.data, test.startsAt, test.endsAt)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateSemesterCreate() error = %v, wantErr %v", err, test.wantErr)
			}
			if status != test.wantStatus {
				t.Fatalf("validateSemesterCreate() status = %q, want %q", status, test.wantStatus)
			}
		})
	}
}

func TestSemesterWriteError(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)
	base := db.Semester{
		Status:    db.SemesterStatusOpen,
		IsCurrent: true,
		StartsAt:  now.Add(-24 * time.Hour),
		EndsAt:    now.Add(24 * time.Hour),
	}

	tests := []struct {
		name    string
		mutate  func(*db.Semester)
		wantErr bool
	}{
		{name: "open current semester is writable"},
		{
			name: "planned semester is read only",
			mutate: func(semester *db.Semester) {
				semester.Status = db.SemesterStatusPlanned
				semester.IsCurrent = false
			},
			wantErr: true,
		},
		{
			name: "non current semester is read only",
			mutate: func(semester *db.Semester) {
				semester.IsCurrent = false
			},
			wantErr: true,
		},
		{
			name: "future semester is read only",
			mutate: func(semester *db.Semester) {
				semester.StartsAt = now.Add(time.Hour)
			},
			wantErr: true,
		},
		{
			name: "ended semester is read only",
			mutate: func(semester *db.Semester) {
				semester.EndsAt = now
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			semester := base
			if test.mutate != nil {
				test.mutate(&semester)
			}
			err := semesterWriteError(semester, now)
			if (err != nil) != test.wantErr {
				t.Fatalf("semesterWriteError() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
