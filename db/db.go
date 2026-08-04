package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"real-time-video-surveillance-system/models"
)

type AuthenticatedUser struct {
	UserID      string
	Email       string
	DisplayName string
}

type PasswordUser struct {
	AuthenticatedUser
	PasswordHash string
}

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

func GetCamerasForUser(dbConn *sql.DB, userID string) ([]map[string]string, error) {
	rows, err := dbConn.Query(`
		SELECT
			c.camera_id,
			c.truck_id,
			c.name,
			c.status
		FROM cameras c
		INNER JOIN user_trucks ut ON ut.truck_id = c.truck_id
		WHERE ut.user_id = $1
		ORDER BY c.truck_id, c.camera_id
	`, userID)
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
		if err := rows.Scan(&cameraID, &truckID, &name, &status); err != nil {
			return nil, err
		}
		cameras = append(cameras, map[string]string{
			"cameraId": cameraID,
			"truckId":  truckID,
			"name":     name,
			"status":   status,
		})
	}
	return cameras, rows.Err()
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

func GetTrucksForUser(dbConn *sql.DB, userID string) ([]map[string]string, error) {
	rows, err := dbConn.Query(`
		SELECT
			t.truck_id,
			t.plate_no,
			t.status
		FROM trucks t
		INNER JOIN user_trucks ut ON ut.truck_id = t.truck_id
		WHERE ut.user_id = $1
		ORDER BY t.truck_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trucks []map[string]string
	for rows.Next() {
		var truckID string
		var plateNo string
		var status string
		if err := rows.Scan(&truckID, &plateNo, &status); err != nil {
			return nil, err
		}
		trucks = append(trucks, map[string]string{
			"truckId": truckID,
			"plateNo": plateNo,
			"status":  status,
		})
	}
	return trucks, rows.Err()
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

func RefreshTruckStatusesFromCameras(dbConn *sql.DB) error {
	_, err := dbConn.Exec(`
		UPDATE trucks t
		SET status = CASE WHEN EXISTS (
			SELECT 1 FROM cameras c
			WHERE c.truck_id = t.truck_id AND c.status = 'online'
		) THEN 'online' ELSE 'offline' END
	`)
	return err
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

func UserByAPIToken(dbConn *sql.DB, token string, now time.Time) (AuthenticatedUser, bool, error) {
	tokenHash := APITokenHash(token)
	var user AuthenticatedUser
	err := dbConn.QueryRow(`
		SELECT u.user_id, u.email, u.display_name
		FROM user_api_tokens t
		INNER JOIN users u ON u.user_id = t.user_id
		WHERE t.token_hash = $1
		  AND u.status = 'active'
		  AND (t.expires_at IS NULL OR t.expires_at > $2)
	`, tokenHash, now).Scan(&user.UserID, &user.Email, &user.DisplayName)
	if err == sql.ErrNoRows {
		return AuthenticatedUser{}, false, nil
	}
	if err != nil {
		return AuthenticatedUser{}, false, err
	}
	return user, true, nil
}

func UserByEmail(dbConn *sql.DB, email string) (PasswordUser, bool, error) {
	var user PasswordUser
	err := dbConn.QueryRow(`
		SELECT user_id, email, display_name, password_hash
		FROM users
		WHERE LOWER(email) = LOWER($1) AND status = 'active' AND password_hash IS NOT NULL
	`, email).Scan(&user.UserID, &user.Email, &user.DisplayName, &user.PasswordHash)
	if err == sql.ErrNoRows {
		return PasswordUser{}, false, nil
	}
	if err != nil {
		return PasswordUser{}, false, err
	}
	return user, true, nil
}

func CreateAPIToken(dbConn *sql.DB, userID, token, label string, expiresAt time.Time) error {
	_, err := dbConn.Exec(`
		INSERT INTO user_api_tokens (token_hash, user_id, label, expires_at)
		VALUES ($1, $2, $3, $4)
	`, APITokenHash(token), userID, label, expiresAt)
	return err
}

func DeleteAPIToken(dbConn *sql.DB, token string) error {
	_, err := dbConn.Exec(`DELETE FROM user_api_tokens WHERE token_hash = $1`, APITokenHash(token))
	return err
}

func UserCanAccessTruck(dbConn *sql.DB, userID string, truckID string) (bool, error) {
	var allowed bool
	err := dbConn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM user_trucks WHERE user_id = $1 AND truck_id = $2
		)
	`, userID, truckID).Scan(&allowed)
	return allowed, err
}

func APITokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
