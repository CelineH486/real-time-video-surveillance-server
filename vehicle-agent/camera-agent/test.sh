#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "${script_dir}/camera-agent-lib.sh"

assert_equal() {
  local expected="$1"
  local actual="$2"
  if [[ "${actual}" != "${expected}" ]]; then
    printf 'expected: %s\nactual:   %s\n' "${expected}" "${actual}" >&2
    exit 1
  fi
}

assert_equal 'abc%40123' "$(percent_encode 'abc@123')"
assert_equal 'abc%3A123' "$(percent_encode 'abc:123')"
assert_equal 'abc%2F123' "$(percent_encode 'abc/123')"
assert_equal 'abc%25123' "$(percent_encode 'abc%123')"
assert_equal \
  'rtsp://truck001:abc%40123@stream.example.com:8554/truck001/cam01' \
  "$(rtsp_base_url 'truck001' 'abc@123' 'rtsp://stream.example.com:8554/' 'cam01')"

echo 'camera-agent tests passed'
