#!/usr/bin/env bash
# ==============================================================================
# Platform/SRE Account Access Verification Script
# Verifies governed JetStream administrative operations and denied guardrails
# ==============================================================================

set -euo pipefail

SERVER="${NATS_SERVER:-nats://localhost:4222}"
USER="platform-user"
PASSWORD="platform-password"
CTX_NAME="platform-user-verify"

PASSED=0
FAILED=0

log_info() {
    printf "\n\033[1;34m[INFO]\033[0m %s\n" "$1"
}

log_pass() {
    printf "  \033[1;32m[PASS]\033[0m %s\n" "$1"
    PASSED=$((PASSED + 1))
}

log_fail() {
    printf "  \033[1;31m[FAIL]\033[0m %s\n" "$1"
    FAILED=$((FAILED + 1))
}

run_allowed() {
    local desc="$1"
    shift
    log_info "Testing Allowed: $desc"
    if "$@" >/dev/null 2>&1; then
        log_pass "$desc succeeded as expected"
    else
        log_fail "$desc failed unexpectedly"
    fi
}

run_denied() {
    local desc="$1"
    shift
    log_info "Testing Denied Guardrail: $desc"
    local output
    output=$("$@" 2>&1) || true
    if echo "$output" | grep -qi "permission\|authorization\|violation\|denied"; then
        log_pass "$desc was properly BLOCKED with authorization error"
    else
        log_fail "$desc succeeded but should have been DENIED"
    fi
}

# Preserve existing context and ensure cleanup on exit
PREV_CTX=$(nats context current 2>/dev/null || echo "")

cleanup() {
    if [ -n "$PREV_CTX" ]; then
        nats context select "$PREV_CTX" >/dev/null 2>&1 || true
    fi
    nats context rm "$CTX_NAME" -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

# Switch to dedicated platform-user context
nats context add "$CTX_NAME" --server "$SERVER" --user "$USER" --password "$PASSWORD" --select >/dev/null

printf "\n==============================================================================\n"
printf "  Platform/SRE Access Verification Catalog (Server: %s)\n" "$SERVER"
printf "  Account: payment-system | User: %s\n" "$USER"
printf "==============================================================================\n"

# ------------------------------------------------------------------------------
# SECTION 1: ALLOWED OPERATIONS (JetStream Lifecycle)
# ------------------------------------------------------------------------------

run_allowed '1. Account JetStream Info ($JS.API.INFO)' \
    nats account info

# Clean up any leftover DELIVERY stream from previous runs
nats stream rm DELIVERY -f >/dev/null 2>&1 || true

run_allowed '2. Stream Provisioning ($JS.API.STREAM.CREATE.DELIVERY)' \
    nats stream add DELIVERY --subjects "delivery.>" --storage file --replicas 3 --defaults

run_allowed '3. Stream Inspection ($JS.API.STREAM.INFO.DELIVERY)' \
    nats stream info DELIVERY

run_allowed '4. Stream Configuration Update ($JS.API.STREAM.UPDATE.DELIVERY)' \
    nats stream edit DELIVERY --max-msgs 10000 -f

run_allowed '5. Consumer Provisioning ($JS.API.CONSUMER.CREATE.DELIVERY.at-least-once)' \
    nats consumer add DELIVERY at-least-once --filter "delivery.atleast-once" --pull --ack explicit --defaults

run_allowed '6. Consumer Inspection ($JS.API.CONSUMER.INFO.DELIVERY.at-least-once)' \
    nats consumer info DELIVERY at-least-once

run_allowed '7. Consumer Pause ($JS.API.CONSUMER.PAUSE.DELIVERY.at-least-once)' \
    nats consumer pause DELIVERY at-least-once 10m -f

run_allowed '8. Consumer Resume ($JS.API.CONSUMER.PAUSE.DELIVERY.at-least-once)' \
    nats consumer resume DELIVERY at-least-once -f

run_allowed '9. Stream Purge ($JS.API.STREAM.PURGE.DELIVERY)' \
    nats stream purge DELIVERY -f

run_allowed '10. Clustered Stream Leader Stepdown ($JS.API.STREAM.LEADER.STEPDOWN.DELIVERY)' \
    nats stream cluster step-down DELIVERY

# ------------------------------------------------------------------------------
# SECTION 2: DENIED OPERATIONS (Guardrails: What won't work for Platform Admin)
# ------------------------------------------------------------------------------

run_denied '11. Indiscriminate Account Purge ($JS.API.ACCOUNT.PURGE)' \
    nats req '$JS.API.ACCOUNT.PURGE' '' --timeout 1s

# Publish sample message so message sequence exists before testing deletion
nats pub "delivery.test" "sample payload" >/dev/null 2>&1 || true

run_denied '12. Audit Log Point Message Deletion ($JS.API.STREAM.MSG.DELETE.DELIVERY)' \
    nats stream rmm DELIVERY 1 -f

run_denied '13. Zero-Trust Business Secret Publishing (*.secret.>)' \
    nats req "dev.secret.keys" "unauthorized payload" --timeout 1s

run_denied '14. Cross-Account Server Telemetry Access ($SYS.REQ.SERVER.PING)' \
    nats server ping

# ------------------------------------------------------------------------------
# SECTION 3: DECOMMISSIONING & RESOURCE CLEANUP
# ------------------------------------------------------------------------------

run_allowed '15. Consumer Decommissioning ($JS.API.CONSUMER.DELETE.DELIVERY.at-least-once)' \
    nats consumer rm DELIVERY at-least-once -f

run_allowed '16. Stream Decommissioning ($JS.API.STREAM.DELETE.DELIVERY)' \
    nats stream rm DELIVERY -f

# ------------------------------------------------------------------------------
# FINAL REPORT
# ------------------------------------------------------------------------------
printf "\n==============================================================================\n"
printf "  Platform Verification Summary: %d Passed, %d Failed\n" "$PASSED" "$FAILED"
printf "==============================================================================\n\n"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
