#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 10 ]]; then
  echo "Usage: results.sh RESULT_DIR TEST_ID SUBJECT TARGET_RATE MESSAGE_SIZE PUBS SUBS DURATION TOTAL_MESSAGES METRICS_BACKEND" >&2
  exit 2
fi

RESULT_DIR="$1"
TEST_ID="$2"
SUBJECT="$3"
TARGET_RATE="$4"
MESSAGE_SIZE="$5"
PUBS="$6"
SUBS="$7"
DURATION="$8"
TOTAL_MESSAGES="$9"
METRICS_BACKEND="${10:-unknown}"

SUMMARY="${RESULT_DIR}/summary.csv"

# Configuration and artifact references. Raw workload output remains in workload.log;
# raw per-container samples remain in metrics.csv for later analysis.
echo 'test_id,subject,target_publish_msgs_per_sec,message_size_bytes,publisher_concurrency,consumer_concurrency,target_duration_seconds,total_messages,metrics_backend,workload_log,metrics_file' > "$SUMMARY"
printf '%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n' \
  "$TEST_ID" "$SUBJECT" "$TARGET_RATE" "$MESSAGE_SIZE" "$PUBS" "$SUBS" \
  "$DURATION" "$TOTAL_MESSAGES" "$METRICS_BACKEND" 'workload.log' 'metrics.csv' >> "$SUMMARY"
