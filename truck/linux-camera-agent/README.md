# Cam01 H.265 recording pilot

This package matches the `camera-agent@.service` unit currently running on the
LPA3588 vehicle. It keeps `main` and `sub` as H.264 live streams. Only a camera
with `CAMERA_ARCHIVE_ENABLED=true` adds an H.265 `archive` stream. The camera is
opened once; GStreamer splits its raw frames before encoding.

For the first pilot, set only `/etc/camera-agent/cam01.env` to enable the archive
stream. Do not put the publish password in this repository.

```ini
CAMERA_ARCHIVE_ENABLED=true
CAMERA_ARCHIVE_CONTENT_WIDTH=854
CAMERA_ARCHIVE_WIDTH=864
CAMERA_ARCHIVE_HEIGHT=480
CAMERA_ARCHIVE_FPS=8
CAMERA_ARCHIVE_BITRATE_BPS=300000
CAMERA_ARCHIVE_GOP=24
```

The 864-pixel encoded width includes a 10-pixel black border on the right. The
854-pixel picture matches the current main stream. The aligned width avoids the
RGA errors observed in the 854-pixel hardware-encoder test.

Before installing on the vehicle, check that GStreamer has `mpph265enc`,
`h265parse`, `mpegtsmux`, and `videobox`. Keep a copy of the current
`/usr/local/bin/camera-agent` before replacing it. Restart only
`camera-agent@cam01.service` for the pilot and check that `main`, `sub`, and
`archive` all publish. Other camera units continue using the same script with
the archive branch disabled.

The server's MediaMTX configuration records only `truck001/cam01/archive` for
the pilot. Existing H.264 `main` recordings remain available until the normal
seven-day deletion period. No server-side transcoding is required.
