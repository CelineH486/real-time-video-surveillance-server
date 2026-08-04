# Real-time video surveillance server

Go control API for truck status, camera metadata, live-stream access, and recording indexes. Live video flows directly between the truck, MediaMTX, and the browser. Historical MP4 responses are authorized and proxied by the Go API.

## Project structure

```text
controllers/  HTTP request validation and responses
services/     streaming tokens, recordings, and status synchronization
routes/       all HTTP route registration
db/           PostgreSQL queries and migrations
models/       shared data models
apps/mobile/  shared Flutter/Dart application for Web and mobile
web/          generated Flutter Web assets (not committed)
truck/        truck-side FFmpeg publisher scripts
main.go       dependency wiring and server startup only
```

## Streaming layout

Each truck publishes a main and sub stream for each of its nine cameras:

```text
truck001/cam01/main  high-quality single-camera view
truck001/cam01/sub   low-bandwidth overview view
```

The truck publishes RTSP. MediaMTX exposes WebRTC/WHEP to the shared Flutter application. The responsive overview requests nine `sub` streams; opening one camera requests its `main` stream.

## Configuration

Copy `.env.example` values into the process environment. `STREAM_PUBLIC_BASE_URL` is the browser-facing WebRTC endpoint of the streaming server. `STREAM_SIGNING_KEY` signs five-minute stream tokens and must be replaced in production. Configure the streaming server to validate the same token before exposing it publicly.

User-facing API routes require an `Authorization: Bearer <api-token>` header. API tokens are stored only as SHA-256 hashes in `user_api_tokens`, and truck visibility is granted through `user_trucks`. The local Docker seed creates a development user that can view `truck001` with token `dev-user-token`; replace this with a real login/JWT flow before production.

API exceptions are returned as JSON:

```json
{
  "error": {
    "code": "truck_access_denied",
    "message": "The authenticated user is not assigned to this truck"
  }
}
```

Run the files in `db/migrations/` in numeric order when installing against an existing PostgreSQL server. The Docker Compose stack applies them automatically to a new database volume.

## Environment and Compose profiles

Copy `.env.example` to the Git-ignored `.env` file and replace every
`change-me` value. Secrets must not be written into Compose files or committed
to Git.

- `compose.yaml` contains the shared application architecture.
- `compose.dev.yaml` publishes local ports, loads development seed data, and
  enables the test-video publisher.
- `compose.prod.yaml` publishes only the host ports needed by the production
  reverse proxy, truck heartbeat, RTSP publishing, and WebRTC.

Start development:

```powershell
docker compose -f compose.yaml -f compose.dev.yaml up -d --build
```

Validate and start production:

```bash
cp .env.production.example .env.production
docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml config
docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml up -d --build
```

The complete DNS, TLS, NGINX, MediaMTX, and firewall procedure is documented
in [`deploy/PRODUCTION.md`](deploy/PRODUCTION.md).

## API

- `GET /api/trucks`
- `GET /api/trucks/{truckID}/cameras` — cameras plus overview `subUrl`
- `POST /api/trucks/{truckID}/cameras/{cameraID}/play` — validated main-stream information
- `GET /api/trucks/{truckID}/recordings?cameraId=cam01&start=...&end=...` — recorded spans with signed playback URLs
- `POST /api/trucks/{truckID}/cameras/{cameraID}/recordings/play` — signed MP4 playback URL
- `GET /health`

Truck status continues to arrive over UDP port 5000. Camera heartbeats can be included without breaking the original payload:

```json
{
  "truckId": "truck001",
  "status": "online",
  "cameras": [
    {"cameraId": "cam01", "status": "online"},
    {"cameraId": "cam02", "status": "offline"}
  ]
}
```

The server updates `last_seen_at` for each camera heartbeat. A camera without a heartbeat for `CAMERA_OFFLINE_TIMEOUT` (15 seconds by default) is marked offline.

## Start MediaMTX

To start PostgreSQL, the Go API, and MediaMTX together:

```powershell
docker compose -f compose.yaml -f compose.dev.yaml up -d --build
```

Open the responsive Flutter dashboard at `http://localhost:8080/web/`. It shows nine low-bandwidth sub-streams, switches to the high-quality main stream when a camera is opened, and provides historical recordings in the same viewer. The Docker build compiles `apps/mobile` for Web and embeds the generated assets in the Go server.

For Flutter development without rebuilding the Go image:

```powershell
cd apps/mobile
flutter run -d chrome --dart-define=API_BASE_URL=http://localhost:8080
```

The Web and mobile targets share the same Dart screens, API client, session
storage, and WHEP player. There is no separate handwritten JavaScript
dashboard to maintain.

The development database is seeded with `truck001`, `cam01` through `cam09`, and a development viewer token. These values are for local testing only.

Open RTSP `8554/tcp`, WebRTC signaling `8889/tcp`, and WebRTC ICE `8189/tcp+udp` on the server firewall. For access across the internet, set MediaMTX `webrtcAdditionalHosts` to the server's public IP or DNS name.

MediaMTX calls `POST /internal/mediamtx/auth` for every publish, live-read, and playback request. Truck publishers use their `truckId` as the RTSP username and `STREAM_PUBLISH_PASSWORD` as the password. Web viewers use the five-minute token returned by the Go API.

Live API responses contain a WHEP URL ending in `/whep` and a separate `accessToken`. The Web client must pass that token to `MediaMTXWebRTCReader`; it must not append it to the iframe or URL query string.

## Start nine truck streams

On the truck computer, copy `truck/streams.example.json` to the ignored file `truck/streams.json`, fill in the nine camera RTSP URLs and server address, then run:

```powershell
powershell -ExecutionPolicy Bypass -File truck/start-streams.ps1
```

Each camera publishes an unchanged high-quality `main` stream and a 360p/12fps `sub` stream. MediaMTX records only `main` in five-minute fMP4 segments; files are retained until an explicit retention policy is chosen.

## End-to-end video test

Place local test files `cam1.mp4` through `cam9.mp4` in the ignored directory `testdata/videos/`. The test script publishes them as nine main/sub camera pairs:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/e2e-video-test.ps1 `
  -LoopCount 20
```

`-VideoDirectory` remains available when you want to test a different set of `cam1.mp4` through `cam9.mp4` files.

During publishing, `GET /api/trucks/truck001/cameras` reports all cameras as `online`. They become `offline` after the configured timeout when the publishers stop. Completed main streams are available through the recordings API.

## Authorization E2E test

After starting the target stack with seeded or equivalent test data, run the authorization E2E test with an explicit API base URL and API token:

```bash
E2E_API_BASE_URL=http://localhost:8080 \
E2E_API_TOKEN=dev-user-token \
go test -tags=e2e ./tests/e2e
```

The test uses AAA structure and validates the viewer authorization flow only: missing API tokens are rejected, the configured user can access `truck001`, unassigned trucks are forbidden, stream tokens are issued for assigned cameras, and the issued token passes the MediaMTX auth hook. `E2E_API_BASE_URL` and `E2E_API_TOKEN` are required. Override seeded identifiers with `E2E_TRUCK_ID`, `E2E_CAMERA_ID`, or `E2E_UNASSIGNED_TRUCK_ID`.

The Web dashboard authenticates with a `.com` email address and password through `POST /api/auth/login` and revokes the session through `POST /api/auth/logout`. Passwords are stored as bcrypt hashes; the server requires 8-72 characters with at least one uppercase letter, one lowercase letter, and one number. The development login is `dev@example.com` / `Dev12345`.
