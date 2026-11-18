#!/usr/bin/env bash
# ==============================================================================
# Platform Provisioning Script (enable-app-access.sh)
# Uses platform-user credentials to provision streams and consumers
# required by application microservices and workers.
# ==============================================================================

set -euo pipefail

SERVER="${NATS_SERVER:-nats://localhost:4222}"
PLATFORM_USER="platform-user"
PLATFORM_PASSWORD="platform-password"
CTX_NAME="platform-provision-ctx"

ACTION="${1:-provision}"

log_info() {
    printf "\n\033[1;34m[INFO]\033[0m %s\n" "$1"
}

log_pass() {
    printf "  \033[1;32m[DONE]\033[0m %s\n" "$1"
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

# Switch to platform-user context
nats context add "$CTX_NAME" --server "$SERVER" --user "$PLATFORM_USER" --password "$PLATFORM_PASSWORD" --select >/dev/null

teardown_resources() {
    log_info "Teardown: Removing existing streams and consumers..."
    nats stream rm DELIVERY -f >/dev/null 2>&1 || true
    nats stream rm PAYMENTS -f >/dev/null 2>&1 || true
    nats stream rm dlq_stream -f >/dev/null 2>&1 || true
    log_pass "Teardown completed"
}

if [ "$ACTION" = "teardown" ] || [ "$ACTION" = "--teardown" ] || [ "$ACTION" = "clean" ]; then
    teardown_resources
    exit 0
fi

printf "\n==============================================================================\n"
printf "  Platform Resource Provisioning (Server: %s)\n" "$SERVER"
printf "  Account: payment-system | Role: %s (Control Plane)\n" "$PLATFORM_USER"
printf "==============================================================================\n"

# ------------------------------------------------------------------------------
# 1. STREAM: DELIVERY (Processing Service & Delivery Workflows)
# ------------------------------------------------------------------------------
log_info "1. Provisioning DELIVERY Stream & Consumers..."
nats stream rm DELIVERY -f >/dev/null 2>&1 || true
nats stream add DELIVERY \
    --subjects "delivery.>" \
    --storage file \
    --retention limits \
    --discard old \
    --dupe-window 2m \
    --replicas 3 \
    --defaults >/dev/null
log_pass "Stream DELIVERY created (Subjects: delivery.>, Replicas: 3, DupeWindow: 2m)"

nats consumer add DELIVERY order-processor-at-least-once \
    --filter "delivery.at-least-once" \
    --pull \
    --ack explicit \
    --max-deliver 3 \
    --wait 10s \
    --defaults >/dev/null
log_pass "Consumer order-processor-at-least-once created on DELIVERY"

nats consumer add DELIVERY at-least-once \
    --filter "delivery.atleast-once" \
    --pull \
    --ack explicit \
    --max-deliver 3 \
    --wait 10s \
    --defaults >/dev/null
log_pass "Consumer at-least-once created on DELIVERY (for processing-service)"

nats consumer add DELIVERY exactly-once \
    --filter "delivery.exactly-once" \
    --pull \
    --ack explicit \
    --max-deliver 3 \
    --wait 10s \
    --defaults >/dev/null
log_pass "Consumer exactly-once created on DELIVERY (for processing-service)"

# ------------------------------------------------------------------------------
# 2. STREAM: PAYMENTS (Payment Worker & Notification Worker)
# ------------------------------------------------------------------------------
log_info "2. Provisioning PAYMENTS Stream & Consumers..."
nats stream rm PAYMENTS -f >/dev/null 2>&1 || true
nats stream add PAYMENTS \
    --subjects "payments.>" \
    --storage file \
    --retention limits \
    --discard old \
    --replicas 3 \
    --defaults >/dev/null
log_pass "Stream PAYMENTS created (Subjects: payments.>, Replicas: 3)"

nats consumer add PAYMENTS payments-pull \
    --filter "payments.>" \
    --pull \
    --ack explicit \
    --defaults >/dev/null
log_pass "Pull Consumer payments-pull created on PAYMENTS"

nats consumer add PAYMENTS payments-push \
    --filter "payments.>" \
    --target "notifications.payments" \
    --deliver-group "notification-workers" \
    --deliver all \
    --ack explicit \
    --defaults >/dev/null
log_pass "Push Consumer payments-push created on PAYMENTS (Deliver: notifications.payments)"

# ------------------------------------------------------------------------------
# 3. STREAM: dlq_stream (Dead Letter Queue & Advisory Consumer)
# ------------------------------------------------------------------------------
log_info "3. Provisioning Dead Letter Queue Stream & Consumer..."
nats stream rm dlq_stream -f >/dev/null 2>&1 || true
nats stream add dlq_stream \
    --subjects '$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>' \
    --storage file \
    --retention limits \
    --discard old \
    --replicas 3 \
    --defaults >/dev/null
log_pass "Stream dlq_stream created (Subjects: \$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>, Replicas: 3)"

nats consumer add dlq_stream dlq-consumer \
    --pull \
    --ack explicit \
    --deliver all \
    --defaults >/dev/null
log_pass "Consumer dlq-consumer created on dlq_stream (for dlq-worker)"

nats consumer add dlq_stream dlq_consumer \
    --pull \
    --ack explicit \
    --deliver all \
    --defaults >/dev/null
log_pass "Consumer dlq_consumer created on dlq_stream"

printf "\n==============================================================================\n"
printf "  All prerequisite streams and consumers provisioned successfully.\n"
printf "  Application microservices and verify-app-access.sh are ready to run.\n"
printf "==============================================================================\n\n"
