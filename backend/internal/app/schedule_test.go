package app

import (
	"testing"
	"time"
)

func TestScheduleDate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, appTimeLocation)
	tests := []struct {
		name    string
		rawDate string
		want    string
		wantErr bool
	}{
		{name: "defaults to today", want: "2026-08-20"},
		{name: "start of current week", rawDate: "2026-08-17", want: "2026-08-17"},
		{name: "end of next week", rawDate: "2026-08-30", want: "2026-08-30"},
		{name: "date in past", rawDate: "2026-08-16", want: "2026-08-16"},
		{name: "date in future", rawDate: "2026-08-31", want: "2026-08-31"},
		{name: "invalid format", rawDate: "20.08.2026", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := scheduleDate(test.rawDate, now)
			if (err != nil) != test.wantErr {
				t.Fatalf("scheduleDate() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && got.Format("2006-01-02") != test.want {
				t.Fatalf("scheduleDate() = %s, want %s", got.Format("2006-01-02"), test.want)
			}
		})
	}
}
