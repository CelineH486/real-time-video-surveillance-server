package controllers

import "testing"

func TestValidPassword(t *testing.T) {
	for _, password := range []string{"Password1", "Abcdefg9", "Monitor2026"} {
		if !validPassword(password) {
			t.Fatalf("expected %q to be valid", password)
		}
	}
}

func TestInvalidPassword(t *testing.T) {
	for _, password := range []string{"short1A", "lowercase1", "UPPERCASE1", "NoNumbers"} {
		if validPassword(password) {
			t.Fatalf("expected %q to be invalid", password)
		}
	}
}

func TestEmailMustUseDotComDomain(t *testing.T) {
	for _, email := range []string{"user@example.com", "USER@COMPANY.COM"} {
		if !emailPattern.MatchString(email) {
			t.Fatalf("expected %q to be valid", email)
		}
	}
	for _, email := range []string{"user@example.local", "user@example.org", "invalid"} {
		if emailPattern.MatchString(email) {
			t.Fatalf("expected %q to be invalid", email)
		}
	}
}
