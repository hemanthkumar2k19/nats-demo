# NATS Client & SDK Developer Guide

This guide provides application developers with reference patterns for initializing and managing NATS client connections and JetStream API access.

The concepts and descriptions are language agnostic. Code examples feature the official NATS Go and Java SDKs.

---

## Prerequisites

### Go NATS Client SDK
- **Go Version:** Go 1.22+
- **SDK Installation:**
  ```bash
  go get github.com/nats-io/nats.go
  ```

### Java NATS Client SDK
- **Java Version:** Java 17+
- **Dependency Management (Maven):**
  ```xml
  <dependency>
      <groupId>io.nats</groupId>
      <artifactId>jnats</artifactId>
      <version>2.20.0</version>
  </dependency>
  ```
- **Dependency Management (Gradle):**
  ```groovy
  implementation 'io.nats:jnats:2.20.0'
  ```

---

# 1. NATS Client Initialization

## 1.1 Basic Connection

**Description:**
Establish a NATS client connection to the platform-provided NATS endpoint. The connection should be initialized during application startup and reused throughout the application lifecycle.

### Go

```go
package main

import (
	"log"

	"github.com/nats-io/nats.go"
)

func main() {
	// Use the NATS endpoint provided by the platform.
	nc, err := nats.Connect("nats://nats.example.com:4222")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	log.Printf("Connected to NATS: %s", nc.ConnectedUrl())
}
```

### Java

```java
import io.nats.client.Connection;
import io.nats.client.Nats;

public class BasicConnection {
    public static void main(String[] args) {
        // Use the NATS endpoint provided by the platform.
        try (Connection nc = Nats.connect("nats://nats.example.com:4222")) {
            System.out.printf("Connected to NATS: %s%n", nc.getConnectedUrl());
        } catch (Exception e) {
            System.err.printf("Failed to connect to NATS: %s%n", e.getMessage());
        }
    }
}
```

---

## 1.2 Cluster Connection

**Description:**
Establish a NATS client connection using the platform-provided cluster endpoints. The client can connect to an available server and discover additional servers in the cluster.

### Go

```go
package main

import (
	"log"

	"github.com/nats-io/nats.go"
)

func main() {
	// Use the cluster endpoints provided by the platform.
	servers := "nats://node1.example.com:4222,nats://node2.example.com:4222,nats://node3.example.com:4222"

	nc, err := nats.Connect(servers)
	if err != nil {
		log.Fatalf("Failed to connect to NATS cluster: %v", err)
	}
	defer nc.Close()

	log.Printf("Connected server: %s", nc.ConnectedUrl())
	log.Printf("Discovered servers: %v", nc.DiscoveredServers())
}
```

### Java

```java
import io.nats.client.Connection;
import io.nats.client.Nats;
import io.nats.client.Options;

public class ClusterConnection {
    public static void main(String[] args) {
        // Use the cluster endpoints provided by the platform.
        String[] servers = new String[] {
            "nats://node1.example.com:4222",
            "nats://node2.example.com:4222",
            "nats://node3.example.com:4222"
        };

        Options options = new Options.Builder()
                .servers(servers)
                .build();

        try (Connection nc = Nats.connect(options)) {
            System.out.printf("Connected server: %s%n", nc.getConnectedUrl());
            System.out.printf("Discovered servers: %s%n", nc.getServers());
        } catch (Exception e) {
            System.err.printf("Failed to connect to NATS cluster: %s%n", e.getMessage());
        }
    }
}
```

---

# 2. NATS Connection Lifecycle

## 2.1 Connection States

**Description:**
A NATS client connection is a long-lived resource that can transition between connected, disconnected/reconnecting, draining, and closed states during the application lifecycle.

```text
        ┌─────────────┐
        │  CONNECTED  │
        └──────┬──────┘
               │
       Network / Server
          disruption
               ↓
        ┌─────────────┐
        │ RECONNECTING│
        └──────┬──────┘
               │
       ┌───────┴────────┐
       ↓                ↓
 Successful          Reconnect
 reconnect           exhausted
       │                │
       ↓                ↓
 CONNECTED           CLOSED

Application Shutdown
       ↓
    DRAINING
       ↓
     CLOSED
```

---

## 2.2 Lifecycle Event Handling

**Description:**
Applications can register lifecycle handlers to observe connection establishment, disconnection, reconnection, closure, and server discovery events.

### Go

```go
opts := []nats.Option{
	nats.ConnectHandler(func(c *nats.Conn) {
		log.Printf("Connected to %s", c.ConnectedUrl())
	}),

	nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
		log.Printf("Disconnected: %v", err)
	}),

	nats.ReconnectHandler(func(c *nats.Conn) {
		log.Printf("Reconnected to %s", c.ConnectedUrl())
	}),

	nats.ClosedHandler(func(c *nats.Conn) {
		log.Printf("NATS connection closed")
	}),
}

nc, err := nats.Connect(serverURL, opts...)
```

