package db

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"real-time-video-surveillance-system/models"
)

func newDatabaseTestConnection(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	databaseConnection, databaseMock, connectionError := sqlmock.New()
	if connectionError != nil {
		t.Fatalf("create SQL mock: %v", connectionError)
	}
	t.Cleanup(func() {
		databaseMock.ExpectClose()
		if closeError := databaseConnection.Close(); closeError != nil {
			t.Errorf("close SQL mock: %v", closeError)
		}
		if expectationError := databaseMock.ExpectationsWereMet(); expectationError != nil {
			t.Errorf("unmet SQL expectations: %v", expectationError)
		}
	})
	return databaseConnection, databaseMock
}

func TestConnect_InvalidConnectionString(t *testing.T) {
	// Arrange
	t.Setenv("DATABASE_URL", "invalid connection string")

	// Act
	databaseConnection, connectionError := Connect()

	// Assert
	if connectionError == nil || databaseConnection != nil {
		t.Fatalf("Connect = (%v, %v), want nil and a connection string error", databaseConnection, connectionError)
	}
}

func TestGetCameras_Success(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	cameraRows := sqlmock.NewRows([]string{"camera_id", "truck_id", "name", "status"}).
		AddRow("cam01", "truck001", "Front Camera", "online").
		AddRow("cam02", "truck001", "Rear Camera", "offline")
	databaseMock.ExpectQuery(`FROM cameras`).WillReturnRows(cameraRows).RowsWillBeClosed()
	expectedCameras := []map[string]string{
		{"cameraId": "cam01", "truckId": "truck001", "name": "Front Camera", "status": "online"},
		{"cameraId": "cam02", "truckId": "truck001", "name": "Rear Camera", "status": "offline"},
	}

	// Act
	returnedCameras, queryError := GetCameras(databaseConnection)

	// Assert
	if queryError != nil || !reflect.DeepEqual(returnedCameras, expectedCameras) {
		t.Errorf("GetCameras = (%+v, %v), want (%+v, nil)", returnedCameras, queryError, expectedCameras)
	}
}

func TestGetCameras_QueryError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedDatabaseError := errors.New("camera query failed")
	databaseMock.ExpectQuery(`FROM cameras`).WillReturnError(expectedDatabaseError)

	// Act
	returnedCameras, queryError := GetCameras(databaseConnection)

	// Assert
	if !errors.Is(queryError, expectedDatabaseError) || returnedCameras != nil {
		t.Errorf("GetCameras = (%+v, %v), want nil and the database error", returnedCameras, queryError)
	}
}

func TestGetCameras_ScanError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	cameraRows := sqlmock.NewRows([]string{"camera_id", "truck_id", "name", "status"}).AddRow(nil, "truck001", "Front Camera", "online")
	databaseMock.ExpectQuery(`FROM cameras`).WillReturnRows(cameraRows).RowsWillBeClosed()

	// Act
	returnedCameras, queryError := GetCameras(databaseConnection)

	// Assert
	if queryError == nil || returnedCameras != nil {
		t.Errorf("GetCameras = (%+v, %v), want nil and a scan error", returnedCameras, queryError)
	}
}

func TestGetTrucks_Success(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	truckRows := sqlmock.NewRows([]string{"truck_id", "plate_no", "status"}).AddRow("truck001", "ABC-1234", "online")
	databaseMock.ExpectQuery(`FROM trucks`).WillReturnRows(truckRows).RowsWillBeClosed()
	expectedTrucks := []map[string]string{{"truckId": "truck001", "plateNo": "ABC-1234", "status": "online"}}

	// Act
	returnedTrucks, queryError := GetTrucks(databaseConnection)

	// Assert
	if queryError != nil || !reflect.DeepEqual(returnedTrucks, expectedTrucks) {
		t.Errorf("GetTrucks = (%+v, %v), want (%+v, nil)", returnedTrucks, queryError, expectedTrucks)
	}
}

func TestGetTrucks_QueryError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedDatabaseError := errors.New("truck query failed")
	databaseMock.ExpectQuery(`FROM trucks`).WillReturnError(expectedDatabaseError)

	// Act
	returnedTrucks, queryError := GetTrucks(databaseConnection)

	// Assert
	if !errors.Is(queryError, expectedDatabaseError) || returnedTrucks != nil {
		t.Errorf("GetTrucks = (%+v, %v), want nil and the database error", returnedTrucks, queryError)
	}
}

