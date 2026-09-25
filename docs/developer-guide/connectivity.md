# NATS Connectivity Guide

This guide covers establishing NATS server connections, configuring connection options, connecting to clusters, inspecting health, and initializing JetStream.

---

## Part 1: Core NATS Connection

### 1. Basic Connection and Graceful Closing

#### Generic Description
Establish a TCP connection to a NATS server URL. Always perform a graceful drain when closing a connection to flush pending outbound messages and complete active processing before releasing network resources.

#### Go SDK Implementation

```go
package main

import (
	"log"

	"github.com/nats-io/nats.go"
)

func main() {
	// 1. Establish connection
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}

	// 2. Perform application messaging...

	// 3. Graceful shutdown
	if err := nc.Drain(); err != nil {
		log.Printf("Error during connection drain: %v", err)
	}
}
```

---

### 2. Available Connection Options

#### Generic Description
Configure production settings including client identification, reconnection limits, backoff delays, buffer limits during network outages, heartbeat pings, dial timeouts, and lifecycle event callbacks.

#### Go SDK Implementation

```go
package main

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func ConnectWithOptions() (*nats.Conn, error) {
	return nats.Connect("nats://localhost:4222",
		nats.Name("demo-service"),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			delay := time.Duration(attempts) * time.Second
			if delay > 10*time.Second {
				return 10 * time.Second
			}
			return delay
		}),
		nats.ReconnectBufSize(8*1024*1024),
		nats.PingInterval(15*time.Second),
		nats.MaxPingsOutstanding(3),
		nats.Timeout(5*time.Second),

		// Lifecycle Callbacks
		nats.ConnectHandler(func(c *nats.Conn) {
			log.Printf("Connected: %s", c.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			log.Printf("Disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Printf("Reconnected: %s", c.ConnectedUrl())
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			log.Printf("Connection closed: %s", c.ConnectedUrl())
		}),
		nats.ErrorHandler(func(c *nats.Conn, sub *nats.Subscription, err error) {
			log.Printf("Async error: %v", err)
		}),
	)
}
```

---

### 3. How to Connect to a Cluster

#### Generic Description
Supply one or more initial seed server URLs. Upon connection, the NATS server advertises the cluster topology, allowing the client to dynamically discover additional cluster nodes and automatically failover if a server goes down.

#### Go SDK Implementation

```go
package main

import (
	"log"

	"github.com/nats-io/nats.go"
)

func ConnectCluster() (*nats.Conn, error) {
	// Seed URLs
	servers := "nats://localhost:4222,nats://localhost:4223,nats://localhost:4224"

	nc, err := nats.Connect(servers,
		nats.DiscoveredServersHandler(func(c *nats.Conn) {
			log.Printf("Discovered cluster nodes: %v", c.DiscoveredServers())
		}),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Connected Node: %s", nc.ConnectedUrl())
	log.Printf("Known Pool:     %v", nc.Servers())

	return nc, nil
}
```

---

### 4. Testing Connectivity & Health Inspection

#### Generic Description
Query connection state, server metadata, and socket statistics to implement health and readiness probes for monitoring.

#### Go SDK Connection Methods

| Method | Return Type | Description |
| :--- | :--- | :--- |
| `nc.Status()` | `nats.Status` | Connection state (`CONNECTED`, `RECONNECTING`, `DRAINING`, `CLOSED`). |
| `nc.IsConnected()` | `bool` | Returns `true` if TCP socket is active. |
| `nc.IsReconnecting()` | `bool` | Returns `true` if reconnecting. |
| `nc.IsDraining()` | `bool` | Returns `true` if connection is draining. |
| `nc.IsClosed()` | `bool` | Returns `true` if connection is closed. |
| `nc.LastError()` | `error` | Returns the last protocol or network error. |
| `nc.ConnectedUrl()` | `string` | Currently connected server URL. |
| `nc.ConnectedAddr()` | `string` | Connected socket IP address and port. |
| `nc.ConnectedServerId()` | `string` | Connected server node ID. |
| `nc.ConnectedServerName()` | `string` | Connected server name. |
| `nc.ConnectedServerVersion()` | `string` | NATS server software version. |
| `nc.ConnectedClusterName()` | `string` | NATS cluster name. |
| `nc.ConnectedServerJetStream()` | `(bool, int)` | Returns JetStream enabled status. |
| `nc.MaxPayload()` | `int64` | Max payload size in bytes. |
| `nc.DiscoveredServers()` | `[]string` | Dynamically discovered cluster server URLs. |
| `nc.Servers()` | `[]string` | Full pool of known server URLs. |
| `nc.Stats()` | `nats.Statistics` | Traffic statistics (`InMsgs`, `OutMsgs`, `InBytes`, `OutBytes`, `Reconnects`). |

#### Go Health Check Implementation

```go
func TestConnectivity(nc *nats.Conn) {
	if nc == nil || !nc.IsConnected() {
		log.Println("Health Check: FAILED")
		return
	}

	log.Printf("Connected Server: %s (%s)", nc.ConnectedServerName(), nc.ConnectedUrl())
	stats := nc.Stats()
	log.Printf("Traffic Stats:    In=%d msgs, Out=%d msgs", stats.InMsgs, stats.OutMsgs)
}
```

---

## Part 2: JetStream Connection

### 1. Connection and Graceful Closing

#### Generic Description
Initialize a JetStream context from an active NATS connection. Because JetStream operations rely on the underlying NATS transport, draining the NATS connection gracefully closes all active JetStream operations.

#### Go SDK Implementation

```go
package main

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	// 1. Connect to NATS
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// 2. Initialize JetStream Context
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}
	_ = js

	// 3. Graceful shutdown (drains both NATS and JetStream operations)
	if err := nc.Drain(); err != nil {
		log.Printf("Error during drain: %v", err)
	}
}
```
