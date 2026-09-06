package app

import "strings"

// SetDemoAccountsEnabled must be called before starting workers. This opt-in is
// intended for an isolated presentation/test environment, never real student data.
func (s *Service) SetDemoAccountsEnabled(enabled bool) {
	s.allowDemoAccounts = enabled
}

func (s *Service) accountLoginAllowed(login string) bool {
	if s.allowDemoAccounts {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(login)) {
	case "student_test", "teacher_test", "admin_test", "head_test", "dean_test":
		return false
	default:
		return true
	}
}
