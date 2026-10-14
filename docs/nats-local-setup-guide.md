# NATS Local Setup

Developer guide for running NATS locally for development and testing.

The guide is organized into two primary tracks designed for tabbed navigation:
- **Track 1: Podman Workflow (Containerized)** -- Container execution via Podman. Recommended for containerized workflows; does not require Homebrew or native NATS tools. Applications interact directly using NATS SDK clients or HTTP monitoring endpoints.
- **Track 2: NATS CLI / Local Binary Workflow (Homebrew)** -- Direct binary execution on macOS/Linux. Recommended for lightweight, fast iteration using `nats-server` and `nats` CLI without a container engine.

Within each track, two deployment topologies are provided:
- **Single-Node Server:** Fast, minimal setup with JetStream for day-to-day development.
- **3-Node Cluster:** Multi-node cluster for validating JetStream replication (R3), leader election, failover, and cluster routing.

---

## Track 1: Podman Workflow

Run NATS inside isolated containers using Podman. This workflow does not require macOS Homebrew or locally installed NATS binaries; applications can interact with the server directly through NATS SDK clients (Go, Java, Python, Node.js) or via HTTP monitoring endpoints.

### 1. Prerequisites

| Tool | Purpose | Install |
|------|---------|---------|
| Podman | Container engine for running NATS containers | [podman.io](https://podman.io/getting-started/installation) or system package manager |
| podman-compose | Multi-container compose orchestration (optional) | `pip install podman-compose` or `brew install podman-compose` |
| Application SDK Client | Application interaction with NATS | Go (`nats.go`), Java (`jnats`), Python (`nats-py`), etc. |

Pull the official NATS container image:

```bash
podman pull nats:latest
```

*(Optional)* If you have Homebrew or the NATS CLI installed, you can also use `nats` CLI commands, but it is not mandatory.

---

### 2. Single-Node Server (Podman)

A single NATS container with JetStream enabled. Suitable for standard application development and unit/integration testing.

**Source Files:** `deploy/local-nats/`
- `deploy/local-nats/docker-compose.yaml` -- Container compose specification
- `deploy/local-nats/nats.conf` -- Server configuration

Server configuration reference (`deploy/local-nats/nats.conf`):

```
# NATS client port
port: 4222

# HTTP monitoring port
http_port: 8222

# Server name
server_name: "local-nats"

jetstream {
    store_dir: "/data/jetstream"
}
```

Key settings:
- **Port 4222:** Standard client connection port exposed to host.
- **Port 8222:** HTTP monitoring endpoint (`/varz`, `/connz`, `/jsz`).
- **JetStream store_dir:** `/data/jetstream` inside the container, mapped to host directory `./deploy/local-nats/data`.

#### Option A: Start with podman-compose

From the repository root:

```bash
cd deploy/local-nats
podman-compose up -d
```

Stop the server:

```bash
cd deploy/local-nats
podman-compose down
```

Stop and wipe persisted JetStream storage:

```bash
cd deploy/local-nats
podman-compose down
rm -rf ./data
```

#### Option B: Start with podman run

Run directly from the repository root:

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

Stop and remove container:

```bash
podman stop local-nats
podman rm local-nats
```

Remove persisted JetStream data:

```bash
rm -rf ./deploy/local-nats/data
```

#### Verification (Podman Single-Node)

1. Verify container status:
   ```bash
   podman ps --filter name=local-nats
   ```

2. Check health via HTTP monitoring (no CLI required):
   ```bash
   curl -s http://localhost:8222/varz | grep version
   curl -s http://localhost:8222/jsz
   ```

3. Connect via Application SDK client:
   - Target URL: `nats://localhost:4222`

---

### 3. 3-Node Cluster (Podman)

A 3-node containerized NATS cluster for validating cluster behavior: JetStream replication (R3 streams), leader election, seamless failover, and cluster routing.

**Source Files:** `deploy/local-nats-cluster/`
- `deploy/local-nats-cluster/docker-compose.yaml` -- 3-node cluster specification
- `deploy/local-nats-cluster/nats-1.conf` -- Node 1 configuration
- `deploy/local-nats-cluster/nats-2.conf` -- Node 2 configuration
- `deploy/local-nats-cluster/nats-3.conf` -- Node 3 configuration

Cluster Port Mapping:
| Node | Client Port (Host) | HTTP Monitoring Port | Cluster Route Port (Internal) |
|------|--------------------|----------------------|--------------------------------|
| nats-1 | 4222 | 8222 | 6222 |
| nats-2 | 4223 | 8223 | 6222 |
| nats-3 | 4224 | 8224 | 6222 |

#### Option A: Start with podman-compose

From the repository root:

```bash
cd deploy/local-nats-cluster
podman-compose up -d
```

This starts `nats-1`, `nats-2`, and `nats-3`. Observability tools (Prometheus, Grafana, Fluent-bit) are under the `observability` profile and remain stopped unless explicitly started.

Stop the cluster:

```bash
cd deploy/local-nats-cluster
podman-compose down
```

Stop and remove persisted cluster data:

```bash
cd deploy/local-nats-cluster
podman-compose down
rm -rf ./data
```

#### Option B: Start with podman run

Create a user-defined container network for node discovery:

```bash
podman network create nats-cluster
```

Start all 3 nodes from the repository root:

```bash
podman run -d \
  --name nats-1 \
  --network nats-cluster \
  -p 4222:4222 \
  -v ./deploy/local-nats-cluster/nats-1.conf:/etc/nats/nats.conf:ro \
  -v ./deploy/local-nats-cluster/data/nats-1:/data \
  nats:latest -c /etc/nats/nats.conf
```

```bash
podman run -d \
  --name nats-2 \
  --network nats-cluster \
  -p 4223:4222 \
  -v ./deploy/local-nats-cluster/nats-2.conf:/etc/nats/nats.conf:ro \
  -v ./deploy/local-nats-cluster/data/nats-2:/data \
  nats:latest -c /etc/nats/nats.conf
```

```bash
podman run -d \
  --name nats-3 \
  --network nats-cluster \
  -p 4224:4222 \
  -v ./deploy/local-nats-cluster/nats-3.conf:/etc/nats/nats.conf:ro \
  -v ./deploy/local-nats-cluster/data/nats-3:/data \
  nats:latest -c /etc/nats/nats.conf
```

Stop and clean up containers:

```bash
podman stop nats-1 nats-2 nats-3
podman rm nats-1 nats-2 nats-3
podman network rm nats-cluster
```

Remove persisted cluster data:

```bash
rm -rf ./deploy/local-nats-cluster/data
```

#### Verification (Podman 3-Node Cluster)

1. Verify container status:
   ```bash
   podman ps --filter name=nats
   ```

2. Check cluster route mesh via HTTP monitoring:
   ```bash
   curl -s http://localhost:8222/routez
   ```

3. Connect via Application SDK client (providing all seed endpoints for automatic failover):
   - Target URL: `nats://localhost:4222,nats://localhost:4223,nats://localhost:4224`

---

## Track 2: NATS CLI / Local Binary Workflow

Run NATS locally as native processes on macOS or Linux. Best for fast iteration without container overhead or VM resource requirements.

### 1. Prerequisites

| Tool | Purpose | Install (macOS / Linux) |
|------|---------|-------------------------|
| NATS Server (`nats-server`) | Local NATS server binary | `brew install nats-server` |
| NATS CLI (`nats`) | Management and messaging CLI tool | `brew tap nats-io/nats-tools && brew install nats-io/nats-tools/nats` |

Verify installation:

```bash
nats-server -v
nats --version
```

---

### 2. Single-Node Server (CLI)

#### Option A: Ephemeral Dev Server (Quick Start)

Start an immediate foreground server with JetStream enabled (stores data in a temporary directory):

```bash
nats-server -js
```

Stop the server:
Press `Ctrl+C` in the terminal.

#### Option B: Config-backed Server (Persistent Storage)

Run with the repository configuration file and an explicit local storage directory:

```bash
nats-server -c deploy/local-nats/nats.conf -sd ./deploy/local-nats/data/jetstream
```

To run in the background:

```bash
nats-server -c deploy/local-nats/nats.conf -sd ./deploy/local-nats/data/jetstream &
```

Stop background server:

```bash
pkill nats-server
```

Remove persisted JetStream data:

```bash
rm -rf ./deploy/local-nats/data
```

#### Verification (CLI Single-Node)

Check connectivity and JetStream availability:

```bash
nats server ping
nats server report jetstream
```

---

### 3. 3-Node Cluster (CLI)

Run 3 separate `nats-server` processes locally with dedicated client, monitoring, and intra-cluster routing ports.

#### Start Nodes

Open 3 terminal windows and execute one command per terminal from the repository root:

Terminal 1 (Node 1):
```bash
nats-server -n nats-1 -p 4222 -m 8222 \
  -cluster nats://127.0.0.1:6222 \
  -routes nats://127.0.0.1:6223,nats://127.0.0.1:6224 \
  --cluster_name local-nats-cluster \
  -js -sd ./deploy/local-nats-cluster/data/nats-1
```

Terminal 2 (Node 2):
```bash
nats-server -n nats-2 -p 4223 -m 8223 \
  -cluster nats://127.0.0.1:6223 \
  -routes nats://127.0.0.1:6222,nats://127.0.0.1:6224 \
  --cluster_name local-nats-cluster \
  -js -sd ./deploy/local-nats-cluster/data/nats-2
```

Terminal 3 (Node 3):
```bash
nats-server -n nats-3 -p 4224 -m 8224 \
  -cluster nats://127.0.0.1:6224 \
  -routes nats://127.0.0.1:6222,nats://127.0.0.1:6223 \
  --cluster_name local-nats-cluster \
  -js -sd ./deploy/local-nats-cluster/data/nats-3
```

#### Stop Nodes

Press `Ctrl+C` in each terminal window, or stop all running server processes:

```bash
pkill nats-server
```

Remove persisted cluster data:

```bash
rm -rf ./deploy/local-nats-cluster/data/nats-{1,2,3}
```

#### Verification (CLI 3-Node Cluster)

Verify all nodes are online and clustered:

```bash
nats server list -s localhost:4222
nats server ping -s localhost:4222
```

---

## Interacting and Testing

### Approach 1: Application SDK Clients (No NATS CLI Required)

Applications connect directly using NATS client libraries. No Homebrew or local NATS CLI installation is required.

Connection Endpoints:
- **Single-Node:** `nats://localhost:4222`
- **3-Node Cluster:** `nats://localhost:4222,nats://localhost:4223,nats://localhost:4224`

Example Connection Patterns:

**Go (`nats.go`):**
```go
// Single node
nc, err := nats.Connect("nats://localhost:4222")

// Cluster with automatic failover
nc, err := nats.Connect("nats://localhost:4222,nats://localhost:4223,nats://localhost:4224")
```

**Java (`jnats`):**
```java
// Single node
Connection nc = Nats.connect("nats://localhost:4222");

// Cluster with automatic failover
Options options = new Options.Builder()
    .servers(new String[]{"nats://localhost:4222", "nats://localhost:4223", "nats://localhost:4224"})
    .build();
Connection nc = Nats.connect(options);
```

### Approach 2: NATS CLI Utility

If you have the NATS CLI installed, configure connection contexts:

Save Single-Node Context:
```bash
nats context save local --server localhost:4222 --select
```

Save Cluster Context (all 3 nodes):
```bash
nats context save local-cluster --server localhost:4222,localhost:4223,localhost:4224 --select
```

#### Pub / Sub Verification

Terminal 1 -- Subscribe:
```bash
nats sub "demo.>"
```

Terminal 2 -- Publish:
```bash
nats pub demo.hello "Hello from NATS"
```

#### Request / Reply Verification

Terminal 1 -- Reply:
```bash
nats reply demo.greet "Hello, requester"
```

Terminal 2 -- Request:
```bash
nats request demo.greet ""
```

---

## Quick Reference Summary

| Property | Single-Node (Podman) | 3-Node Cluster (Podman) | Single-Node (CLI) | 3-Node Cluster (CLI) |
|----------|----------------------|-------------------------|-------------------|----------------------|
| Primary Requirement | Podman / Compose | Podman / Compose | `nats-server` (Brew) | `nats-server` (Brew) |
| Start Command | `podman-compose up -d` in `deploy/local-nats` | `podman-compose up -d` in `deploy/local-nats-cluster` | `nats-server -js` | 3 `nats-server` commands with route ports |
| Stop Command | `podman-compose down` | `podman-compose down` | `Ctrl+C` or `pkill nats-server` | `Ctrl+C` or `pkill nats-server` |
| Client URL | `localhost:4222` | `localhost:4222,localhost:4223,localhost:4224` | `localhost:4222` | `localhost:4222,localhost:4223,localhost:4224` |
| HTTP Monitoring | `http://localhost:8222` | `http://localhost:8222`, `8223`, `8224` | `http://localhost:8222` (if `-m 8222`) | `http://localhost:8222`, `8223`, `8224` |
| Configuration | `deploy/local-nats/nats.conf` | `deploy/local-nats-cluster/nats-{1,2,3}.conf` | `deploy/local-nats/nats.conf` | CLI flags or local config files |
| Persisted Storage | `./deploy/local-nats/data` | `./deploy/local-nats-cluster/data/nats-{1,2,3}` | `./deploy/local-nats/data/jetstream` | `./deploy/local-nats-cluster/data/nats-{1,2,3}` |


