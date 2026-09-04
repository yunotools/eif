#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

IMAGE=${1:?usage: trivy-image.sh IMAGE [OUT_DIR]}
OUT_DIR=${2:-dist/security/image}
mkdir -p "$OUT_DIR"

require_cmd trivy

log "Creating image vulnerability report for $IMAGE"
trivy image \
  --scanners vuln \
  --format json \
  --output "$OUT_DIR/trivy-image.json" \
  "$IMAGE"

log "Creating image SBOM"
trivy image \
  --scanners vuln \
  --format cyclonedx \
  --output "$OUT_DIR/sbom-image.cdx.json" \
  "$IMAGE"

status=0

log "Failing on CRITICAL image vulnerabilities"
if ! trivy image \
  --scanners vuln \
  --severity CRITICAL \
  --exit-code 1 \
  "$IMAGE"; then
  status=1
fi

log "Failing on secrets embedded in image layers"
if ! trivy image \
  --scanners secret \
  --exit-code 1 \
  --format table \
  "$IMAGE"; then
  status=1
fi

exit "$status"
