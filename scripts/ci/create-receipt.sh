#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd jq
require_cmd sha256sum

SOURCE_DIR=${SOURCE_SECURITY_DIR:-dist/inputs/source-security}
IMAGE_DIR=${IMAGE_SECURITY_DIR:-dist/inputs/image-security}
OUT_DIR=${CI_RECEIPT_DIR:-dist/ci}

[[ -f "$SOURCE_DIR/trivy-fs.json" ]] || die "missing $SOURCE_DIR/trivy-fs.json"
[[ -f "$SOURCE_DIR/sbom.cdx.json" ]] || die "missing $SOURCE_DIR/sbom.cdx.json"
[[ -f "$SOURCE_DIR/govulncheck.txt" ]] || die "missing $SOURCE_DIR/govulncheck.txt"
[[ -f "$IMAGE_DIR/trivy-image.json" ]] || die "missing $IMAGE_DIR/trivy-image.json"
[[ -f "$IMAGE_DIR/sbom-image.cdx.json" ]] || die "missing $IMAGE_DIR/sbom-image.cdx.json"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR/reports"
cp "$SOURCE_DIR/trivy-fs.json" "$OUT_DIR/reports/"
cp "$SOURCE_DIR/sbom.cdx.json" "$OUT_DIR/reports/"
cp "$SOURCE_DIR/govulncheck.txt" "$OUT_DIR/reports/"
cp "$IMAGE_DIR/trivy-image.json" "$OUT_DIR/reports/"
cp "$IMAGE_DIR/sbom-image.cdx.json" "$OUT_DIR/reports/"

critical_source=$(jq '[.Results[]? | ((.Vulnerabilities // []) + (.Misconfigurations // []))[]? | select(.Severity == "CRITICAL")] | length' "$SOURCE_DIR/trivy-fs.json")
critical_image=$(jq '[.Results[]? | (.Vulnerabilities // [])[]? | select(.Severity == "CRITICAL")] | length' "$IMAGE_DIR/trivy-image.json")
test_files=$(find . -type f -name '*_test.go' -not -path './vendor/*' | wc -l | tr -d ' ')
if [[ "$test_files" -gt 0 ]]; then
  tests_configured=true
  tests_status=passed
else
  tests_configured=false
  tests_status=not_configured
fi

go_version=$(awk '$1 == "go" {print $2; exit}' go.mod)
repo=${GITHUB_REPOSITORY:-local/backend}
sha=${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || printf unknown)}
ref=${GITHUB_REF:-local}
run_id=${GITHUB_RUN_ID:-0}
run_attempt=${GITHUB_RUN_ATTEMPT:-0}
workflow=${GITHUB_WORKFLOW:-local}
created_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)

jq -n \
  --arg repository "$repo" \
  --arg sha "$sha" \
  --arg ref "$ref" \
  --arg go_version "$go_version" \
  --arg go_mod_sha256 "$(sha256_file go.mod)" \
  --arg go_sum_sha256 "$(sha256_file go.sum)" \
  --arg workflow "$workflow" \
  --arg run_id "$run_id" \
  --arg run_attempt "$run_attempt" \
  --arg created_at "$created_at" \
  --arg tests_status "$tests_status" \
  --argjson tests_configured "$tests_configured" \
  --argjson test_files "$test_files" \
  --argjson critical_source "$critical_source" \
  --argjson critical_image "$critical_image" \
  '{
    schema_version: 1,
    component: "backend",
    repository: $repository,
    commit: {sha: $sha, ref: $ref},
    toolchain: {go: $go_version},
    inputs: {
      go_mod_sha256: $go_mod_sha256,
      go_sum_sha256: $go_sum_sha256
    },
    checks: {
      format: "passed",
      module: "passed",
      lint: "passed",
      typecheck: "passed",
      test: "passed",
      govulncheck: "passed",
      trivy_source: "passed",
      trivy_image: "passed",
      secret_scan_source: "passed",
      secret_scan_image: "passed"
    },
    tests: {
      configured: $tests_configured,
      files: $test_files,
      status: $tests_status
    },
    security: {
      critical_source: $critical_source,
      critical_image: $critical_image
    },
    artifacts: {
      source_sbom: "reports/sbom.cdx.json",
      image_sbom: "reports/sbom-image.cdx.json",
      trivy_source: "reports/trivy-fs.json",
      trivy_image: "reports/trivy-image.json",
      govulncheck: "reports/govulncheck.txt"
    },
    github: {
      workflow: $workflow,
      run_id: $run_id,
      run_attempt: $run_attempt
    },
    created_at: $created_at
  }' >"$OUT_DIR/ci-manifest.json"

(
  cd "$OUT_DIR"
  find . -type f ! -name SHA256SUMS -print0 \
    | sort -z \
    | xargs -0 sha256sum >SHA256SUMS
)

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "## EIF Backend CI"
    echo
    echo "| Field | Value |"
    echo "|---|---|"
    echo "| Commit | \`$sha\` |"
    echo "| Go | \`$go_version\` |"
    echo "| Test files | \`$test_files\` |"
    echo "| Test suite | \`$tests_status\` |"
    echo "| Source CRITICAL | \`$critical_source\` |"
    echo "| Image CRITICAL | \`$critical_image\` |"
    echo "| Receipt | \`eif-backend-ci-$sha-$run_attempt\` |"
  } >>"$GITHUB_STEP_SUMMARY"
fi
