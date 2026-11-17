#!/usr/bin/env bash
# ==============================================================================
# System Account Access Verification Script
# Verifies standard untweaked SYS account capabilities across cluster nodes
# ==============================================================================

set -euo pipefail

SERVER="${NATS_SERVER:-nats://localhost:4222}"
USER="sys-admin"
PASSWORD="sys-password"
CTX_NAME="sys-admin-verify"

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

run_check() {
    local desc="$1"
    shift
    log_info "Testing System Operation: $desc"
    if "$@" >/dev/null 2>&1; then
        log_pass "$desc succeeded as expected"
    else
        log_fail "$desc failed unexpectedly"
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

# Switch to dedicated sys-admin context
nats context add "$CTX_NAME" --server "$SERVER" --user "$USER" --password "$PASSWORD" --select >/dev/null

printf "\n==============================================================================\n"
printf "  System Account (SYS) Verification Catalog (Server: %s)\n" "$SERVER"
printf "  User: %s (Standard, untweaked system account privileges)\n" "$USER"
printf "==============================================================================\n"

run_check '1. Cluster Server Ping (Telemetry $SYS.REQ.SERVER.PING)' \
    nats server ping

run_check '2. Server JetStream Report ($SYS.REQ.SERVER.STATZ)' \
    nats server report jetstream

run_check '3. Server Connections Report ($SYS.REQ.SERVER.CONNZ)' \
    nats server report connections

run_check '4. Server Accounts Report ($SYS.REQ.ACCOUNT.STATZ)' \
    nats server report accounts

run_check '5. Cluster Node Inventory ($SYS.REQ.SERVER.PING / STATZ)' \
    nats server list

printf "\n==============================================================================\n"
printf "  System Verification Summary: %d Passed, %d Failed\n" "$PASSED" "$FAILED"
printf "==============================================================================\n\n"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