func TestGetTrucks_ScanError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	truckRows := sqlmock.NewRows([]string{"truck_id", "plate_no", "status"}).AddRow("truck001", nil, "online")
	databaseMock.ExpectQuery(`FROM trucks`).WillReturnRows(truckRows).RowsWillBeClosed()

	// Act
	returnedTrucks, queryError := GetTrucks(databaseConnection)

	// Assert
	if queryError == nil || returnedTrucks != nil {
		t.Errorf("GetTrucks = (%+v, %v), want nil and a scan error", returnedTrucks, queryError)
	}
}

func TestUpdateTruckStatus_Success(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	truckID := "truck001"
	newTruckStatus := "offline"
	databaseMock.ExpectExec(`UPDATE trucks\s+SET status = \$1\s+WHERE truck_id = \$2`).
		WithArgs(newTruckStatus, truckID).WillReturnResult(sqlmock.NewResult(0, 1))

	// Act
	updateError := UpdateTruckStatus(databaseConnection, truckID, newTruckStatus)

	// Assert
	if updateError != nil {
		t.Errorf("update truck status: %v", updateError)
	}
}

func TestUpdateTruckStatus_DatabaseError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedDatabaseError := errors.New("truck update failed")
	databaseMock.ExpectExec(`UPDATE trucks`).WithArgs("offline", "truck001").WillReturnError(expectedDatabaseError)

	// Act
	updateError := UpdateTruckStatus(databaseConnection, "truck001", "offline")

	// Assert
	if !errors.Is(updateError, expectedDatabaseError) {
		t.Errorf("update error = %v, want %v", updateError, expectedDatabaseError)
	}
}

func TestUpdateCameraStatus_Success(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	cameraID := "cam01"
	truckID := "truck001"
	newCameraStatus := "online"
	databaseMock.ExpectExec(`UPDATE cameras\s+SET status = \$1, last_seen_at = NOW\(\)\s+WHERE camera_id = \$2 AND truck_id = \$3`).
		WithArgs(newCameraStatus, cameraID, truckID).WillReturnResult(sqlmock.NewResult(0, 1))

	// Act
	updateError := UpdateCameraStatus(databaseConnection, cameraID, truckID, newCameraStatus)

	// Assert
	if updateError != nil {
		t.Errorf("update camera status: %v", updateError)
	}
}

func TestUpdateCameraStatus_DatabaseError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedDatabaseError := errors.New("camera update failed")
	databaseMock.ExpectExec(`UPDATE cameras`).WithArgs("online", "cam01", "truck001").WillReturnError(expectedDatabaseError)

	// Act
	updateError := UpdateCameraStatus(databaseConnection, "cam01", "truck001", "online")

	// Assert
	if !errors.Is(updateError, expectedDatabaseError) {
		t.Errorf("update error = %v, want %v", updateError, expectedDatabaseError)
	}
}

func TestMarkStaleCamerasOffline_ReturnsAffectedCount(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	offlineTimeout := "15s"
	databaseMock.ExpectExec(`UPDATE cameras[\s\S]+last_seen_at < NOW\(\) - \$1::interval`).
		WithArgs(offlineTimeout).WillReturnResult(sqlmock.NewResult(0, 3))

	// Act
	updatedCameraCount, updateError := MarkStaleCamerasOffline(databaseConnection, offlineTimeout)

	// Assert
	if updateError != nil || updatedCameraCount != 3 {
		t.Errorf("offline update = (%d, %v), want (3, nil)", updatedCameraCount, updateError)
	}
}

func TestMarkStaleCamerasOffline_DatabaseError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedDatabaseError := errors.New("offline update failed")
	databaseMock.ExpectExec(`UPDATE cameras`).WithArgs("15s").WillReturnError(expectedDatabaseError)

	// Act
	updatedCameraCount, updateError := MarkStaleCamerasOffline(databaseConnection, "15s")

	// Assert
	if !errors.Is(updateError, expectedDatabaseError) || updatedCameraCount != 0 {
		t.Errorf("offline update = (%d, %v), want zero and the database error", updatedCameraCount, updateError)
	}
}

func TestMarkStaleCamerasOffline_RowsAffectedError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedResultError := errors.New("affected row count unavailable")
	databaseMock.ExpectExec(`UPDATE cameras`).WithArgs("15s").WillReturnResult(sqlmock.NewErrorResult(expectedResultError))

	// Act
	updatedCameraCount, updateError := MarkStaleCamerasOffline(databaseConnection, "15s")

	// Assert
	if !errors.Is(updateError, expectedResultError) || updatedCameraCount != 0 {
		t.Errorf("offline update = (%d, %v), want zero and the result error", updatedCameraCount, updateError)
	}
}

