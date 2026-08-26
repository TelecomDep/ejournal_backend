package app

import (
	"database/sql"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// StaffAnalyticsData contains the optional scope selected by a supervisory
// user. ScopeType is one of faculty, stream, group or student. A stream is
// identified by its display name because streams are optional catalog data.
type StaffAnalyticsData struct {
	SemesterID *int32 `json:"semester_id,omitempty"`
	ScopeType  string `json:"scope_type,omitempty"`
	ScopeID    string `json:"scope_id,omitempty"`
	SubjectID  *int32 `json:"subject_id,omitempty"`
}

type StaffAnalyticsSemester struct {
	SemesterID int32  `json:"semester_id"`
	Title      string `json:"title"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
}

type StaffAnalyticsScope struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

type StaffAnalyticsSummary struct {
	AverageRating      *float64 `json:"average_rating"`
	MedianRating       *float64 `json:"median_rating"`
	P25Rating          *float64 `json:"p25_rating"`
	P75Rating          *float64 `json:"p75_rating"`
	AttendancePercent  *float64 `json:"attendance_percent"`
	RatingSpread       *float64 `json:"rating_spread"`
	GradeCoverage      *float64 `json:"grade_coverage"`
	StudentCount       int32    `json:"student_count"`
	StudentsWithRating int32    `json:"students_with_rating"`
	StudentsWithAttend int32    `json:"students_with_attendance"`
	AtRiskRating       int32    `json:"at_risk_rating"`
	AtRiskAttendance   int32    `json:"at_risk_attendance"`
	AtRiskStudents     int32    `json:"at_risk_students"`
	CriticalRisk       int32    `json:"critical_risk"`
	DueItems           int32    `json:"due_items"`
	GradedItems        int32    `json:"graded_items"`
	AttendedSessions   int32    `json:"attended_sessions"`
	CountedSessions    int32    `json:"counted_sessions"`
	LateSessions       int32    `json:"late_sessions"`
	AbsentSessions     int32    `json:"absent_sessions"`
	ExcusedSessions    int32    `json:"excused_sessions"`
}

type StaffAnalyticsStreamOption struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	GroupCount   int32  `json:"group_count"`
	StudentCount int32  `json:"student_count"`
}

type StaffAnalyticsGroupOption struct {
	ID         int32  `json:"id"`
	Name       string `json:"name"`
	StreamID   string `json:"stream_id"`
	StreamName string `json:"stream_name"`
}

type StaffAnalyticsStudentOption struct {
	ID         int32  `json:"id"`
	Label      string `json:"label"`
	GroupID    int32  `json:"group_id"`
	GroupName  string `json:"group_name"`
	StreamName string `json:"stream_name"`
}

type StaffAnalyticsSubjectOption struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type StaffAnalyticsOptions struct {
	Streams  []StaffAnalyticsStreamOption  `json:"streams"`
	Groups   []StaffAnalyticsGroupOption   `json:"groups"`
	Students []StaffAnalyticsStudentOption `json:"students"`
	Subjects []StaffAnalyticsSubjectOption `json:"subjects"`
}

type StaffAnalyticsGroup struct {
	ID                 int32    `json:"id"`
	Name               string   `json:"name"`
	StreamID           string   `json:"stream_id"`
	StreamName         string   `json:"stream_name"`
	AverageRating      *float64 `json:"average_rating"`
	MedianRating       *float64 `json:"median_rating"`
	P25Rating          *float64 `json:"p25_rating"`
	P75Rating          *float64 `json:"p75_rating"`
	AttendancePercent  *float64 `json:"attendance_percent"`
	StudentCount       int32    `json:"student_count"`
	StudentsWithRating int32    `json:"students_with_rating"`
	AtRiskCount        int32    `json:"at_risk_count"`
	CriticalRisk       int32    `json:"critical_risk"`
}

type StaffAnalyticsStream struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	AverageRating      *float64 `json:"average_rating"`
	MedianRating       *float64 `json:"median_rating"`
	P25Rating          *float64 `json:"p25_rating"`
	P75Rating          *float64 `json:"p75_rating"`
	AttendancePercent  *float64 `json:"attendance_percent"`
	StudentCount       int32    `json:"student_count"`
	GroupCount         int32    `json:"group_count"`
	StudentsWithRating int32    `json:"students_with_rating"`
	AtRiskCount        int32    `json:"at_risk_count"`
	CriticalRisk       int32    `json:"critical_risk"`
}

type StaffAnalyticsStudent struct {
	ID                int32    `json:"id"`
	Label             string   `json:"label"`
	HasConsent        bool     `json:"has_consent"`
	GroupID           int32    `json:"group_id"`
	GroupName         string   `json:"group_name"`
	StreamName        string   `json:"stream_name"`
	OverallRating     *float64 `json:"overall_rating"`
	AttendancePercent *float64 `json:"attendance_percent"`
	Rank              int32    `json:"rank"`
	Percentile        *float64 `json:"percentile"`
	Risk              string   `json:"risk"`
}

type StaffAnalyticsStudentSubject struct {
	ID          int32    `json:"id"`
	Name        string   `json:"name"`
	Rating      *float64 `json:"rating"`
	Attendance  *float64 `json:"attendance"`
	DueItems    int32    `json:"due_items"`
	GradedItems int32    `json:"graded_items"`
	Counted     int32    `json:"counted_sessions"`
	Attended    int32    `json:"attended_sessions"`
	Late        int32    `json:"late_sessions"`
}

type StaffAnalyticsStudentDetail struct {
	StaffAnalyticsStudent
	Subjects []StaffAnalyticsStudentSubject `json:"subjects"`
}

type StaffAnalyticsSubject struct {
	ID                 int32    `json:"id"`
	Name               string   `json:"name"`
	AverageRating      *float64 `json:"average_rating"`
	AverageAttendance  *float64 `json:"average_attendance"`
	StudentsWithRating int32    `json:"students_with_rating"`
	StudentsWithAttend int32    `json:"students_with_attendance"`
	AtRiskCount        int32    `json:"at_risk_count"`
}

type StaffAnalyticsHeatmapPoint struct {
	GroupID     int32    `json:"group_id"`
	GroupName   string   `json:"group_name"`
	SubjectID   int32    `json:"subject_id"`
	SubjectName string   `json:"subject_name"`
	Rating      *float64 `json:"rating"`
	Attendance  *float64 `json:"attendance"`
}

type StaffAnalyticsWeeklyPoint struct {
	WeekStart          string   `json:"week_start"`
	WeekEnd            string   `json:"week_end"`
	AverageRating      *float64 `json:"average_rating"`
	MedianRating       *float64 `json:"median_rating"`
	AttendancePercent  *float64 `json:"attendance_percent"`
	StudentsWithRating int32    `json:"students_with_rating"`
	StudentsWithAttend int32    `json:"students_with_attendance"`
	GradeCoverage      *float64 `json:"grade_coverage"`
}

type StaffAnalyticsDistributionBucket struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int32  `json:"count"`
}

type StaffAnalyticsAttendanceBreakdown struct {
	Present int32 `json:"present"`
	Late    int32 `json:"late"`
	Absent  int32 `json:"absent"`
	Excused int32 `json:"excused"`
}

type StaffAnalyticsResult struct {
	Semester            StaffAnalyticsSemester             `json:"semester"`
	Scope               StaffAnalyticsScope                `json:"scope"`
	Summary             StaffAnalyticsSummary              `json:"summary"`
	Options             StaffAnalyticsOptions              `json:"options"`
	Groups              []StaffAnalyticsGroup              `json:"groups"`
	Streams             []StaffAnalyticsStream             `json:"streams"`
	Students            []StaffAnalyticsStudent            `json:"students"`
	Subjects            []StaffAnalyticsSubject            `json:"subjects"`
	Heatmap             []StaffAnalyticsHeatmapPoint       `json:"heatmap"`
	Weekly              []StaffAnalyticsWeeklyPoint        `json:"weekly"`
	Distribution        []StaffAnalyticsDistributionBucket `json:"distribution"`
	AttendanceBreakdown StaffAnalyticsAttendanceBreakdown  `json:"attendance_breakdown"`
	Student             *StaffAnalyticsStudentDetail       `json:"student,omitempty"`
}

type analyticsGradePoint struct {
	maxScore int64
	score    int64
	hasScore bool
	deadline time.Time
	gradedAt *time.Time
}

type analyticsAttendancePoint struct {
	when   time.Time
	status string
}

type analyticsSubjectCalc struct {
	id         int32
	name       string
	grades     []analyticsGradePoint
	attendance []analyticsAttendancePoint
}

type analyticsStudentCalc struct {
	id         int32
	label      string
	consent    bool
	groupID    int32
	groupName  string
	streamName string
	subjects   map[int32]*analyticsSubjectCalc
}

type analyticsSnapshot struct {
	rating           float64
	hasRating        bool
	attendance       float64
	hasAttendance    bool
	dueItems         int32
	gradedItems      int32
	attendedSessions int32
	countedSessions  int32
	lateSessions     int32
	absentSessions   int32
	excusedSessions  int32
}

type analyticsGroupMeta struct {
	id         int32
	name       string
	streamName string
}

const (
	analyticsScopeFaculty  = "faculty"
	analyticsScopeStream   = "stream"
	analyticsScopeGroup    = "group"
	analyticsScopeStudent  = "student"
	analyticsNoStreamID    = "__none__"
	analyticsNoStreamLabel = "Без потока"
)

func normalizeAnalyticsScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case analyticsScopeStream, analyticsScopeGroup, analyticsScopeStudent:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return analyticsScopeFaculty
	}
}

func analyticsStreamID(name string) string {
	if strings.TrimSpace(name) == "" {
		return analyticsNoStreamID
	}
	return strings.TrimSpace(name)
}

func analyticsStreamLabel(name string) string {
	if strings.TrimSpace(name) == "" {
		return analyticsNoStreamLabel
	}
	return strings.TrimSpace(name)
}

func analyticsRound(value float64) float64 {
	return math.Round(value*10) / 10
}

func analyticsFloat(value float64, ok bool) *float64 {
	if !ok {
		return nil
	}
	result := analyticsRound(value)
	return &result
}

func analyticsMedian(values []float64, percentile float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	if percentile <= 0 {
		return ordered[0], true
	}
	if percentile >= 1 {
		return ordered[len(ordered)-1], true
	}
	position := percentile * float64(len(ordered)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return ordered[lower], true
	}
	weight := position - float64(lower)
	return ordered[lower] + (ordered[upper]-ordered[lower])*weight, true
}

func analyticsRisk(rating, attendance *float64) string {
	if rating != nil && attendance != nil && *rating < 60 && *attendance < 60 {
		return "critical"
	}
	if (rating != nil && *rating < 60) || (attendance != nil && *attendance < 60) {
		return "risk"
	}
	if (rating != nil && *rating < 80) || (attendance != nil && *attendance < 80) {
		return "watch"
	}
	return "stable"
}

func (student *analyticsStudentCalc) snapshot(asOf time.Time, subjectID *int32) analyticsSnapshot {
	result := analyticsSnapshot{}
	var ratingValues []float64
	for id, subject := range student.subjects {
		if subjectID != nil && id != *subjectID {
			continue
		}
		var dueMax, earned int64
		var dueItems, gradedItems int32
		for _, grade := range subject.grades {
			if grade.deadline.IsZero() || grade.deadline.After(asOf) {
				continue
			}
			dueItems++
			dueMax += grade.maxScore
			if grade.hasScore && (grade.gradedAt == nil || !grade.gradedAt.After(asOf)) {
				gradedItems++
				earned += grade.score
			}
		}
		result.dueItems += dueItems
		result.gradedItems += gradedItems
		if dueMax > 0 {
			ratingValues = append(ratingValues, 100*float64(earned)/float64(dueMax))
		}

		for _, attendance := range subject.attendance {
			if attendance.when.After(asOf) {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(attendance.status)) {
			case "excused":
				result.excusedSessions++
			case "present":
				result.attendedSessions++
				result.countedSessions++
			case "late":
				result.attendedSessions++
				result.countedSessions++
				result.lateSessions++
			default:
				result.countedSessions++
				result.absentSessions++
			}
		}
	}
	if len(ratingValues) > 0 {
		var sum float64
		for _, value := range ratingValues {
			sum += value
		}
		result.rating = sum / float64(len(ratingValues))
		result.hasRating = true
	}
	if result.countedSessions > 0 {
		result.attendance = 100 * float64(result.attendedSessions) / float64(result.countedSessions)
		result.hasAttendance = true
	}
	return result
}

func analyticsSummary(students []*analyticsStudentCalc, asOf time.Time, subjectID *int32) StaffAnalyticsSummary {
	result := StaffAnalyticsSummary{StudentCount: int32(len(students))}
	var ratings []float64
	var totalAttended, totalCounted int32
	var totalDue, totalGraded int32
	var ratingSum float64
	var ratingMin, ratingMax float64
	for _, student := range students {
		snapshot := student.snapshot(asOf, subjectID)
		totalAttended += snapshot.attendedSessions
		totalCounted += snapshot.countedSessions
		totalDue += snapshot.dueItems
		totalGraded += snapshot.gradedItems
		result.AttendedSessions += snapshot.attendedSessions
		result.CountedSessions += snapshot.countedSessions
		result.LateSessions += snapshot.lateSessions
		result.AbsentSessions += snapshot.absentSessions
		result.ExcusedSessions += snapshot.excusedSessions
		if snapshot.hasRating {
			result.StudentsWithRating++
			ratings = append(ratings, snapshot.rating)
			ratingSum += snapshot.rating
			if len(ratings) == 1 || snapshot.rating < ratingMin {
				ratingMin = snapshot.rating
			}
			if len(ratings) == 1 || snapshot.rating > ratingMax {
				ratingMax = snapshot.rating
			}
			if snapshot.rating < 60 {
				result.AtRiskRating++
			}
		}
		if snapshot.hasAttendance {
			result.StudentsWithAttend++
			if snapshot.attendance < 60 {
				result.AtRiskAttendance++
			}
		}
		if (snapshot.hasRating && snapshot.rating < 60) || (snapshot.hasAttendance && snapshot.attendance < 60) {
			result.AtRiskStudents++
		}
		if snapshot.hasRating && snapshot.hasAttendance && snapshot.rating < 60 && snapshot.attendance < 60 {
			result.CriticalRisk++
		}
	}
	result.DueItems = totalDue
	result.GradedItems = totalGraded
	if len(ratings) > 0 {
		result.AverageRating = analyticsFloat(ratingSum/float64(len(ratings)), true)
		if value, ok := analyticsMedian(ratings, 0.5); ok {
			result.MedianRating = analyticsFloat(value, true)
		}
		if value, ok := analyticsMedian(ratings, 0.25); ok {
			result.P25Rating = analyticsFloat(value, true)
		}
		if value, ok := analyticsMedian(ratings, 0.75); ok {
			result.P75Rating = analyticsFloat(value, true)
		}
		result.RatingSpread = analyticsFloat(ratingMax-ratingMin, true)
	}
	if totalCounted > 0 {
		result.AttendancePercent = analyticsFloat(100*float64(totalAttended)/float64(totalCounted), true)
	}
	if totalDue > 0 {
		result.GradeCoverage = analyticsFloat(100*float64(totalGraded)/float64(totalDue), true)
	}
	return result
}

func analyticsStudentLabel(service *Service, student *analyticsStudentCalc) string {
	if student.consent && strings.TrimSpace(student.label) != "" {
		return student.label
	}
	return service.studentReference(student.id)
}

func analyticsStudentRank(student *analyticsStudentCalc, pool []*analyticsStudentCalc, asOf time.Time, subjectID *int32) (int32, *float64) {
	type ranked struct {
		student *analyticsStudentCalc
		rating  float64
	}
	items := make([]ranked, 0, len(pool))
	for _, candidate := range pool {
		snapshot := candidate.snapshot(asOf, subjectID)
		if snapshot.hasRating {
			items = append(items, ranked{student: candidate, rating: snapshot.rating})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].rating != items[j].rating {
			return items[i].rating > items[j].rating
		}
		return items[i].student.id < items[j].student.id
	})
	for index, item := range items {
		if item.student.id != student.id {
			continue
		}
		rank := int32(index + 1)
		if len(items) <= 1 {
			return rank, nil
		}
		percentile := 100 * float64(len(items)-index-1) / float64(len(items)-1)
		return rank, analyticsFloat(percentile, true)
	}
	return 0, nil
}

func (s *Service) renderAnalyticsStudent(student *analyticsStudentCalc, rankPool []*analyticsStudentCalc, asOf time.Time, subjectID *int32) StaffAnalyticsStudent {
	snapshot := student.snapshot(asOf, subjectID)
	rating := analyticsFloat(snapshot.rating, snapshot.hasRating)
	attendance := analyticsFloat(snapshot.attendance, snapshot.hasAttendance)
	rank, percentile := analyticsStudentRank(student, rankPool, asOf, subjectID)
	return StaffAnalyticsStudent{
		ID:                student.id,
		Label:             analyticsStudentLabel(s, student),
		HasConsent:        student.consent,
		GroupID:           student.groupID,
		GroupName:         student.groupName,
		StreamName:        analyticsStreamLabel(student.streamName),
		OverallRating:     rating,
		AttendancePercent: attendance,
		Rank:              rank,
		Percentile:        percentile,
		Risk:              analyticsRisk(rating, attendance),
	}
}

func (student *analyticsStudentCalc) renderSubjects(asOf time.Time, subjectID *int32) []StaffAnalyticsStudentSubject {
	ids := make([]int32, 0, len(student.subjects))
	for id := range student.subjects {
		if subjectID == nil || *subjectID == id {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]StaffAnalyticsStudentSubject, 0, len(ids))
	for _, id := range ids {
		subject := student.subjects[id]
		snapshot := (&analyticsStudentCalc{subjects: map[int32]*analyticsSubjectCalc{id: subject}}).snapshot(asOf, &id)
		result = append(result, StaffAnalyticsStudentSubject{
			ID:          subject.id,
			Name:        subject.name,
			Rating:      analyticsFloat(snapshot.rating, snapshot.hasRating),
			Attendance:  analyticsFloat(snapshot.attendance, snapshot.hasAttendance),
			DueItems:    snapshot.dueItems,
			GradedItems: snapshot.gradedItems,
			Counted:     snapshot.countedSessions,
			Attended:    snapshot.attendedSessions,
			Late:        snapshot.lateSessions,
		})
	}
	return result
}

func analyticsGroupStudents(students []*analyticsStudentCalc) map[int32][]*analyticsStudentCalc {
	result := make(map[int32][]*analyticsStudentCalc)
	for _, student := range students {
		result[student.groupID] = append(result[student.groupID], student)
	}
	return result
}

func analyticsStreamStudents(students []*analyticsStudentCalc) map[string][]*analyticsStudentCalc {
	result := make(map[string][]*analyticsStudentCalc)
	for _, student := range students {
		result[analyticsStreamID(student.streamName)] = append(result[analyticsStreamID(student.streamName)], student)
	}
	return result
}

func analyticsGroupValue(meta analyticsGroupMeta, students []*analyticsStudentCalc, asOf time.Time, subjectID *int32) StaffAnalyticsGroup {
	summary := analyticsSummary(students, asOf, subjectID)
	return StaffAnalyticsGroup{
		ID:                 meta.id,
		Name:               meta.name,
		StreamID:           analyticsStreamID(meta.streamName),
		StreamName:         analyticsStreamLabel(meta.streamName),
		AverageRating:      summary.AverageRating,
		MedianRating:       summary.MedianRating,
		P25Rating:          summary.P25Rating,
		P75Rating:          summary.P75Rating,
		AttendancePercent:  summary.AttendancePercent,
		StudentCount:       summary.StudentCount,
		StudentsWithRating: summary.StudentsWithRating,
		AtRiskCount:        analyticsRiskCount(students, asOf, subjectID),
		CriticalRisk:       summary.CriticalRisk,
	}
}

func analyticsStreamValue(id, name string, students []*analyticsStudentCalc, groupCount int32, asOf time.Time, subjectID *int32) StaffAnalyticsStream {
	summary := analyticsSummary(students, asOf, subjectID)
	return StaffAnalyticsStream{
		ID:                 id,
		Name:               name,
		AverageRating:      summary.AverageRating,
		MedianRating:       summary.MedianRating,
		P25Rating:          summary.P25Rating,
		P75Rating:          summary.P75Rating,
		AttendancePercent:  summary.AttendancePercent,
		StudentCount:       summary.StudentCount,
		GroupCount:         groupCount,
		StudentsWithRating: summary.StudentsWithRating,
		AtRiskCount:        analyticsRiskCount(students, asOf, subjectID),
		CriticalRisk:       summary.CriticalRisk,
	}
}

func analyticsRiskCount(students []*analyticsStudentCalc, asOf time.Time, subjectID *int32) int32 {
	var result int32
	for _, student := range students {
		snapshot := student.snapshot(asOf, subjectID)
		if (snapshot.hasRating && snapshot.rating < 60) || (snapshot.hasAttendance && snapshot.attendance < 60) {
			result++
		}
	}
	return result
}

func analyticsSubjectValues(students []*analyticsStudentCalc, subjectID int32, asOf time.Time) ([]float64, []float64, int32) {
	var ratings, attendance []float64
	var atRisk int32
	for _, student := range students {
		snapshot := student.snapshot(asOf, &subjectID)
		if snapshot.hasRating {
			ratings = append(ratings, snapshot.rating)
			if snapshot.rating < 60 {
				atRisk++
			}
		}
		if snapshot.hasAttendance {
			attendance = append(attendance, snapshot.attendance)
		}
	}
	return ratings, attendance, atRisk
}

func analyticsWeekStart(value time.Time) time.Time {
	local := value.In(appTimeLocation)
	day := int(local.Weekday())
	if day == 0 {
		day = 7
	}
	return time.Date(local.Year(), local.Month(), local.Day()-day+1, 0, 0, 0, 0, appTimeLocation)
}

func analyticsWeekly(students []*analyticsStudentCalc, semesterStart, cutoff time.Time, subjectID *int32) []StaffAnalyticsWeeklyPoint {
	if len(students) == 0 || !semesterStart.Before(cutoff) {
		return []StaffAnalyticsWeeklyPoint{}
	}
	result := make([]StaffAnalyticsWeeklyPoint, 0)
	week := analyticsWeekStart(semesterStart)
	for week.Before(cutoff) {
		weekEnd := week.AddDate(0, 0, 7)
		if weekEnd.After(cutoff) {
			weekEnd = cutoff
		}
		periodStart := week
		if periodStart.Before(semesterStart) {
			periodStart = semesterStart
		}
		summary := analyticsSummary(students, weekEnd, subjectID)
		result = append(result, StaffAnalyticsWeeklyPoint{
			WeekStart:          periodStart.In(appTimeLocation).Format("2006-01-02"),
			WeekEnd:            weekEnd.In(appTimeLocation).Format("2006-01-02"),
			AverageRating:      summary.AverageRating,
			MedianRating:       summary.MedianRating,
			AttendancePercent:  summary.AttendancePercent,
			StudentsWithRating: summary.StudentsWithRating,
			StudentsWithAttend: summary.StudentsWithAttend,
			GradeCoverage:      summary.GradeCoverage,
		})
		week = week.AddDate(0, 0, 7)
	}
	return result
}

func analyticsDistribution(students []*analyticsStudentCalc, asOf time.Time, subjectID *int32) []StaffAnalyticsDistributionBucket {
	buckets := []StaffAnalyticsDistributionBucket{
		{Key: "0-39", Label: "0–39%", Count: 0},
		{Key: "40-59", Label: "40–59%", Count: 0},
		{Key: "60-79", Label: "60–79%", Count: 0},
		{Key: "80-100", Label: "80–100%", Count: 0},
	}
	for _, student := range students {
		snapshot := student.snapshot(asOf, subjectID)
		if !snapshot.hasRating {
			continue
		}
		switch {
		case snapshot.rating < 40:
			buckets[0].Count++
		case snapshot.rating < 60:
			buckets[1].Count++
		case snapshot.rating < 80:
			buckets[2].Count++
		default:
			buckets[3].Count++
		}
	}
	return buckets
}

func analyticsStatusBreakdown(students []*analyticsStudentCalc, asOf time.Time, subjectID *int32) StaffAnalyticsAttendanceBreakdown {
	result := StaffAnalyticsAttendanceBreakdown{}
	for _, student := range students {
		for id, subject := range student.subjects {
			if subjectID != nil && id != *subjectID {
				continue
			}
			for _, attendance := range subject.attendance {
				if attendance.when.After(asOf) {
					continue
				}
				switch strings.ToLower(strings.TrimSpace(attendance.status)) {
				case "present":
					result.Present++
				case "late":
					result.Late++
				case "excused":
					result.Excused++
				default:
					result.Absent++
				}
			}
		}
	}
	return result
}

func (s *Service) staffAnalytics(token string, data StaffAnalyticsData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if user.Role == RoleStudent {
		return Response{OK: false, Error: "forbidden: staff role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()
	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	scope, err := s.scopeForUser(ctx, user)
	if err != nil {
		return Response{OK: false, Error: "failed to resolve scope"}
	}

	groupPredicate, groupArgs := scopeGroupPredicate(scope, "g")
	groupRows, err := s.store.Pool().Query(ctx, `
		SELECT g.group_id, COALESCE(g.group_name, ''), COALESCE(g.stream_name, '')
		FROM groups g
		WHERE `+groupPredicate+`
		ORDER BY g.group_name, g.group_id`, groupArgs...)
	if err != nil {
		return Response{OK: false, Error: "failed to load analytics groups"}
	}
	defer groupRows.Close()
	groupMeta := make(map[int32]analyticsGroupMeta)
	groupIDs := make([]int32, 0)
	for groupRows.Next() {
		var meta analyticsGroupMeta
		if err := groupRows.Scan(&meta.id, &meta.name, &meta.streamName); err != nil {
			return Response{OK: false, Error: "failed to scan analytics groups"}
		}
		groupMeta[meta.id] = meta
		groupIDs = append(groupIDs, meta.id)
	}
	if err := groupRows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate analytics groups"}
	}

	studentPredicate, studentArgs := scopeStudentPredicate(scope, "st")
	studentPredicate = strings.ReplaceAll(studentPredicate, "$1", "$3")
	studentRows, err := s.store.Pool().Query(ctx, `
		WITH latest_consent AS (
			SELECT DISTINCT ON (user_id) user_id, decision
			FROM user_agreement_decisions
			WHERE agreement_key = $1 AND version = $2
			ORDER BY user_id, decided_at DESC, decision_id DESC
		)
		SELECT st.student_id, COALESCE(st.student_name, ''), st.group_id,
		       COALESCE(g.group_name, ''), COALESCE(g.stream_name, ''),
		       COALESCE(lc.decision = 'accepted', FALSE)
		FROM students st
		LEFT JOIN groups g ON g.group_id = st.group_id
		LEFT JOIN latest_consent lc ON lc.user_id = st.user_id
		WHERE `+studentPredicate+`
		ORDER BY g.group_name, st.student_name, st.student_id`, append([]any{userAgreementKey, currentAgreementVersion}, studentArgs...)...)
	if err != nil {
		return Response{OK: false, Error: "failed to load analytics students"}
	}
	defer studentRows.Close()
	studentsByID := make(map[int32]*analyticsStudentCalc)
	allStudents := make([]*analyticsStudentCalc, 0)
	for studentRows.Next() {
		student := &analyticsStudentCalc{subjects: make(map[int32]*analyticsSubjectCalc)}
		if err := studentRows.Scan(&student.id, &student.label, &student.groupID, &student.groupName, &student.streamName, &student.consent); err != nil {
			return Response{OK: false, Error: "failed to scan analytics students"}
		}
		studentsByID[student.id] = student
		allStudents = append(allStudents, student)
	}
	if err := studentRows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate analytics students"}
	}

	assignmentRows, err := s.store.Pool().Query(ctx, `
		SELECT DISTINCT sch.group_id, sch.subject_id, COALESCE(sub.name, '')
		FROM schedules sch
		JOIN subjects sub ON sub.subject_id = sch.subject_id
		WHERE sch.semester_id = $1 AND sch.group_id = ANY($2)
		ORDER BY sch.subject_id`, semester.ID, groupIDs)
	if err != nil {
		return Response{OK: false, Error: "failed to load analytics subjects"}
	}
	defer assignmentRows.Close()
	subjectNames := make(map[int32]string)
	for assignmentRows.Next() {
		var groupID, subjectID int32
		var subjectName string
		if err := assignmentRows.Scan(&groupID, &subjectID, &subjectName); err != nil {
			return Response{OK: false, Error: "failed to scan analytics subjects"}
		}
		subjectNames[subjectID] = subjectName
		for _, student := range allStudents {
			if student.groupID != groupID {
				continue
			}
			if student.subjects[subjectID] == nil {
				student.subjects[subjectID] = &analyticsSubjectCalc{id: subjectID, name: subjectName}
			}
		}
	}
	if err := assignmentRows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate analytics subjects"}
	}

	studentIDs := make([]int32, 0, len(allStudents))
	for _, student := range allStudents {
		studentIDs = append(studentIDs, student.id)
	}
	subjectIDs := make([]int32, 0, len(subjectNames))
	for subjectID := range subjectNames {
		subjectIDs = append(subjectIDs, subjectID)
	}
	sort.Slice(subjectIDs, func(i, j int) bool { return subjectIDs[i] < subjectIDs[j] })
	if len(studentIDs) > 0 && len(subjectIDs) > 0 {
		gradeRows, err := s.store.Pool().Query(ctx, `
			SELECT st.student_id, gi.subject_id, gi.max_score, gi.deadline,
			       gr.score, gr.updated_at
			FROM students st
			JOIN grade_items gi ON gi.semester_id = $1
			                         AND gi.subject_id = ANY($2)
			                         AND gi.deleted_at IS NULL
			LEFT JOIN grades gr ON gr.item_id = gi.item_id
			                       AND gr.student_id = st.student_id
			                       AND gr.deleted_at IS NULL
			WHERE st.student_id = ANY($3)
			ORDER BY st.student_id, gi.subject_id, gi.deadline NULLS LAST, gi.item_id`, semester.ID, subjectIDs, studentIDs)
		if err != nil {
			return Response{OK: false, Error: "failed to load analytics grades"}
		}
		for gradeRows.Next() {
			var studentID, subjectID, maxScore int32
			var deadline, gradedAt sql.NullTime
			var score sql.NullInt32
			if err := gradeRows.Scan(&studentID, &subjectID, &maxScore, &deadline, &score, &gradedAt); err != nil {
				gradeRows.Close()
				return Response{OK: false, Error: "failed to scan analytics grades"}
			}
			student := studentsByID[studentID]
			if student == nil || !deadline.Valid {
				continue
			}
			subject := student.subjects[subjectID]
			if subject == nil {
				continue
			}
			point := analyticsGradePoint{maxScore: int64(maxScore), deadline: deadline.Time}
			if score.Valid {
				point.score = int64(score.Int32)
				point.hasScore = true
			}
			if gradedAt.Valid {
				value := gradedAt.Time
				point.gradedAt = &value
			}
			subject.grades = append(subject.grades, point)
		}
		gradeRows.Close()
		if err := gradeRows.Err(); err != nil {
			return Response{OK: false, Error: "failed to iterate analytics grades"}
		}

		cutoff := time.Now().UTC()
		if cutoff.After(semester.EndsAt) {
			cutoff = semester.EndsAt
		}
		attendanceRows, err := s.store.Pool().Query(ctx, `
			SELECT ass.student_id, ats.subject_id, ats.created_at, ass.status
			FROM attendance_session_students ass
			JOIN attendance_sessions ats ON ats.session_id = ass.session_id
			WHERE ats.semester_id = $1
			  AND ats.created_at >= $2
			  AND ats.created_at <= $3
			  AND ass.student_id = ANY($4)
			  AND ats.subject_id = ANY($5)
			ORDER BY ass.student_id, ats.created_at, ats.session_id`, semester.ID, semester.StartsAt, cutoff, studentIDs, subjectIDs)
		if err != nil {
			return Response{OK: false, Error: "failed to load analytics attendance"}
		}
		for attendanceRows.Next() {
			var studentID, subjectID int32
			var occurredAt time.Time
			var status string
			if err := attendanceRows.Scan(&studentID, &subjectID, &occurredAt, &status); err != nil {
				attendanceRows.Close()
				return Response{OK: false, Error: "failed to scan analytics attendance"}
			}
			student := studentsByID[studentID]
			if student == nil || student.subjects[subjectID] == nil {
				continue
			}
			student.subjects[subjectID].attendance = append(student.subjects[subjectID].attendance, analyticsAttendancePoint{when: occurredAt, status: status})
		}
		attendanceRows.Close()
		if err := attendanceRows.Err(); err != nil {
			return Response{OK: false, Error: "failed to iterate analytics attendance"}
		}
	}

	scopeType := normalizeAnalyticsScope(data.ScopeType)
	scopeID := strings.TrimSpace(data.ScopeID)
	selectedStudents := allStudents
	selectedGroupIDs := make(map[int32]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		selectedGroupIDs[groupID] = struct{}{}
	}
	switch scopeType {
	case analyticsScopeStream:
		if scopeID == "" {
			return Response{OK: false, Error: "analytics stream not found"}
		}
		streamFound := false
		for groupID, meta := range groupMeta {
			if analyticsStreamID(meta.streamName) == scopeID {
				streamFound = true
				selectedGroupIDs[groupID] = struct{}{}
			} else {
				delete(selectedGroupIDs, groupID)
			}
		}
		if !streamFound {
			return Response{OK: false, Error: "analytics stream not found"}
		}
	case analyticsScopeGroup:
		groupID, parseErr := strconv.ParseInt(scopeID, 10, 32)
		if parseErr != nil || groupMeta[int32(groupID)].id == 0 {
			return Response{OK: false, Error: "analytics group not found"}
		}
		selectedGroupIDs = map[int32]struct{}{int32(groupID): struct{}{}}
	case analyticsScopeStudent:
		studentID, parseErr := strconv.ParseInt(scopeID, 10, 32)
		if parseErr != nil || studentsByID[int32(studentID)] == nil {
			return Response{OK: false, Error: "analytics student not found"}
		}
		selectedGroupIDs = map[int32]struct{}{studentsByID[int32(studentID)].groupID: struct{}{}}
	}
	selectedStudents = make([]*analyticsStudentCalc, 0, len(allStudents))
	for _, student := range allStudents {
		if _, ok := selectedGroupIDs[student.groupID]; !ok {
			continue
		}
		if scopeType == analyticsScopeStudent && strconv.FormatInt(int64(student.id), 10) != scopeID {
			continue
		}
		selectedStudents = append(selectedStudents, student)
	}

	now := time.Now().UTC()
	cutoff := now
	if cutoff.After(semester.EndsAt) {
		cutoff = semester.EndsAt
	}
	groupStudents := analyticsGroupStudents(allStudents)
	streamStudents := analyticsStreamStudents(allStudents)
	selectedGroupStudentMap := analyticsGroupStudents(selectedStudents)
	selectedStreamStudentMap := analyticsStreamStudents(selectedStudents)

	options := StaffAnalyticsOptions{
		Streams:  make([]StaffAnalyticsStreamOption, 0),
		Groups:   make([]StaffAnalyticsGroupOption, 0),
		Students: make([]StaffAnalyticsStudentOption, 0),
		Subjects: make([]StaffAnalyticsSubjectOption, 0),
	}
	streamGroupCount := make(map[string]int32)
	for _, meta := range groupMeta {
		streamGroupCount[analyticsStreamID(meta.streamName)]++
		options.Groups = append(options.Groups, StaffAnalyticsGroupOption{ID: meta.id, Name: meta.name, StreamID: analyticsStreamID(meta.streamName), StreamName: analyticsStreamLabel(meta.streamName)})
	}
	for _, student := range allStudents {
		options.Students = append(options.Students, StaffAnalyticsStudentOption{ID: student.id, Label: analyticsStudentLabel(s, student), GroupID: student.groupID, GroupName: student.groupName, StreamName: analyticsStreamLabel(student.streamName)})
	}
	for subjectID, subjectName := range subjectNames {
		options.Subjects = append(options.Subjects, StaffAnalyticsSubjectOption{ID: subjectID, Name: subjectName})
	}
	for streamID, students := range streamStudents {
		options.Streams = append(options.Streams, StaffAnalyticsStreamOption{ID: streamID, Name: analyticsStreamLabel(streamID), GroupCount: streamGroupCount[streamID], StudentCount: int32(len(students))})
	}
	sort.Slice(options.Streams, func(i, j int) bool { return options.Streams[i].Name < options.Streams[j].Name })
	sort.Slice(options.Groups, func(i, j int) bool { return options.Groups[i].Name < options.Groups[j].Name })
	sort.Slice(options.Students, func(i, j int) bool { return options.Students[i].Label < options.Students[j].Label })
	sort.Slice(options.Subjects, func(i, j int) bool { return options.Subjects[i].Name < options.Subjects[j].Name })

	selectedGroupMeta := make([]analyticsGroupMeta, 0, len(selectedGroupIDs))
	for groupID := range selectedGroupIDs {
		if meta, ok := groupMeta[groupID]; ok {
			selectedGroupMeta = append(selectedGroupMeta, meta)
		}
	}
	sort.Slice(selectedGroupMeta, func(i, j int) bool { return selectedGroupMeta[i].name < selectedGroupMeta[j].name })
	selectedStreamIDs := make(map[string]struct{})
	for _, meta := range selectedGroupMeta {
		selectedStreamIDs[analyticsStreamID(meta.streamName)] = struct{}{}
	}

	groupValues := make([]StaffAnalyticsGroup, 0, len(selectedGroupMeta))
	for _, meta := range selectedGroupMeta {
		groupValues = append(groupValues, analyticsGroupValue(meta, selectedGroupStudentMap[meta.id], cutoff, data.SubjectID))
	}
	streamValues := make([]StaffAnalyticsStream, 0, len(selectedStreamIDs))
	for streamID := range selectedStreamIDs {
		streamValues = append(streamValues, analyticsStreamValue(streamID, analyticsStreamLabel(streamID), selectedStreamStudentMap[streamID], streamGroupCount[streamID], cutoff, data.SubjectID))
	}
	sort.Slice(streamValues, func(i, j int) bool { return streamValues[i].Name < streamValues[j].Name })

	rankingPool := selectedStudents
	if scopeType == analyticsScopeStudent && len(selectedStudents) == 1 {
		rankingPool = groupStudents[selectedStudents[0].groupID]
	}
	studentValues := make([]StaffAnalyticsStudent, 0, len(selectedStudents))
	sort.SliceStable(selectedStudents, func(i, j int) bool {
		left := selectedStudents[i].snapshot(cutoff, data.SubjectID)
		right := selectedStudents[j].snapshot(cutoff, data.SubjectID)
		if left.hasRating != right.hasRating {
			return left.hasRating
		}
		if left.hasRating && left.rating != right.rating {
			return left.rating > right.rating
		}
		return analyticsStudentLabel(s, selectedStudents[i]) < analyticsStudentLabel(s, selectedStudents[j])
	})
	for _, student := range selectedStudents {
		studentValues = append(studentValues, s.renderAnalyticsStudent(student, rankingPool, cutoff, data.SubjectID))
	}

	studentDetail := (*StaffAnalyticsStudentDetail)(nil)
	if scopeType == analyticsScopeStudent && len(selectedStudents) == 1 {
		value := s.renderAnalyticsStudent(selectedStudents[0], rankingPool, cutoff, data.SubjectID)
		studentDetail = &StaffAnalyticsStudentDetail{StaffAnalyticsStudent: value, Subjects: selectedStudents[0].renderSubjects(cutoff, data.SubjectID)}
	}

	selectedSubjectIDs := subjectIDs
	if data.SubjectID != nil {
		if _, ok := subjectNames[*data.SubjectID]; !ok {
			return Response{OK: false, Error: "analytics subject not found"}
		}
		selectedSubjectIDs = []int32{*data.SubjectID}
	}
	subjectValues := make([]StaffAnalyticsSubject, 0, len(selectedSubjectIDs))
	heatmap := make([]StaffAnalyticsHeatmapPoint, 0, len(selectedGroupMeta)*len(selectedSubjectIDs))
	for _, subjectID := range selectedSubjectIDs {
		ratings, attendance, atRisk := analyticsSubjectValues(selectedStudents, subjectID, cutoff)
		var avgRating, avgAttendance *float64
		if len(ratings) > 0 {
			var sum float64
			for _, value := range ratings {
				sum += value
			}
			avgRating = analyticsFloat(sum/float64(len(ratings)), true)
		}
		if len(attendance) > 0 {
			var sum float64
			for _, value := range attendance {
				sum += value
			}
			avgAttendance = analyticsFloat(sum/float64(len(attendance)), true)
		}
		subjectValues = append(subjectValues, StaffAnalyticsSubject{ID: subjectID, Name: subjectNames[subjectID], AverageRating: avgRating, AverageAttendance: avgAttendance, StudentsWithRating: int32(len(ratings)), StudentsWithAttend: int32(len(attendance)), AtRiskCount: atRisk})
		for _, meta := range selectedGroupMeta {
			groupRatings, groupAttendance, _ := analyticsSubjectValues(selectedGroupStudentMap[meta.id], subjectID, cutoff)
			var groupRating, groupAttend *float64
			if len(groupRatings) > 0 {
				var sum float64
				for _, value := range groupRatings {
					sum += value
				}
				groupRating = analyticsFloat(sum/float64(len(groupRatings)), true)
			}
			if len(groupAttendance) > 0 {
				var sum float64
				for _, value := range groupAttendance {
					sum += value
				}
				groupAttend = analyticsFloat(sum/float64(len(groupAttendance)), true)
			}
			heatmap = append(heatmap, StaffAnalyticsHeatmapPoint{GroupID: meta.id, GroupName: meta.name, SubjectID: subjectID, SubjectName: subjectNames[subjectID], Rating: groupRating, Attendance: groupAttend})
		}
	}
	sort.Slice(subjectValues, func(i, j int) bool { return subjectValues[i].Name < subjectValues[j].Name })

	scopeLabel := scope.Label
	switch scopeType {
	case analyticsScopeStream:
		scopeLabel = "Поток: " + analyticsStreamLabel(scopeID)
	case analyticsScopeGroup:
		if meta, ok := groupMeta[selectedGroupMeta[0].id]; ok {
			scopeLabel = "Группа: " + meta.name
		}
	case analyticsScopeStudent:
		if len(selectedStudents) == 1 {
			scopeLabel = "Студент: " + analyticsStudentLabel(s, selectedStudents[0])
		}
	}

	result := StaffAnalyticsResult{
		Semester:            StaffAnalyticsSemester{SemesterID: semester.ID, Title: semester.Name, StartsAt: semester.StartsAt.In(appTimeLocation).Format(time.RFC3339), EndsAt: semester.EndsAt.In(appTimeLocation).Format(time.RFC3339)},
		Scope:               StaffAnalyticsScope{Type: scopeType, ID: scopeID, Label: scopeLabel},
		Summary:             analyticsSummary(selectedStudents, cutoff, data.SubjectID),
		Options:             options,
		Groups:              groupValues,
		Streams:             streamValues,
		Students:            studentValues,
		Subjects:            subjectValues,
		Heatmap:             heatmap,
		Weekly:              analyticsWeekly(selectedStudents, semester.StartsAt, cutoff, data.SubjectID),
		Distribution:        analyticsDistribution(selectedStudents, cutoff, data.SubjectID),
		AttendanceBreakdown: analyticsStatusBreakdown(selectedStudents, cutoff, data.SubjectID),
		Student:             studentDetail,
	}
	return Response{OK: true, Result: result}
}
