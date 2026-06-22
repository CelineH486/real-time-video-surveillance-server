# Real-time video surveillance server

Go control API for truck status, camera metadata, live-stream access, and recording indexes. Video bytes flow between the truck, the RTSP server, storage, and the browser; the Go server does not proxy video.

## Streaming layout

Each truck publishes a main and sub stream for each of its eight cameras:

```text
truck001/cam01/main  high-quality single-camera view
truck001/cam01/sub   low-bandwidth overview view
```

The truck publishes RTSP. The streaming server exposes WebRTC to the responsive Web client. The overview requests eight `sub` streams; opening one camera requests its `main` stream.

## Configuration

Copy `.env.example` values into the process environment. `STREAM_PUBLIC_BASE_URL` is the browser-facing WebRTC endpoint of the streaming server. `STREAM_SIGNING_KEY` signs five-minute stream tokens and must be replaced in production. Configure the streaming server to validate the same token before exposing it publicly.

Run `db/migrations/001_streaming.sql` once against the existing database before using camera heartbeat or recording APIs.

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

With the Go server already listening on port 8080:

```powershell
docker compose up -d mediamtx
```

Open RTSP `8554/tcp`, WebRTC signaling `8889/tcp`, and WebRTC ICE `8189/tcp+udp` on the server firewall. For access across the internet, set MediaMTX `webrtcAdditionalHosts` to the server's public IP or DNS name.

MediaMTX calls `POST /internal/mediamtx/auth` for every publish, live-read, and playback request. Truck publishers use their `truckId` as the RTSP username and `STREAM_PUBLISH_PASSWORD` as the password. Web viewers use the five-minute token returned by the Go API.

## Start eight truck streams

On the truck computer, copy `truck/streams.example.json` to the ignored file `truck/streams.json`, fill in the eight camera RTSP URLs and server address, then run:

```powershell
powershell -ExecutionPolicy Bypass -File truck/start-streams.ps1
```

Each camera publishes an unchanged high-quality `main` stream and a 360p/12fps `sub` stream. MediaMTX records only `main` in five-minute fMP4 segments; files are retained until an explicit retention policy is chosen.

User login authentication is still required before production exposure. The current viewing API verifies the truck/camera and issues MediaMTX-compatible short-lived stream tokens.
