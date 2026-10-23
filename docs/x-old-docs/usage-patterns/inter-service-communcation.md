# NATS Inter-Service Communication Usage Pattern

## Purpose

This guide defines application-level practices for **Inter-Service Communication** using NATS Core messaging, JetStream persistent streams, and the NATS Micro Services framework.

It covers synchronous request-reply queries, asynchronous event-driven notifications, native load balancing via queue groups, security and architectural advantages over traditional HTTP/REST microservice stacks, and complete Go and Java SDK reference code.

---

## 1. Overview & Architectural Principles

Traditional microservices architectures rely on direct HTTP/REST or gRPC calls between services. While simple initially, this introduces operational complexity:

* **Service Discovery Tax**: Requires dedicated infrastructure components (e.g. Eureka, Consul, or Kubernetes DNS) to track dynamic IP addresses and ports.
* **External Load Balancers**: Requires sidecar proxies (Envoy) or dedicated load balancers (NGINX, Ribbon) to distribute traffic.
* **Ingress Security Risks**: Requires opening inbound TCP listening ports on every microservice container, increasing attack surfaces.
* **Tight Temporal Coupling**: Synchronous HTTP calls chain latency together; if one downstream service experiences slowdowns, upstream callers block or fail.

NATS provides a **unified messaging backbone** that replaces this complex mesh with lightweight, subject-based addressability.

```text
[Traditional Microservices Architecture]
Client ---> API Gateway ---> [Eureka Registry / Envoy Proxy]
                                  |                 |
                                  v (Inbound HTTP)  v (Inbound HTTP)
                            Service A ----------> Service B

[NATS Unified Inter-Service Backbone]
                            Service A           Service B
                               |                    ^
              (Outbound TCP)   v                    | (Outbound TCP)
           =================================================
                            NATS Core / JetStream Broker
           =================================================
```

### Architectural Advantages of NATS Inter-Service Communication

1. **Location-Transparent Subject Addressability**: Services communicate using logical subjects (e.g., `inventory.query.stock` or `orders.event.created`). Microservices never need to know the IP address, host, or port of other services.
2. **Native Load Balancing**: Core NATS Queue Groups (`QueueSubscribe`) and JetStream Durable Consumer Groups automatically load-balance requests across multiple service instances without extra proxies or sidecars.
3. **Watertight Perimeter Security**: Microservices connect *outward* to the NATS cluster (default TCP 4222). Microservices require **zero open inbound network ports**, shielding containers from external intrusion.
4. **Seamless Hybrid Communication**: Supports both synchronous Request-Reply (querying data) and asynchronous Event-Driven Pub/Sub (decoupled notifications) over a single shared TCP connection.

---

## 2. NATS Inter-Service Communication Paradigms

NATS supports three primary interaction patterns for inter-service communication:

| Messaging Pattern | NATS Mechanism | Use Case | Lifecycle & Delivery |
| :--- | :--- | :--- | :--- |
| **Synchronous Request-Reply** | `nc.Request` / `nc.request()` | Direct point-to-point queries (e.g. `inventory.query.stock`). | Ephemeral inbox (`_INBOX.xxx`). Sender blocks until responder replies or timeout occurs. |
| **Asynchronous Event-Driven** | Core NATS `Publish` / JetStream `js.Publish` | Broadcasting domain events (e.g. `orders.event.created`). | Decoupled fire-and-forget or JetStream stream persistence for multiple subscribers. |
| **Microservices Framework (`NATS Micro`)** | `NATS Micro` API | Production REST-like service endpoints with auto-discovery and health metrics. | Endpoint routing with standard monitoring subjects (`$SRV.PING`, `$SRV.INFO`, `$SRV.STATS`). |

---

## 3. Code Examples: Go & Java SDKs

The following examples demonstrate inter-service communication between an **Order Service** and an **Inventory Service**:
1. **Synchronous Query**: Order Service queries Inventory Service for item stock availability using Request-Reply.
2. **Asynchronous Broadcast**: Order Service publishes an order creation event to JetStream, which is consumed independently by Inventory Service and Notification Service.

---

### 3.1 Synchronous Request-Reply Inter-Service Communication