func TestRefreshTruckStatusesFromCameras_Success(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	databaseMock.ExpectExec(`UPDATE trucks t[\s\S]+FROM cameras c[\s\S]+c.status = 'online'`).
		WillReturnResult(sqlmock.NewResult(0, 2))

	// Act
	refreshError := RefreshTruckStatusesFromCameras(databaseConnection)

	// Assert
	if refreshError != nil {
		t.Errorf("refresh truck statuses: %v", refreshError)
	}
}

func TestRefreshTruckStatusesFromCameras_DatabaseError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedDatabaseError := errors.New("truck status refresh failed")
	databaseMock.ExpectExec(`UPDATE trucks t`).WillReturnError(expectedDatabaseError)

	// Act
	refreshError := RefreshTruckStatusesFromCameras(databaseConnection)

	// Assert
	if !errors.Is(refreshError, expectedDatabaseError) {
		t.Errorf("refresh error = %v, want %v", refreshError, expectedDatabaseError)
	}
}

func TestGetRecordings_Success(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	recordingStartedAt := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	recordingEndedAt := recordingStartedAt.Add(time.Minute)
	recordingRows := sqlmock.NewRows([]string{"recording_id", "truck_id", "camera_id", "started_at", "ended_at", "file_path", "file_size", "status"}).
		AddRow(42, "truck001", "cam01", recordingStartedAt, recordingEndedAt, "recordings/clip.mp4", 4096, "complete")
	databaseMock.ExpectQuery(`FROM recordings[\s\S]+WHERE truck_id = \$1`).
		WithArgs("truck001", "cam01").WillReturnRows(recordingRows).RowsWillBeClosed()
	expectedRecordings := []models.Recording{{RecordingID: 42, TruckID: "truck001", CameraID: "cam01",
		StartedAt: recordingStartedAt, EndedAt: recordingEndedAt, FilePath: "recordings/clip.mp4", FileSize: 4096, Status: "complete"}}

	// Act
	returnedRecordings, queryError := GetRecordings(databaseConnection, "truck001", "cam01")

	// Assert
	if queryError != nil || !reflect.DeepEqual(returnedRecordings, expectedRecordings) {
		t.Errorf("recordings = (%+v, %v), want (%+v, nil)", returnedRecordings, queryError, expectedRecordings)
	}
}

func TestGetRecordings_EmptyResult(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	recordingRows := sqlmock.NewRows([]string{"recording_id", "truck_id", "camera_id", "started_at", "ended_at", "file_path", "file_size", "status"})
	databaseMock.ExpectQuery(`FROM recordings`).WithArgs("truck001", "").WillReturnRows(recordingRows)

	// Act
	returnedRecordings, queryError := GetRecordings(databaseConnection, "truck001", "")

	// Assert
	if queryError != nil || returnedRecordings == nil || len(returnedRecordings) != 0 {
		t.Errorf("recordings = (%+v, %v), want an empty non-nil slice and nil", returnedRecordings, queryError)
	}
}

func TestGetRecordings_QueryError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedDatabaseError := errors.New("recording query failed")
	databaseMock.ExpectQuery(`FROM recordings`).WithArgs("truck001", "cam01").WillReturnError(expectedDatabaseError)

	// Act
	returnedRecordings, queryError := GetRecordings(databaseConnection, "truck001", "cam01")

	// Assert
	if !errors.Is(queryError, expectedDatabaseError) || returnedRecordings != nil {
		t.Errorf("recordings = (%+v, %v), want nil and the database error", returnedRecordings, queryError)
	}
}

func TestGetRecordings_ScanError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	recordingRows := sqlmock.NewRows([]string{"recording_id", "truck_id", "camera_id", "started_at", "ended_at", "file_path", "file_size", "status"}).
		AddRow("invalid-id", "truck001", "cam01", time.Now(), time.Now(), "clip.mp4", 4096, "complete")
	databaseMock.ExpectQuery(`FROM recordings`).WithArgs("truck001", "cam01").WillReturnRows(recordingRows).RowsWillBeClosed()

	// Act
	returnedRecordings, queryError := GetRecordings(databaseConnection, "truck001", "cam01")

	// Assert
	if queryError == nil || returnedRecordings != nil {
		t.Errorf("recordings = (%+v, %v), want nil and a scan error", returnedRecordings, queryError)
	}
}

func TestGetRecordings_RowIterationError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	expectedReadError := errors.New("recording row read failed")
	recordingRows := sqlmock.NewRows([]string{"recording_id", "truck_id", "camera_id", "started_at", "ended_at", "file_path", "file_size", "status"}).
		AddRow(42, "truck001", "cam01", time.Now(), time.Now(), "clip.mp4", 4096, "complete").RowError(0, expectedReadError)
	databaseMock.ExpectQuery(`FROM recordings`).WithArgs("truck001", "cam01").WillReturnRows(recordingRows)

	// Act
	_, queryError := GetRecordings(databaseConnection, "truck001", "cam01")

	// Assert
	if !errors.Is(queryError, expectedReadError) {
		t.Errorf("query error = %v, want %v", queryError, expectedReadError)
	}
}

