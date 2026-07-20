#!/bin/sh
set -eu

publish() {
  number="$1"
  camera_id=$(printf 'cam%02d' "$number")
  video="/videos/cam${number}.mp4"
  target="rtsp://truck001:${STREAM_PUBLISH_PASSWORD}@mediamtx:8554/truck001/${camera_id}"

  ffmpeg -hide_banner -loglevel warning -nostdin -stream_loop -1 -re -i "$video" \
    -map 0:v:0 -an -c:v libx264 -preset veryfast -tune zerolatency \
    -pix_fmt yuv420p -b:v 2500k -maxrate 3000k -bufsize 5000k -g 60 \
    -f rtsp -rtsp_transport tcp "$target/main" &

  ffmpeg -hide_banner -loglevel warning -nostdin -stream_loop -1 -re -i "$video" \
    -map 0:v:0 -an -c:v libx264 -preset veryfast -tune zerolatency \
    -pix_fmt yuv420p -vf 'scale=-2:360,fps=12' \
    -b:v 500k -maxrate 600k -bufsize 1000k -g 24 \
    -f rtsp -rtsp_transport tcp "$target/sub" &
}

for number in 1 2 3 4 5 6 7 8 9; do
  publish "$number"
done

wait
