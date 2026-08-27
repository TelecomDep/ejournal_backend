package app

import (
	"testing"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

func TestAttendanceSessionStudentIncludesFraudFields(t *testing.T) {
	now := time.Now().UTC()
	student := db.AttendanceSessionStudent{
		StudentID:   101,
		StudentName: "Тестовый Читер",
		GroupID:     1,
		GroupName:   "TEST-101",
		Status:      "absent",
		MarkedAt:    &now,
		MarkedBy:    "self",
		IsFraud:     true,
		FraudReason: "device_id already used in this lesson",
	}

	if !student.IsFraud {
		t.Fatalf("expected IsFraud to be true")
	}
	if student.FraudReason != "device_id already used in this lesson" {
		t.Fatalf("unexpected fraud reason: %s", student.FraudReason)
	}
}
