package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAttendanceCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   string
	}{
		{status: "present", want: ""},
		{status: "absent", want: "О"},
		{status: "late", want: "ОП"},
		{status: "excused", want: "У"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.status, func(t *testing.T) {
			t.Parallel()
			if got := attendanceCode(test.status); got != test.want {
				t.Fatalf("attendanceCode(%q) = %q, want %q", test.status, got, test.want)
			}
		})
	}
}

func TestRatingActivityKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		itemType string
		want     string
	}{
		{itemType: "Lab", want: "laboratory"},
		{itemType: "laboratory", want: "laboratory"},
		{itemType: "Лабораторная работа", want: "laboratory"},
		{itemType: "practice", want: "practice"},
		{itemType: "Практика", want: "practice"},
		{itemType: "test", want: ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.itemType, func(t *testing.T) {
			t.Parallel()
			if got := ratingActivityKind(test.itemType); got != test.want {
				t.Fatalf("ratingActivityKind(%q) = %q, want %q", test.itemType, got, test.want)
			}
		})
	}
}

func TestRatingPercent(t *testing.T) {
	t.Parallel()

	if got := ratingPercent(87, 100); got != 87 {
		t.Fatalf("ratingPercent(87, 100) = %v, want 87", got)
	}
	if got := ratingPercent(12, 13); got != 92.3 {
		t.Fatalf("ratingPercent(12, 13) = %v, want 92.3", got)
	}
	if got := ratingPercent(1, 0); got != 0 {
		t.Fatalf("ratingPercent(1, 0) = %v, want 0", got)
	}
}

func TestStudentReference(t *testing.T) {
	t.Parallel()
	svc := &Service{jwtSecret: []byte("test-secret")}
	got := svc.studentReference(1)
	if got == "STU-0001" || len(got) != len("STU-")+12 {
		t.Fatalf("studentReference(1) = %q, want a non-reversible stable pseudonym", got)
	}
	if got != svc.studentReference(1) {
		t.Fatalf("studentReference(1) is not stable")
	}
}

func TestGeneralRatingJSONKeepsEmptyArraysAndNullScore(t *testing.T) {
	t.Parallel()

	subject := newGeneralRatingStudentSubject(102).value
	subject.Activities.Practices = append(subject.Activities.Practices, GeneralRatingGradedActivity{
		Date:     "2026-02-07",
		Title:    "Практическое занятие №1",
		Score:    nil,
		MaxScore: 100,
	})
	payload := GeneralRatingPayload{
		SchemaVersion: generalRatingSchemaVersion,
		Departments:   make([]GeneralRatingDepartment, 0),
		Subjects:      make([]GeneralRatingSubject, 0),
		Groups: []GeneralRatingGroup{{
			Students: []GeneralRatingStudent{{Subjects: []GeneralRatingStudentSubject{subject}}},
		}},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	got := string(raw)
	for _, fragment := range []string{
		`"departments":[]`,
		`"subjects":[]`,
		`"is_current_user":false`,
		`"lectures":[]`,
		`"laboratory_works":[]`,
		`"score":null`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("marshalled payload does not contain %s: %s", fragment, got)
		}
	}
}
