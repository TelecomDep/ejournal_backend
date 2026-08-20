package app

import (
	"strings"
)

type CatalogGroup struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	LecternName string `json:"lectern_name,omitempty"`
}

type CatalogLectern struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	FacultyName string `json:"faculty_name,omitempty"`
}

type CatalogFaculty struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type CatalogTeacher struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	JobTitle    string `json:"job_title,omitempty"`
	LecternName string `json:"lectern_name,omitempty"`
}

type CatalogStudent struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"group_name,omitempty"`
}

type AdminCatalogsResult struct {
	Groups    []CatalogGroup   `json:"groups"`
	Lecterns  []CatalogLectern `json:"lecterns"`
	Faculties []CatalogFaculty `json:"faculties"`
	Teachers  []CatalogTeacher `json:"teachers"`
	Students  []CatalogStudent `json:"students"`
}

func (s *Service) adminListCatalogs(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if user.Role != RoleAdmin && user.Role != RoleMinister && user.Role != RoleDean && user.Role != RoleHead {
		return Response{OK: false, Error: "forbidden"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	result := AdminCatalogsResult{
		Groups:    make([]CatalogGroup, 0),
		Lecterns:  make([]CatalogLectern, 0),
		Faculties: make([]CatalogFaculty, 0),
		Teachers:  make([]CatalogTeacher, 0),
		Students:  make([]CatalogStudent, 0),
	}

	// 1. Fetch Groups
	gRows, err := s.store.Pool().Query(ctx, `
		SELECT g.group_id, COALESCE(g.group_name, ''), COALESCE(l.name, '')
		FROM groups g
		LEFT JOIN lecterns l ON l.lectern_id = g.lectern_id
		ORDER BY g.group_name`)
	if err == nil {
		for gRows.Next() {
			var g CatalogGroup
			if err := gRows.Scan(&g.ID, &g.Name, &g.LecternName); err == nil && strings.TrimSpace(g.Name) != "" {
				result.Groups = append(result.Groups, g)
			}
		}
		gRows.Close()
	}

	// 2. Fetch Lecterns
	lRows, err := s.store.Pool().Query(ctx, `
		SELECT l.lectern_id, COALESCE(l.name, ''), COALESCE(f.name, '')
		FROM lecterns l
		LEFT JOIN faculties f ON f.faculty_id = l.faculty_id
		ORDER BY l.name`)
	if err == nil {
		for lRows.Next() {
			var l CatalogLectern
			if err := lRows.Scan(&l.ID, &l.Name, &l.FacultyName); err == nil && strings.TrimSpace(l.Name) != "" {
				result.Lecterns = append(result.Lecterns, l)
			}
		}
		lRows.Close()
	}

	// 3. Fetch Faculties
	fRows, err := s.store.Pool().Query(ctx, `
		SELECT f.faculty_id, COALESCE(f.name, '')
		FROM faculties f
		ORDER BY f.name`)
	if err == nil {
		for fRows.Next() {
			var f CatalogFaculty
			if err := fRows.Scan(&f.ID, &f.Name); err == nil && strings.TrimSpace(f.Name) != "" {
				result.Faculties = append(result.Faculties, f)
			}
		}
		fRows.Close()
	}

	// 4. Fetch Teachers
	tRows, err := s.store.Pool().Query(ctx, `
		SELECT t.teacher_id, COALESCE(t.name, ''), COALESCE(t.job_title, ''), COALESCE(l.name, '')
		FROM teachers t
		LEFT JOIN lecterns l ON l.lectern_id = t.lectern_id
		ORDER BY t.name`)
	if err == nil {
		for tRows.Next() {
			var t CatalogTeacher
			if err := tRows.Scan(&t.ID, &t.Name, &t.JobTitle, &t.LecternName); err == nil && strings.TrimSpace(t.Name) != "" {
				result.Teachers = append(result.Teachers, t)
			}
		}
		tRows.Close()
	}

	// 5. Fetch Students
	sRows, err := s.store.Pool().Query(ctx, `
		SELECT s.student_id, COALESCE(s.student_name, ''), COALESCE(g.group_name, '')
		FROM students s
		LEFT JOIN groups g ON g.group_id = s.group_id
		ORDER BY s.student_name`)
	if err == nil {
		for sRows.Next() {
			var st CatalogStudent
			if err := sRows.Scan(&st.ID, &st.Name, &st.GroupName); err == nil && strings.TrimSpace(st.Name) != "" {
				result.Students = append(result.Students, st)
			}
		}
		sRows.Close()
	}

	return Response{OK: true, Result: result}
}
