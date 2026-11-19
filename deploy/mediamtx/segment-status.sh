#!/bin/sh
set -eu
file=${1:?missing segment}
case "$file" in /recordings/*.mp4) ;; *) exit 1 ;; esac
[ -f "$file" ] || exit 1
before=$(stat -c %s "$file")
# Decode the complete available file before marking it downloadable.
ffmpeg -v error -xerror -i "$file" -map 0:v:0 -f null - </dev/null >/dev/null 2>&1 || exit 1
size=$(stat -c %s "$file")
[ "$size" -gt 0 ] || exit 1
[ "$before" = "$size" ] || exit 1
marker="/recording-status/${file#/recordings/}.complete"
mkdir -p "$(dirname "$marker")"
tmp="$marker.tmp.$$"
printf '%s\n' "$size" > "$tmp"
mv -f "$tmp" "$marker"
