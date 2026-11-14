package services

import (
	"database/sql"
	"testing"
)

func TestNewStatusService(t *testing.T) {
	// Arrange
	var databaseConnection *sql.DB

	// Act
	statusService := NewStatusService(databaseConnection)

	// Assert
	if statusService == nil || statusService.database != databaseConnection {
		t.Fatal("NewStatusService did not initialize the expected database dependency")
	}
}

func TestMonitorMediaMTX_EmptyURLDisablesMonitoring(t *testing.T) {
	// Arrange
	statusService := NewStatusService(nil)
	mediaMTXBaseURL := "///"

	// Act
	statusService.MonitorMediaMTX(mediaMTXBaseURL)

	// Assert
	if statusService.database != nil {
		t.Error("disabled monitoring changed the database dependency")
	}
}
