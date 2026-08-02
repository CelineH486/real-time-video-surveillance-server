package controllers_test

import (
	"testing"

	"real-time-video-surveillance-system/controllers"
)

func TestValidPassword(t *testing.T) {
	for _, password := range []string{"Password1", "Abcdefg9", "Monitor2026"} {
		if !controllers.ValidLoginPassword(password) {
			t.Fatalf("expected %q to be valid", password)
		}
	}
}

func TestInvalidPassword(t *testing.T) {
	for _, password := range []string{"short1A", "lowercase1", "UPPERCASE1", "NoNumbers"} {
		if controllers.ValidLoginPassword(password) {
			t.Fatalf("expected %q to be invalid", password)
		}
	}
}

func TestEmailMustUseDotComDomain(t *testing.T) {
	for _, email := range []string{"user@example.com", "USER@COMPANY.COM"} {
		if !controllers.ValidLoginEmail(email) {
			t.Fatalf("expected %q to be valid", email)
		}
	}
	for _, email := range []string{"user@example.local", "user@example.org", "invalid"} {
		if controllers.ValidLoginEmail(email) {
			t.Fatalf("expected %q to be invalid", email)
		}
	}
}
