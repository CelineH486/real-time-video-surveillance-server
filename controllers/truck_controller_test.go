package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"real-time-video-surveillance-system/apiresponse"
)

func TestNewTruckController(t *testing.T) {
	// Arrange
	databaseConnection, _ := newControllerTestDatabase(t)

	// Act
	truckController := NewTruckController(databaseConnection)

	// Assert
	if truckController == nil {
		t.Fatal("NewTruckController returned nil")
	}
	if truckController.database != databaseConnection {
		t.Error("controller did not retain the supplied database connection")
	}
}

func TestTruckController_List_Unauthenticated(t *testing.T) {
	// Arrange
	truckController := NewTruckController(nil)
	unauthenticatedRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	truckController.List(responseRecorder, unauthenticatedRequest)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, http.StatusUnauthorized,
		apiresponse.CodeAuthenticationRequired, apiresponse.MessageAuthenticationRequired)
}

func TestTruckController_List_DatabaseError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodGet, "/api/trucks")
	databaseMock.ExpectQuery(`FROM trucks t\s+INNER JOIN user_trucks ut`).
		WithArgs(controllerTestUserID).
		WillReturnError(errors.New("truck list query failed"))
	truckController := NewTruckController(databaseConnection)
	responseRecorder := httptest.NewRecorder()

	// Act
	truckController.List(responseRecorder, authenticatedRequest)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, http.StatusInternalServerError,
		apiresponse.CodeTrucksUnavailable, apiresponse.MessageTrucksUnavailable)
}

func TestTruckController_List_Success(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodGet, "/api/trucks")
	assignedTruckRows := sqlmock.NewRows([]string{"truck_id", "plate_no", "status"}).
		AddRow("truck001", "ABC-1234", "online").
		AddRow("truck002", "XYZ-5678", "offline")
	databaseMock.ExpectQuery(`FROM trucks t\s+INNER JOIN user_trucks ut`).
		WithArgs(controllerTestUserID).
		WillReturnRows(assignedTruckRows)
	truckController := NewTruckController(databaseConnection)
	responseRecorder := httptest.NewRecorder()

	// Act
	truckController.List(responseRecorder, authenticatedRequest)

	// Assert
	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	var returnedTrucks []map[string]string
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &returnedTrucks); decodeError != nil {
		t.Fatalf("decode truck list response: %v", decodeError)
	}
	if len(returnedTrucks) != 2 {
		t.Fatalf("truck count = %d, want 2", len(returnedTrucks))
	}
	if returnedTrucks[0]["truckId"] != "truck001" || returnedTrucks[0]["plateNo"] != "ABC-1234" || returnedTrucks[0]["status"] != "online" {
		t.Errorf("unexpected first truck: %+v", returnedTrucks[0])
	}
	if returnedTrucks[1]["truckId"] != "truck002" || returnedTrucks[1]["plateNo"] != "XYZ-5678" || returnedTrucks[1]["status"] != "offline" {
		t.Errorf("unexpected second truck: %+v", returnedTrucks[1])
	}
}

func TestTruckController_List_NoAssignedTrucks(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodGet, "/api/trucks")
	databaseMock.ExpectQuery(`FROM trucks t\s+INNER JOIN user_trucks ut`).
		WithArgs(controllerTestUserID).
		WillReturnRows(sqlmock.NewRows([]string{"truck_id", "plate_no", "status"}))
	truckController := NewTruckController(databaseConnection)
	responseRecorder := httptest.NewRecorder()

	// Act
	truckController.List(responseRecorder, authenticatedRequest)

	// Assert
	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	var returnedTrucks []map[string]string
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &returnedTrucks); decodeError != nil {
		t.Fatalf("decode truck list response: %v", decodeError)
	}
	if len(returnedTrucks) != 0 {
		t.Errorf("truck count = %d, want 0", len(returnedTrucks))
	}
}
