package app

import (
	"strings"
	"testing"
)

func TestFraudFilterWhereLimitsTeacherToAssignedGroups(t *testing.T) {
	where, args := fraudFilterWhere(
		VisibilityScope{Role: RoleTeacher, GroupIDs: []int32{12, 18}},
		FraudLogsQuery{Search: "Иванов", Reason: "device"},
	)

	if !strings.Contains(where, "g.group_id = ANY($1)") {
		t.Fatalf("teacher scope is missing from query: %s", where)
	}
	if strings.Contains(where, "g.lectern_id") {
		t.Fatalf("teacher query must not use lectern scope: %s", where)
	}
	if !strings.Contains(where, "ass.fraud_reason = 'device_id already used in this lesson'") {
		t.Fatalf("device reason filter is missing: %s", where)
	}
	if len(args) != 2 {
		t.Fatalf("expected group scope and search arguments, got %d", len(args))
	}
}

func TestFraudFilterWhereBuildsAdminFilters(t *testing.T) {
	where, args := fraudFilterWhere(
		VisibilityScope{Role: RoleAdmin, All: true},
		FraudLogsQuery{
			GroupID:   7,
			TeacherID: 4,
			Reason:    "distance",
			DateFrom:  "2026-08-01",
			DateTo:    "2026-08-31",
		},
	)

	for _, expected := range []string{
		"g.group_id = $1",
		"s.teacher_id = $2",
		"student is too far from lesson location",
		"ass.marked_at >= $3",
		"ass.marked_at < $4",
	} {
		if !strings.Contains(where, expected) {
			t.Fatalf("expected %q in query: %s", expected, where)
		}
	}
	if len(args) != 4 {
		t.Fatalf("expected four filter arguments, got %d", len(args))
	}
}
