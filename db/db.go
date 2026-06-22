package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"real-time-video-surveillance-system/models"
)

func Connect() (*sql.DB, error) {

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "host=localhost port=5432 user=postgres password=123456 dbname=mydb sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("PostgreSQL Connected")

	return db, nil
}

func GetCameras(dbConn *sql.DB) ([]map[string]string, error) {

	rows, err := dbConn.Query(`
		SELECT
			camera_id,
			truck_id,
			name,
			status
		FROM cameras
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cameras []map[string]string

	for rows.Next() {

		var cameraID string
		var truckID string
		var name string
		var status string

		err := rows.Scan(
			&cameraID,
			&truckID,
			&name,
			&status,
		)
		if err != nil {
			return nil, err
		}

		cameras = append(cameras, map[string]string{
			"cameraId": cameraID,
			"truckId":  truckID,
			"name":     name,
			"status":   status,
		})
	}

	return cameras, nil
}

func GetTrucks(dbConn *sql.DB) ([]map[string]string, error) {

	rows, err := dbConn.Query(`
		SELECT
			truck_id,
			plate_no,
			status
		FROM trucks
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trucks []map[string]string

	for rows.Next() {

		var truckID string
		var plateNo string
		var status string

		err := rows.Scan(
			&truckID,
			&plateNo,
			&status,
		)
		if err != nil {
			return nil, err
		}

		trucks = append(trucks, map[string]string{
			"truckId": truckID,
			"plateNo": plateNo,
			"status":  status,
		})
	}

	return trucks, nil
}

func GetCamerasByTruckID(dbConn *sql.DB, truckID string) ([]map[string]string, error) {
	rows, err := dbConn.Query(`
		SELECT
			camera_id,
			truck_id,
			name,
			status
		FROM cameras
		WHERE truck_id = $1
		ORDER BY camera_id
	`, truckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cameras []map[string]string

	for rows.Next() {
		var cameraID string
		var truckID string
		var name string
		var status string

		err := rows.Scan(&cameraID, &truckID, &name, &status)
		if err != nil {
			return nil, err
		}

		cameras = append(cameras, map[string]string{
			"cameraId": cameraID,
			"truckId":  truckID,
			"name":     name,
			"status":   status,
		})
	}

	return cameras, nil
}

func UpdateTruckStatus(dbConn *sql.DB, truckID string, status string) error {
	_, err := dbConn.Exec(`
		UPDATE trucks
		SET status = $1
		WHERE truck_id = $2
	`, status, truckID)

	return err
}

func UpdateCameraStatus(dbConn *sql.DB, cameraID string, truckID string, status string) error {
	_, err := dbConn.Exec(`
		UPDATE cameras
		SET status = $1, last_seen_at = NOW()
		WHERE camera_id = $2 AND truck_id = $3
	`, status, cameraID, truckID)

	return err
}

func MarkStaleCamerasOffline(dbConn *sql.DB, timeout string) (int64, error) {
	result, err := dbConn.Exec(`
		UPDATE cameras
		SET status = 'offline'
		WHERE status <> 'offline'
		  AND last_seen_at IS NOT NULL
		  AND last_seen_at < NOW() - $1::interval
	`, timeout)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func CameraExists(dbConn *sql.DB, truckID string, cameraID string) (bool, error) {
	var exists bool
	err := dbConn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM cameras WHERE truck_id = $1 AND camera_id = $2
		)
	`, truckID, cameraID).Scan(&exists)
	return exists, err
}

func GetRecordings(dbConn *sql.DB, truckID string, cameraID string) ([]models.Recording, error) {
	rows, err := dbConn.Query(`
		SELECT recording_id, truck_id, camera_id, started_at, ended_at,
		       file_path, file_size, status
		FROM recordings
		WHERE truck_id = $1 AND ($2 = '' OR camera_id = $2)
		ORDER BY started_at DESC
	`, truckID, cameraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recordings := make([]models.Recording, 0)
	for rows.Next() {
		var recording models.Recording
		if err := rows.Scan(
			&recording.RecordingID, &recording.TruckID, &recording.CameraID,
			&recording.StartedAt, &recording.EndedAt, &recording.FilePath,
			&recording.FileSize, &recording.Status,
		); err != nil {
			return nil, err
		}
		recordings = append(recordings, recording)
	}
	return recordings, rows.Err()
}
