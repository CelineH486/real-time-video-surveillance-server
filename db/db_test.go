package db

import "testing"

func TestAPITokenHash(t *testing.T) {
	expected := "ff194a51405eb34180b91ed9d9130ec5ddec108174c6806fc333ec3c33d83870"
	if result := APITokenHash("dev-user-token"); result != expected {
		t.Fatalf("unexpected token hash: %s", result)
	}
}