### Java

```java
ConnectionListener connectionListener = (conn, type) -> {
    switch (type) {
        case CONNECTED -> System.out.printf("Connected to %s%n", conn.getConnectedUrl());
        case DISCONNECTED -> System.out.println("Disconnected from NATS server");
        case RECONNECTED -> System.out.printf("Reconnected to %s%n", conn.getConnectedUrl());
        case CLOSED -> System.out.println("NATS connection closed");
        case DISCOVERED_SERVERS -> System.out.println("Discovered new servers");
    }
};

Options options = new Options.Builder()
        .server(serverUrl)
        .connectionListener(connectionListener)
        .build();

Connection nc = Nats.connect(options);
```

---

## 2.3 Graceful Shutdown

**Description:**
Gracefully terminate the NATS client during application shutdown so that active subscriptions and pending operations can be handled before the connection is closed.

### Go

```go
// Use Drain during application shutdown when graceful termination is required.
if err := nc.Drain(); err != nil {
	log.Printf("NATS drain failed: %v", err)
}
```

### Java

```java
// Use drain during application shutdown when graceful termination is required.
try {
    nc.drain(Duration.ofSeconds(5)).get();
} catch (Exception e) {
    System.err.printf("NATS drain failed: %s%n", e.getMessage());
}
```

---

# 3. NATS Connection Configuration

## 3.1 Client Identification

**Description:**
Provide a meaningful client name so that application connections can be identified during NATS operational diagnostics.

### Go

```go
nc, err := nats.Connect(
	serverURL,
	nats.Name("order-processing-service"),
)
```

### Java

```java
Options options = new Options.Builder()
        .server(serverUrl)
        .connectionName("order-processing-service")
        .build();

Connection nc = Nats.connect(options);
```

---

## 3.2 Connection Timeout

**Description:**
Configure the timeout used when establishing the initial connection to NATS.

### Go

```go
nc, err := nats.Connect(
	serverURL,
	nats.Timeout(5*time.Second),
)
```

### Java

```java
Options options = new Options.Builder()
        .server(serverUrl)
        .connectionTimeout(Duration.ofSeconds(5))
        .build();

Connection nc = Nats.connect(options);
```

---

## 3.3 Reconnection

**Description:**
Configure client reconnection behavior for temporary network or server disruptions.

### Go

```go
nc, err := nats.Connect(
	serverURL,
	nats.MaxReconnects(10),
	nats.ReconnectWait(2*time.Second),
)
```

### Java

```java
Options options = new Options.Builder()
        .server(serverUrl)
        .maxReconnects(10)
        .reconnectWait(Duration.ofSeconds(2))
        .build();

Connection nc = Nats.connect(options);
```

Applications should use the platform-recommended reconnect configuration rather than independently defining arbitrary retry behavior.

---

## 3.4 Connection Liveness

**Description:**
Configure client-side liveness checks to detect an unresponsive NATS connection.

### Go

```go
nc, err := nats.Connect(
	serverURL,
	nats.PingInterval(15*time.Second),
	nats.MaxPingsOutstanding(3),
)
```

### Java

```java
Options options = new Options.Builder()
        .server(serverUrl)
        .pingInterval(Duration.ofSeconds(15))
        .maxPingsOutstanding(3)
        .build();

Connection nc = Nats.connect(options);
```

---

## 3.5 Recommended Connection Configuration

The platform may provide recommended values for:

* Client identification
* Connection timeout
* Reconnect attempts
* Reconnect wait/backoff
* Connection liveness

Application teams should use the platform defaults/recommendations unless a specific requirement requires deviation.

---

# 4. NATS Connection Validation

## 4.1 Connection State

**Description:**
Check the current client connection state before performing operations that require an active NATS connection.

### Go

```go
switch nc.Status() {
case nats.CONNECTED:
	log.Println("NATS connection is active")

case nats.RECONNECTING:
	log.Println("NATS connection is reconnecting")

case nats.DRAINING:
	log.Println("NATS connection is draining")

case nats.CLOSED:
	log.Println("NATS connection is closed")
}
```

### Java

```java
Connection.Status status = nc.getStatus();
switch (status) {
    case CONNECTED -> System.out.println("NATS connection is active");
    case RECONNECTING -> System.out.println("NATS connection is reconnecting");
    case DRAINING -> System.out.println("NATS connection is draining");
    case CLOSED -> System.out.println("NATS connection is closed");
    default -> System.out.println("Status: " + status);
}
```

---

## 4.2 Round-Trip Time

**Description:**
Measure the NATS protocol round-trip time to help identify connection latency.

### Go

