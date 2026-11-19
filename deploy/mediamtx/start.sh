#!/bin/sh
set -eu
# Snapshot paths before MediaMTX starts, then verify them in the background so
# an existing archive does not delay incoming streams after a restart.
find /recordings -type f -name '*.mp4' > /tmp/recording-recovery-list
while IFS= read -r file; do
  marker="/recording-status/${file#/recordings/}.complete"
  if [ ! -f "$marker" ] || [ "$(cat "$marker" 2>/dev/null || true)" != "$(stat -c %s "$file")" ]; then
    /segment-status.sh "$file" || echo "Could not validate recording: $file" >&2
  fi
done < /tmp/recording-recovery-list &
exec /mediamtx "$@"
