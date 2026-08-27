package app

import (
	"testing"
	"time"
)

func TestAttendanceInviteTokenRoundTrip(t *testing.T) {
	svc := &Service{
		jwtSecret: []byte("dev-secret-change-me"),
	}

	token, exp, err := svc.generateAttendanceInviteToken("123", "456", 20)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if token == "" {
		t.Fatal("empty token")
	}

	claims, err := svc.parseAttendanceInviteToken(token)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if claims.LessonID != "123" {
		t.Errorf("expected LessonID 123, got %s", claims.LessonID)
	}
	if claims.TeacherID != "456" {
		t.Errorf("expected TeacherID 456, got %s", claims.TeacherID)
	}
	if time.Now().After(exp) {
		t.Error("token already expired")
	}
}