```go
rtt, err := nc.RTT()
if err != nil {
	log.Printf("Failed to measure NATS RTT: %v", err)
	return
}

log.Printf("NATS RTT: %v", rtt)
```

### Java

```java
try {
    Duration rtt = nc.getRTT();
    System.out.printf("NATS RTT: %s%n", rtt);
} catch (Exception e) {
    System.err.printf("Failed to measure NATS RTT: %s%n", e.getMessage());
}
```

---

## 4.3 Server and Connection Information

**Description:**
Inspect information about the currently connected NATS server and connection traffic.

### Go

```go
log.Printf("Server ID: %s", nc.ConnectedServerId())
log.Printf("Server Name: %s", nc.ConnectedServerName())
log.Printf("Server Version: %s", nc.ConnectedServerVersion())
log.Printf("Cluster: %s", nc.ConnectedClusterName())

stats := nc.Stats()

log.Printf(
	"InBytes=%d OutBytes=%d InMsgs=%d OutMsgs=%d Reconnects=%d",
	stats.InBytes,
	stats.OutBytes,
	stats.InMsgs,
	stats.OutMsgs,
	stats.Reconnects,
)
```

### Java

```java
ServerInfo info = nc.getServerInfo();
if (info != null) {
    System.out.printf("Server ID: %s%n", info.getServerId());
    System.out.printf("Server Name: %s%n", info.getServerName());
    System.out.printf("Server Version: %s%n", info.getVersion());
    System.out.printf("Cluster: %s%n", info.getClusterName());
}

Statistics stats = nc.getStatistics();
System.out.printf(
    "InBytes=%d OutBytes=%d InMsgs=%d OutMsgs=%d Reconnects=%d%n",
    stats.getInBytes(),
    stats.getOutBytes(),
    stats.getInMsgs(),
    stats.getOutMsgs(),
    stats.getReconnects()
);
```

---

# 5. JetStream API Context

## 5.1 Initialize JetStream Context

**Description:**
Create a JetStream API context from an established NATS client connection when the application requires JetStream functionality.

### Go

```go
nc, err := nats.Connect(serverURL)
if err != nil {
	log.Fatalf("Failed to connect to NATS: %v", err)
}
defer nc.Close()

js, err := jetstream.New(nc)
if err != nil {
	log.Fatalf("Failed to initialize JetStream: %v", err)
}
```

### Java

```java
try (Connection nc = Nats.connect(serverUrl)) {
    JetStream js = nc.jetStream();
    JetStreamManagement jsm = nc.jetStreamManagement();
} catch (Exception e) {
    System.err.printf("Failed to initialize JetStream: %s%n", e.getMessage());
}
```

---

## 5.2 Validate JetStream API Access

**Description:**
Verify that the application can access the JetStream API and retrieve account-level JetStream information.

### Go

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

info, err := js.AccountInfo(ctx)
if err != nil {
	log.Fatalf("JetStream API access failed: %v", err)
}

log.Printf("JetStream streams: %d", info.Streams)
```

### Java

```java
try {
    JetStreamManagement jsm = nc.jetStreamManagement();
    AccountStatistics stats = jsm.getAccountStatistics();
    System.out.printf("JetStream streams: %d%n", stats.getStreams());
} catch (Exception e) {
    System.err.printf("JetStream API access failed: %s%n", e.getMessage());
}
```

---

# 6. Client Resource Management

## 6.1 Long-Lived Connection

**Description:**
Maintain a long-lived NATS client connection and reuse it across application operations. NATS clients are designed to multiplex application traffic over the established connection.

### Go

```go
type Service struct {
	nc *nats.Conn
}

func NewService(nc *nats.Conn) *Service {
	return &Service{
		nc: nc,
	}
}
```

### Java

```java
public class Service {
    private final Connection nc;

    public Service(Connection nc) {
        this.nc = nc;
    }
}
```

---

## 6.2 Avoid Connection Per Operation

**Description:**
Do not create and close a NATS connection for every application operation. Reuse the established client connection.

### Go (Avoid)

```go
// Do not create a new NATS connection for every request.
func handleRequest() {
	nc, _ := nats.Connect(serverURL)
	defer nc.Close()

	// Application operation...
}
```

### Go (Recommended)

```go
// Reuse the application-level NATS connection.
type Service struct {
	nc *nats.Conn
}

func (s *Service) HandleRequest() {
	// Use the existing connection.
}
```

### Java (Avoid)

```java
// Do not create a new NATS connection for every request.
public void handleRequest() {
    try (Connection nc = Nats.connect(serverUrl)) {
        // Application operation...
    } catch (Exception e) {
        e.printStackTrace();
    }
}
```

### Java (Recommended)

```java
// Reuse the application-level NATS connection.
public class Service {
    private final Connection nc;

    public Service(Connection nc) {
        this.nc = nc;
    }

