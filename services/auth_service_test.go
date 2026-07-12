package services

import "testing"

func TestBearerToken(t *testing.T) {
	token, ok := bearerToken("Bearer test-token")
	if !ok || token != "test-token" {
		t.Fatalf("unexpected bearer token result: %q %v", token, ok)
	}

	token, ok = bearerToken("bearer spaced-token ")
	if !ok || token != "spaced-token" {
		t.Fatalf("unexpected lowercase bearer token result: %q %v", token, ok)
	}
}

func TestBearerTokenRejectsInvalidHeader(t *testing.T) {
	for _, header := range []string{"", "Basic token", "Bearer", "Bearer "} {
		if token, ok := bearerToken(header); ok {
			t.Fatalf("accepted invalid header %q as token %q", header, token)
		}
	}
}
