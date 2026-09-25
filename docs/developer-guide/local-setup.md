# NATS Local Setup

Developer guide for running NATS locally for development and testing.
Two approaches are provided: a single-node server for simple development and a 3-node cluster for testing replication, failover, and cluster-aware behavior.

Each approach can be started using `podman-compose` with the provided `compose.yaml`, or by running containers directly with `podman run`.

---

## Prerequisites

### Required Tools

| Tool | Purpose | Install |
|------|---------|---------|
| Podman | Run NATS containers | [podman.io](https://podman.io/getting-started/installation) |
| podman-compose | Orchestrate multi-container setups | `pip install podman-compose` or [github.com/containers/podman-compose](https://github.com/containers/podman-compose) |
| NATS CLI (`nats`) | Interact with NATS from the terminal | See below |

### Install NATS CLI

macOS (Homebrew):

```bash
brew tap nats-io/nats-tools
brew install nats-io/nats-tools/nats
```

Verify the installation:

```bash
nats --version
```

### Pull Container Image

```bash
podman pull nats:latest
```

---

## Single-Node Server

A single NATS server with JetStream enabled. Suitable for most local development and testing.

**Source:** `deploy/local-nats/`

### Directory Contents

```
deploy/local-nats/
  compose.yaml   # Container definition (works with podman-compose)
  nats.conf              # Server configuration
```

### Configuration

The server configuration in `nats.conf`:

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
- **port 4222** -- Standard NATS client connection port.
- **jetstream** -- Enabled with data stored at `/data/jetstream` inside the container, mapped to `./data` on the host via a volume mount.

### Start with podman-compose

```bash
cd deploy/local-nats
podman-compose up -d
```

Stop:

```bash
cd deploy/local-nats
podman-compose down
```

To remove persisted JetStream data as well:

```bash
cd deploy/local-nats
podman-compose down
rm -rf ./data
```

### Start with podman run

Run the server directly without compose:

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

Run from the repository root so the volume paths resolve correctly.

Stop and remove:

```bash
podman stop local-nats
podman rm local-nats
```

To remove persisted data as well:

```bash
rm -rf ./deploy/local-nats/data
```

### Start with nats server (CLI)

The NATS CLI includes a lightweight built-in server. No containers required -- useful for quick local development and testing.

Start with JetStream enabled:

```bash
nats server run --jetstream
```

The server runs in the foreground on port 4222 by default. Press Ctrl+C to stop it.

To use a custom configuration file:

```bash
nats server run -c deploy/local-nats/nats.conf
```

This is the simplest option for single-node development but does not persist data across restarts unless a store directory is configured.

### Verify

Check the container is running (podman-compose or podman run):

```bash
podman ps --filter name=local-nats
```

For `nats server run`, verify connectivity directly:

```bash
nats server ping
```

---

## 3-Node Cluster

A 3-node NATS cluster for testing cluster behavior: replication, leader election, failover, and multi-node JetStream.

**Source:** `deploy/local-nats-cluster/`

### Directory Contents

```
deploy/local-nats-cluster/
  compose.yaml   # 3 NATS nodes (works with podman-compose)
  nats-1.conf            # Node 1 configuration
  nats-2.conf            # Node 2 configuration
  nats-3.conf            # Node 3 configuration
```

### Configuration

Each node has its own configuration file. The structure is the same across nodes with the server name and route peers adjusted per node.

Example (`nats-1.conf`):

```
port: 4222
http_port: 8222

server_name: "nats-1"

server_tags: ["stream:primary"]

log_file: "/data/nats.log"
logtime: true

jetstream {
    store_dir: "/data/jetstream"
}

cluster {
    name: "local-nats-cluster"

    listen: "0.0.0.0:6222"

    routes [
        "nats://nats-2:6222"
        "nats://nats-3:6222"
    ]
}
```

Key settings:
- **server_name** -- Unique name per node (`nats-1`, `nats-2`, `nats-3`).
- **server_tags** -- `nats-1` and `nats-2` are tagged `stream:primary`. `nats-3` has no stream tag. Tags can be used for JetStream stream placement.
- **log_file** -- Each node writes logs to `/data/nats.log` inside its container, mapped to `./data/nats-{n}/` on the host.
- **cluster.name** -- All nodes share the cluster name `local-nats-cluster`.
- **cluster.listen** -- Intra-cluster communication on port 6222.
- **cluster.routes** -- Each node lists the other two nodes as route peers. NATS uses gossip to form a full mesh.

### Exposed Ports

| Host Port | Container (Node) | Purpose |
|-----------|-------------------|---------|
| 4222 | nats-1:4222 | Client connections (node 1) |
| 4223 | nats-2:4222 | Client connections (node 2) |
| 4224 | nats-3:4222 | Client connections (node 3) |

Cluster routing ports (6222) are internal to the Podman network and not exposed to the host.

### Start with podman-compose

```bash
cd deploy/local-nats-cluster
podman-compose up -d
```

This starts only the 3 NATS nodes. Observability services are behind the `observability` profile and are not started by default.

Stop:

```bash
cd deploy/local-nats-cluster
podman-compose down
```

To remove persisted data and logs:

```bash
cd deploy/local-nats-cluster
podman-compose down
rm -rf ./data
```

### Start with podman run

Create a shared network so the nodes can discover each other:

```bash
podman network create nats-cluster
```

Start all 3 nodes:

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

Run from the repository root so the volume paths resolve correctly.

Stop and remove:

```bash
podman stop nats-1 nats-2 nats-3
podman rm nats-1 nats-2 nats-3
podman network rm nats-cluster
```

To remove persisted data:

```bash
rm -rf ./deploy/local-nats-cluster/data
```

### Verify

Check all 3 containers are running:

```bash
podman ps --filter name=nats
```

---

## Testing Connectivity

After starting either the single-node server or the cluster, use the NATS CLI to verify the server is reachable and messaging works.

### Add a CLI Context (Optional)

Contexts store connection details so you do not have to pass `--server` every time.

Single-node:

```bash
nats context save local --server localhost:4222 --select
```

Cluster (all 3 nodes):

```bash
nats context save local-cluster --server localhost:4222,localhost:4223,localhost:4224 --select
```

### Check Connectivity

```bash
nats server ping
```

For a cluster, verify all nodes respond:

```bash
nats server list
```

### Publish and Subscribe

Open two terminals.

Terminal 1 -- Subscribe:

```bash
nats sub "demo.>"
```

Terminal 2 -- Publish:

```bash
nats pub demo.hello "Hello from NATS"
```

Terminal 1 should display the received message. Press Ctrl+C to stop the subscriber.

### Request and Reply

Terminal 1 -- Start a replier:

```bash
nats reply demo.greet "Hello, requester"
```

Terminal 2 -- Send a request:

```bash
nats request demo.greet ""
```

Terminal 2 should display the reply message.

---

## Quick Reference

| Action | Single-Node | Cluster |
|--------|-------------|---------|
| Start (compose) | `cd deploy/local-nats && podman-compose up -d` | `cd deploy/local-nats-cluster && podman-compose up -d` |
| Stop (compose) | `cd deploy/local-nats && podman-compose down` | `cd deploy/local-nats-cluster && podman-compose down` |
| Client URL | `localhost:4222` | `localhost:4222,localhost:4223,localhost:4224` |
| Config | `deploy/local-nats/nats.conf` | `deploy/local-nats-cluster/nats-{1,2,3}.conf` |
| Data | `deploy/local-nats/data/` | `deploy/local-nats-cluster/data/nats-{1,2,3}/` |
