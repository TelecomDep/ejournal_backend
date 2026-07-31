package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func scanInt32Column(rows pgx.Rows) ([]int32, error) {
	var out []int32
	for rows.Next() {
		var v int32
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// VisibilityScope describes what a user is allowed to see. It is derived from
// the user's role and (for head/dean) their org_scopes assignment.
//
//	student -> only own record (handled by the dedicated student endpoints)
//	teacher -> students of the groups they teach (GroupIDs)
//	head    -> teachers + students of their lectern (LecternIDs, one entry)
//	dean    -> everything under their faculty (LecternIDs = lecterns of faculty)
//	admin   -> everything (All = true)
type VisibilityScope struct {
	Role       string
	All        bool
	LecternIDs []int32
	GroupIDs   []int32
	Label      string
}

// scopeForUser resolves the visibility scope for a supervisory user
// (teacher/head/dean/admin). Students are rejected — they use self-scoped
// endpoints only.
func (s *Service) scopeForUser(ctx context.Context, user User) (VisibilityScope, error) {
	switch user.Role {
	case RoleAdmin:
		return VisibilityScope{Role: RoleAdmin, All: true, Label: "Вся организация"}, nil

	case RoleDean:
		facultyIDs, err := s.scopeFacultyIDs(ctx, user.ID)
		if err != nil {
			return VisibilityScope{}, err
		}
		if len(facultyIDs) == 0 {
			return VisibilityScope{Role: RoleDean, Label: "Факультет не назначен"}, nil
		}
		lecternIDs, err := s.lecternIDsByFaculty(ctx, facultyIDs)
		if err != nil {
			return VisibilityScope{}, err
		}
		label, _ := s.facultyLabel(ctx, facultyIDs)
		return VisibilityScope{Role: RoleDean, LecternIDs: lecternIDs, Label: label}, nil

	case RoleHead:
		lecternIDs, err := s.scopeLecternIDs(ctx, user.ID)
		if err != nil {
			return VisibilityScope{}, err
		}
		if len(lecternIDs) == 0 {
			return VisibilityScope{Role: RoleHead, Label: "Кафедра не назначена"}, nil
		}
		label, _ := s.lecternLabel(ctx, lecternIDs)
		return VisibilityScope{Role: RoleHead, LecternIDs: lecternIDs, Label: label}, nil

	case RoleTeacher:
		teacherID, err := s.teacherIDForUser(ctx, user.ID)
		if err != nil {
			return VisibilityScope{}, err
		}
		groupIDs, err := s.teacherGroupIDs(ctx, teacherID)
		if err != nil {
			return VisibilityScope{}, err
		}
		return VisibilityScope{Role: RoleTeacher, GroupIDs: groupIDs, Label: "Мои группы"}, nil

	default:
		return VisibilityScope{}, fmt.Errorf("role %q has no supervisory scope", user.Role)
	}
}

func (s *Service) scopeFacultyIDs(ctx context.Context, userID int32) ([]int32, error) {
	rows, err := s.store.Pool().Query(ctx,
		`SELECT faculty_id FROM org_scopes WHERE user_id = $1 AND faculty_id IS NOT NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt32Column(rows)
}

func (s *Service) scopeLecternIDs(ctx context.Context, userID int32) ([]int32, error) {
	rows, err := s.store.Pool().Query(ctx,
		`SELECT lectern_id FROM org_scopes WHERE user_id = $1 AND lectern_id IS NOT NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt32Column(rows)
}

func (s *Service) lecternIDsByFaculty(ctx context.Context, facultyIDs []int32) ([]int32, error) {
	rows, err := s.store.Pool().Query(ctx,
		`SELECT lectern_id FROM lecterns WHERE faculty_id = ANY($1)`, facultyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt32Column(rows)
}

func (s *Service) teacherIDForUser(ctx context.Context, userID int32) (int32, error) {
	var teacherID int32
	err := s.store.Pool().QueryRow(ctx,
		`SELECT teacher_id FROM teachers
		 WHERE user_id = $1 OR teacher_id = $1
		 ORDER BY CASE WHEN user_id = $1 THEN 0 ELSE 1 END
		 LIMIT 1`, userID).Scan(&teacherID)
	return teacherID, err
}

func (s *Service) teacherGroupIDs(ctx context.Context, teacherID int32) ([]int32, error) {
	rows, err := s.store.Pool().Query(ctx,
		`SELECT DISTINCT sch.group_id
		 FROM schedules sch
		 JOIN semesters sem ON sem.semester_id = sch.semester_id
		 WHERE sch.teacher_id = $1
		   AND sch.group_id IS NOT NULL
		   AND sem.status = 'open'`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt32Column(rows)
}

func (s *Service) facultyLabel(ctx context.Context, facultyIDs []int32) (string, error) {
	var name string
	err := s.store.Pool().QueryRow(ctx,
		`SELECT string_agg(name, ', ') FROM faculties WHERE faculty_id = ANY($1)`, facultyIDs).Scan(&name)
	if name == "" {
		name = "Факультет"
	}
	return "Факультет: " + name, err
}

func (s *Service) lecternLabel(ctx context.Context, lecternIDs []int32) (string, error) {
	var name string
	err := s.store.Pool().QueryRow(ctx,
		`SELECT string_agg(name, ', ') FROM lecterns WHERE lectern_id = ANY($1)`, lecternIDs).Scan(&name)
	if name == "" {
		name = "Кафедра"
	}
	return "Кафедра: " + name, err
}

// --- staff overview payload ---

type staffCounts struct {
	Lecterns int32 `json:"lecterns"`
	Groups   int32 `json:"groups"`
	Teachers int32 `json:"teachers"`
	Students int32 `json:"students"`
}

type staffGroupRow struct {
	GroupID       int32  `json:"group_id"`
	GroupName     string `json:"group_name"`
	LecternName   string `json:"lectern_name"`
	StudentCount  int32  `json:"student_count"`
	AttendancePct int32  `json:"attendance_pct"`
}

type staffTeacherRow struct {
	TeacherID   int32  `json:"teacher_id"`
	Name        string `json:"name"`
	JobTitle    string `json:"job_title"`
	LecternName string `json:"lectern_name"`
}

type staffStudentRow struct {
	StudentID     int32  `json:"student_id"`
	Name          string `json:"name"`
	GroupName     string `json:"group_name"`
	LecternName   string `json:"lectern_name"`
	AttendancePct int32  `json:"attendance_pct"`
}

type staffOverviewResult struct {
	Role     string            `json:"role"`
	Label    string            `json:"label"`
	CanEdit  bool              `json:"can_edit"`
	Counts   staffCounts       `json:"counts"`
	Groups   []staffGroupRow   `json:"groups"`
	Teachers []staffTeacherRow `json:"teachers"`
	Students []staffStudentRow `json:"students"`
}

type StaffStudentsPageData struct {
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	GroupID  *int32 `json:"group_id,omitempty"`
	Search   string `json:"search,omitempty"`
	Sort     string `json:"sort,omitempty"`
	Order    string `json:"order,omitempty"`
}

type staffStudentsPageResult struct {
	Students   []staffStudentRow `json:"students"`
	Page       int32             `json:"page"`
	PageSize   int32             `json:"page_size"`
	Total      int32             `json:"total"`
	TotalPages int32             `json:"total_pages"`
}

// staffOverview builds the supervisory dashboard payload, scoped to what the
// caller is allowed to see.
func (s *Service) staffOverview(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if user.Role == RoleStudent {
		return Response{OK: false, Error: "forbidden"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	scope, err := s.scopeForUser(ctx, user)
	if err != nil {
		return Response{OK: false, Error: "failed to resolve scope"}
	}

	// "where" predicates per entity, parameterised by scope.
	groupPred, groupArgs := scopeGroupPredicate(scope, "g")
	teacherPred, teacherArgs := scopeTeacherPredicate(scope, "t")
	studentPred, studentArgs := scopeStudentPredicate(scope, "s")

	result := staffOverviewResult{
		Role:    user.Role,
		Label:   scope.Label,
		CanEdit: user.Role == RoleAdmin,
	}

	// Aggregate each relation before joining it to groups. Joining students and
	// attendance rows directly would multiply every student by every mark.
	groupRows, err := s.store.Pool().Query(ctx, `
		WITH student_counts AS (
		    SELECT group_id, COUNT(*)::INTEGER AS student_count
		    FROM students
		    GROUP BY group_id
		), attendance_stats AS (
		    SELECT group_id_snapshot AS group_id,
		           COUNT(*) FILTER (WHERE status IN ('present', 'late'))::NUMERIC AS attended_count,
		           COUNT(*) FILTER (WHERE status <> 'excused')::NUMERIC AS counted_count
		    FROM attendance_session_students
		    GROUP BY group_id_snapshot
		)
		SELECT g.group_id, COALESCE(g.group_name, ''), COALESCE(l.name, ''),
		       COALESCE(sc.student_count, 0) AS student_count,
		       COALESCE(ROUND(100.0 * ast.attended_count / NULLIF(ast.counted_count, 0)), 0) AS attendance_pct
		FROM groups g
		LEFT JOIN lecterns l ON l.lectern_id = g.lectern_id
		LEFT JOIN student_counts sc ON sc.group_id = g.group_id
		LEFT JOIN attendance_stats ast ON ast.group_id = g.group_id
		WHERE `+groupPred+`
		ORDER BY l.name, g.group_name`, groupArgs...)
	if err != nil {
		return Response{OK: false, Error: "failed to load groups"}
	}
	for groupRows.Next() {
		var row staffGroupRow
		if err := groupRows.Scan(&row.GroupID, &row.GroupName, &row.LecternName, &row.StudentCount, &row.AttendancePct); err != nil {
			groupRows.Close()
			return Response{OK: false, Error: "failed to scan groups"}
		}
		result.Groups = append(result.Groups, row)
	}
	groupRows.Close()

	// Teachers (only meaningful for head/dean/admin; for teacher returns self).
	teacherRows, err := s.store.Pool().Query(ctx, `
		SELECT t.teacher_id, COALESCE(t.name, ''), COALESCE(t.job_title, ''), COALESCE(l.name, '')
		FROM teachers t
		LEFT JOIN lecterns l ON l.lectern_id = t.lectern_id
		WHERE `+teacherPred+`
		ORDER BY l.name, t.name`, teacherArgs...)
	if err != nil {
		return Response{OK: false, Error: "failed to load teachers"}
	}
	for teacherRows.Next() {
		var row staffTeacherRow
		if err := teacherRows.Scan(&row.TeacherID, &row.Name, &row.JobTitle, &row.LecternName); err != nil {
			teacherRows.Close()
			return Response{OK: false, Error: "failed to scan teachers"}
		}
		result.Teachers = append(result.Teachers, row)
	}
	teacherRows.Close()

	// Students (scoped) with attendance %.
	studentRows, err := s.store.Pool().Query(ctx, `
		SELECT s.student_id, COALESCE(s.student_name, ''), COALESCE(g.group_name, ''), COALESCE(l.name, ''),
		       COALESCE(ROUND(
		           100.0 * COUNT(ass.student_id) FILTER (WHERE ass.status IN ('present','late'))
		           / NULLIF(COUNT(ass.student_id) FILTER (WHERE ass.status <> 'excused'), 0)
		       ), 0) AS attendance_pct
		FROM students s
		LEFT JOIN groups g ON g.group_id = s.group_id
		LEFT JOIN lecterns l ON l.lectern_id = g.lectern_id
		LEFT JOIN attendance_session_students ass ON ass.student_id = s.student_id
		WHERE `+studentPred+`
		GROUP BY s.student_id, s.student_name, g.group_name, l.name
		ORDER BY g.group_name, s.student_name`, studentArgs...)
	if err != nil {
		return Response{OK: false, Error: "failed to load students"}
	}
	for studentRows.Next() {
		var row staffStudentRow
		if err := studentRows.Scan(&row.StudentID, &row.Name, &row.GroupName, &row.LecternName, &row.AttendancePct); err != nil {
			studentRows.Close()
			return Response{OK: false, Error: "failed to scan students"}
		}
		result.Students = append(result.Students, row)
	}
	studentRows.Close()

	result.Counts = staffCounts{
		Groups:   int32(len(result.Groups)),
		Teachers: int32(len(result.Teachers)),
		Students: int32(len(result.Students)),
		Lecterns: int32(countDistinctLecterns(result)),
	}

	return Response{OK: true, Result: result}
}

func normalizeStaffStudentsPage(data StaffStudentsPageData) StaffStudentsPageData {
	if data.Page < 1 {
		data.Page = 1
	}
	if data.PageSize < 1 {
		data.PageSize = 100
	}
	if data.PageSize > 200 {
		data.PageSize = 200
	}
	data.Search = strings.TrimSpace(data.Search)
	data.Sort = strings.ToLower(strings.TrimSpace(data.Sort))
	switch data.Sort {
	case "name", "group", "attendance":
	default:
		data.Sort = "name"
	}
	data.Order = strings.ToLower(strings.TrimSpace(data.Order))
	if data.Order != "desc" {
		data.Order = "asc"
	}
	if data.GroupID != nil && *data.GroupID <= 0 {
		data.GroupID = nil
	}
	return data
}

func (s *Service) staffStudentsPage(token string, data StaffStudentsPageData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if user.Role == RoleStudent {
		return Response{OK: false, Error: "forbidden"}
	}
	data = normalizeStaffStudentsPage(data)

	ctx, cancel := s.dbContext()
	defer cancel()
	scope, err := s.scopeForUser(ctx, user)
	if err != nil {
		return Response{OK: false, Error: "failed to resolve scope"}
	}

	studentPred, args := scopeStudentPredicate(scope, "s")
	conditions := []string{studentPred}
	if data.GroupID != nil {
		conditions = append(conditions, fmt.Sprintf("s.group_id = $%d", len(args)+1))
		args = append(args, *data.GroupID)
	}
	if data.Search != "" {
		conditions = append(conditions, fmt.Sprintf("s.student_name ILIKE $%d", len(args)+1))
		args = append(args, "%"+data.Search+"%")
	}
	where := strings.Join(conditions, " AND ")

	var total int32
	if err := s.store.Pool().QueryRow(ctx, `SELECT COUNT(*)::INTEGER FROM students s WHERE `+where, args...).Scan(&total); err != nil {
		return Response{OK: false, Error: "failed to count students"}
	}

	sortColumn := map[string]string{
		"name":       "s.student_name",
		"group":      "g.group_name",
		"attendance": "attendance_pct",
	}[data.Sort]
	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, data.PageSize, (data.Page-1)*data.PageSize)
	limitArg := len(queryArgs) - 1
	offsetArg := len(queryArgs)
	rows, err := s.store.Pool().Query(ctx, `
		WITH attendance_stats AS (
		    SELECT student_id,
		           COALESCE(ROUND(
		               100.0 * COUNT(*) FILTER (WHERE status IN ('present', 'late'))
		               / NULLIF(COUNT(*) FILTER (WHERE status <> 'excused'), 0)
		           ), 0)::INTEGER AS attendance_pct
		    FROM attendance_session_students
		    GROUP BY student_id
		)
		SELECT s.student_id, COALESCE(s.student_name, ''), COALESCE(g.group_name, ''), COALESCE(l.name, ''),
		       COALESCE(ast.attendance_pct, 0)
		FROM students s
		LEFT JOIN groups g ON g.group_id = s.group_id
		LEFT JOIN lecterns l ON l.lectern_id = g.lectern_id
		LEFT JOIN attendance_stats ast ON ast.student_id = s.student_id
		WHERE `+where+`
		ORDER BY `+sortColumn+` `+data.Order+`, s.student_id
		LIMIT $`+fmt.Sprint(limitArg)+` OFFSET $`+fmt.Sprint(offsetArg), queryArgs...)
	if err != nil {
		return Response{OK: false, Error: "failed to load students"}
	}
	defer rows.Close()

	result := staffStudentsPageResult{Students: make([]staffStudentRow, 0, data.PageSize), Page: data.Page, PageSize: data.PageSize, Total: total}
	if total > 0 {
		result.TotalPages = (total + data.PageSize - 1) / data.PageSize
	}
	for rows.Next() {
		var student staffStudentRow
		if err := rows.Scan(&student.StudentID, &student.Name, &student.GroupName, &student.LecternName, &student.AttendancePct); err != nil {
			return Response{OK: false, Error: "failed to scan students"}
		}
		result.Students = append(result.Students, student)
	}
	if err := rows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate students"}
	}
	return Response{OK: true, Result: result}
}

// scopeGroupPredicate builds a WHERE fragment + args restricting groups
// (aliased `alias`) to the given scope.
func scopeGroupPredicate(scope VisibilityScope, alias string) (string, []any) {
	switch {
	case scope.All:
		return "TRUE", nil
	case scope.Role == RoleTeacher:
		return fmt.Sprintf("%s.group_id = ANY($1)", alias), []any{nonNil(scope.GroupIDs)}
	default: // head/dean
		return fmt.Sprintf("%s.lectern_id = ANY($1)", alias), []any{nonNil(scope.LecternIDs)}
	}
}

func scopeTeacherPredicate(scope VisibilityScope, alias string) (string, []any) {
	switch {
	case scope.All:
		return "TRUE", nil
	case scope.Role == RoleTeacher:
		// A teacher only sees themselves in the teacher list.
		return fmt.Sprintf("%s.user_id IS NOT NULL AND %s.teacher_id IN "+
				"(SELECT sch.teacher_id FROM schedules sch "+
				"JOIN semesters sem ON sem.semester_id = sch.semester_id "+
				"WHERE sch.group_id = ANY($1) AND sem.status = 'open')", alias, alias),
			[]any{nonNil(scope.GroupIDs)}
	default: // head/dean
		return fmt.Sprintf("%s.lectern_id = ANY($1)", alias), []any{nonNil(scope.LecternIDs)}
	}
}

func scopeStudentPredicate(scope VisibilityScope, alias string) (string, []any) {
	switch {
	case scope.All:
		return "TRUE", nil
	case scope.Role == RoleTeacher:
		return fmt.Sprintf("%s.group_id = ANY($1)", alias), []any{nonNil(scope.GroupIDs)}
	default: // head/dean — students whose group belongs to an in-scope lectern
		return fmt.Sprintf("%s.group_id IN (SELECT group_id FROM groups WHERE lectern_id = ANY($1))", alias),
			[]any{nonNil(scope.LecternIDs)}
	}
}

func countDistinctLecterns(r staffOverviewResult) int {
	seen := map[string]struct{}{}
	for _, g := range r.Groups {
		if g.LecternName != "" {
			seen[g.LecternName] = struct{}{}
		}
	}
	for _, t := range r.Teachers {
		if t.LecternName != "" {
			seen[t.LecternName] = struct{}{}
		}
	}
	return len(seen)
}

// nonNil returns an empty (non-nil) slice for nil input so that `= ANY($1)`
// receives a valid empty array instead of NULL.
func nonNil(ids []int32) []int32 {
	if ids == nil {
		return []int32{}
	}
	return ids
}
