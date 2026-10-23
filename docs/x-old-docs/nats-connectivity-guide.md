# NATS Connectivity & Troubleshooting Guide

## Purpose

This guide provides with a flow-based procedure to validate NATS connectivity, cluster connectivity, server health, and functional communication, and to isolate connectivity failures.

### Prerequisite

NATS server or cluster deployment must already be completed or started. For deployment-specific procedures, refer to the **NATS Deployment Architecture** and platform deployment documentation.

---

## Operational Flow

```text
Server Connectivity
        |
        V
Cluster Connectivity
        |
        V
Health Check
        |
        V
Functional Connectivity Check
        |
     +--+--+
    Pass  Fail
     |      |
     V      V
Connectivity  Troubleshoot
   Ready          |
                  V
            Identify Failure Domain
```

---

## 1. Server Connectivity Check

Verify that the NATS client endpoint is reachable from the SRE environment before attempting a NATS protocol connection.

### 1.1 DNS Resolution

```bash
nslookup <nats-host>
```

**Expected:** NATS hostname resolves to the expected IP address.

### 1.2 TCP Port Reachability

NATS client connections normally use port `4222`. Use the platform-provided endpoint and port.

```bash
nc -zv -w 3 <nats-host> 4222
```

**Expected:** TCP connection succeeds.

### 1.3 Optional Monitoring Port Check

If the NATS HTTP monitoring endpoint is exposed to the SRE environment:

```bash
nc -zv -w 3 <nats-host> 8222
```

**Expected:** Monitoring endpoint is reachable.

### 1.4 TLS Connectivity Check

Use when TLS is enabled and a TLS-specific failure is suspected.

```bash
openssl s_client \
  -connect <nats-host>:4222 \
  -servername <nats-host> \
  </dev/null
```

Use the output to inspect certificate and TLS handshake failures.

> TLS configuration and certificate management are covered by the **NATS Security Guide**. This command is provided here only as a troubleshooting diagnostic.

---

## 2. Cluster Connectivity

Establish a NATS connection using the platform-provided cluster endpoint and verify that the expected NATS server/cluster can be reached.

### 2.1 Configure NATS CLI Context

```bash
nats context add <context-name> \
  --server nats://<nats-endpoint>:4222
```

For multiple endpoints:

```bash
nats context add <context-name> \
  --server nats://<node1>:4222,nats://<node2>:4222,nats://<node3>:4222
```

For authenticated environments, use the credentials mechanism defined by the platform.

### 2.2 Select Context

```bash
nats context select <context-name>
```

### 2.3 Verify Server Connectivity

```bash
nats server ping
```

**Expected:** Successful response from the NATS server.

### 2.4 Check Server Round-Trip Time

```bash
nats server rtt
```

Use this to identify unusually high NATS request latency.

### 2.5 Inspect Server Information

```bash
nats server info
```

Use this to verify the connected server and obtain server/cluster information.

---

## 3. Health Check

Verify that the NATS server and, where applicable, the cluster are healthy and operational.

### 3.1 Server Health

```bash
curl -s -i http://<nats-host>:8222/healthz
```

**Expected:** HTTP `200` response indicating the server is healthy.

### 3.2 Server Status

```bash
curl -s http://<nats-host>:8222/varz | jq .
```

Use `/varz` to inspect server status and runtime information such as:

* Server ID / name
* Version
* Uptime
* Resource utilization
* Connection information

### 3.3 Cluster Routes

```bash
curl -s http://<nats-host>:8222/routez | jq .
```

Use `/routez` to verify the expected server-to-server routes.

### 3.4 Active Connections

Use when investigating client connection issues:

```bash
curl -s "http://<nats-host>:8222/connz?limit=10" | jq .
```

### 3.5 JetStream Health

Use only when the application depends on JetStream:

```bash
curl -s http://<nats-host>:8222/jsz | jq .
```

> JetStream resource management is covered by the **NATS Resource Management Guide**.

---

## 4. Functional Connectivity Check

Verify that NATS is not only reachable but capable of processing NATS operations successfully.

### 4.1 NATS Ping

```bash
nats server ping
```

**Expected:** Successful response.

### 4.2 Request / Response

Start a temporary responder:

```bash
nats reply <test-subject> "NATS_OK"
```

