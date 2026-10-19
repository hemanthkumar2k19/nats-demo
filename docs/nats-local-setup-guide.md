# NATS Local Setup Guide

Developer guide for running and testing NATS locally with JetStream persistence enabled.

This guide is structured to prioritize day-to-day development:
1. **Prerequisites & NATS Tools** -- Foundational runtimes and NATS components.
2. **Repository File Structure** -- Common configuration and deployment layout.
3. **Single-Node Setup (Primary Dev Workflow)** -- Recommended for 80% of application development and integration testing.
4. **Multi-Node Cluster Setup (3-Node Replication & Failover)** -- Advanced setup for evaluating JetStream R3 replication, leader election, and high availability.
5. **Testing & Interaction** -- Quick verification using application SDKs or the NATS CLI.
6. **Quick Reference Summary** -- Lookup table for ports, commands, and storage paths.

---

## 1. Prerequisites & NATS Tools

To keep setup explicit and easy to navigate, requirements are separated into **System Prerequisites & Runtimes** (foundational tools on your machine) and **NATS Components & Utilities** (broker binaries, container images, and CLI tools).

### 1.1 System Prerequisites & Runtimes

| Prerequisite | Status | Purpose | Installation / Check |
|--------------|--------|---------|----------------------|
| **Podman** | Required (Container Workflow) | OCI container runtime for running isolated NATS instances | [podman.io](https://podman.io/getting-started/installation) (`podman --version`) |
| **Homebrew (`brew`)** | Required (Native Binary Workflow) | Package manager to install native NATS binaries on macOS / Linux | [brew.sh](https://brew.sh/) (`brew --version`) |
| **Application Runtimes** (Go / Java / Python) | Optional (Application Testing) | Language runtimes needed if building or running application code | Go 1.22+, Java 17+, or Python 3.8+ |

### 1.2 NATS Components & Utilities

| Component / Tool | Status | Purpose | Installation / Source |
|------------------|--------|---------|-----------------------|
| **NATS Container Image (`nats:latest`)** | Required (Container Workflow) | Official NATS server container image | `podman pull nats:latest` |
| **podman-compose** | Optional (Container Workflow) | Multi-container compose runner (alternative to raw `podman run`) | `pip install podman-compose` or `brew install podman-compose` |
| **NATS Server (`nats-server`)** | Required (Native Binary Workflow) | Native local NATS server binary | `brew install nats-server` |
| **NATS CLI (`nats`)** | Optional (CLI Utilities) | Command-line tool for context management, monitoring, and pub/sub verification | `brew tap nats-io/nats-tools && brew install nats-io/nats-tools/nats` |
| **Application Client SDKs** | Optional (Application Testing) | Official language libraries for connecting from application code | Go (`nats.go`), Java (`jnats`), Python (`nats-py`) |

> [!NOTE]
> **Which tools do you actually need?**
> - **If using the Container Workflow (Podman):**
>   - System Prerequisite: **Podman**
>   - NATS Component: **`nats:latest` container image** (and optionally `podman-compose`)
>   - *Homebrew, native `nats-server`, and the `nats` CLI are not required.*
> - **If using the Native Binary Workflow (Local Process):**
>   - System Prerequisite: **Homebrew**
>   - NATS Component: **`nats-server`** (and optionally `nats` CLI)
>   - *Podman and container tools are not required.*
> - **If developing applications:**
>   - Only install the specific runtime (Go, Java, or Python) and SDK required for your service.

---

## 2. Repository File Structure

All local deployment assets reside in the `deploy/` directory. Both containerized and native binary workflows share this common configuration and storage structure:

```text
deploy/
|-- local-nats/                  # Single-Node Deployment (Primary Dev Setup)
|   |-- docker-compose.yaml      # Compose specification for single-node container
|   |-- nats.conf                # Server configuration (JetStream, ports 4222, 8222)
|   `-- data/                    # Persisted JetStream storage directory (git-ignored)
`-- local-nats-cluster/          # 3-Node Cluster Deployment (Replication & HA)
    |-- docker-compose.yaml      # Compose specification for 3-node cluster
    |-- nats-1.conf              # Node 1 config (client: 4222, http: 8222, route: 6222)
    |-- nats-2.conf              # Node 2 config (client: 4223, http: 8223, route: 6222)
    |-- nats-3.conf              # Node 3 config (client: 4224, http: 8224, route: 6222)
    `-- data/                    # Persisted storage per node (git-ignored)
```

### Configuration and Storage Roles

- **Configuration Files (`nats.conf`, `nats-*.conf`):** Define listening ports, HTTP monitoring endpoints, server names, cluster mesh discovery, and JetStream storage directories (`store_dir: "/data/jetstream"`).
- **Compose Files (`docker-compose.yaml`):** Pre-configured multi-container descriptors mapping host ports and configuration files into container paths.
- **Storage Directories (`data/`):** Host-mounted directories where JetStream messages and consumer state are stored. Deleting these directories safely resets the broker state back to empty.

---

## 3. Single-Node Setup (Primary Dev Workflow)

A single NATS instance with JetStream enabled is the recommended environment for day-to-day application development, unit testing, and service integration.

Choose one of the two execution options below:

### Option A: Container Workflow (Podman)

Run a single NATS container without installing local Go or NATS binaries.

**1. Start via podman-compose:**
From the repository root:
```bash
cd deploy/local-nats
podman-compose up -d
```

**Stop the server:**
```bash
cd deploy/local-nats
podman-compose down
```

**Stop and wipe persisted JetStream storage:**
```bash
cd deploy/local-nats
podman-compose down
rm -rf ./data
```

**Alternative: Start directly with podman run:**
```bash
podman run -d \
  --name local-nats \
  --restart unless-stopped \
  -p 4222:4222 \
  -p 8222:8222 \
  -v ./deploy/local-nats/nats.conf:/etc/nats/nats.conf:ro \
  -v ./deploy/local-nats/data:/data \
  nats:latest -c /etc/nats/nats.conf
```

To stop and remove:
```bash
podman stop local-nats
podman rm local-nats
rm -rf ./deploy/local-nats/data
```

---

### Option B: Local Binary Workflow (nats-server)

Run NATS natively as a local process for rapid iteration without container virtualization.

**1. Ephemeral Dev Server (Fastest Start):**
Starts an immediate foreground server with JetStream enabled in temporary memory/disk:
```bash
nats-server -js
```
Stop with `Ctrl+C`.

**2. Config-backed Persistent Server:**
Runs with the repository configuration file and an explicit local storage directory:
```bash
# Foreground
nats-server -c deploy/local-nats/nats.conf -sd ./deploy/local-nats/data/jetstream

# Background
nats-server -c deploy/local-nats/nats.conf -sd ./deploy/local-nats/data/jetstream &
```

**Stop background server:**
```bash
pkill nats-server
```

**Wipe persisted storage:**
```bash
rm -rf ./deploy/local-nats/data
```

---

### Verification (Single-Node)

Once started, verify your single-node instance:

**1. Check via HTTP Monitoring (No CLI required):**
```bash
curl -s http://localhost:8222/varz | grep version
curl -s http://localhost:8222/jsz
```

**2. Check via NATS CLI (if installed):**
```bash
nats server ping
nats server report jetstream
```

**3. Application SDK Connection Target:**
Applications connect to `nats://localhost:4222`.

---

## 4. Multi-Node Cluster Setup (3-Node Replication & Failover)

This section is for developers testing JetStream stream replication (R3 streams), leader election, cluster routing, and client failover across nodes.

### Cluster Topology & Port Mapping

| Node | Client Port (Host) | HTTP Monitoring Port | Cluster Route Port (Internal) | Config File |
|------|--------------------|----------------------|--------------------------------|-------------|
| nats-1 | 4222 | 8222 | 6222 | `deploy/local-nats-cluster/nats-1.conf` |
| nats-2 | 4223 | 8223 | 6222 | `deploy/local-nats-cluster/nats-2.conf` |
| nats-3 | 4224 | 8224 | 6222 | `deploy/local-nats-cluster/nats-3.conf` |

Choose one of the two execution options below:

### Option A: Container Workflow (Podman)

**1. Start via podman-compose:**
From the repository root:
```bash
cd deploy/local-nats-cluster
podman-compose up -d
```
This starts `nats-1`, `nats-2`, and `nats-3`. Observability tools (Prometheus, Grafana, Fluent-bit) remain stopped under the `observability` profile unless explicitly invoked.

**Stop the cluster:**
```bash
cd deploy/local-nats-cluster
podman-compose down
```

**Stop and wipe persisted cluster storage:**
```bash
cd deploy/local-nats-cluster
podman-compose down
rm -rf ./data
```

**Alternative: Start directly with podman run:**
Create a dedicated bridge network for node resolution:
```bash
podman network create nats-cluster

podman run -d --name nats-1 --network nats-cluster -p 4222:4222 \
  -v ./deploy/local-nats-cluster/nats-1.conf:/etc/nats/nats.conf:ro \
  -v ./deploy/local-nats-cluster/data/nats-1:/data \
  nats:latest -c /etc/nats/nats.conf

podman run -d --name nats-2 --network nats-cluster -p 4223:4222 \
  -v ./deploy/local-nats-cluster/nats-2.conf:/etc/nats/nats.conf:ro \
  -v ./deploy/local-nats-cluster/data/nats-2:/data \
  nats:latest -c /etc/nats/nats.conf

podman run -d --name nats-3 --network nats-cluster -p 4224:4222 \
  -v ./deploy/local-nats-cluster/nats-3.conf:/etc/nats/nats.conf:ro \
  -v ./deploy/local-nats-cluster/data/nats-3:/data \
  nats:latest -c /etc/nats/nats.conf
```

To stop and remove:
```bash
podman stop nats-1 nats-2 nats-3
podman rm nats-1 nats-2 nats-3
podman network rm nats-cluster
rm -rf ./deploy/local-nats-cluster/data
```

---

### Option B: Local Binary Workflow (CLI)

Run 3 local `nats-server` processes across separate terminal windows with route mesh peering:

**Terminal 1 (Node 1):**
```bash
nats-server -n nats-1 -p 4222 -m 8222 \
  -cluster nats://127.0.0.1:6222 \
  -routes nats://127.0.0.1:6223,nats://127.0.0.1:6224 \
  --cluster_name local-nats-cluster \
  -js -sd ./deploy/local-nats-cluster/data/nats-1
```

**Terminal 2 (Node 2):**
```bash
nats-server -n nats-2 -p 4223 -m 8223 \
  -cluster nats://127.0.0.1:6223 \
  -routes nats://127.0.0.1:6222,nats://127.0.0.1:6224 \
  --cluster_name local-nats-cluster \
  -js -sd ./deploy/local-nats-cluster/data/nats-2
```

**Terminal 3 (Node 3):**
```bash
nats-server -n nats-3 -p 4224 -m 8224 \
  -cluster nats://127.0.0.1:6224 \
  -routes nats://127.0.0.1:6222,nats://127.0.0.1:6223 \
  --cluster_name local-nats-cluster \
  -js -sd ./deploy/local-nats-cluster/data/nats-3
```

**Stop all nodes:**
Press `Ctrl+C` in each terminal, or run:
```bash
pkill nats-server
```

**Wipe cluster storage:**
```bash
rm -rf ./deploy/local-nats-cluster/data
```

---

### Verification (3-Node Cluster)

**1. Check Cluster Mesh via HTTP Monitoring:**
```bash
curl -s http://localhost:8222/routez
```

**2. Check Cluster via NATS CLI:**
```bash
nats server list -s localhost:4222
nats server ping -s localhost:4222
```

**3. Application SDK Connection Target:**
Provide all 3 seed nodes so client SDKs can automatically failover:
- Connection URL: `nats://localhost:4222,nats://localhost:4223,nats://localhost:4224`

---

## 5. Testing & Interaction

### Approach 1: Application SDK Clients (Recommended, No CLI Required)

Connect directly to the local broker using your standard application client SDK:

**Go (`nats.go`):**
```go
// Single-Node
nc, err := nats.Connect("nats://localhost:4222")

// 3-Node Cluster (Automatic Failover)
nc, err := nats.Connect("nats://localhost:4222,nats://localhost:4223,nats://localhost:4224")
```

**Java (`jnats`):**
```java
// Single-Node
Connection nc = Nats.connect("nats://localhost:4222");

// 3-Node Cluster (Automatic Failover)
Options options = new Options.Builder()
    .servers(new String[]{"nats://localhost:4222", "nats://localhost:4223", "nats://localhost:4224"})
    .build();
Connection nc = Nats.connect(options);
```

**Python (`nats-py`):**
```python
# Single-Node
nc = await nats.connect("nats://localhost:4222")

# 3-Node Cluster (Automatic Failover)
nc = await nats.connect(servers=["nats://localhost:4222", "nats://localhost:4223", "nats://localhost:4224"])
```

---

### Approach 2: NATS CLI Utility

If you have installed the `nats` CLI, save connection contexts for quick switching:

**Save Contexts:**
```bash
# Single-Node Context
nats context save local --server localhost:4222 --select

# Cluster Context (All 3 Nodes)
nats context save local-cluster --server localhost:4222,localhost:4223,localhost:4224 --select
```

**Pub / Sub Verification:**
```bash
# Terminal 1 - Subscribe
nats sub "demo.>"

# Terminal 2 - Publish
nats pub demo.hello "Hello from NATS"
```

**Request / Reply Verification:**
```bash
# Terminal 1 - Responder
nats reply demo.greet "Hello, requester"

# Terminal 2 - Requester
nats request demo.greet ""
```

---

## 6. Quick Reference Summary

| Property | Single-Node (Podman Container) | Single-Node (Native Binary) | 3-Node Cluster (Podman Container) | 3-Node Cluster (Native Binary) |
|----------|--------------------------------|-----------------------------|-----------------------------------|--------------------------------|
| **Primary Requirement** | Podman | `nats-server` (Brew) | Podman (Compose optional) | `nats-server` (Brew) |
| **Start Command** | `podman-compose up -d` in `deploy/local-nats` | `nats-server -js` or `nats-server -c deploy/local-nats/nats.conf` | `podman-compose up -d` in `deploy/local-nats-cluster` | 3 `nats-server` instances with `-cluster` and `-routes` |
| **Stop Command** | `podman-compose down` | `Ctrl+C` or `pkill nats-server` | `podman-compose down` | `Ctrl+C` or `pkill nats-server` |
| **Client URL** | `localhost:4222` | `localhost:4222` | `localhost:4222, 4223, 4224` | `localhost:4222, 4223, 4224` |
| **HTTP Monitoring** | `http://localhost:8222` | `http://localhost:8222` (if `-m 8222`) | `http://localhost:8222`, `8223`, `8224` | `http://localhost:8222`, `8223`, `8224` |
| **Config Location** | `deploy/local-nats/nats.conf` | `deploy/local-nats/nats.conf` | `deploy/local-nats-cluster/nats-{1,2,3}.conf` | Flags or cluster config files |
| **Persisted Storage** | `./deploy/local-nats/data` | `./deploy/local-nats/data/jetstream` | `./deploy/local-nats-cluster/data/` | `./deploy/local-nats-cluster/data/` |
