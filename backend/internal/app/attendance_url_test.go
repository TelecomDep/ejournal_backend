package app

import "testing"

func TestBuildAttendanceJoinURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		siteBaseURL string
		want        string
	}{
		{
			name:        "base URL without trailing slash",
			siteBaseURL: "https://lms.signal.qlabs.pro",
			want:        "https://lms.signal.qlabs.pro/#/attendance/join?token=test.token",
		},
		{
			name:        "base URL with trailing slash",
			siteBaseURL: "https://lms.signal.qlabs.pro/",
			want:        "https://lms.signal.qlabs.pro/#/attendance/join?token=test.token",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := buildAttendanceJoinURL(test.siteBaseURL, "test.token")
			if got != test.want {
				t.Fatalf("buildAttendanceJoinURL() = %q, want %q", got, test.want)
			}
		})
	}
}
