package routes

import (
	"io/fs"
	"net/http"

	"real-time-video-surveillance-system/controllers"
	"real-time-video-surveillance-system/services"
)

type Controllers struct {
	Health     *controllers.HealthController
	Trucks     *controllers.TruckController
	Cameras    *controllers.CameraController
	Streams    *controllers.StreamController
	Recordings *controllers.RecordingController
	MediaMTX   *controllers.MediaMTXController
	Auth       *services.AuthService
	Sessions   *controllers.SessionController
}

func New(controllers Controllers, webContent fs.FS) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /web/", http.StripPrefix("/web/", http.FileServer(http.FS(webContent))))
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		page, err := fs.ReadFile(webContent, "index.html")
		if err != nil {
			http.Error(w, "Login page unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("GET /health", controllers.Health.Get)
	mux.HandleFunc("POST /api/auth/login", controllers.Sessions.Login)
	mux.HandleFunc("POST /api/auth/logout", controllers.Sessions.Logout)
	mux.HandleFunc("POST /internal/mediamtx/auth", controllers.MediaMTX.Auth)
	mux.HandleFunc("GET /api/cameras", controllers.Auth.Middleware(controllers.Cameras.List))
	mux.HandleFunc("GET /api/trucks", controllers.Auth.Middleware(controllers.Trucks.List))
	mux.HandleFunc("GET /api/trucks/{truckID}/cameras", controllers.Auth.Middleware(controllers.Cameras.ListByTruck))
	mux.HandleFunc("POST /api/trucks/{truckID}/cameras/{cameraID}/play", controllers.Auth.Middleware(controllers.Streams.Play))
	mux.HandleFunc("GET /api/trucks/{truckID}/recordings", controllers.Auth.Middleware(controllers.Recordings.List))
	mux.HandleFunc("POST /api/trucks/{truckID}/cameras/{cameraID}/recordings/play", controllers.Auth.Middleware(controllers.Recordings.Play))
	mux.HandleFunc("GET /api/trucks/{truckID}/cameras/{cameraID}/recordings/content", controllers.Recordings.Content)
	return mux
}
