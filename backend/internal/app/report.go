package app

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
	"golang.org/x/image/font/gofont/goregular"
)

type PerformanceReportSubject struct {
	ID   int32
	Name string
}

type PerformanceReportRow struct {
	StudentID      int32
	StudentName    string
	GroupID        int32
	GroupName      string
	SubjectPercent map[int32]float64
	Rating         float64
	HasRating      bool
	AttendancePct  float64
}

type PerformanceReport struct {
	Label        string
	SemesterID   int32
	SemesterName string
	GeneratedAt  time.Time
	Subjects     []PerformanceReportSubject
	Rows         []PerformanceReportRow
	GroupNames   []string
}

func (s *Service) StaffPerformanceReport(sessionToken string) (*PerformanceReport, Response) {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return nil, Response{OK: false, Error: err.Error()}
	}
	if user.Role != RoleHead && user.Role != RoleDean && user.Role != RoleAdmin {
		return nil, Response{OK: false, Error: "forbidden: head role or higher required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	scope, err := s.scopeForUser(ctx, user)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to resolve scope"}
	}

	semester, err := s.currentSemester(ctx)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to resolve current semester"}
	}

	studentPred, studentArgs := scopeStudentPredicate(scope, "st")
	studentPredSemester := strings.ReplaceAll(studentPred, "$1", "$2")

	report := &PerformanceReport{
		Label:        scope.Label,
		SemesterID:   semester.ID,
		SemesterName: semester.Name,
		GeneratedAt:  time.Now().In(appTimeLocation),
	}

	rowByStudent := map[int32]*PerformanceReportRow{}

	studentRows, err := s.store.Pool().Query(ctx, `
		SELECT st.student_id,
		       COALESCE(st.student_name, ''),
		       COALESCE(g.group_id, 0),
		       COALESCE(g.group_name, ''),
		       COALESCE(100.0 * COUNT(ass.student_id) FILTER (WHERE s.semester_id = $1 AND ass.status IN ('present','late'))
		           / NULLIF(COUNT(ass.student_id) FILTER (WHERE s.semester_id = $1 AND ass.status <> 'excused'), 0), 0)::float8
		FROM students st
		LEFT JOIN groups g ON g.group_id = st.group_id
		LEFT JOIN attendance_session_students ass ON ass.student_id = st.student_id
		LEFT JOIN attendance_sessions s ON s.session_id = ass.session_id
		WHERE `+studentPredSemester+`
		GROUP BY st.student_id, st.student_name, g.group_id, g.group_name`, append([]any{semester.ID}, studentArgs...)...)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to load students"}
	}
	for studentRows.Next() {
		row := PerformanceReportRow{SubjectPercent: map[int32]float64{}}
		if err := studentRows.Scan(&row.StudentID, &row.StudentName, &row.GroupID, &row.GroupName, &row.AttendancePct); err != nil {
			studentRows.Close()
			return nil, Response{OK: false, Error: "failed to scan students"}
		}
		rowByStudent[row.StudentID] = &row
	}
	studentRows.Close()

	subjectSeen := map[int32]string{}

	gradeRows, err := s.store.Pool().Query(ctx, `
		WITH scoped_students AS (
		    SELECT st.student_id, st.group_id
		    FROM students st
		    WHERE `+studentPredSemester+`
		),
		student_subjects AS (
		    SELECT DISTINCT ss.student_id, sch.subject_id
		    FROM scoped_students ss
		    JOIN schedules sch ON sch.group_id = ss.group_id
		)
		SELECT stsub.student_id,
		       sub.subject_id,
		       sub.name,
		       COALESCE(SUM(g.score)      FILTER (WHERE gi.deadline < now()), 0)::float8,
		       COALESCE(SUM(gi.max_score) FILTER (WHERE gi.deadline < now()), 0)::float8
		FROM student_subjects stsub
		JOIN subjects sub ON sub.subject_id = stsub.subject_id
		LEFT JOIN grade_items gi ON gi.subject_id = sub.subject_id AND gi.semester_id = $1
		LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = stsub.student_id AND g.deleted_at IS NULL
		GROUP BY stsub.student_id, sub.subject_id, sub.name`, append([]any{semester.ID}, studentArgs...)...)
	if err != nil {
		return nil, Response{OK: false, Error: "failed to load grades"}
	}
	for gradeRows.Next() {
		var studentID, subjectID int32
		var subjectName string
		var score, max float64
		if err := gradeRows.Scan(&studentID, &subjectID, &subjectName, &score, &max); err != nil {
			gradeRows.Close()
			return nil, Response{OK: false, Error: "failed to scan grades"}
		}
		subjectSeen[subjectID] = subjectName
		if row, ok := rowByStudent[studentID]; ok && max > 0 {
			row.SubjectPercent[subjectID] = 100 * score / max
		}
	}
	gradeRows.Close()

	for id, name := range subjectSeen {
		report.Subjects = append(report.Subjects, PerformanceReportSubject{ID: id, Name: name})
	}
	sort.Slice(report.Subjects, func(i, j int) bool {
		if report.Subjects[i].Name != report.Subjects[j].Name {
			return report.Subjects[i].Name < report.Subjects[j].Name
		}
		return report.Subjects[i].ID < report.Subjects[j].ID
	})

	groupSeen := map[string]struct{}{}
	for _, row := range rowByStudent {
		if len(row.SubjectPercent) > 0 {
			var sum float64
			for _, pct := range row.SubjectPercent {
				sum += pct
			}
			row.Rating = sum / float64(len(row.SubjectPercent))
			row.HasRating = true
		}
		if row.GroupName != "" {
			groupSeen[row.GroupName] = struct{}{}
		}
		report.Rows = append(report.Rows, *row)
	}

	sort.Slice(report.Rows, func(i, j int) bool {
		a, b := report.Rows[i], report.Rows[j]
		if a.HasRating != b.HasRating {
			return a.HasRating
		}
		if a.Rating != b.Rating {
			return a.Rating > b.Rating
		}
		return a.StudentName < b.StudentName
	})

	for name := range groupSeen {
		report.GroupNames = append(report.GroupNames, name)
	}
	sort.Strings(report.GroupNames)

	return report, Response{OK: true}
}

