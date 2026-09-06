package app

import "testing"

func TestDemoAccountsRequireExplicitOptIn(t *testing.T) {
	svc := &Service{}
	for _, login := range []string{"student_test", "teacher_test", "admin_test", "head_test", "dean_test"} {
		if svc.accountLoginAllowed(login) {
			t.Errorf("demo account %s enabled by default", login)
		}
		// Must reject without reaching the (absent) database.
		if response := svc.login(LoginData{Login: login, Password: "123456"}); response.OK {
			t.Errorf("default demo login %s succeeded", login)
		}
	}
	if !svc.accountLoginAllowed("real.teacher") {
		t.Fatal("ordinary university account blocked")
	}
	svc.SetDemoAccountsEnabled(true)
	if !svc.accountLoginAllowed("teacher_test") {
		t.Fatal("explicit presentation mode did not enable demo account")
	}
}

func TestEmptyLegacyCodeCannotGrantTeacherRole(t *testing.T) {
	svc := &Service{}
	for _, code := range []string{"", "  ", "TEACHER-HASH-2026", "STUDENT-HASH-2026"} {
		if _, allowed := svc.resolveRoleByHash(code); allowed {
			t.Fatalf("disabled legacy registration accepted %q", code)
		}
	}
}
