package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {

	connStr :=
		"host=localhost " +
			"port=5432 " +
			"user=postgres " +
			"password=123456 " +
			"dbname=mydb " +
			"sslmode=disable"

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