func sanitizeSheetName(name string) string {
	replacer := strings.NewReplacer(":", "", "\\", "", "/", "", "?", "", "*", "", "[", "", "]", "")
	out := replacer.Replace(name)
	if out == "" {
		out = "Группа"
	}
	runes := []rune(out)
	if len(runes) > 31 {
		out = string(runes[:31])
	}
	return out
}

func BuildPerformanceReportXLSX(report *PerformanceReport) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	const streamSheet = "Поток"
	if err := f.SetSheetName("Sheet1", streamSheet); err != nil {
		return nil, err
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4FC3F7"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "000000"},
			{Type: "right", Style: 1, Color: "000000"},
			{Type: "top", Style: 1, Color: "000000"},
			{Type: "bottom", Style: 1, Color: "000000"},
		},
	})
	if err != nil {
		return nil, err
	}
	numberStyle, err := f.NewStyle(&excelize.Style{NumFmt: 2})
	if err != nil {
		return nil, err
	}
	summaryStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, NumFmt: 2})
	if err != nil {
		return nil, err
	}

	writeSheet := func(sheet string, rows []PerformanceReportRow) error {
		headers := []any{"№", "ФИО", "Группа"}
		for _, subj := range report.Subjects {
			headers = append(headers, subj.Name)
		}
		headers = append(headers, "Итоговый рейтинг", "Посещаемость, %")

		if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
			return err
		}
		lastCol, err := excelize.ColumnNumberToName(len(headers))
		if err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, "A1", lastCol+"1", headerStyle); err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, "B", "B", 30); err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, "C", lastCol, 14); err != nil {
			return err
		}
		if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, ActivePane: "bottomLeft"}); err != nil {
			return err
		}

		subjectSums := make([]float64, len(report.Subjects))
		subjectCounts := make([]int, len(report.Subjects))
		var ratingSum, attendanceSum float64
		var ratingCount int

		for i, row := range rows {
			cells := []any{i + 1, row.StudentName, row.GroupName}
			for j, subj := range report.Subjects {
				if pct, ok := row.SubjectPercent[subj.ID]; ok {
					cells = append(cells, pct)
					subjectSums[j] += pct
					subjectCounts[j]++
				} else {
					cells = append(cells, nil)
				}
			}
			if row.HasRating {
				cells = append(cells, row.Rating)
				ratingSum += row.Rating
				ratingCount++
			} else {
				cells = append(cells, nil)
			}
			cells = append(cells, row.AttendancePct)
			attendanceSum += row.AttendancePct

			cellRef := fmt.Sprintf("A%d", i+2)
			if err := f.SetSheetRow(sheet, cellRef, &cells); err != nil {
				return err
			}
		}

		summary := []any{nil, "Средние значения", nil}
		for j := range report.Subjects {
			if subjectCounts[j] > 0 {
				summary = append(summary, subjectSums[j]/float64(subjectCounts[j]))
			} else {
				summary = append(summary, nil)
			}
		}
		if ratingCount > 0 {
			summary = append(summary, ratingSum/float64(ratingCount))
		} else {
			summary = append(summary, nil)
		}
		if len(rows) > 0 {
			summary = append(summary, attendanceSum/float64(len(rows)))
		} else {
			summary = append(summary, nil)
		}
		summaryRow := len(rows) + 2
		if err := f.SetSheetRow(sheet, fmt.Sprintf("A%d", summaryRow), &summary); err != nil {
			return err
		}

		if len(rows) > 0 {
			dataStart := "D2"
			dataEnd := fmt.Sprintf("%s%d", lastCol, len(rows)+1)
			if err := f.SetCellStyle(sheet, dataStart, dataEnd, numberStyle); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("%s%d", lastCol, summaryRow), summaryStyle); err != nil {
				return err
			}
			_ = f.SetConditionalFormat(sheet, dataStart+":"+dataEnd, []excelize.ConditionalFormatOptions{
				{
					Type:     "3_color_scale",
					Criteria: "=",
					MinType:  "num", MinValue: "0", MinColor: "#F8696B",
					MidType: "num", MidValue: "70", MidColor: "#FFEB84",
					MaxType: "num", MaxValue: "100", MaxColor: "#63BE7B",
				},
			})
		}

		return nil
	}

	if err := writeSheet(streamSheet, report.Rows); err != nil {
		return nil, err
	}

	usedNames := map[string]bool{streamSheet: true}
	for _, groupName := range report.GroupNames {
		sheetName := sanitizeSheetName(groupName)
		for i := 2; usedNames[sheetName]; i++ {
			sheetName = sanitizeSheetName(fmt.Sprintf("%s (%d)", groupName, i))
		}
		usedNames[sheetName] = true

		if _, err := f.NewSheet(sheetName); err != nil {
			return nil, err
		}
		var groupRows []PerformanceReportRow
		for _, row := range report.Rows {
			if row.GroupName == groupName {
				groupRows = append(groupRows, row)
			}
		}
		if err := writeSheet(sheetName, groupRows); err != nil {
			return nil, err
		}
	}

	idx, err := f.GetSheetIndex(streamSheet)
	if err != nil {
		return nil, err
	}
	f.SetActiveSheet(idx)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func gradientColor(pct float64) (int, int, int) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	type stop struct{ pos, r, g, b float64 }
	stops := []stop{
		{0, 0xF8, 0x69, 0x6B},
		{70, 0xFF, 0xEB, 0x84},
		{100, 0x63, 0xBE, 0x7B},
	}
	lo, hi := stops[0], stops[1]
	if pct > stops[1].pos {
		lo, hi = stops[1], stops[2]
	}
	t := 0.0
	if span := hi.pos - lo.pos; span > 0 {
		t = (pct - lo.pos) / span
	}
	r := int(lo.r + t*(hi.r-lo.r))
	g := int(lo.g + t*(hi.g-lo.g))
	b := int(lo.b + t*(hi.b-lo.b))
	return r, g, b
}

