package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const generalRatingSchemaVersion = "1.0"

type GeneralRatingSemester struct {
	SemesterID int32  `json:"semester_id"`
	Title      string `json:"title"`
}

type GeneralRatingAccessControl struct {
	Role    string `json:"role"`
	UserID  int32  `json:"user_id"`
	Allowed bool   `json:"allowed"`
}

type GeneralRatingDepartment struct {
	DepartmentID   int32  `json:"department_id"`
	DepartmentName string `json:"department_name"`
}

type GeneralRatingSubject struct {
	SubjectID      int32  `json:"subject_id"`
	Name           string `json:"name"`
	ShortName      string `json:"short_name"`
	TeacherID      int32  `json:"teacher_id"`
	Teacher        string `json:"teacher"`
	DepartmentID   int32  `json:"department_id"`
	DepartmentName string `json:"department_name"`
}

type GeneralRatingConsent struct {
	Accepted bool `json:"accepted"`
}

type GeneralRatingAttendanceActivity struct {
	Date             string `json:"date"`
	AttendanceStatus string `json:"attendance_status"`
	AttendanceCode   string `json:"attendance_code"`
}

type GeneralRatingGradedActivity struct {
	Date     string `json:"date"`
	Title    string `json:"title"`
	Score    *int32 `json:"score" extensions:"x-nullable"`
	MaxScore int32  `json:"max_score"`
}

type GeneralRatingActivities struct {
	Lectures        []GeneralRatingAttendanceActivity `json:"lectures"`
	LaboratoryWorks []GeneralRatingGradedActivity     `json:"laboratory_works"`
	Practices       []GeneralRatingGradedActivity     `json:"practices"`
}

type GeneralRatingAssessmentSummary struct {
	PerformancePercent float64 `json:"performance_percent"`
}

type GeneralRatingAttendanceSummary struct {
	AttendancePercent float64 `json:"attendance_percent"`
}

type GeneralRatingStudentSubject struct {
	SubjectID         int32                          `json:"subject_id"`
	Activities        GeneralRatingActivities        `json:"activities"`
	AssessmentSummary GeneralRatingAssessmentSummary `json:"assessment_summary"`
	AttendanceSummary GeneralRatingAttendanceSummary `json:"attendance_summary"`
}

type GeneralRatingStudent struct {
	StudentRef          string                        `json:"student_ref"`
	StudentLabel        string                        `json:"student_label"`
	IsCurrentUser       bool                          `json:"is_current_user"`
	PersonalDataConsent GeneralRatingConsent          `json:"personal_data_consent"`
	Subjects            []GeneralRatingStudentSubject `json:"subjects"`
}

type GeneralRatingGroup struct {
	GroupID   int32                  `json:"group_id"`
	GroupName string                 `json:"group_name"`
	Students  []GeneralRatingStudent `json:"students"`
}

type GeneralRatingPagination struct {
	Page       int32 `json:"page"`
	PageSize   int32 `json:"page_size"`
	Total      int32 `json:"total"`
	TotalPages int32 `json:"total_pages"`
}

type GeneralRatingPayload struct {
	SchemaVersion string                     `json:"schema_version"`
	Semester      GeneralRatingSemester      `json:"semester"`
	AccessControl GeneralRatingAccessControl `json:"access_control"`
	Departments   []GeneralRatingDepartment  `json:"departments"`
	Subjects      []GeneralRatingSubject     `json:"subjects"`
	Groups        []GeneralRatingGroup       `json:"groups"`
	Pagination    GeneralRatingPagination    `json:"pagination"`
}

type generalRatingStudentSubjectBuilder struct {
	value             GeneralRatingStudentSubject
	passedScore       int64
	passedMax         int64
	attendancePresent int64
	attendanceCounted int64
}

type generalRatingStudentBuilder struct {
	id           int32
	label        string
	consent      bool
	subjects     map[int32]*generalRatingStudentSubjectBuilder
	subjectOrder []int32
}

