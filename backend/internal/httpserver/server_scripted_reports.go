package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

const scriptedReportPageSize int32 = 50

type scriptedReportSelection struct {
	departmentID *int32
	subjectID    *int32
	groupIDs     []int32
}

func optionalPositiveQueryID(c *fiber.Ctx, name string) (*int32, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || parsed <= 0 {
		return nil, fmt.Errorf("invalid %s", name)
	}
	value := int32(parsed)
	return &value, nil
}

func positiveQueryIDs(c *fiber.Ctx, name string) ([]int32, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' })
	values := make([]int32, 0, len(parts))
	seen := make(map[int32]struct{}, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseInt(part, 10, 32)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("invalid %s", name)
		}
		value := int32(parsed)
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values, nil
}

func reportSelectionFromQuery(c *fiber.Ctx) (scriptedReportSelection, error) {
	departmentID, err := optionalPositiveQueryID(c, "department_id")
	if err != nil {
		return scriptedReportSelection{}, err
	}
	subjectID, err := optionalPositiveQueryID(c, "subject_id")
	if err != nil {
		return scriptedReportSelection{}, err
	}
	groupIDs, err := positiveQueryIDs(c, "group_ids")
	if err != nil {
		return scriptedReportSelection{}, err
	}
	return scriptedReportSelection{departmentID: departmentID, subjectID: subjectID, groupIDs: groupIDs}, nil
}

func (s *Server) allGeneralRatingPages(token string, semesterID *int32) (*app.GeneralRatingPayload, app.Response) {
	payload, resp := s.svc.GeneralRating(token, semesterID, 1, scriptedReportPageSize)
	if payload == nil {
		return nil, resp
	}
	for page := int32(2); page <= payload.Pagination.TotalPages; page++ {
		next, nextResp := s.svc.GeneralRating(token, semesterID, page, scriptedReportPageSize)
		if next == nil {
			return nil, nextResp
		}
		payload.Groups = append(payload.Groups, next.Groups...)
	}
	payload.Pagination.Page = 1
	payload.Pagination.PageSize = int32(len(payload.Groups))
	return payload, app.Response{OK: true}
}

func findReportSubject(payload *app.GeneralRatingPayload, subjectID int32) (*app.GeneralRatingSubject, bool) {
	for index := range payload.Subjects {
		if payload.Subjects[index].SubjectID == subjectID {
			return &payload.Subjects[index], true
		}
	}
	return nil, false
}

func resolveDepartmentID(payload *app.GeneralRatingPayload, requested *int32) (int32, error) {
	if requested != nil {
		for _, department := range payload.Departments {
			if department.DepartmentID == *requested {
				return *requested, nil
			}
		}
		return 0, fmt.Errorf("department is outside the available scope")
	}
	if len(payload.Departments) == 1 {
		return payload.Departments[0].DepartmentID, nil
	}
	if len(payload.Departments) == 0 {
		return 0, fmt.Errorf("no departments available for report")
	}
	return 0, fmt.Errorf("department_id is required")
}

func appendIntArgs(args []string, flag string, values []int32) []string {
	if len(values) == 0 {
		return args
	}
	args = append(args, flag)
	for _, value := range values {
		args = append(args, strconv.FormatInt(int64(value), 10))
	}
	return args
}

func scriptedReportCommand(
	payload *app.GeneralRatingPayload,
	selection scriptedReportSelection,
	inputPath string,
	outputPath string,
) (string, []string, string, error) {
	role := payload.AccessControl.Role
	if role == app.RoleTeacher {
		if selection.subjectID == nil {
			return "", nil, "", fmt.Errorf("subject_id is required for teacher report")
		}
		subject, found := findReportSubject(payload, *selection.subjectID)
		if !found {
			return "", nil, "", fmt.Errorf("subject is outside the available scope")
		}
		args := []string{
			"--teacher-id", strconv.FormatInt(int64(subject.TeacherID), 10),
			"--subject-id", strconv.FormatInt(int64(subject.SubjectID), 10),
			"--input", inputPath,
			"--output", outputPath,
		}
		args = appendIntArgs(args, "--group-ids", selection.groupIDs)
		return "teacher_report.py", args, fmt.Sprintf("teacher_subject_%d", subject.SubjectID), nil
	}

	if role != app.RoleHead && role != app.RoleDean && role != app.RoleAdmin {
		return "", nil, "", fmt.Errorf("role cannot export reports")
	}
	departmentID, err := resolveDepartmentID(payload, selection.departmentID)
	if err != nil {
		return "", nil, "", err
	}
	args := []string{
		"--department-id", strconv.FormatInt(int64(departmentID), 10),
		"--input", inputPath,
		"--output", outputPath,
	}
	if selection.subjectID != nil {
		subject, found := findReportSubject(payload, *selection.subjectID)
		if !found || subject.DepartmentID != departmentID {
			return "", nil, "", fmt.Errorf("subject is outside the selected department")
		}
		args = appendIntArgs(args, "--subject-ids", []int32{*selection.subjectID})
	}
	args = appendIntArgs(args, "--group-ids", selection.groupIDs)
	return "department_head_report.py", args, fmt.Sprintf("department_%d", departmentID), nil
}

func (s *Server) buildScriptedPerformanceReport(c *fiber.Ctx) ([]byte, string, error) {
	token := c.Get("Authorization")
	if token == "" {
		return nil, "", c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}
	semesterID, err := optionalSemesterIDFromQuery(c)
	if err != nil {
		return nil, "", c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: err.Error()})
	}
	selection, err := reportSelectionFromQuery(c)
	if err != nil {
		return nil, "", c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: err.Error()})
	}
	payload, resp := s.allGeneralRatingPages(token, semesterID)
	if payload == nil {
		return nil, "", c.Status(generalRatingHTTPStatus(resp)).JSON(resp)
	}

	tempDir, err := os.MkdirTemp("", "ejournal-report-*")
	if err != nil {
		return nil, "", c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to prepare report"})
	}
	defer os.RemoveAll(tempDir)
	inputPath := filepath.Join(tempDir, "metrics.json")
	outputPath := filepath.Join(tempDir, "report.xlsx")
	raw, err := json.Marshal(payload)
	if err != nil || os.WriteFile(inputPath, raw, 0o600) != nil {
		return nil, "", c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to prepare report data"})
	}

	script, args, namePart, err := scriptedReportCommand(payload, selection, inputPath, outputPath)
	if err != nil {
		return nil, "", c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: err.Error()})
	}
	pythonBin := strings.TrimSpace(os.Getenv("REPORT_PYTHON_BIN"))
	if pythonBin == "" {
		pythonBin = "python3"
	}
	scriptDir := strings.TrimSpace(os.Getenv("REPORT_SCRIPTS_DIR"))
	if scriptDir == "" {
		scriptDir = "report_scripts"
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 45*time.Second)
	defer cancel()
	commandArgs := append([]string{filepath.Join(scriptDir, script)}, args...)
	command := exec.CommandContext(ctx, pythonBin, commandArgs...)
	if output, runErr := command.CombinedOutput(); runErr != nil {
		log.Printf("scripted report failed: %v: %s", runErr, strings.TrimSpace(string(output)))
		return nil, "", c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to build xlsx report"})
	}
	content, err := os.ReadFile(outputPath)
	if err != nil || len(content) == 0 {
		return nil, "", c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "xlsx report was not created"})
	}
	filename := fmt.Sprintf("performance_%s_semester_%d_%s.xlsx", namePart, payload.Semester.SemesterID, time.Now().Format("2006-01-02"))
	return content, filename, nil
}