In this pattern, the requesting service publishes a request to a concrete subject (e.g. `inventory.query.stock`) and waits synchronously for a response. The NATS SDK automatically generates a unique temporary reply inbox (`_INBOX.xxx`).

#### Workflow

```text
Order Service                                 NATS Server                             Inventory Service
      |                                            |                                          |
      |-- 1. Request (inventory.query.stock) ----->|                                          |
      |      ReplyTo: _INBOX.772A                |-- 2. Deliver Request ------------------->|
      |                                            |                                          |-- 3. Check Stock DB
      |                                            |<-- 4. Publish Response (_INBOX.772A) ----|
      |<-- 5. Deliver Response --------------------|                                          |
```

#### Go

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Data Models
type StockCheckReq struct {
	SKU string `json:"sku"`
}

type StockCheckResp struct {
	SKU       string `json:"sku"`
	Available int    `json:"available"`
	InStock   bool   `json:"in_stock"`
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("NATS connect error: %v", err)
	}
	defer nc.Close()

	// ------------------------------------------------------------------
	// 1. Inventory Service: Responder Subscription (Queue Group for Load Balancing)
	// ------------------------------------------------------------------
	_, err = nc.QueueSubscribe("inventory.query.stock", "inventory-service-group", func(msg *nats.Msg) {
		var req StockCheckReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[Inventory Service] Invalid request payload: %v", err)
			return
		}

		log.Printf("[Inventory Service] Received stock query for SKU: %s", req.SKU)

		// Business Logic: Check stock database
		resp := StockCheckResp{
			SKU:       req.SKU,
			Available: 42,
			InStock:   true,
		}
		respBytes, _ := json.Marshal(resp)

		// Respond directly to the caller's inbox (_INBOX.xxx)
		if err := msg.Respond(respBytes); err != nil {
			log.Printf("[Inventory Service] Failed to send response: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to subscribe responder: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// ------------------------------------------------------------------
	// 2. Order Service: Requester (Synchronous Inter-Service Call)
	// ------------------------------------------------------------------
	log.Println("[Order Service] Querying Inventory Service for SKU-9901...")
	reqPayload, _ := json.Marshal(StockCheckReq{SKU: "SKU-9901"})

	// Perform synchronous Request-Reply with a 2-second timeout
	respMsg, err := nc.Request("inventory.query.stock", reqPayload, 2*time.Second)
	if err != nil {
		if err == nats.ErrNoResponders {
			log.Fatalf("[Order Service] Error: No active Inventory Service instances available!")
		}
		log.Fatalf("[Order Service] Inter-service query failed: %v", err)
	}

	var stockResp StockCheckResp
	_ = json.Unmarshal(respMsg.Data, &stockResp)

	fmt.Printf("[Order Service] Response received! SKU: %s | InStock: %t | Quantity: %d\n",
		stockResp.SKU, stockResp.InStock, stockResp.Available)
}
```

#### Java

```java
package io.nats.demo.interservice;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.nats.client.*;

import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

public class RequestReplyInterServiceDemo {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    public static class StockCheckReq {
        public String sku;

        public StockCheckReq() {}
        public StockCheckReq(String sku) {
            this.sku = sku;
        }
    }

    public static class StockCheckResp {
        public String sku;
        public int available;
        public boolean inStock;

        public StockCheckResp() {}
        public StockCheckResp(String sku, int available, boolean inStock) {
            this.sku = sku;
            this.available = available;
            this.inStock = inStock;
        }
    }

    public static void main(String[] args) throws Exception {
        try (Connection nc = Nats.connect("nats://localhost:4222")) {

            // 1. Inventory Service Responder (Queue Subscription for Load Balancing)
            Dispatcher dispatcher = nc.createDispatcher(msg -> {
                try {
                    StockCheckReq req = MAPPER.readValue(msg.getData(), StockCheckReq.class);
                    System.out.printf("[Inventory Service] Received stock query for SKU: %s%n", req.sku);

                    StockCheckResp resp = new StockCheckResp(req.sku, 42, true);
                    byte[] respBytes = MAPPER.writeValueAsBytes(resp);

                    // Respond to caller inbox
                    nc.publish(msg.getReplyTo(), respBytes);
                } catch (Exception e) {
                    e.printStackTrace();
                }
            });
            dispatcher.subscribe("inventory.query.stock", "inventory-service-group");

            Thread.sleep(200);

            // 2. Order Service Requester (Synchronous Inter-Service Query)
            System.out.println("[Order Service] Querying Inventory Service for SKU-9901...");
            byte[] reqBytes = MAPPER.writeValueAsBytes(new StockCheckReq("SKU-9901"));

            CompletableFuture<Message> future = nc.request("inventory.query.stock", reqBytes);
            try {
                Message respMsg = future.get(2, TimeUnit.SECONDS);
                StockCheckResp stockResp = MAPPER.readValue(respMsg.getData(), StockCheckResp.class);

                System.out.printf("[Order Service] Response received! SKU: %s | InStock: %b | Quantity: %d%n",
                        stockResp.sku, stockResp.inStock, stockResp.available);
            } catch (Exception e) {
                System.err.printf("[Order Service] Inter-service query failed: %s%n", e.getMessage());
            }
        }
    }
}
```

---

### 3.2 Asynchronous Event-Driven Inter-Service Communication

In this pattern, a publishing microservice broadcasts domain events to a JetStream stream. Subscribed microservices consume events independently with durable subscriptions, ensuring persistent delivery, redelivery on failure, and message replay.

#### Event-Driven Workflow

```text
                                       +---> Inventory Service (Reserves Stock)
                                       |
Order Service ---> [JetStream Stream] -+
  (Publishes)      (ORDERS_EVENTS)     |
                                       +---> Notification Service (Sends Email)
```

#### Go

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type OrderCreatedEvent struct {
	OrderID     string    `json:"order_id"`
	SKU         string    `json:"sku"`
	Quantity    int       `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

func main() {
	ctx := context.Background()
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("NATS connect error: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("JetStream init error: %v", err)
	}

	// 1. Create JetStream Stream for Inter-Service Events
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "ORDERS_EVENTS",
		Subjects: []string{"orders.event.>"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		log.Fatalf("Stream creation error: %v", err)
	}

	// 2. Inventory Service Durable Consumer
	invCons, _ := js.CreateOrUpdateConsumer(ctx, "ORDERS_EVENTS", jetstream.ConsumerConfig{
		Durable:       "InventoryServiceConsumer",
		FilterSubject: "orders.event.created",
	})

	go func() {
		cc, _ := invCons.Consume(func(msg jetstream.Msg) {
			var evt OrderCreatedEvent
			_ = json.Unmarshal(msg.Data(), &evt)
			log.Printf("[Inventory Service] Deducting %d units of %s for Order %s", evt.Quantity, evt.SKU, evt.OrderID)
			_ = msg.Ack()
		})
		defer cc.Stop()
		select {}
	}()

	// 3. Notification Service Durable Consumer
	notifCons, _ := js.CreateOrUpdateConsumer(ctx, "ORDERS_EVENTS", jetstream.ConsumerConfig{
		Durable:       "NotificationServiceConsumer",
		FilterSubject: "orders.event.created",
	})

	go func() {
		cc, _ := notifCons.Consume(func(msg jetstream.Msg) {
			var evt OrderCreatedEvent
			_ = json.Unmarshal(msg.Data(), &evt)
			log.Printf("[Notification Service] Sending order confirmation email for Order %s", evt.OrderID)
			_ = msg.Ack()
		})
		defer cc.Stop()
		select {}
	}()

	time.Sleep(300 * time.Millisecond)

	// 4. Order Service Publishes Event
	evt := OrderCreatedEvent{
		OrderID:   "ORD-5541",
		SKU:       "SKU-9901",
		Quantity:  2,
		Timestamp: time.Now(),
	}
	evtBytes, _ := json.Marshal(evt)

	// Attach Nats-Msg-Id header for JetStream duplicate detection
	msg := nats.NewMsg("orders.event.created")
	msg.Data = evtBytes
	msg.Header.Set("Nats-Msg-Id", "order-created-ORD-5541")

	ack, err := js.PublishMsg(ctx, msg)
	if err != nil {
		log.Fatalf("Event publish failed: %v", err)
	}

	log.Printf("[Order Service] Event published to stream %s (Seq: %d)", ack.Stream, ack.Sequence)
	time.Sleep(1 * time.Second)
}
```

#### Java

```java
package io.nats.demo.interservice;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.nats.client.*;
import io.nats.client.api.ConsumerConfiguration;
import io.nats.client.api.StorageType;
import io.nats.client.api.StreamConfiguration;

import java.nio.charset.StandardCharsets;
import java.time.Instant;

public class EventDrivenInterServiceDemo {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    public static class OrderCreatedEvent {
        public String orderId;
        public String sku;
        public int quantity;
        public String timestamp;

        public OrderCreatedEvent() {}
        public OrderCreatedEvent(String orderId, String sku, int quantity, String timestamp) {
            this.orderId = orderId;
            this.sku = sku;
            this.quantity = quantity;
            this.timestamp = timestamp;
        }
    }

    public static void main(String[] args) throws Exception {
        try (Connection nc = Nats.connect("nats://localhost:4222")) {
            Management jsm = nc.createManagementContext();
            JetStream js = nc.jetStream();

            // 1. Create JetStream Stream for Inter-Service Events
            StreamConfiguration streamConfig = StreamConfiguration.builder()
                    .name("ORDERS_EVENTS")
                    .subjects("orders.event.>")
                    .storageType(StorageType.File)
                    .build();
            jsm.addOrUpdateStream(streamConfig);

            // 2. Inventory Service Durable Consumer
            ConsumerConfiguration invConfig = ConsumerConfiguration.builder()
                    .durable("InventoryServiceConsumer")
                    .filterSubject("orders.event.created")
                    .build();
            jsm.addOrUpdateConsumer("ORDERS_EVENTS", invConfig);

            Dispatcher dispatcher = nc.createDispatcher();
            js.subscribe("orders.event.created", dispatcher, msg -> {
                try {
                    OrderCreatedEvent evt = MAPPER.readValue(msg.getData(), OrderCreatedEvent.class);
                    System.out.printf("[Inventory Service] Deducting %d units of %s for Order %s%n",
                            evt.quantity, evt.sku, evt.orderId);
                    msg.ack();
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }, false, PushSubscribeOptions.bind("ORDERS_EVENTS", "InventoryServiceConsumer"));

            // 3. Notification Service Durable Consumer
            ConsumerConfiguration notifConfig = ConsumerConfiguration.builder()
                    .durable("NotificationServiceConsumer")
                    .filterSubject("orders.event.created")
                    .build();
            jsm.addOrUpdateConsumer("ORDERS_EVENTS", notifConfig);

            js.subscribe("orders.event.created", dispatcher, msg -> {
                try {
                    OrderCreatedEvent evt = MAPPER.readValue(msg.getData(), OrderCreatedEvent.class);
                    System.out.printf("[Notification Service] Sending order confirmation email for Order %s%n", evt.orderId);
                    msg.ack();
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }, false, PushSubscribeOptions.bind("ORDERS_EVENTS", "NotificationServiceConsumer"));

            Thread.sleep(300);

            // 4. Order Service Publishes Event
            OrderCreatedEvent evt = new OrderCreatedEvent("ORD-5542", "SKU-9901", 2, Instant.now().toString());
            byte[] payload = MAPPER.writeValueAsBytes(evt);

            Headers headers = new Headers();
            headers.set("Nats-Msg-Id", "order-created-ORD-5542");

            Message msg = NatsMessage.builder()
                    .subject("orders.event.created")
                    .headers(headers)
                    .data(payload)
                    .build();

            PublishAck ack = js.publish(msg);
            System.out.printf("[Order Service] Event published to stream %s (Seq: %d)%n", ack.getStream(), ack.getSequence());

            Thread.sleep(1000);
        }
    }
}
```

---

### 3.3 Microservice Framework (`NATS Micro`) Endpoint Example

The **NATS Micro Services Framework** standardizes service endpoint registration, auto-discovery, and health monitoring. Every microservice registered with `NATS Micro` automatically handles system discovery queries (`$SRV.PING`, `$SRV.INFO`, `$SRV.STATS`) without custom code.

#### Go

```go
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("NATS connect error: %v", err)
	}
	defer nc.Close()

	// 1. Create NATS Micro Service Config
	srvConfig := micro.Config{
		Name:        "InventoryService",
		Version:     "1.0.0",
		Description: "Handles inventory stock checks and reservations",
	}

	svc, err := micro.AddService(nc, srvConfig)
	if err != nil {
		log.Fatalf("Failed to add micro service: %v", err)
	}

	// 2. Add Service Endpoint
	err = svc.AddEndpoint("StockCheck", micro.HandlerFunc(func(req micro.Request) {
		var body StockCheckReq
		_ = json.Unmarshal(req.Data(), &body)

		resp := StockCheckResp{
			SKU:       body.SKU,
			Available: 100,
			InStock:   true,
		}
		respBytes, _ := json.Marshal(resp)
		_ = req.Respond(respBytes)
	}), micro.WithEndpointSubject("inventory.query.stock"))

	if err != nil {
		log.Fatalf("Failed to add endpoint: %v", err)
	}

	log.Printf("NATS Micro Service %s (v%s) running...", svc.Info().Name, svc.Info().Version)
	log.Println("Responds to queries on subject: inventory.query.stock")
	log.Println("Monitoring subjects active: $SRV.PING, $SRV.INFO, $SRV.STATS")

	// Keep running
	select {}
}
```

#### Java

```java
package io.nats.demo.interservice;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.nats.client.*;
import io.nats.client.service.*;

public class MicroServiceFrameworkDemo {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    public static void main(String[] args) throws Exception {
        try (Connection nc = Nats.connect("nats://localhost:4222")) {

            // 1. Define Service Endpoint Handler
            ServiceMessageHandler handler = msg -> {
                try {
                    StockCheckReq req = MAPPER.readValue(msg.getData(), StockCheckReq.class);
                    StockCheckResp resp = new StockCheckResp(req.sku, 100, true);
                    byte[] respBytes = MAPPER.writeValueAsBytes(resp);

                    msg.respond(nc, respBytes);
                } catch (Exception e) {
                    e.printStackTrace();
                }
            };

            // 2. Build Service Endpoint
            Endpoint endpoint = Endpoint.builder()
                    .name("StockCheck")
                    .subject("inventory.query.stock")
                    .build();

            // 3. Build and Start Micro Service
            Service service = new ServiceBuilder()
                    .connection(nc)
                    .name("InventoryService")
                    .version("1.0.0")
                    .description("Handles inventory stock checks and reservations")
                    .addSegment(endpoint, handler)
                    .build();

            service.startService();
            System.out.printf("NATS Micro Service %s (v%s) running...%n", service.getName(), service.getVersion());
            System.out.println("Monitoring subjects active: $SRV.PING, $SRV.INFO, $SRV.STATS");

            Thread.sleep(5000);
        }
    }
}
```

---

## 4. Best Practices for Enterprise Inter-Service Communication

### 4.1 Distributed Tracing & Correlation Headers
Pass W3C standard trace context or custom correlation headers across inter-service calls to maintain visibility:

```text
Content-Type: application/json
X-Correlation-ID: corr-88123-abc
X-Trace-ID: 4bf92f3577b34da6a3ce929d0e0e4736
```

### 4.2 Handling Service Unavailability (`ErrNoResponders`)
In Core NATS Request-Reply, if no subscribers are listening on the target request subject, NATS immediately returns a `503 No Responders` error (`nats.ErrNoResponders`) rather than timing out. Callers should handle this error to return circuit-breaker status or fallback responses instantly.

### 4.3 Deduplication & Idempotency
Use `Nats-Msg-Id` headers when publishing inter-service events to JetStream. This prevents duplicate event processing if network retries occur.

---

## 5. Official References

* [NATS Go Client Documentation](https://github.com/nats-io/nats.go)
* [NATS Java Client Documentation](https://github.com/nats-io/nats.java)
* [Rethinking Microservices with NATS](https://www.synadia.com/blog/rethinking-microservices)
* [NATS Micro Services Framework](https://docs.nats.io/nats-concepts/services)