type generalRatingGroupBuilder struct {
	id           int32
	name         string
	students     map[int32]*generalRatingStudentBuilder
	studentOrder []int32
}

func newGeneralRatingStudentSubject(subjectID int32) *generalRatingStudentSubjectBuilder {
	return &generalRatingStudentSubjectBuilder{value: GeneralRatingStudentSubject{
		SubjectID: subjectID,
		Activities: GeneralRatingActivities{
			Lectures:        make([]GeneralRatingAttendanceActivity, 0),
			LaboratoryWorks: make([]GeneralRatingGradedActivity, 0),
			Practices:       make([]GeneralRatingGradedActivity, 0),
		},
	}}
}

func (s *Service) studentReference(studentID int32) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	_, _ = fmt.Fprintf(mac, "general-rating-student:%d", studentID)
	return "STU-" + strings.ToUpper(hex.EncodeToString(mac.Sum(nil)[:6]))
}

func attendanceCode(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "absent":
		return "О"
	case "late":
		return "ОП"
	case "excused":
		return "У"
	default:
		return ""
	}
}

func ratingActivityKind(itemType string) string {
	normalized := strings.ToLower(strings.TrimSpace(itemType))
	switch {
	case strings.Contains(normalized, "lab"), strings.Contains(normalized, "лабор"):
		return "laboratory"
	case strings.Contains(normalized, "pract"), strings.Contains(normalized, "практ"):
		return "practice"
	default:
		return ""
	}
}

func ratingPercent(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return math.Round((1000*float64(numerator))/float64(denominator)) / 10
}

// generalRatingScopePredicate returns a schedule predicate and its arguments.
// The first argument is always the selected semester ID, so the predicate may
// safely start using placeholders at $2.
func (s *Service) generalRatingScopePredicate(
	ctx context.Context,
	user User,
	semesterID int32,
) (string, []any, int32, error) {
	args := []any{semesterID}
	if user.Role == RoleStudent {
		var studentID, groupID int32
		err := s.store.Pool().QueryRow(ctx, `
			SELECT student_id, group_id
			FROM students
			WHERE user_id = $1 OR student_id = $1
			ORDER BY CASE WHEN user_id = $1 THEN 0 ELSE 1 END
			LIMIT 1`, user.ID).Scan(&studentID, &groupID)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, 0, fmt.Errorf("student profile not found")
		}
		if err != nil {
			return "", nil, 0, fmt.Errorf("failed to load student profile")
		}
		return "sch.group_id = $2", append(args, groupID), studentID, nil
	}
	if user.Role == RoleTeacher {
		teacherID, err := s.teacherIDForUser(ctx, user.ID)
		if err != nil {
			return "", nil, 0, fmt.Errorf("failed to resolve teacher profile")
		}
		return "sch.teacher_id = $2", append(args, teacherID), 0, nil
	}

	scope, err := s.scopeForUser(ctx, user)
	if err != nil {
		return "", nil, 0, err
	}
	if scope.All {
		return "TRUE", args, 0, nil
	}
	return "g.lectern_id = ANY($2)", append(args, nonNil(scope.LecternIDs)), 0, nil
}

