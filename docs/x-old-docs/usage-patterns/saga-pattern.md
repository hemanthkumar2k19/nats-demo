# Event-Driven Saga Pattern with NATS

## Purpose

This guide defines application-level practices for implementing the **Event-Driven Saga Pattern** using NATS Core messaging and JetStream persistence.

It covers the architectural principles of distributed transactions without two-phase commit (2PC), subject hierarchy for commands and events, saga orchestration vs. choreography, compensating transaction flows, and concrete code implementations in Go and Java.

---

## 1. What is the Saga Pattern?

### 1.1 Overview & Distributed Transaction Problem

In a distributed microservices architecture, a single business operation often spans multiple independent services, each managing its own isolated database. Traditional database transactions (ACID with two-phase commit / 2PC) do not scale well across microservice boundaries due to tight coupling, network latency, lock contention, and single points of failure.

The **Saga Pattern** solves this problem by breaking a global distributed transaction into a sequence of **local transactions**:

1. Each saga participant executes a local transaction that updates its internal database.
2. Participant services publish messages (events or commands) to trigger the next step in the saga.
3. If a local transaction fails or business rules are violated, the saga executes a series of **compensating transactions** in reverse order to undo changes committed by earlier steps.

```text
[Successful Saga Execution]
Service A (Tx 1) ---> Service B (Tx 2) ---> Service C (Tx 3) ---> [Saga Completed]

[Failed Saga Execution & Compensation Flow]
Service A (Tx 1) ---> Service B (Tx 2) ---> Service C (Tx 3 FAIL!)
      |                     |                     |
      v                     v                     v
Compensate A <------- Compensate B <--------------+               [Saga Aborted]
```

### 1.2 Core Saga Concepts

#### Local Transactions
A distinct atomic transaction executed inside a single service boundary. Once a local transaction commits, its changes are durable within that service.

#### Compensating Actions
Explicit undo operations executed when a downstream step in the saga fails. A compensating action must be **idempotent** and **eventually consistent**, restoring the system to a clean, balanced state (e.g., refunding a credit card charge or canceling an unfulfilled order).

#### Commands vs. Events
* **Commands**: Explicit requests for an action to be performed by a target service (e.g., `payment.command.charge`). Commands express intent and expect a direct outcome or reply.
* **Events**: Facts stating that an action has already occurred within a service (e.g., `orders.event.created` or `payment.event.failed`). Events are broadcast to notify interested listeners.

### 1.3 Saga Coordination Approaches

There are two primary approaches to coordinating a Saga:

| Approach | Description | Control Model | NATS Messaging Model |
| :--- | :--- | :--- | :--- |
| **Saga Orchestration** | A central Saga Coordinator manages execution, sends commands to participants, and handles compensation if a step fails. | Centralized control, explicit state transitions. | Core NATS Request-Reply or JetStream command streams. |
| **Saga Choreography** | Decoupled services listen to domain events and autonomously execute their local step or compensating action. | Decentralized, event-driven reactive flow. | JetStream persistent streams and event subscriptions. |

---

## 2. How to Implement the Saga Pattern using NATS

NATS provides complementary capabilities for building scalable, resilient Sagas:

* **Core NATS Request-Reply**: Enables fast, synchronous or asynchronous command invocation with built-in response waiting (`nc.Request`), ideal for Orchestrator-to-Participant commands.
* **JetStream Persistent Streams**: Delivers guaranteed event storage, sequence tracking, message deduplication (`Nats-Msg-Id`), and durable pull/push consumers for Choreography flows.
* **JetStream Key-Value Store**: Serves as an ideal lightweight state store for tracking active Saga state, correlation IDs, completed steps, and execution status.

### 2.1 Subject Architecture for Sagas

Enterprise Saga implementations use clear hierarchical subject names separating commands and events across business domains:

```text
<domain>.<message_type>.<action>
```

#### Subject Hierarchy Examples