From another terminal:

```bash
nats req <test-subject> "ping"
```

**Expected:** Response is received successfully.

### 4.3 Optional Core Pub/Sub Test

Use when publish/subscribe functionality needs to be explicitly validated.

Terminal 1:

```bash
nats sub <test-subject>
```

Terminal 2:

```bash
nats pub <test-subject> "NATS_CONNECTIVITY_TEST"
```

**Expected:** Subscriber receives the test message.

### 4.4 Optional JetStream Validation

Perform only when JetStream functionality is part of the connectivity validation requirement.

Use an existing platform-approved test stream rather than creating production resources from this guide.

---

## 5. Troubleshooting

When any connectivity or health check fails, isolate the failure progressively from the lowest dependency layer to the NATS service layer.

### 5.1 Failure Isolation Flow

```text
DNS
 |
 V
Network / TCP
 |
 V
NATS Endpoint
 |
 V
TLS / Authentication
 |
 V
NATS Server
 |
 V
Cluster
 |
 V
JetStream (if applicable)
```

### 5.2 Common Failure Scenarios

| Symptom                   | Failure Domain             | Diagnostic                           |
| ------------------------- | -------------------------- | ------------------------------------ |
| DNS resolution failure    | DNS / Network              | `dig`, `nslookup`                    |
| Connection timeout        | Network / Firewall         | `nc -zv`, routing/firewall checks    |
| Connection refused        | Endpoint / NATS Server     | `nc -zv`, server status, server logs |
| TLS handshake failure     | TLS / Security             | `openssl s_client`                   |
| Authentication failure    | Authentication             | NATS CLI context / server logs       |
| Permission violation      | Authorization              | NATS CLI operation / server logs     |
| Server ping fails         | NATS Server / Connectivity | `nats server ping`                   |
| Unexpected RTT            | Network / Server           | `nats server rtt`, `/varz`           |
| Cluster route unavailable | Cluster                    | `/routez`, server logs               |
| NATS unhealthy            | NATS Server                | `/healthz`, `/varz`                  |
| JetStream unavailable     | JetStream / Server         | `/jsz`, JetStream health checks      |

### 5.3 Step-by-Step Troubleshooting

#### 5.3.1 Step 1 - Network

```bash
dig +short <nats-host>
nc -zv -w 3 <nats-host> 4222
```

If TCP connectivity fails, investigate:

* DNS
* Routing
* Firewall / security rules
* Load balancer / ingress
* Kubernetes service/network policy
* Server listener configuration

#### 5.3.2 Step 2 - NATS Connection

```bash
nats server ping
```

If the TCP connection succeeds but the NATS operation fails, investigate:

* NATS endpoint configuration
* TLS configuration
* Authentication
* Authorization
* NATS server logs

#### 5.3.3 Step 3 - Server Health

```bash
curl -s -i http://<nats-host>:8222/healthz
curl -s http://<nats-host>:8222/varz | jq .
```

If the server is unhealthy, investigate server logs and runtime/resource conditions.

#### 5.3.4 Step 4 - Cluster

```bash
curl -s http://<nats-host>:8222/routez | jq .
```

Investigate:

* Missing/disconnected routes
* Node availability
* Cluster configuration
* Inter-node network connectivity

#### 5.3.5 Step 5 - JetStream

Only when JetStream is required:

```bash
curl -s http://<nats-host>:8222/jsz | jq .
```

Investigate JetStream state, storage, replicas, and server/resource conditions.

---

## 6. Evidence Collection & Escalation

When escalating a connectivity issue, collect:

* Environment
* NATS endpoint
* Timestamp
* Affected server/node, if known
* Failed command and output
* Client/server error message
* `/healthz` result
* Relevant `/varz` and `/routez` information
* Relevant server logs
* Recent infrastructure or configuration changes

---

## Official References

* [NATS Server Configuration](https://docs.nats.io/reference/config/)
* [NATS Server Monitoring](https://docs.nats.io/running-a-nats-service/nats_admin/monitoring)
* [NATS CLI](https://docs.nats.io/using-nats/nats-tools/nats_cli)
* [NATS Core Concepts](https://docs.nats.io/learn/core-nats/)
* [NATS Security / TLS](https://docs.nats.io/learn/security/encryption)
