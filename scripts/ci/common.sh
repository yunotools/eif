#!/usr/bin/env bash
set -euo pipefail

log() {
  printf '[eif-ci] %s\n' "$*"
}

die() {
  printf '[eif-ci] ERROR: %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

sha256_file() {
  sha256sum "$1" | awk '{print $1}'
}

write_output() {
  local key=$1
  local value=$2
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    printf '%s=%s\n' "$key" "$value" >>"$GITHUB_OUTPUT"
  fi
}
