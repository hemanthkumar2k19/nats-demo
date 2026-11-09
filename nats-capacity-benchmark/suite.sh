#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Usage:
#   ./suite.sh [SUITE_NAME_OR_FILE] [BASE_CONFIG_FILE]
#
# Arguments:
#   $1 - Suite file or ID (e.g. core-ladder, jetstream-guarantees, or path to .conf).
#        Defaults to suites/jetstream-guarantees.conf.
#   $2 - Base environment configuration file (defaults to benchmark.conf).
#        Sourced here to establish suite-level defaults (e.g. RESULTS_DIR, NATS_URL).

SUITE_INPUT="${1:-${ROOT_DIR}/suites/jetstream-guarantees.conf}"
ENVIRONMENT_CONFIG="${2:-${ROOT_DIR}/benchmark.conf}"

# Resolve suite config file
if [[ -f "$SUITE_INPUT" ]]; then
  SUITE_FILE="$SUITE_INPUT"
elif [[ -f "${ROOT_DIR}/suites/${SUITE_INPUT}.conf" ]]; then
  SUITE_FILE="${ROOT_DIR}/suites/${SUITE_INPUT}.conf"
elif [[ -f "${ROOT_DIR}/suites/${SUITE_INPUT}" ]]; then
  SUITE_FILE="${ROOT_DIR}/suites/${SUITE_INPUT}"
else
  echo "ERROR: Suite configuration not found: $SUITE_INPUT" >&2
  echo "Available suites in suites/:" >&2
  ls -1 "${ROOT_DIR}/suites/"*.conf 2>/dev/null | xargs -n 1 basename | sed 's/\.conf$//' >&2
  exit 1
fi

# Preserve environment overrides prior to sourcing base configuration
_ENV_NATS_URL="${NATS_URL:-}"
_ENV_RESULTS_DIR="${RESULTS_DIR:-}"
_ENV_SETTLE_SECONDS="${SETTLE_SECONDS:-}"

if [[ -f "$ENVIRONMENT_CONFIG" ]]; then
  source "$ENVIRONMENT_CONFIG"
fi

# Reapply caller overrides so benchmark.conf does not clobber them
[[ -n "$_ENV_NATS_URL" ]] && NATS_URL="$_ENV_NATS_URL"
[[ -n "$_ENV_RESULTS_DIR" ]] && RESULTS_DIR="$_ENV_RESULTS_DIR"
[[ -n "$_ENV_SETTLE_SECONDS" ]] && SETTLE_SECONDS="$_ENV_SETTLE_SECONDS"

source "$SUITE_FILE"

: "${SUITE_ID:=custom-suite}"
: "${SUITE_TITLE:=NATS Benchmark Suite}"
: "${SUITE_DESCRIPTION:=}"
: "${SETTLE_SECONDS:=10}"

