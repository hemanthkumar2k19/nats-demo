
#!/usr/bin/env bash

preflight_checks() {
  local failures=0
  local node_count=0
  local expected_routes=$((EXPECTED_NATS_NODES - 1))

  echo
  echo "===================================="
  echo " NATS Infrastructure Preflight Check"
  echo "===================================="

  # 1. Cluster health
  echo
  echo "[1/3] Checking NATS cluster health..."

  for monitor_url in $NATS_MONITOR_URLS; do
    node_count=$((node_count + 1))

    if ! curl -fsS "${monitor_url}/healthz" >/dev/null; then
      echo "FAIL: NATS node is unhealthy: ${monitor_url}"
      failures=$((failures + 1))
      continue
    fi

    route_count=$(curl -fsS "${monitor_url}/routez" 2>/dev/null |
      jq -r '.num_routes // (.routes // [] | length)' 2>/dev/null || echo 0)

    if [[ "$route_count" -lt "$expected_routes" ]]; then
      echo "FAIL: ${monitor_url} sees ${route_count} routes; expected ${expected_routes}"
      failures=$((failures + 1))
    else
      echo "OK: ${monitor_url} healthy; ${route_count} cluster routes"
    fi
  done

  if [[ "$node_count" -ne "$EXPECTED_NATS_NODES" ]]; then
    echo "FAIL: Configured ${node_count} monitoring endpoints; expected ${EXPECTED_NATS_NODES}"
    failures=$((failures + 1))
  fi

  # 2. JetStream replication
  echo
  echo "[2/3] Checking JetStream replication..."

  if [[ -z "${REPLICATION_STREAMS:-}" ]]; then
    echo "SKIP: REPLICATION_STREAMS is not configured"
  else
    for stream in $REPLICATION_STREAMS; do
      stream_info=$(nats --server "$NATS_URL" stream info "$stream" --json 2>/dev/null) || {
        echo "FAIL: Unable to retrieve stream information: ${stream}"
        failures=$((failures + 1))
        continue
      }

      configured=$(echo "$stream_info" | jq -r '.config.num_replicas // 0')
      leader=$(echo "$stream_info" | jq -r '.cluster.leader // empty')
      healthy_replicas=$(echo "$stream_info" |
        jq '[.cluster.replicas[]? | select(.current == true)] | length')

      if [[ "$configured" -ne "$EXPECTED_REPLICAS" ]]; then
        echo "FAIL: ${stream} configured with ${configured} replicas; expected ${EXPECTED_REPLICAS}"
        failures=$((failures + 1))
      elif [[ -z "$leader" ]]; then
        echo "FAIL: ${stream} has no reported JetStream leader"
        failures=$((failures + 1))
      elif [[ "$healthy_replicas" -lt $((EXPECTED_REPLICAS - 1)) ]]; then
        echo "FAIL: ${stream} has ${healthy_replicas} healthy replica peers; expected $((EXPECTED_REPLICAS - 1))"
        failures=$((failures + 1))
      else
        echo "OK: ${stream} has ${configured} replicas; leader=${leader}"
      fi
    done
  fi

  # 3. Client connectivity
  echo
  echo "[3/3] Checking client connectivity..."

  if nats server check connection -s "$NATS_URL"; then
    echo "OK: Client connectivity verified: ${NATS_URL}"
  else
    echo "FAIL: Cannot connect to ${NATS_URL}"
    failures=$((failures + 1))
  fi

  # Final result
  echo
  echo "===================================="

  if [[ "$failures" -eq 0 ]]; then
    echo "PREFLIGHT PASSED"
    echo "===================================="
    return 0
  else
    echo "PREFLIGHT FAILED: ${failures} check(s) failed"
    echo "===================================="
    return 1
  fi
}
