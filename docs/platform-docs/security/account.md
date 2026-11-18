# NATS Accounts

- Core Principle - A NATS Account is the primary multi-tenancy and security boundary in NATS.
- Namespace Isolation - Subjects, streams, and consumers are completely isolated within an Account.
- Identity Association - Every NATS User belongs to exactly one NATS Account.

---

## 1. Entity Hierarchy

Distinguishes the enterprise application concept from native NATS security primitives:

| Entity | Domain | Responsibility | Mapping |
|---|---|---|---|
| Client | Enterprise Application | Logical service (e.g. `payments-api`) owned by application teams. | 1:1 or 1:N with NATS Users |
| NATS User | NATS Security Runtime | Authenticated credential and subject permission holder. | Exactly 1 NATS Account |
| NATS Account | NATS Security Container | Multi-tenant namespace, resource quotas, and isolation boundary. | 1:N NATS Users |

```text
Client (payments-api) -> NATS User (payments-api-user) -> NATS Account (PAYMENTS)
```

---

## 2. Namespace & Resource Isolation

Accounts enforce complete zero-trust isolation by default:

- Subject Isolation - Subjects in Account A cannot be published or subscribed to by Account B. Identical subject names (e.g. `orders.created`) in different accounts represent disjoint namespaces without collisions.
- JetStream Isolation - Streams, consumers, and storage allocations belong strictly to their parent Account. A stream cannot capture messages from another Account by subject matching.
- User Uniqueness - In configuration-based multi-account setups, username strings must be globally unique across all accounts.

---

## 3. Cross-Account Communication (Exports & Imports)

Cross-account data exchange requires explicit, bidirectional contract configuration:

### Stream Sharing (Pub/Sub Events)
One-way event streaming across account boundaries. The exporting account grants access; the importing account binds to a local subject.

- Stream Export - Exposes selected subjects (e.g. `payments.completed`) to specific accounts or publicly.
- Stream Import - Subscribes to the exported stream, mapping it into the importing account's local subject hierarchy.

### Service Sharing (Request-Reply RPC)
Point-to-point request-reply across account boundaries.

- Service Export - Exposes an RPC endpoint (e.g. `payments.process`) with optional response permissions.
- Service Import - Allows the importing account to send requests to the exported service and receive replies.

---

## 4. Multi-Account Configuration Pattern

Declarative configuration defining accounts, JetStream limits, users, and cross-account exports/imports:

```conf
accounts {
  PAYMENTS {
    jetstream: enabled
    users: [
      { user: "payments-user", password: "pwd" }
    ]
    exports [
      { stream: "payments.completed" }
      { service: "payments.charge" }
    ]
  }

  AUDIT {
    jetstream: enabled
    users: [
      { user: "audit-user", password: "pwd" }
    ]
    imports [
      { stream: { account: "PAYMENTS", subject: "payments.completed" } }
    ]
  }
}
```

---

## 5. Architectural Standards

- Boundary Strategy - Assign one NATS Account per tenant or application.
- JetStream Quotas - Enable JetStream explicitly per account; configure account-level byte/stream limits to prevent noisy-neighbor exhaustion.
- Explicit Contracts - Prohibit wildcard exports; require approved service contracts for all cross-account communication.
- Identity Administration - Platform/SRE team provisions Accounts and JetStream allocations; application teams define service consumers and subjects.