SUITE_TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
if [[ "$RESULTS_DIR" == /* ]]; then
  SUITE_OUT_DIR="${RESULTS_DIR}/suites/${SUITE_ID}/${SUITE_TIMESTAMP}"
else
  SUITE_OUT_DIR="${ROOT_DIR}/${RESULTS_DIR:-results}/suites/${SUITE_ID}/${SUITE_TIMESTAMP}"
fi
mkdir -p "$SUITE_OUT_DIR"

TOTAL_STEPS="${#TEST_CASES[@]}"
(( TOTAL_STEPS > 0 )) || {
  echo "ERROR: No test cases defined in suite file $SUITE_FILE" >&2
  exit 1
}

echo "================================================================================"
echo "NATS Benchmark Executive Suite: ${SUITE_TITLE}"
echo "================================================================================"
echo "Suite ID      : ${SUITE_ID}"
echo "Description   : ${SUITE_DESCRIPTION}"
echo "Total Tests   : ${TOTAL_STEPS}"
echo "Output Folder : ${SUITE_OUT_DIR}"
echo "Settling Pause: ${SETTLE_SECONDS}s between runs"
echo "================================================================================"
echo ""

STEP_NUM=0
for ENTRY in "${TEST_CASES[@]}"; do
  (( STEP_NUM++ ))

  IFS='|' read -r CASE_ID CASE_DESC CASE_WORKLOAD CASE_OVERRIDES <<< "$ENTRY"

  CASE_ID="$(echo "$CASE_ID" | xargs)"
  CASE_DESC="$(echo "$CASE_DESC" | xargs)"
  CASE_WORKLOAD="$(echo "$CASE_WORKLOAD" | xargs)"
  CASE_OVERRIDES="$(echo "$CASE_OVERRIDES" | xargs)"

  # Resolve workload config path
  if [[ "$CASE_WORKLOAD" != /* ]]; then
    CASE_WORKLOAD="${ROOT_DIR}/${CASE_WORKLOAD}"
  fi
  if [[ ! -f "$CASE_WORKLOAD" ]]; then
    CASE_WORKLOAD="${ROOT_DIR}/benchmark.conf"
  fi

  echo "--------------------------------------------------------------------------------"
  echo "[${STEP_NUM}/${TOTAL_STEPS}] Starting Test Case: ${CASE_ID}"
  echo "Description : ${CASE_DESC}"
  echo "Workload    : $(basename "$CASE_WORKLOAD")"
  echo "Overrides   : ${CASE_OVERRIDES}"
  echo "--------------------------------------------------------------------------------"

  # Execute benchmark in an isolated subshell.
  # Running in a subshell ensures environment overrides exported specifically
  # for this test case (e.g. PUBS, THROUGHPUT, QUEUE) do not bleed into subsequent
  # test cases in the loop.
  set +e
  (
    # Export overrides
    export TEST_ID="${CASE_ID}"
    export TEST_DESCRIPTION="${CASE_DESC}"
    export RESULTS_DIR="${SUITE_OUT_DIR}"
    export TIMESTAMP_SUBFOLDER=false
    export NATS_URL="${NATS_URL}"

    # Parse and export key=value pairs
    for PAIR in $CASE_OVERRIDES; do
      if [[ "$PAIR" == *"="* ]]; then
        export "$PAIR"
      fi
    done

    bash "${ROOT_DIR}/benchmark.sh" \
      "$CASE_WORKLOAD"
  )
  STEP_EXIT=$?
  set -e

  if [[ "$STEP_EXIT" -ne 0 ]]; then
    echo "WARNING: Test ${CASE_ID} finished with non-zero exit code: ${STEP_EXIT}"
  else
    echo "SUCCESS: Test ${CASE_ID} finished cleanly."
  fi

  # Quiescence Cooldown between tests (skip after the last test)
  if [[ "$STEP_NUM" -lt "$TOTAL_STEPS" ]]; then
    echo ""
    echo "Waiting ${SETTLE_SECONDS}s for cluster settling (quiescence cooldown)..."
    sleep "$SETTLE_SECONDS"
    echo ""
  fi
done

echo ""
echo "================================================================================"
echo "Suite Workload Execution Complete. Generating Executive Comparison Report..."
echo "================================================================================"

python3 "${ROOT_DIR}/scripts/aggregate_suite.py" \
  "$SUITE_OUT_DIR" \
  "$SUITE_ID" \
  "$SUITE_TITLE" \
  "$SUITE_DESCRIPTION"

# Post-suite stream cleanup to leave the cluster in a pristine state
bash "${ROOT_DIR}/scripts/js_stream.sh" cleanup "$NATS_URL" "BENCH_STREAM" >/dev/null 2>&1 || true

echo ""
echo "================================================================================"
echo "Executive Suite Summary Complete"
echo "================================================================================"
echo "Suite Results    : ${SUITE_OUT_DIR}"
echo "Executive Report : ${SUITE_OUT_DIR}/executive-report.html"
echo "Suite Report     : ${SUITE_OUT_DIR}/suite-report.html"
echo "================================================================================"

