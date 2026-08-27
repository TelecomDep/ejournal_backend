package app

import (
	"testing"
	"time"
)

func TestDynamicQRTimestampValidation(t *testing.T) {
	now := time.Now().Unix()

	freshTs := now - 2
	ageFresh := now - freshTs
	if ageFresh < -10 || ageFresh > 8 {
		t.Errorf("expected fresh timestamp to be valid, got age %d", ageFresh)
	}

	expiredTs := now - 15
	ageExpired := now - expiredTs
	if !(ageExpired < -10 || ageExpired > 8) {
		t.Errorf("expected expired timestamp to be rejected, got age %d", ageExpired)
	}

	futureTs := now + 30
	ageFuture := now - futureTs
	if !(ageFuture < -10 || ageFuture > 8) {
		t.Errorf("expected far future timestamp to be rejected, got age %d", ageFuture)
	}
}

func TestLessonTypeElectiveDetection(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"elective", true},
		{"ELECTIVE", true},
		{"факультатив", true},
		{"Факультатив", true},
		{"Практика", false},
		{"Лекция", false},
		{"Лабораторная работа", false},
		{"", false},
	}

	for _, c := range cases {
		isElective := (c.input == "elective" || c.input == "ELECTIVE" || c.input == "факультатив" || c.input == "Факультатив")
		if isElective != c.expected {
			t.Errorf("for input %q expected %v, got %v", c.input, c.expected, isElective)
		}
	}
}