// GeneralRating returns the detailed, role-scoped source data used to build a
// common student rating. The HTTP handler places this value in Response.result.
func (s *Service) GeneralRating(token string, semesterID *int32, page, pageSize int32) (*GeneralRatingPayload, Response) {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return nil, Response{OK: false, Error: err.Error()}
	}
	ctx, cancel := s.dbContext()
	defer cancel()

	semester, err := s.semesterForOptionalID(ctx, semesterID)
	if err != nil {
		return nil, Response{OK: false, Error: err.Error()}
	}
	predicate, queryArgs, currentStudentID, err := s.generalRatingScopePredicate(ctx, user, semester.ID)
	if err != nil {
		return nil, Response{OK: false, Error: err.Error()}
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	var totalGroups int32
	if err := s.store.Pool().QueryRow(ctx, `
		SELECT COUNT(DISTINCT g.group_id)::INTEGER
		FROM schedules sch JOIN groups g ON g.group_id = sch.group_id
		WHERE sch.semester_id = $1 AND `+predicate, queryArgs...).Scan(&totalGroups); err != nil {
		return nil, Response{OK: false, Error: "failed to count rating groups"}
	}
	totalPages := int32(0)
	if totalGroups > 0 {
		totalPages = (totalGroups + pageSize - 1) / pageSize
	}

	payload := &GeneralRatingPayload{
		SchemaVersion: generalRatingSchemaVersion,
		Semester: GeneralRatingSemester{
			SemesterID: semester.ID,
			Title:      semester.Name,
		},
		AccessControl: GeneralRatingAccessControl{
			Role:    user.Role,
			UserID:  user.ID,
			Allowed: true,
		},
		Departments: make([]GeneralRatingDepartment, 0),
		Subjects:    make([]GeneralRatingSubject, 0),
		Groups:      make([]GeneralRatingGroup, 0),
		Pagination: GeneralRatingPagination{
			Page: page, PageSize: pageSize, Total: totalGroups, TotalPages: totalPages,
		},
	}

	departmentByID := make(map[int32]GeneralRatingDepartment)

	subjectRows, err := s.store.Pool().Query(ctx, `
		SELECT DISTINCT ON (sub.subject_id)
		       sub.subject_id,
		       COALESCE(sub.name, ''),
		       COALESCE(sub.subject_index, ''),
		       t.teacher_id,
		       COALESCE(t.name, ''),
		       COALESCE(sub_l.lectern_id, teacher_l.lectern_id, group_l.lectern_id, 0),
		       COALESCE(sub_l.name, teacher_l.name, group_l.name, '')
		FROM schedules sch
		JOIN groups g ON g.group_id = sch.group_id
		JOIN subjects sub ON sub.subject_id = sch.subject_id
		JOIN teachers t ON t.teacher_id = sch.teacher_id
		LEFT JOIN lecterns sub_l ON sub_l.lectern_id = sub.lectern_id
		LEFT JOIN lecterns teacher_l ON teacher_l.lectern_id = t.lectern_id
		LEFT JOIN lecterns group_l ON group_l.lectern_id = g.lectern_id
		WHERE sch.semester_id = $1 AND `+predicate+`
		ORDER BY sub.subject_id, t.name, t.teacher_id`, queryArgs...)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to load rating subjects"}
	}
	for subjectRows.Next() {
		var subject GeneralRatingSubject
		if err := subjectRows.Scan(
			&subject.SubjectID,
			&subject.Name,
			&subject.ShortName,
			&subject.TeacherID,
			&subject.Teacher,
			&subject.DepartmentID,
			&subject.DepartmentName,
		); err != nil {
			subjectRows.Close()
			return nil, Response{OK: false, Error: "failed to scan rating subjects"}
		}
		payload.Subjects = append(payload.Subjects, subject)
		if subject.DepartmentID > 0 {
			departmentByID[subject.DepartmentID] = GeneralRatingDepartment{
				DepartmentID:   subject.DepartmentID,
				DepartmentName: subject.DepartmentName,
			}
		}
	}
	if err := subjectRows.Err(); err != nil {
		subjectRows.Close()
		return nil, Response{OK: false, Error: "failed to iterate rating subjects"}
	}
	subjectRows.Close()
	sort.Slice(payload.Subjects, func(i, j int) bool {
		if payload.Subjects[i].Name != payload.Subjects[j].Name {
			return payload.Subjects[i].Name < payload.Subjects[j].Name
		}
		return payload.Subjects[i].SubjectID < payload.Subjects[j].SubjectID
	})

	groupBuilders := make(map[int32]*generalRatingGroupBuilder)
	groupOrder := make([]int32, 0)
	groupQueryArgs := append(append([]any{}, queryArgs...), pageSize, (page-1)*pageSize)
	limitArg := len(queryArgs) + 1
	offsetArg := limitArg + 1
	groupRows, err := s.store.Pool().Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT g.group_id, COALESCE(g.group_name, ''),
		       COALESCE(l.lectern_id, 0), COALESCE(l.name, '')
		FROM schedules sch
		JOIN groups g ON g.group_id = sch.group_id
		LEFT JOIN lecterns l ON l.lectern_id = g.lectern_id
		WHERE sch.semester_id = $1 AND `+predicate+`
		ORDER BY COALESCE(g.group_name, ''), g.group_id
		LIMIT $%d OFFSET $%d`, limitArg, offsetArg), groupQueryArgs...)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to load rating groups"}
	}
	for groupRows.Next() {
		var groupID, departmentID int32
		var groupName, departmentName string
		if err := groupRows.Scan(&groupID, &groupName, &departmentID, &departmentName); err != nil {
			groupRows.Close()
			return nil, Response{OK: false, Error: "failed to scan rating groups"}
		}
		groupBuilders[groupID] = &generalRatingGroupBuilder{
			id:       groupID,
			name:     groupName,
			students: make(map[int32]*generalRatingStudentBuilder),
		}
		groupOrder = append(groupOrder, groupID)
		if departmentID > 0 {
			departmentByID[departmentID] = GeneralRatingDepartment{
				DepartmentID:   departmentID,
				DepartmentName: departmentName,
			}
		}
	}
	if err := groupRows.Err(); err != nil {
		groupRows.Close()
		return nil, Response{OK: false, Error: "failed to iterate rating groups"}
	}
	groupRows.Close()
	dataArgs := append(append([]any{}, queryArgs...), groupOrder)
	groupIDsArg := len(dataArgs)
	dataPredicate := fmt.Sprintf("%s AND sch.group_id = ANY($%d)", predicate, groupIDsArg)

	agreementKeyArg := len(dataArgs) + 1
	agreementVersionArg := agreementKeyArg + 1
	studentArgs := append(append([]any{}, dataArgs...), userAgreementKey, currentAgreementVersion)
	studentRows, err := s.store.Pool().Query(ctx, fmt.Sprintf(`
		WITH latest_consent AS (
		    SELECT DISTINCT ON (user_id) user_id, decision
		    FROM user_agreement_decisions
		    WHERE agreement_key = $%d AND version = $%d
		    ORDER BY user_id, decided_at DESC, decision_id DESC
		)
		SELECT DISTINCT g.group_id, st.student_id, COALESCE(st.student_name, ''),
		       sub.subject_id, COALESCE(lc.decision = 'accepted', FALSE)
		FROM schedules sch
		JOIN groups g ON g.group_id = sch.group_id
		JOIN students st ON st.group_id = g.group_id
		JOIN subjects sub ON sub.subject_id = sch.subject_id
		LEFT JOIN latest_consent lc ON lc.user_id = st.user_id
		WHERE sch.semester_id = $1 AND %s
		ORDER BY g.group_id, st.student_id, sub.subject_id`, agreementKeyArg, agreementVersionArg, dataPredicate), studentArgs...)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to load rating students"}
	}
	for studentRows.Next() {
		var groupID, studentID, subjectID int32
		var studentName string
		var consent bool
		if err := studentRows.Scan(&groupID, &studentID, &studentName, &subjectID, &consent); err != nil {
			studentRows.Close()
			return nil, Response{OK: false, Error: "failed to scan rating students"}
		}
		group := groupBuilders[groupID]
		if group == nil {
			continue
		}
		student := group.students[studentID]
		if student == nil {
			student = &generalRatingStudentBuilder{
				id:       studentID,
				label:    studentName,
				consent:  consent,
				subjects: make(map[int32]*generalRatingStudentSubjectBuilder),
			}
			group.students[studentID] = student
			group.studentOrder = append(group.studentOrder, studentID)
		}
		if student.subjects[subjectID] == nil {
			student.subjects[subjectID] = newGeneralRatingStudentSubject(subjectID)
			student.subjectOrder = append(student.subjectOrder, subjectID)
		}
	}
	if err := studentRows.Err(); err != nil {
		studentRows.Close()
		return nil, Response{OK: false, Error: "failed to iterate rating students"}
	}
	studentRows.Close()

	attendanceRows, err := s.store.Pool().Query(ctx, `
		WITH scoped_assignments AS (
		    SELECT DISTINCT sch.group_id, sch.subject_id, sch.teacher_id
		    FROM schedules sch
		    JOIN groups g ON g.group_id = sch.group_id
		    WHERE sch.semester_id = $1 AND `+dataPredicate+`
		)
		SELECT ass.group_id_snapshot, ass.student_id, ats.subject_id, ats.created_at, ass.status
		FROM attendance_session_students ass
		JOIN attendance_sessions ats ON ats.session_id = ass.session_id
		JOIN scoped_assignments sa
		  ON sa.group_id = ass.group_id_snapshot
		 AND sa.subject_id = ats.subject_id
		 AND sa.teacher_id = ats.teacher_id
		WHERE ats.semester_id = $1
		ORDER BY ass.group_id_snapshot, ass.student_id, ats.subject_id, ats.created_at, ats.session_id`, dataArgs...)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to load rating attendance"}
	}
	for attendanceRows.Next() {
		var groupID, studentID, subjectID int32
		var occurredAt time.Time
		var status string
		if err := attendanceRows.Scan(&groupID, &studentID, &subjectID, &occurredAt, &status); err != nil {
			attendanceRows.Close()
			return nil, Response{OK: false, Error: "failed to scan rating attendance"}
		}
		subject := ratingSubjectBuilder(groupBuilders, groupID, studentID, subjectID)
		if subject == nil {
			continue
		}
		normalizedStatus := strings.ToLower(strings.TrimSpace(status))
		subject.value.Activities.Lectures = append(subject.value.Activities.Lectures, GeneralRatingAttendanceActivity{
			Date:             occurredAt.In(appTimeLocation).Format("2006-01-02"),
			AttendanceStatus: normalizedStatus,
			AttendanceCode:   attendanceCode(normalizedStatus),
		})
		if normalizedStatus != "excused" {
			subject.attendanceCounted++
			if normalizedStatus == "present" || normalizedStatus == "late" {
				subject.attendancePresent++
			}
		}
	}
	if err := attendanceRows.Err(); err != nil {
		attendanceRows.Close()
		return nil, Response{OK: false, Error: "failed to iterate rating attendance"}
	}
	attendanceRows.Close()

	gradeRows, err := s.store.Pool().Query(ctx, `
		WITH scoped_assignments AS (
		    SELECT DISTINCT sch.group_id, sch.subject_id
		    FROM schedules sch
		    JOIN groups g ON g.group_id = sch.group_id
		    WHERE sch.semester_id = $1 AND `+dataPredicate+`
		)
		SELECT sa.group_id, st.student_id, sa.subject_id,
		       gi.item_id, gi.title, gi.max_score, gi.item_type,
		       gi.deadline, gi.created_at, g.score
		FROM scoped_assignments sa
		JOIN students st ON st.group_id = sa.group_id
		LEFT JOIN grade_items gi
		  ON gi.subject_id = sa.subject_id
		 AND gi.semester_id = $1
		 AND gi.deleted_at IS NULL
		LEFT JOIN grades g
		  ON g.item_id = gi.item_id
		 AND g.student_id = st.student_id
		 AND g.deleted_at IS NULL
		ORDER BY sa.group_id, st.student_id, sa.subject_id,
		         gi.deadline NULLS LAST, gi.created_at, gi.item_id`, dataArgs...)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to load rating grades"}
	}
	now := time.Now()
	for gradeRows.Next() {
		var groupID, studentID, subjectID int32
		var itemID, maxScore, score sql.NullInt32
		var title, itemType sql.NullString
		var deadline, createdAt sql.NullTime
		if err := gradeRows.Scan(
			&groupID,
			&studentID,
			&subjectID,
			&itemID,
			&title,
			&maxScore,
			&itemType,
			&deadline,
			&createdAt,
			&score,
		); err != nil {
			gradeRows.Close()
			return nil, Response{OK: false, Error: "failed to scan rating grades"}
		}
		subject := ratingSubjectBuilder(groupBuilders, groupID, studentID, subjectID)
		if subject == nil || !itemID.Valid {
			continue
		}
		if deadline.Valid && deadline.Time.Before(now) {
			subject.passedMax += int64(maxScore.Int32)
			if score.Valid {
				subject.passedScore += int64(score.Int32)
			}
		}

		kind := ratingActivityKind(itemType.String)
		if kind == "" {
			continue
		}
		activityDate := createdAt.Time
		if deadline.Valid {
			activityDate = deadline.Time
		}
		var activityScore *int32
		if score.Valid {
			value := score.Int32
			activityScore = &value
		}
		activity := GeneralRatingGradedActivity{
			Date:     activityDate.In(appTimeLocation).Format("2006-01-02"),
			Title:    title.String,
			Score:    activityScore,
			MaxScore: maxScore.Int32,
		}
		if kind == "laboratory" {
			subject.value.Activities.LaboratoryWorks = append(subject.value.Activities.LaboratoryWorks, activity)
		} else {
			subject.value.Activities.Practices = append(subject.value.Activities.Practices, activity)
		}
	}
	if err := gradeRows.Err(); err != nil {
		gradeRows.Close()
		return nil, Response{OK: false, Error: "failed to iterate rating grades"}
	}
	gradeRows.Close()

	for _, department := range departmentByID {
		payload.Departments = append(payload.Departments, department)
	}
	sort.Slice(payload.Departments, func(i, j int) bool {
		if payload.Departments[i].DepartmentName != payload.Departments[j].DepartmentName {
			return payload.Departments[i].DepartmentName < payload.Departments[j].DepartmentName
		}
		return payload.Departments[i].DepartmentID < payload.Departments[j].DepartmentID
	})

	for _, groupID := range groupOrder {
		builder := groupBuilders[groupID]
		group := GeneralRatingGroup{
			GroupID:   builder.id,
			GroupName: builder.name,
			Students:  make([]GeneralRatingStudent, 0, len(builder.studentOrder)),
		}
		for _, studentID := range builder.studentOrder {
			studentBuilder := builder.students[studentID]
			studentRef := s.studentReference(studentBuilder.id)
			studentLabel := studentBuilder.label
			if !studentBuilder.consent {
				studentLabel = studentRef
			}
			student := GeneralRatingStudent{
				StudentRef:          studentRef,
				StudentLabel:        studentLabel,
				IsCurrentUser:       currentStudentID > 0 && studentBuilder.id == currentStudentID,
				PersonalDataConsent: GeneralRatingConsent{Accepted: studentBuilder.consent},
				Subjects:            make([]GeneralRatingStudentSubject, 0, len(studentBuilder.subjectOrder)),
			}
			for _, subjectID := range studentBuilder.subjectOrder {
				subjectBuilder := studentBuilder.subjects[subjectID]
				subjectBuilder.value.AssessmentSummary.PerformancePercent = ratingPercent(
					subjectBuilder.passedScore,
					subjectBuilder.passedMax,
				)
				subjectBuilder.value.AttendanceSummary.AttendancePercent = ratingPercent(
					subjectBuilder.attendancePresent,
					subjectBuilder.attendanceCounted,
				)
				student.Subjects = append(student.Subjects, subjectBuilder.value)
			}
			group.Students = append(group.Students, student)
		}
		payload.Groups = append(payload.Groups, group)
	}

	return payload, Response{OK: true}
}

func ratingSubjectBuilder(
	groups map[int32]*generalRatingGroupBuilder,
	groupID, studentID, subjectID int32,
) *generalRatingStudentSubjectBuilder {
	group := groups[groupID]
	if group == nil {
		return nil
	}
	student := group.students[studentID]
	if student == nil {
		return nil
	}
	return student.subjects[subjectID]
}
