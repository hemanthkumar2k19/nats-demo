# NATS Security Overview

NATS security is organized around three core capabilities: Authentication (AuthN), Authorization (AuthZ), and Encryption.

```text
NATS Security Model
├── 1. Authentication (AuthN)
│   ├── Token
│   ├── Username / Password
│   ├── TLS / mTLS
│   ├── NKey
│   ├── JWT-Based Authentication
│   └── Auth Callout
│
├── 2. Authorization (AuthZ)
│   ├── Subject Permissions
│   ├── Account Isolation
│   ├── Imports / Exports
│   ├── Connection & Resource Limits
│   └── JetStream API Permissions
│
└── 3. Encryption
    ├── Data in Transit (TLS / mTLS)
    └── Data at Rest (JetStream Storage Encryption)
```

---

## 1. Authentication (AuthN)

### Significance
Authentication establishes and verifies the identity of connecting entities (clients, backend services, server nodes, or leaf nodes) before allowing them to join the NATS event mesh. Reliable authentication ensures that only known, trusted actors can establish connection sockets with NATS servers.

### NATS Offerings

NATS provides multiple authentication methods to support both centralized configuration and decentralized trust models.

#### Authentication Methods
- **Token Authentication:** Clients present a single shared secret token or user-specific static token during initial connection.
- **Username and Password Authentication:** Clients provide username and password credentials. NATS servers can store passwords as plain text or bcrypt hashes within server configuration files.
- **TLS Client Certificates (mTLS Authentication):** Uses TLS X.509 client certificates to identify clients cryptographically, mapping certificate Subject Common Names (CN) or Subject Alternative Names (SAN) to user identities.
- **NKey Authentication:** Uses Ed25519 digital signature key pairs (NKeys). Clients authenticate by signing a challenge nonce issued by the server during connection setup, proving identity without transmitting secrets over the wire.
- **JWT-Based Authentication:** Operator, Account, and User identities form a hierarchical trust model using JSON Web Tokens (JWTs) signed by NKeys. NATS supports a decentralized, cryptographically verifiable authentication model.
- **External Authentication Callout (Auth Callout):** NATS delegates connection authentication and permission assignment to a dedicated external microservice via NATS Request/Reply, enabling integration with external IAM systems such as OAuth2, OIDC, LDAP, or custom authentication providers.

---

## 2. Authorization (AuthZ)

### Significance
Authorization defines access control policies, determining exactly what actions an authenticated identity is permitted to execute. In a messaging architecture, authorization governs which subjects can be published or subscribed to, which services can be invoked, which JetStream streams can be accessed, and how data can be shared across tenant boundaries.

### NATS Offerings

#### Subject-Based Permissions
NATS enforces granular, subject-level access control rules for publish and subscribe operations.

- **Publish and Subscribe Permissions:** Per-user or per-account rules specifying exact subjects or wildcard patterns (`*` for single token, `>` for multi-token tail matching) that a client may publish to or subscribe from. Allow and deny rules can be combined to define the permitted subject space.
- **Dynamic Subject Variables:** Permissions can use NATS-supported identity variables (such as `{{username}}`, `{{account}}`, or client NKey public key) to create dynamic subject-based access policies.

#### Account Isolation
NATS natively supports multi-tenant logical accounts within a single server cluster. Each account represents an independent tenant boundary with isolated subject namespaces, message flows, and JetStream resources.

#### Controlled Sharing (Imports and Exports)
Accounts are completely isolated by default. Data sharing across account boundaries requires explicit opt-in configuration.

- **Service Exports and Imports:** An account can export a service endpoint subject to allow other accounts to send requests and receive responses without exposing internal account subscriptions.
- **Stream Exports and Imports:** An account can export a pub/sub event stream to allow designated consumer accounts to subscribe to event streams across tenant boundaries.

#### Connection and Resource Limits
NATS allows administrators to scope operational resource consumption per user or per account to maintain cluster stability and prevent noisy-neighbor impact.

- **Resource Scoping:** Configurable limits for maximum message payload size, maximum pending bytes, maximum pending messages, maximum connections, and maximum subscriptions.

#### JetStream API Permissions
JetStream capabilities are exposed via system NATS subjects (`$JS.API.>`).

- **API Subject Access Control:** Fine-grained permissions can restrict users to specific JetStream endpoints (e.g. allowing publish to a JetStream subject while denying administrative API access like `$JS.API.STREAM.DELETE.>`), enforcing separation between application data producers/consumers and cluster operators.

---

## 3. Encryption

### Significance
TLS protects data in transit; JetStream storage encryption protects persisted data at rest. Encryption safeguards data confidentiality and integrity against network eavesdropping, tampering, and unauthorized storage access.

### NATS Offerings

#### Encryption in Transit (Data in Motion)
NATS secures network communication channels across all topology connections.

- **Transport Layer Security (TLS):** NATS supports TLS encryption for client, cluster, gateway, and leaf-node connections.
- **Mutual TLS (mTLS):** TLS can additionally authenticate both sides using X.509 certificates, securing transport encryption alongside cryptographic peer verification.

#### Encryption at Rest (Data at Rest)
NATS JetStream protects persisted data on disk against unauthorized physical or filesystem access.

- **JetStream Storage Encryption:** JetStream supports encryption of persisted stream data, Key-Value (KV) stores, and Object Stores at rest.