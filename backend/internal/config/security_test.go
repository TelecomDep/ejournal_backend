package config

import "testing"

func TestRejectPlaceholderJWTSecrets(t *testing.T) {
	for _, secret := range []string{"", "dev-secret-change-me", "generate-a-secure-random-jwt-secret-for-production", "replace-with-a-very-long-secret-value", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		if err := (AppConfig{JWTSecret: secret}).ValidateSecurity(); err == nil {
			t.Errorf("accepted unsafe secret %q", secret)
		}
	}
	if err := (AppConfig{JWTSecret: "682d4a1bed795fc05d643413f8fb5f642d095b4fcddda05e166ea1eb230da91f"}).ValidateSecurity(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyRegistrationRequiresExplicitCodes(t *testing.T) {
	cfg := AppConfig{JWTSecret: "682d4a1bed795fc05d643413f8fb5f642d095b4fcddda05e166ea1eb230da91f"}
	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatal(err)
	}
	cfg.AllowLegacyRoleRegistration = true
	if err := cfg.ValidateSecurity(); err == nil {
		t.Fatal("accepted enabled legacy registration with empty role codes")
	}
	cfg.RoleHashTeacher, cfg.RoleHashStudent = "same-code", "SAME-CODE"
	if err := cfg.ValidateSecurity(); err == nil {
		t.Fatal("accepted matching teacher/student role codes")
	}
}
