#!/usr/bin/env bash
# ==============================================================================
# Application Data Plane Access Verification Script (verify-app-access.sh)
# Pure verification suite running strictly as app-user (payment-system tenant).
# Validates data-plane messaging (catalogue.md) and asserts denied guardrails.
#
# Prerequisite: Run ./scripts/enable-app-access.sh first to provision streams/consumers.
# ==============================================================================

set -euo pipefail

SERVER="${NATS_SERVER:-nats://localhost:4222}"
APP_USER="app-user"
APP_PASSWORD="app-password"
CTX_APP="app-verify-ctx"

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
    log_info "Testing Allowed Data-Plane Operation: $desc"
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

PREV_CTX=$(nats context current 2>/dev/null || echo "")

cleanup() {
    if [ -n "$PREV_CTX" ]; then
        nats context select "$PREV_CTX" >/dev/null 2>&1 || true
    fi
    nats context rm "$CTX_APP" -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

# ------------------------------------------------------------------------------
# CONFIGURE APPLICATION CONTEXT (Data Plane)
# ------------------------------------------------------------------------------
nats context add "$CTX_APP" --server "$SERVER" --user "$APP_USER" --password "$APP_PASSWORD" --select >/dev/null

printf "\n==============================================================================\n"
printf "  Application (app-user) Access Verification Catalog (Server: %s)\n" "$SERVER"
printf "  Account: payment-system | Role: %s (Data Plane)\n" "$APP_USER"
printf "==============================================================================\n"

# ------------------------------------------------------------------------------
# PREREQUISITE CHECK
# ------------------------------------------------------------------------------
if ! nats stream info DELIVERY >/dev/null 2>&1; then
    printf "\n\033[1;31m[ERROR]\033[0m Prerequisite stream 'DELIVERY' not found on server %s.\n" "$SERVER"
    printf "Please run the platform provisioning script first:\n"
    printf "    ./scripts/enable-app-access.sh\n\n"
    exit 1
fi

# ------------------------------------------------------------------------------
# SECTION 1: ALLOWED DATA PLANE OPERATIONS (catalogue.md)
# ------------------------------------------------------------------------------

run_allowed '1. Account JetStream Information ($JS.API.INFO)' \
    nats account info

run_allowed '2. Transient Core NATS Publishing (orders.created)' \
    nats pub "orders.created" "{\"order_id\":\"ORD-101\",\"amount\":49.99}"

run_allowed '3. Transient Core NATS Publishing (payments.requested)' \
    nats pub "payments.requested" "{\"payment_id\":\"PAY-202\",\"status\":\"pending\"}"

run_allowed '4. JetStream Persistent Publishing (delivery.dispatched)' \
    nats pub "delivery.dispatched" "{\"delivery_id\":\"DEL-303\",\"address\":\"100 NATS Way\"}"

run_allowed '5. Header Enrichment & Deduplication (Nats-Msg-Id)' \
    nats pub "delivery.dispatched" "{\"delivery_id\":\"DEL-304\",\"status\":\"in-transit\"}" -H "Nats-Msg-Id:msg-app-dedup-001"

run_allowed '6. Stream Sequence Discovery ($JS.API.STREAM.INFO.DELIVERY)' \
    nats stream info DELIVERY

run_allowed '7. Consumer Metadata Inspection ($JS.API.CONSUMER.INFO.DELIVERY.order-processor-at-least-once)' \
    nats consumer info DELIVERY order-processor-at-least-once

run_allowed '8. Direct Stream Point-Message Retrieval ($JS.API.DIRECT.GET.DELIVERY)' \
    nats stream get DELIVERY 1

# Publish a delivery work item and consume it via pull consumer with explicit ACK
nats pub "delivery.at-least-once" "{\"work_item\":\"PKG-555\"}" >/dev/null 2>&1

run_allowed '9. Pull Consumer Fetch & Explicit ACK ($JS.API.CONSUMER.MSG.NEXT)' \
    nats consumer next DELIVERY order-processor-at-least-once --ack

# ------------------------------------------------------------------------------
# SECTION 2: DENIED OPERATIONS (Guardrails: What won't work for App Microservices)
# ------------------------------------------------------------------------------

run_denied '10. Stream Creation Mutation ($JS.API.STREAM.CREATE.TEST_STREAM)' \
    nats stream add TEST_STREAM --subjects "test.>" --defaults

run_denied '11. Stream Deletion Mutation ($JS.API.STREAM.DELETE.DELIVERY)' \
    nats stream rm DELIVERY -f

run_denied '12. Stream Purge Mutation ($JS.API.STREAM.PURGE.DELIVERY)' \
    nats stream purge DELIVERY -f

run_denied '13. Audit Log Point Message Deletion ($JS.API.STREAM.MSG.DELETE.DELIVERY)' \
    nats stream rmm DELIVERY 1 -f

run_denied '14. Consumer Creation Mutation ($JS.API.CONSUMER.CREATE.DELIVERY.rogue-consumer)' \
    nats consumer add DELIVERY rogue-consumer --pull --defaults

run_denied '15. Consumer Deletion Mutation ($JS.API.CONSUMER.DELETE.DELIVERY.order-processor-at-least-once)' \
    nats consumer rm DELIVERY order-processor-at-least-once -f

run_denied '16. Consumer Pause Mutation ($JS.API.CONSUMER.PAUSE.DELIVERY.order-processor-at-least-once)' \
    nats consumer pause DELIVERY order-processor-at-least-once 10m -f

run_denied '17. Indiscriminate Account Purge ($JS.API.ACCOUNT.PURGE)' \
    nats req '$JS.API.ACCOUNT.PURGE' '' --timeout 1s

run_denied '18. Cross-Account Server Telemetry Access ($SYS.REQ.SERVER.PING)' \
    nats server ping

run_denied '19. Zero-Trust Business Secret Publishing (*.secret.>)' \
    nats req "orders.secret.keys" "unauthorized payload" --timeout 1s

# ------------------------------------------------------------------------------
# FINAL REPORT
# ------------------------------------------------------------------------------
printf "\n==============================================================================\n"
printf "  Application Verification Summary: %d Passed, %d Failed\n" "$PASSED" "$FAILED"
printf "==============================================================================\n\n"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
