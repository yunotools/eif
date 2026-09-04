#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

OUT_DIR=${1:-dist/security/source}
mkdir -p "$OUT_DIR"
require_cmd govulncheck

log "Running govulncheck"
set +e
govulncheck ./... 2>&1 | tee "$OUT_DIR/govulncheck.txt"
status=${PIPESTATUS[0]}
set -e
exit "$status"