| Domain / Step | Subject | Type | Purpose |
| :--- | :--- | :--- | :--- |
| Order Service | `orders.command.create` | Command | Request creation of a new pending order. |
| Order Service | `orders.event.created` | Event | Emitted when order record is created. |
| Payment Service | `payment.command.charge` | Command | Request credit card payment authorization. |
| Payment Service | `payment.event.charged` | Event | Emitted on successful payment processing. |
| Payment Service | `payment.event.failed` | Event | Emitted when payment fails (e.g. card declined). |
| Order Service | `orders.command.cancel` | Command (Compensating) | Request order cancellation due to downstream failure. |
| Inventory Service | `inventory.command.reserve` | Command | Request inventory allocation. |
| Inventory Service | `inventory.command.release` | Command (Compensating) | Request release of reserved inventory items. |

---

## 3. Saga Pattern Code Examples

The following examples demonstrate an **E-Commerce Order Fulfillment Saga**:
1. **Order Service** creates a pending order.
2. **Payment Service** attempts to charge payment.
3. If payment fails (e.g. insufficient funds), the system executes a **compensating transaction** to cancel the pending order.

### 3.1 Saga Orchestration Example

In Orchestration, a central `OrderSagaCoordinator` sends commands to participant services via Core NATS Request-Reply. If any command returns a failure or times out, the coordinator sends compensating commands to roll back previous steps.

#### Orchestration Workflow

```text
Coordinator                   Payment Service               Order Service
     |                               |                            |
     |-- 1. Charge Payment --------->|                            |
     |<-- 2. Payment Declined -------|                            |
     |                                                            |
     |-- 3. Trigger Compensation (Cancel Order) ----------------->|
     |<-- 4. Order Canceled Ack ----------------------------------|
```