func TestUserByAPIToken_NotFound(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	lookupTime := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	databaseMock.ExpectQuery(`FROM user_api_tokens`).WithArgs(APITokenHash("missing-token"), lookupTime).WillReturnError(sql.ErrNoRows)

	// Act
	returnedUser, userFound, queryError := UserByAPIToken(databaseConnection, "missing-token", lookupTime)

	// Assert
	if queryError != nil || userFound || returnedUser != (AuthenticatedUser{}) {
		t.Errorf("token lookup = (%+v, %v, %v), want an empty user, false, and nil", returnedUser, userFound, queryError)
	}
}

func TestUserByAPIToken_DatabaseError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	lookupTime := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	expectedDatabaseError := errors.New("token lookup failed")
	databaseMock.ExpectQuery(`FROM user_api_tokens`).WithArgs(APITokenHash("test-token"), lookupTime).WillReturnError(expectedDatabaseError)

	// Act
	returnedUser, userFound, queryError := UserByAPIToken(databaseConnection, "test-token", lookupTime)

	// Assert
	if !errors.Is(queryError, expectedDatabaseError) || userFound || returnedUser != (AuthenticatedUser{}) {
		t.Errorf("token lookup = (%+v, %v, %v), want an empty user, false, and the database error", returnedUser, userFound, queryError)
	}
}

func TestSaveTruckLocation_HistoryInsertErrorRollsBack(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, SpeedKmh: 10, FixQuality: 1}
	recordedAt := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	expectedDatabaseError := errors.New("location history insert failed")
	databaseMock.ExpectBegin()
	databaseMock.ExpectExec(`INSERT INTO truck_locations`).
		WithArgs("truck001", float64(25), float64(121), nil, float64(10), nil, nil, nil, 1, recordedAt, recordedAt).
		WillReturnError(expectedDatabaseError)
	databaseMock.ExpectRollback()

	// Act
	_, saveError := SaveTruckLocation(databaseConnection, "truck001", locationInput, recordedAt, recordedAt, nil, nil, nil)

	// Assert
	if !errors.Is(saveError, expectedDatabaseError) {
		t.Errorf("save error = %v, want %v", saveError, expectedDatabaseError)
	}
}

func TestSaveTruckLocation_StateInsertErrorRollsBack(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, SpeedKmh: 10, FixQuality: 1}
	recordedAt := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	expectedDatabaseError := errors.New("location state insert failed")
	databaseMock.ExpectBegin()
	databaseMock.ExpectExec(`INSERT INTO truck_locations`).
		WithArgs("truck001", float64(25), float64(121), nil, float64(10), nil, nil, nil, 1, recordedAt, recordedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	databaseMock.ExpectExec(`INSERT INTO truck_location_state`).
		WithArgs("truck001", float64(25), float64(121), nil, float64(10), nil, nil, nil, 1, recordedAt, recordedAt, nil, nil, nil).
		WillReturnError(expectedDatabaseError)
	databaseMock.ExpectRollback()

	// Act
	_, saveError := SaveTruckLocation(databaseConnection, "truck001", locationInput, recordedAt, recordedAt, nil, nil, nil)

	// Assert
	if !errors.Is(saveError, expectedDatabaseError) {
		t.Errorf("save error = %v, want %v", saveError, expectedDatabaseError)
	}
}

func TestSaveTruckLocation_CommitError(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newDatabaseTestConnection(t)
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, SpeedKmh: 10, FixQuality: 1}
	recordedAt := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	expectedDatabaseError := errors.New("location commit failed")
	databaseMock.ExpectBegin()
	databaseMock.ExpectExec(`INSERT INTO truck_locations`).
		WithArgs("truck001", float64(25), float64(121), nil, float64(10), nil, nil, nil, 1, recordedAt, recordedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	databaseMock.ExpectExec(`INSERT INTO truck_location_state`).
		WithArgs("truck001", float64(25), float64(121), nil, float64(10), nil, nil, nil, 1, recordedAt, recordedAt, nil, nil, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	databaseMock.ExpectCommit().WillReturnError(expectedDatabaseError)

	// Act
	_, saveError := SaveTruckLocation(databaseConnection, "truck001", locationInput, recordedAt, recordedAt, nil, nil, nil)

	// Assert
	if !errors.Is(saveError, expectedDatabaseError) {
		t.Errorf("save error = %v, want %v", saveError, expectedDatabaseError)
	}
}