func BuildPerformanceReportPDF(report *PerformanceReport) (*bytes.Buffer, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", goregular.TTF)
	pdf.SetFont("Go", "", 8)
	pdf.SetAutoPageBreak(true, 15)

	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	usableW := pageW - left - right

	headers := []string{"№", "ФИО", "Группа"}
	for _, subj := range report.Subjects {
		headers = append(headers, subj.Name)
	}
	headers = append(headers, "Рейтинг", "Посещ. %")

	numW, nameW, groupW := 10.0, 55.0, 25.0
	flexCols := len(report.Subjects) + 2
	flexW := (usableW - numW - nameW - groupW) / float64(flexCols)
	if flexW < 16 {
		flexW = 16
	}
	colWidths := []float64{numW, nameW, groupW}
	for i := 0; i < flexCols; i++ {
		colWidths = append(colWidths, flexW)
	}

	const rowHeight = 7.0

	drawHeaderRow := func() {
		pdf.SetFont("Go", "B", 8)
		pdf.SetFillColor(0x4F, 0xC3, 0xF7)
		for i, h := range headers {
			pdf.CellFormat(colWidths[i], rowHeight, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Go", "", 8)
	}

	var currentTitle string
	pdf.SetHeaderFunc(func() {
		if currentTitle != "" {
			pdf.SetFont("Go", "B", 12)
			pdf.CellFormat(0, 8, currentTitle, "", 1, "L", false, 0, "")
			pdf.SetFont("Go", "", 8)
		}
		drawHeaderRow()
	})

	writeSection := func(title string, rows []PerformanceReportRow) {
		currentTitle = title
		pdf.AddPage()
		currentTitle = ""

		for i, row := range rows {
			pdf.CellFormat(colWidths[0], rowHeight, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colWidths[1], rowHeight, row.StudentName, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colWidths[2], rowHeight, row.GroupName, "1", 0, "L", false, 0, "")

			for j, subj := range report.Subjects {
				w := colWidths[3+j]
				if pct, ok := row.SubjectPercent[subj.ID]; ok {
					r, g, b := gradientColor(pct)
					pdf.SetFillColor(r, g, b)
					pdf.CellFormat(w, rowHeight, fmt.Sprintf("%.1f", pct), "1", 0, "C", true, 0, "")
				} else {
					pdf.CellFormat(w, rowHeight, "", "1", 0, "C", false, 0, "")
				}
			}

			ratingIdx := 3 + len(report.Subjects)
			if row.HasRating {
				r, g, b := gradientColor(row.Rating)
				pdf.SetFillColor(r, g, b)
				pdf.CellFormat(colWidths[ratingIdx], rowHeight, fmt.Sprintf("%.1f", row.Rating), "1", 0, "C", true, 0, "")
			} else {
				pdf.CellFormat(colWidths[ratingIdx], rowHeight, "", "1", 0, "C", false, 0, "")
			}

			r, g, b := gradientColor(row.AttendancePct)
			pdf.SetFillColor(r, g, b)
			pdf.CellFormat(colWidths[ratingIdx+1], rowHeight, fmt.Sprintf("%.1f", row.AttendancePct), "1", 0, "C", true, 0, "")
			pdf.Ln(-1)
		}
	}

	writeSection("Поток — "+report.Label, report.Rows)
	for _, groupName := range report.GroupNames {
		var groupRows []PerformanceReportRow
		for _, row := range report.Rows {
			if row.GroupName == groupName {
				groupRows = append(groupRows, row)
			}
		}
		writeSection("Группа "+groupName, groupRows)
	}

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		return nil, err
	}
	return buf, nil
}
