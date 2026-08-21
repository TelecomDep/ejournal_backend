package app

import "testing"

func TestMakeStudentAttendanceMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		total          int32
		attended       int32
		excused        int32
		wantMissed     int32
		wantPercentage float64
	}{
		{name: "excused sessions are excluded", total: 10, attended: 7, excused: 1, wantMissed: 2, wantPercentage: 77.78},
		{name: "no sessions", wantMissed: 0, wantPercentage: 0},
		{name: "all sessions excused", total: 3, excused: 3, wantMissed: 0, wantPercentage: 0},
		{name: "all counted sessions attended", total: 5, attended: 4, excused: 1, wantMissed: 0, wantPercentage: 100},
		{name: "inconsistent totals do not produce negative missed count", total: 1, attended: 1, excused: 1, wantMissed: 0, wantPercentage: 0},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := makeStudentAttendanceMetrics(test.total, test.attended, test.excused)
			if got.MissedSessions != test.wantMissed {
				t.Fatalf("MissedSessions = %d, want %d", got.MissedSessions, test.wantMissed)
			}
			if got.Percent != test.wantPercentage {
				t.Fatalf("Percent = %v, want %v", got.Percent, test.wantPercentage)
			}
		})
	}
}
