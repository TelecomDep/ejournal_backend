package httpserver

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TelecomDep/ejournal_backend/internal/app"
)

func int32Pointer(value int32) *int32 {
	return &value
}

func TestScriptedReportCommandForTeacher(t *testing.T) {
	t.Parallel()

	payload := &app.GeneralRatingPayload{
		AccessControl: app.GeneralRatingAccessControl{Role: app.RoleTeacher},
		Subjects: []app.GeneralRatingSubject{
			{SubjectID: 12, TeacherID: 7, DepartmentID: 3},
		},
	}
	selection := scriptedReportSelection{
		subjectID: int32Pointer(12),
		groupIDs:  []int32{4, 9},
	}

	script, args, namePart, err := scriptedReportCommand(payload, selection, "input.json", "report.xlsx")
	if err != nil {
		t.Fatalf("scriptedReportCommand() error = %v", err)
	}
	if script != "teacher_report.py" || namePart != "teacher_subject_12" {
		t.Fatalf("unexpected command: script=%q namePart=%q", script, namePart)
	}
	wantArgs := []string{
		"--teacher-id", "7", "--subject-id", "12",
		"--input", "input.json", "--output", "report.xlsx",
		"--group-ids", "4", "9",
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
}

func TestScriptedReportCommandRejectsDepartmentOutsideScope(t *testing.T) {
	t.Parallel()

	payload := &app.GeneralRatingPayload{
		AccessControl: app.GeneralRatingAccessControl{Role: app.RoleHead},
		Departments: []app.GeneralRatingDepartment{
			{DepartmentID: 3, DepartmentName: "Allowed"},
		},
	}
	selection := scriptedReportSelection{departmentID: int32Pointer(8)}

	_, _, _, err := scriptedReportCommand(payload, selection, "input.json", "report.xlsx")
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("scriptedReportCommand() error = %v, want scope error", err)
	}
}
