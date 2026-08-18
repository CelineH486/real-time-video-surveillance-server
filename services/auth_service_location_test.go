package services

import "testing"

func TestWebSocketBearerToken(t *testing.T) {
	token, ok := websocketBearerToken("bearer, test-token")
	if !ok || token != "test-token" {
		t.Fatalf("unexpected WebSocket bearer token result: %q %v", token, ok)
	}
	for _, header := range []string{"", "bearer", "other, test-token"} {
		if token, ok := websocketBearerToken(header); ok {
			t.Fatalf("accepted invalid WebSocket protocol %q as token %q", header, token)
		}
	}
}
