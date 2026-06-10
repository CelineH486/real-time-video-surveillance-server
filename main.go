package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/models"
)

var databaseConn *sql.DB

func startUDPServer() {
	addr, err := net.ResolveUDPAddr("udp", ":5000")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("UDP Server Listening :5000")

	buffer := make([]byte, 65535)

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println(err)
			continue
		}

		var truckStatus models.TruckStatus

		err = json.Unmarshal(buffer[:n], &truckStatus)
		if err != nil {
			log.Println("Invalid JSON:", err)
			continue
		}

		err = db.UpdateTruckStatus(
			databaseConn,
			truckStatus.TruckID,
			truckStatus.Status,
		)

		if err != nil {
			log.Println(err)
			continue
		}

		fmt.Printf(
			"TruckID=%s Status=%s Updated DB\n",
			truckStatus.TruckID,
			truckStatus.Status,
		)
	}
}

func startHTTPServer() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]string{
			"status": "ok",
		}

		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/api/cameras", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		cameras, err := db.GetCameras(databaseConn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(cameras)
	})

	http.HandleFunc("/api/trucks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		trucks, err := db.GetTrucks(databaseConn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(trucks)
	})

	http.HandleFunc("/api/trucks/truck001/cameras", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		cameras, err := db.GetCamerasByTruckID(databaseConn, "truck001")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(cameras)
	})

	fmt.Println("HTTP Server Listening :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	var err error

	databaseConn, err = db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer databaseConn.Close()

	go startUDPServer()

	startHTTPServer()
}