#### Go

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Request and Response Models
type ChargePaymentReq struct {
	SagaID  string  `json:"saga_id"`
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

type ChargePaymentResp struct {
	Success string `json:"status"` // "SUCCESS" or "DECLINED"
	Reason  string `json:"reason,omitempty"`
}

type CancelOrderReq struct {
	SagaID  string `json:"saga_id"`
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type CancelOrderResp struct {
	Status string `json:"status"` // "CANCELED"
}

// OrderSagaCoordinator executes and coordinates the saga steps
type OrderSagaCoordinator struct {
	nc *nats.Conn
}

func NewOrderSagaCoordinator(nc *nats.Conn) *OrderSagaCoordinator {
	return &OrderSagaCoordinator{nc: nc}
}

// ExecuteSaga executes forward steps and triggers compensation if any step fails
func (c *OrderSagaCoordinator) ExecuteSaga(sagaID, orderID string, amount float64) error {
	log.Printf("[Saga %s] Starting Order Fulfillment Saga for Order: %s", sagaID, orderID)

	// Step 1: Forward Action - Charge Payment via Core NATS Request-Reply
	payReq := ChargePaymentReq{
		SagaID:  sagaID,
		OrderID: orderID,
		Amount:  amount,
	}
	payBytes, _ := json.Marshal(payReq)

	log.Printf("[Saga %s] Step 1: Requesting payment charge...", sagaID)
	respMsg, err := c.nc.Request("payment.command.charge", payBytes, 3*time.Second)
	if err != nil {
		log.Printf("[Saga %s] Step 1 Failed (Network/Timeout): %v", sagaID, err)
		c.compensateCancelOrder(sagaID, orderID, "Payment request timed out")
		return fmt.Errorf("payment command failed: %w", err)
	}

	var payResp ChargePaymentResp
	if err := json.Unmarshal(respMsg.Data, &payResp); err != nil {
		c.compensateCancelOrder(sagaID, orderID, "Invalid payment response payload")
		return fmt.Errorf("unmarshal error: %w", err)
	}

	if payResp.Success != "SUCCESS" {
		log.Printf("[Saga %s] Step 1 Declined: %s. Initiating Compensation Flow...", sagaID, payResp.Reason)
		c.compensateCancelOrder(sagaID, orderID, payResp.Reason)
		return fmt.Errorf("saga failed at step 1: %s", payResp.Reason)
	}

	log.Printf("[Saga %s] Step 1 Succeeded. Order Fulfillment Saga Completed Successfully.", sagaID)
	return nil
}

// compensateCancelOrder sends a compensating command to cancel the pending order
func (c *OrderSagaCoordinator) compensateCancelOrder(sagaID, orderID, reason string) {
	log.Printf("[Saga %s] COMPENSATING: Sending cancel command for Order %s...", sagaID, orderID)

	cancelReq := CancelOrderReq{
		SagaID:  sagaID,
		OrderID: orderID,
		Reason:  reason,
	}
	cancelBytes, _ := json.Marshal(cancelReq)

	// Send compensating command
	respMsg, err := c.nc.Request("orders.command.cancel", cancelBytes, 3*time.Second)
	if err != nil {
		log.Printf("[Saga %s] CRITICAL: Compensation request failed: %v", sagaID, err)
		return
	}

	var cancelResp CancelOrderResp
	_ = json.Unmarshal(respMsg.Data, &cancelResp)
	log.Printf("[Saga %s] COMPENSATED: Order %s status updated to %s", sagaID, orderID, cancelResp.Status)
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Service Responders (Simulated)
	// 1. Payment Service Responder (Simulates a declined payment)
	_, _ = nc.Subscribe("payment.command.charge", func(msg *nats.Msg) {
		resp := ChargePaymentResp{
			Success: "DECLINED",
			Reason:  "Insufficient funds on credit card",
		}
		data, _ := json.Marshal(resp)
		_ = msg.Respond(data)
	})

	// 2. Order Service Responder (Handles compensating cancel order command)
	_, _ = nc.Subscribe("orders.command.cancel", func(msg *nats.Msg) {
		resp := CancelOrderResp{
			Status: "CANCELED",
		}
		data, _ := json.Marshal(resp)
		_ = msg.Respond(data)
	})

	// Run Saga Coordinator
	coordinator := NewOrderSagaCoordinator(nc)
	_ = coordinator.ExecuteSaga("SAGA-1001", "ORD-8842", 250.00)
}
```

#### Java

```java
package io.nats.demo.saga;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.nats.client.Connection;
import io.nats.client.Message;
import io.nats.client.Nats;
import io.nats.client.Subscription;

import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

public class SagaOrchestratorDemo {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    // Data Transfer Objects
    public static class ChargePaymentReq {
        public String sagaId;
        public String orderId;
        public double amount;

        public ChargePaymentReq() {}
        public ChargePaymentReq(String sagaId, String orderId, double amount) {
            this.sagaId = sagaId;
            this.orderId = orderId;
            this.amount = amount;
        }
    }

    public static class ChargePaymentResp {
        public String status; // "SUCCESS" or "DECLINED"
        public String reason;
    }

    public static class CancelOrderReq {
        public String sagaId;
        public String orderId;
        public String reason;

        public CancelOrderReq() {}
        public CancelOrderReq(String sagaId, String orderId, String reason) {
            this.sagaId = sagaId;
            this.orderId = orderId;
            this.reason = reason;
        }
    }

    public static class CancelOrderResp {
        public String status;
    }

    public static class OrderSagaCoordinator {
        private final Connection nc;

        public OrderSagaCoordinator(Connection nc) {
            this.nc = nc;
        }

        public void executeSaga(String sagaId, String orderId, double amount) throws Exception {
            System.out.printf("[Saga %s] Starting Order Fulfillment Saga for Order: %s%n", sagaId, orderId);

            // Step 1: Forward Action - Charge Payment via NATS Request-Reply
            ChargePaymentReq payReq = new ChargePaymentReq(sagaId, orderId, amount);
            byte[] payPayload = MAPPER.writeValueAsBytes(payReq);

            System.out.printf("[Saga %s] Step 1: Requesting payment charge...%n", sagaId);
            CompletableFuture<Message> future = nc.request("payment.command.charge", payPayload);

            Message respMsg;
            try {
                respMsg = future.get(3, TimeUnit.SECONDS);
            } catch (Exception e) {
                System.err.printf("[Saga %s] Step 1 Failed (Timeout/Error): %s%n", sagaId, e.getMessage());
                compensateCancelOrder(sagaId, orderId, "Payment request timed out");
                return;
            }

            ChargePaymentResp payResp = MAPPER.readValue(respMsg.getData(), ChargePaymentResp.class);
            if (!"SUCCESS".equalsIgnoreCase(payResp.status)) {
                System.out.printf("[Saga %s] Step 1 Declined: %s. Initiating Compensation Flow...%n", sagaId, payResp.reason);
                compensateCancelOrder(sagaId, orderId, payResp.reason);
                return;
            }

            System.out.printf("[Saga %s] Step 1 Succeeded. Saga Completed Successfully.%n", sagaId);
        }

        private void compensateCancelOrder(String sagaId, String orderId, String reason) {
            System.out.printf("[Saga %s] COMPENSATING: Sending cancel command for Order %s...%n", sagaId, orderId);
            try {
                CancelOrderReq cancelReq = new CancelOrderReq(sagaId, orderId, reason);
                byte[] cancelPayload = MAPPER.writeValueAsBytes(cancelReq);

                CompletableFuture<Message> future = nc.request("orders.command.cancel", cancelPayload);
                Message respMsg = future.get(3, TimeUnit.SECONDS);

                CancelOrderResp cancelResp = MAPPER.readValue(respMsg.getData(), CancelOrderResp.class);
                System.out.printf("[Saga %s] COMPENSATED: Order %s status updated to %s%n", sagaId, orderId, cancelResp.status);
            } catch (Exception e) {
                System.err.printf("[Saga %s] CRITICAL: Compensation failed: %s%n", sagaId, e.getMessage());
            }
        }
    }

    public static void main(String[] args) throws Exception {
        try (Connection nc = Nats.connect("nats://localhost:4222")) {

            // Mock Service Responders
            // 1. Payment Service Responder (Simulates decline)
            Subscription paySub = nc.createDispatcher(msg -> {
                try {
                    ChargePaymentResp resp = new ChargePaymentResp();
                    resp.status = "DECLINED";
                    resp.reason = "Card limit exceeded";
                    nc.publish(msg.getReplyTo(), MAPPER.writeValueAsBytes(resp));
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }).subscribe("payment.command.charge");

            // 2. Order Service Responder (Handles compensation)
            Subscription cancelSub = nc.createDispatcher(msg -> {
                try {
                    CancelOrderResp resp = new CancelOrderResp();
                    resp.status = "CANCELED";
                    nc.publish(msg.getReplyTo(), MAPPER.writeValueAsBytes(resp));
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }).subscribe("orders.command.cancel");

            // Run Saga Coordinator
            OrderSagaCoordinator coordinator = new OrderSagaCoordinator(nc);
            coordinator.executeSaga("SAGA-1002", "ORD-9910", 450.00);

            Thread.sleep(1000);
        }
    }
}
```

---

### 3.2 Saga Choreography Example

In Choreography, there is no central orchestrator. Services publish domain events to JetStream streams. Subscribed services react to these events by executing local transactions and emitting follow-up events or failure compensation events.

#### Choreography Event Flow

```text
Order Service                  JetStream (EVENTS Stream)            Payment Service
      |                                   |                                |
      |-- 1. Publish orders.event.created->|                                |
      |                                   |-- 2. Consume Event ----------->|
      |                                   |                                |-- 3. Process Charge (Declined)
      |                                   |<-- 4. Publish payment.event.failed
      |<-- 5. Consume Payment Failed Event|                                |
      |-- 6. Cancel Order Local Tx ------|                                |
```

#### Go

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type OrderCreatedEvent struct {
	SagaID  string  `json:"saga_id"`
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

type PaymentFailedEvent struct {
	SagaID  string `json:"saga_id"`
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
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

	// Step 1: Ensure JetStream Stream for Saga Events exists
	cfg := jetstream.StreamConfig{
		Name:     "SAGA_EVENTS",
		Subjects: []string{"orders.event.*", "payment.event.*"},
		Storage:  jetstream.FileStorage,
	}
	_, err = js.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		log.Fatalf("Stream creation failed: %v", err)
	}

	// Step 2: Payment Service Consumer (Reacts to orders.event.created)
	payCons, _ := js.CreateOrUpdateConsumer(ctx, "SAGA_EVENTS", jetstream.ConsumerConfig{
		Durable:       "PaymentServiceConsumer",
		FilterSubject: "orders.event.created",
	})

	go func() {
		cc, _ := payCons.Consume(func(msg jetstream.Msg) {
			var evt OrderCreatedEvent
			_ = json.Unmarshal(msg.Data(), &evt)
			log.Printf("[Payment Service] Received OrderCreatedEvent for Order %s", evt.OrderID)

			// Simulate payment failure
			failEvt := PaymentFailedEvent{
				SagaID:  evt.SagaID,
				OrderID: evt.OrderID,
				Reason:  "Insufficient funds",
			}
			failBytes, _ := json.Marshal(failEvt)

			// Publish PaymentFailedEvent to JetStream
			_, _ = js.Publish(ctx, "payment.event.failed", failBytes)
			_ = msg.Ack()
			log.Printf("[Payment Service] Published PaymentFailedEvent for Order %s", evt.OrderID)
		})
		defer cc.Stop()
		select {}
	}()

	// Step 3: Order Service Compensating Listener (Reacts to payment.event.failed)
	orderCompCons, _ := js.CreateOrUpdateConsumer(ctx, "SAGA_EVENTS", jetstream.ConsumerConfig{
		Durable:       "OrderServiceCompensatingConsumer",
		FilterSubject: "payment.event.failed",
	})

	go func() {
		cc, _ := orderCompCons.Consume(func(msg jetstream.Msg) {
			var evt PaymentFailedEvent
			_ = json.Unmarshal(msg.Data(), &evt)

			// Execute local compensating action: Update Order status to CANCELED
			log.Printf("[Order Service COMPENSATING] Executing rollback for Order %s. Reason: %s", evt.OrderID, evt.Reason)
			_ = msg.Ack()
			log.Printf("[Order Service COMPENSATED] Order %s state successfully set to CANCELED.", evt.OrderID)
		})
		defer cc.Stop()
		select {}
	}()

	// Trigger Saga Choreography by publishing OrderCreatedEvent
	time.Sleep(500 * time.Millisecond)
	orderEvt := OrderCreatedEvent{
		SagaID:  "SAGA-CHOR-2001",
		OrderID: "ORD-7711",
		Amount:  120.00,
	}
	evtBytes, _ := json.Marshal(orderEvt)

	// Set Nats-Msg-Id header for JetStream publish deduplication
	msg := nats.NewMsg("orders.event.created")
	msg.Data = evtBytes
	msg.Header.Set("Nats-Msg-Id", "order-created-ORD-7711")

	_, err = js.PublishMsg(ctx, msg)
	if err != nil {
		log.Printf("Publish error: %v", err)
	}

	time.Sleep(2 * time.Second)
}
```

#### Java

```java
package io.nats.demo.saga;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.nats.client.*;
import io.nats.client.api.ConsumerConfiguration;
import io.nats.client.api.StreamConfiguration;
import io.nats.client.api.StorageType;

import java.nio.charset.StandardCharsets;
import java.time.Duration;

public class SagaChoreographyDemo {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    public static class OrderCreatedEvent {
        public String sagaId;
        public String orderId;
        public double amount;

        public OrderCreatedEvent() {}
        public OrderCreatedEvent(String sagaId, String orderId, double amount) {
            this.sagaId = sagaId;
            this.orderId = orderId;
            this.amount = amount;
        }
    }

    public static class PaymentFailedEvent {
        public String sagaId;
        public String orderId;
        public String reason;

        public PaymentFailedEvent() {}
        public PaymentFailedEvent(String sagaId, String orderId, String reason) {
            this.sagaId = sagaId;
            this.orderId = orderId;
            this.reason = reason;
        }
    }

    public static void main(String[] args) throws Exception {
        try (Connection nc = Nats.connect("nats://localhost:4222")) {
            Management jsm = nc.createManagementContext();
            JetStream js = nc.jetStream();

            // Step 1: Create JetStream Stream for Saga events
            StreamConfiguration streamConfig = StreamConfiguration.builder()
                    .name("SAGA_EVENTS")
                    .subjects("orders.event.*", "payment.event.*")
                    .storageType(StorageType.File)
                    .build();
            jsm.addOrUpdateStream(streamConfig);

            // Step 2: Payment Service Consumer (Reacts to orders.event.created)
            ConsumerConfiguration payConsumerConfig = ConsumerConfiguration.builder()
                    .durable("PaymentServiceConsumer")
                    .filterSubject("orders.event.created")
                    .build();
            jsm.addOrUpdateConsumer("SAGA_EVENTS", payConsumerConfig);

            JetStreamSubscription paySub = js.subscribe("orders.event.created",
                    PushSubscribeOptions.bind("SAGA_EVENTS", "PaymentServiceConsumer"));

            nc.createDispatcher(msg -> {
                try {
                    OrderCreatedEvent evt = MAPPER.readValue(msg.getData(), OrderCreatedEvent.class);
                    System.out.printf("[Payment Service] Received OrderCreatedEvent for Order %s%n", evt.orderId);

                    PaymentFailedEvent failEvt = new PaymentFailedEvent(evt.sagaId, evt.orderId, "Insufficient funds");
                    byte[] failBytes = MAPPER.writeValueAsBytes(failEvt);

                    js.publish("payment.event.failed", failBytes);
                    msg.ack();
                    System.out.printf("[Payment Service] Published PaymentFailedEvent for Order %s%n", evt.orderId);
                } catch (Exception e) {
                    e.printStackTrace();
                }
            });

            // Step 3: Order Service Compensating Listener (Reacts to payment.event.failed)
            ConsumerConfiguration orderCompConfig = ConsumerConfiguration.builder()
                    .durable("OrderServiceCompensatingConsumer")
                    .filterSubject("payment.event.failed")
                    .build();
            jsm.addOrUpdateConsumer("SAGA_EVENTS", orderCompConfig);

            Dispatcher dispatcher = nc.createDispatcher();
            JetStreamSubscription compSub = js.subscribe("payment.event.failed",
                    dispatcher,
                    msg -> {
                        try {
                            PaymentFailedEvent evt = MAPPER.readValue(msg.getData(), PaymentFailedEvent.class);
                            System.out.printf("[Order Service COMPENSATING] Rolling back Order %s. Reason: %s%n", evt.orderId, evt.reason);
                            msg.ack();
                            System.out.printf("[Order Service COMPENSATED] Order %s state set to CANCELED.%n", evt.orderId);
                        } catch (Exception e) {
                            e.printStackTrace();
                        }
                    },
                    false,
                    PushSubscribeOptions.bind("SAGA_EVENTS", "OrderServiceCompensatingConsumer"));

            Thread.sleep(500);

            // Trigger Saga Choreography by publishing OrderCreatedEvent
            OrderCreatedEvent orderEvt = new OrderCreatedEvent("SAGA-CHOR-2002", "ORD-8822", 199.99);
            byte[] payload = MAPPER.writeValueAsBytes(orderEvt);

            Headers headers = new Headers();
            headers.set("Nats-Msg-Id", "order-created-ORD-8822");

            Message msg = NatsMessage.builder()
                    .subject("orders.event.created")
                    .headers(headers)
                    .data(payload)
                    .build();

            js.publish(msg);
            System.out.println("Published initial OrderCreatedEvent to JetStream.");

            Thread.sleep(2000);
        }
    }
}
```

---

## 4. Operational & Enterprise Best Practices

### 4.1 Message Deduplication and Idempotency
Saga participants and compensating handlers **must be idempotent**. Because network timeouts can lead to message redelivery in JetStream, handlers must check if an event or command was already processed. Use `Nats-Msg-Id` headers when publishing events to enforce server-side deduplication within JetStream.

### 4.2 Handling Transient vs. Permanent Failures
* **Transient Failures** (e.g. temporary network partition or database lock timeout): Use NATS JetStream consumer redelivery with backoff rather than immediately triggering compensation.
* **Permanent Business Failures** (e.g. credit card declined, invalid customer ID, insufficient stock): Immediately publish failure events or return error responses to trigger compensating transactions.

### 4.3 Correlation IDs and Observability
Pass standard correlation headers across all Saga steps to enable end-to-end tracing across distributed microservices:

```text
X-Saga-ID: saga-uuid-1001
X-Correlation-ID: corr-uuid-5502
```

---

## 5. Official References

* [NATS Go Client Documentation](https://github.com/nats-io/nats.go)
* [NATS Java Client Documentation](https://github.com/nats-io/nats.java)
* [Synadia NATS Weekly #19](https://www.synadia.com/newsletter/nats-weekly-19)
* [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
