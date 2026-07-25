# Real-time video surveillance server

Go control API for truck status, camera metadata, live-stream access, and recording indexes. Live video flows directly between the truck, MediaMTX, and the browser. Historical MP4 responses are authorized and proxied by the Go API.

## Project structure

```text
controllers/  HTTP request validation and responses
services/     streaming tokens, recordings, and status synchronization
routes/       all HTTP route registration
db/           PostgreSQL queries and migrations
models/       shared data models
web/          responsive monitoring dashboard
truck/        truck-side FFmpeg publisher scripts
main.go       dependency wiring and server startup only
```

## Streaming layout

Each truck publishes a main and sub stream for each of its nine cameras:

```text
truck001/cam01/main  high-quality single-camera view
truck001/cam01/sub   low-bandwidth overview view
```

The truck publishes RTSP. The streaming server exposes WebRTC to the responsive Web client. The overview requests nine `sub` streams; opening one camera requests its `main` stream.

## Configuration

Copy `.env.example` to `.env` before running Docker Compose:

```bash
cp .env.example .env
```

Update `.env` with the VM public IP or DNS name and replace every secret value. `API_PUBLIC_BASE_URL` is the browser-facing Go API endpoint, `STREAM_PUBLIC_BASE_URL` is the browser-facing MediaMTX WebRTC endpoint, and `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` tells MediaMTX which public IP or DNS name WebRTC clients should use. `STREAM_SIGNING_KEY`, `STREAM_PUBLISH_PASSWORD`, and `POSTGRES_PASSWORD` must be replaced before exposing the service.

Run `db/migrations/000_base.sql`, `db/migrations/001_streaming.sql`, and `db/migrations/002_composite_camera_key.sql` in order when installing against an existing PostgreSQL server. The Docker Compose development stack applies them automatically.

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

```bash
cp .env.example .env
# Edit .env before starting the stack.
docker compose up -d --build
```

On older Ubuntu packages, use `docker-compose up -d --build` instead of `docker compose up -d --build`.

Open the responsive monitoring dashboard at the `API_PUBLIC_BASE_URL` configured in `.env`, for example `http://YOUR_VM_EXTERNAL_IP_OR_DNS:8080/web/`. It shows nine low-bandwidth sub-streams, switches to the high-quality main stream when a camera is opened, and provides historical recordings in the same viewer.

The development database is seeded with `truck001` and `cam01` through `cam09`. These values are for local testing only.

Open RTSP `8554/tcp`, WebRTC signaling `8889/tcp`, WebRTC ICE `8189/tcp+udp`, API `8080/tcp`, and truck status `5000/udp` on the server firewall. Do not open PostgreSQL `5432/tcp` or MediaMTX playback `9996/tcp` to the internet; the Compose stack keeps them internal. For access across the internet, set `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` in `.env` to the server's public IP or DNS name.

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

User login authentication is still required before production exposure. The current viewing API verifies the truck/camera and issues MediaMTX-compatible short-lived stream tokens.
