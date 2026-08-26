package app

import (
	"math"
	"testing"
	"time"
)

func TestAnalyticsSnapshotUsesDueItemsAndExcludesExcusedAttendance(t *testing.T) {
	asOf := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	gradedAfterSnapshot := asOf.Add(24 * time.Hour)
	student := &analyticsStudentCalc{
		subjects: map[int32]*analyticsSubjectCalc{
			1: {
				id:   1,
				name: "Математика",
				grades: []analyticsGradePoint{
					{maxScore: 10, score: 8, hasScore: true, deadline: asOf.Add(-48 * time.Hour)},
					{maxScore: 10, deadline: asOf.Add(-24 * time.Hour)},
					{maxScore: 10, score: 10, hasScore: true, deadline: asOf.Add(-72 * time.Hour), gradedAt: &gradedAfterSnapshot},
					{maxScore: 10, score: 10, hasScore: true, deadline: asOf.Add(24 * time.Hour)},
				},
				attendance: []analyticsAttendancePoint{
					{when: asOf.Add(-72 * time.Hour), status: "present"},
					{when: asOf.Add(-48 * time.Hour), status: "late"},
					{when: asOf.Add(-24 * time.Hour), status: "absent"},
					{when: asOf.Add(-12 * time.Hour), status: "excused"},
					{when: asOf.Add(24 * time.Hour), status: "present"},
				},
			},
		},
	}

	got := student.snapshot(asOf, nil)
	if !got.hasRating || math.Abs(got.rating-26.666666666666668) > 0.000001 {
		t.Fatalf("snapshot rating = %v (has=%v), want 26.6667", got.rating, got.hasRating)
	}
	if got.dueItems != 3 || got.gradedItems != 1 {
		t.Fatalf("snapshot grade counts = %d/%d, want 3/1", got.gradedItems, got.dueItems)
	}
	if !got.hasAttendance || math.Abs(got.attendance-66.66666666666667) > 0.000001 {
		t.Fatalf("snapshot attendance = %v (has=%v), want 66.6667", got.attendance, got.hasAttendance)
	}
	if got.countedSessions != 3 || got.excusedSessions != 1 || got.lateSessions != 1 || got.absentSessions != 1 {
		t.Fatalf("snapshot attendance counts = counted:%d excused:%d late:%d absent:%d", got.countedSessions, got.excusedSessions, got.lateSessions, got.absentSessions)
	}
}

func TestAnalyticsSummaryMinMaxIgnoresStudentsWithoutRating(t *testing.T) {
	asOf := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	students := []*analyticsStudentCalc{
		{subjects: map[int32]*analyticsSubjectCalc{}},
		{subjects: map[int32]*analyticsSubjectCalc{1: {id: 1, grades: []analyticsGradePoint{{maxScore: 10, score: 7, hasScore: true, deadline: asOf.Add(-time.Hour)}}}}},
		{subjects: map[int32]*analyticsSubjectCalc{1: {id: 1, grades: []analyticsGradePoint{{maxScore: 10, score: 9, hasScore: true, deadline: asOf.Add(-time.Hour)}}}}},
	}

	got := analyticsSummary(students, asOf, nil)
	if got.RatingSpread == nil || *got.RatingSpread != 20 {
		t.Fatalf("rating spread = %v, want 20", got.RatingSpread)
	}
}

func TestAnalyticsWeeklyBuildsOnePointPerCompletedWeek(t *testing.T) {
	semesterStart := time.Date(2026, 8, 3, 0, 0, 0, 0, appTimeLocation)
	cutoff := semesterStart.AddDate(0, 0, 21)
	student := &analyticsStudentCalc{
		subjects: map[int32]*analyticsSubjectCalc{
			1: {
				id:     1,
				grades: []analyticsGradePoint{{maxScore: 10, score: 8, hasScore: true, deadline: semesterStart.AddDate(0, 0, 2)}},
			},
		},
	}

	got := analyticsWeekly([]*analyticsStudentCalc{student}, semesterStart, cutoff, nil)
	if len(got) != 3 {
		t.Fatalf("weekly points = %d, want 3", len(got))
	}
	if got[0].WeekStart != "2026-08-03" || got[0].AverageRating == nil || *got[0].AverageRating != 80 {
		t.Fatalf("first weekly point = %+v", got[0])
	}
}