    public void handleRequest() {
        // Use the existing connection.
    }
}
```

---

# 7. Client Error Handling

## 7.1 Connection Errors

**Description:**
Handle errors returned during initial connection establishment and explicitly determine the application behavior when the connection cannot be established.

### Go

```go
nc, err := nats.Connect(serverURL)
if err != nil {
	log.Printf("Unable to connect to NATS: %v", err)
	return
}
```

### Java

```java
try {
    Connection nc = Nats.connect(serverUrl);
} catch (Exception e) {
    System.out.printf("Unable to connect to NATS: %s%n", e.getMessage());
}
```

---

## 7.2 Asynchronous Errors

**Description:**
Register an asynchronous error handler when the application needs to observe protocol or subscription-related errors reported by the client.

### Go

```go
opts := []nats.Option{
	nats.ErrorHandler(func(
		c *nats.Conn,
		sub *nats.Subscription,
		err error,
	) {
		log.Printf("NATS asynchronous error: %v", err)
	}),
}

nc, err := nats.Connect(serverURL, opts...)
```

### Java

```java
ErrorListener errorListener = new ErrorListener() {
    @Override
    public void errorOccurred(Connection conn, String error) {
        System.out.printf("NATS asynchronous error: %s%n", error);
    }

    @Override
    public void exceptionOccurred(Connection conn, Exception exp) {
        System.out.printf("NATS exception: %s%n", exp.getMessage());
    }

    @Override
    public void slowConsumerDetected(Connection conn, Consumer consumer) {
        System.out.println("Slow consumer detected");
    }
};

Options options = new Options.Builder()
        .server(serverUrl)
        .errorListener(errorListener)
        .build();

Connection nc = Nats.connect(options);
```

---

## 7.3 Reconnection-Related Errors

**Description:**
Applications should use connection lifecycle events to detect disconnection and reconnection rather than implementing an independent connection-retry loop.

### Go

```go
nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
	log.Printf("NATS disconnected: %v", err)
})

nats.ReconnectHandler(func(c *nats.Conn) {
	log.Printf("NATS reconnected to %s", c.ConnectedUrl())
})
```

### Java

```java
ConnectionListener connectionListener = (conn, type) -> {
    if (type == ConnectionListener.Events.DISCONNECTED) {
        System.out.println("NATS disconnected");
    } else if (type == ConnectionListener.Events.RECONNECTED) {
        System.out.printf("NATS reconnected to %s%n", conn.getConnectedUrl());
    }
};

Options options = new Options.Builder()
        .server(serverUrl)
        .connectionListener(connectionListener)
        .build();

Connection nc = Nats.connect(options);
```

---

# 8. Client Testing

## 8.1 Connection Test

**Description:**
Validate that the application can establish a NATS connection against a test NATS server.

### Go

```go
func TestNATSConnection(t *testing.T) {
	nc, err := nats.Connect(
		"nats://localhost:4222",
		nats.Timeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	if !nc.IsConnected() {
		t.Fatal("Expected NATS connection to be active")
	}
}
```

### Java

```java
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class NatsConnectionTest {
    @Test
    public void testNatsConnection() {
        Options options = new Options.Builder()
                .server("nats://localhost:4222")
                .connectionTimeout(Duration.ofSeconds(2))
                .build();

        try (Connection nc = Nats.connect(options)) {
            assertEquals(Connection.Status.CONNECTED, nc.getStatus(), "Expected NATS connection to be active");
        } catch (Exception e) {
            fail("Failed to connect to NATS: " + e.getMessage());
        }
    }
}
```

---

## 8.2 JetStream API Test

**Description:**
Validate that the application can initialize and access the JetStream API when JetStream is enabled in the test environment.

### Go

```go
func TestJetStreamAPI(t *testing.T) {
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to initialize JetStream: %v", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	_, err = js.AccountInfo(ctx)
	if err != nil {
		t.Fatalf("JetStream API access failed: %v", err)
	}
}
```

### Java

```java
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class JetStreamApiTest {
    @Test
    public void testJetStreamApi() {
        try (Connection nc = Nats.connect("nats://localhost:4222")) {
            JetStreamManagement jsm = nc.jetStreamManagement();
            AccountStatistics stats = jsm.getAccountStatistics();
            assertNotNull(stats, "JetStream API access failed");
        } catch (Exception e) {
            fail("JetStream API test failed: " + e.getMessage());
        }
    }
}
```

---

## Official References

* [Official NATS Go Client Repository](https://github.com/nats-io/nats.go)
* [NATS Go Client API Documentation (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go)
* [NATS Go JetStream API Documentation (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)
* [Official NATS Java Client Repository](https://github.com/nats-io/nats.java)
* [NATS Java Client Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/index.html)
* [Official NATS Developer Documentation](https://docs.nats.io/using-nats/developer)

