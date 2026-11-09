#!/usr/bin/env bash

percent_encode() {
  local input="${1-}"
  local output=""
  local character hex index

  LC_ALL=C
  for ((index = 0; index < ${#input}; index++)); do
    character="${input:index:1}"
    case "${character}" in
      [a-zA-Z0-9.~_-]) output+="${character}" ;;
      *)
        printf -v hex '%02X' "'${character}"
        output+="%${hex}"
        ;;
    esac
  done
  printf '%s' "${output}"
}

rtsp_base_url() {
  local truck_id="$1"
  local password="$2"
  local server="$3"
  local camera_id="$4"

  server="${server#rtsp://}"
  server="${server%/}"
  printf 'rtsp://%s:%s@%s/%s/%s' \
    "$(percent_encode "${truck_id}")" \
    "$(percent_encode "${password}")" \
    "${server}" \
    "$(percent_encode "${truck_id}")" \
    "$(percent_encode "${camera_id}")"
}
