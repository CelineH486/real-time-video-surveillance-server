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
