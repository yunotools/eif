#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd gh
require_cmd jq

: "${GH_TOKEN:?GH_TOKEN is required}"
: "${EIF_GITHUB_OWNER:?EIF_GITHUB_OWNER is required}"
: "${EIF_CI_REPO:?EIF_CI_REPO is required}"
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GITHUB_SHA:?GITHUB_SHA is required}"
: "${GITHUB_RUN_ID:?GITHUB_RUN_ID is required}"

[[ "$GITHUB_SHA" =~ ^[0-9a-f]{40}$ ]] || die "GITHUB_SHA is not a full 40-character SHA"

payload=$(jq -n \
  --arg component backend \
  --arg repository "$GITHUB_REPOSITORY" \
  --arg sha "$GITHUB_SHA" \
  --arg ref "${GITHUB_REF:-}" \
  --arg run_id "$GITHUB_RUN_ID" \
  --arg run_attempt "${GITHUB_RUN_ATTEMPT:-1}" \
  --arg receipt_artifact "eif-backend-ci-$GITHUB_SHA-${GITHUB_RUN_ATTEMPT:-1}" \
  '{
    event_type: "component-ci-passed",
    client_payload: {
      schema_version: "1",
      component: $component,
      repository: $repository,
      sha: $sha,
      ref: $ref,
      run_id: $run_id,
      run_attempt: $run_attempt,
      receipt_artifact: $receipt_artifact
    }
  }')

log "Dispatching backend candidate $GITHUB_SHA to $EIF_GITHUB_OWNER/$EIF_CI_REPO"
printf '%s' "$payload" | gh api \
  --method POST \
  -H 'Accept: application/vnd.github+json' \
  "repos/$EIF_GITHUB_OWNER/$EIF_CI_REPO/dispatches" \
  --input -
