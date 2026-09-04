#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

OUT_DIR=${1:-dist/security/source}
mkdir -p "$OUT_DIR"

require_cmd trivy

log "Creating source vulnerability/misconfiguration report"
trivy fs \
  --scanners vuln,misconfig \
  --format json \
  --output "$OUT_DIR/trivy-fs.json" \
  .

log "Creating source SBOM"
trivy fs \
  --scanners vuln \
  --format cyclonedx \
  --output "$OUT_DIR/sbom.cdx.json" \
  .

status=0

log "Failing on CRITICAL source vulnerabilities/misconfigurations"
if ! trivy fs \
  --scanners vuln,misconfig \
  --severity CRITICAL \
  --exit-code 1 \
  .; then
  status=1
fi

# Do not persist a JSON secret report: it can itself contain sensitive snippets.
log "Failing on any detected secret"
if ! trivy fs \
  --scanners secret \
  --exit-code 1 \
  --format table \
  .; then
  status=1
fi

exit "$status"
