package db_test

import (
	"testing"

	"real-time-video-surveillance-system/db"
)

func TestAPITokenHash(t *testing.T) {
	expected := "ff194a51405eb34180b91ed9d9130ec5ddec108174c6806fc333ec3c33d83870"
	if result := db.APITokenHash("dev-user-token"); result != expected {
		t.Fatalf("unexpected token hash: %s", result)
	}
}
