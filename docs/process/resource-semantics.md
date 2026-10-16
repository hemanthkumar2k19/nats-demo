# NATS Resource Semantics

NATS resources have distinct architectural responsibilities. Enterprise naming MUST reflect the semantic purpose of each resource rather than implementation or configuration details.

The enterprise resource model is:

```bash
Cluster
   |
   \-- Server

Account
   |
   \-- Stream
         |
         +-- Consumer
         +-- Consumer
         \-- Consumer
```

## Resource Semantic Model

| Resource | Semantic Role | Name Represents |
|---|---|---|
| **Account** | Enterprise / security boundary | Ownership and isolation boundary |
| **Stream** | Persistence / operational boundary | Logical persisted workload |
| **Consumer** | Consumption relationship / delivery state | Logical consumption purpose or role |
| **Cluster** | Infrastructure / deployment boundary | NATS deployment identity |
| **Server** | Runtime / node identity | Individual NATS server within a cluster |

The following sections define the semantic meaning and enterprise naming principles for each resource.

---

## 1. Stream

A **Stream** is a JetStream persistence and operational boundary that stores messages and manages their persisted lifecycle.

A Stream determines **what persisted workload belongs together and how that workload is managed**, including retention, storage, replication, and lifecycle characteristics.

A Stream can contain messages from multiple subjects when those messages share compatible persistence and operational requirements.

```bash
ORDER_EVENTS
|
+-- persisted message type A
+-- persisted message type B
+-- persisted message type C
\-- persisted message type D
```

Consumers then read persisted messages from the Stream according to their own consumption requirements.

```bash
Stream
   |
   +-- Consumer A
   +-- Consumer B
   \-- Consumer C
```

### Design Principle

> **Streams define persistence and operational boundaries, while Consumers define independent consumption relationships and state.**

A new application workload or Consumer **does not automatically require a new Stream**. An existing Stream SHOULD be reused when its persistence, retention, storage, lifecycle, and operational requirements are compatible.

For detailed Stream boundary decisions, see **Stream Design**.

### Enterprise Standard for Naming

Stream names MUST identify the **logical persisted workload or persistence boundary**, rather than an individual Consumer, deployment, or configuration detail.

| Principle | Standard |
|---|---|
| **Semantic identity** | Name the logical persisted workload represented by the Stream. |
| **Application independence** | Do not use application/service names unless they represent the intended persistence boundary. |
| **Consumer independence** | Do not include Consumer names or consumption behavior. |
| **Configuration independence** | Do not encode retention, storage, replica count, or other Stream configuration. |
| **Environment independence** | Do not encode environment when it is already represented by the deployment or Account boundary. |
| **Stability** | Prefer names that remain valid when applications, deployments, or implementation details change. |

### Naming Format

```bash
UPPER_SNAKE_CASE
```

Examples:

```bash
ORDER_EVENTS
CUSTOMER_EVENTS
INVENTORY_EVENTS
PAYMENT_EVENTS
```

Avoid:

```bash
ORDERS_CREATED_STREAM
PAYMENT_SERVICE_PROD_STREAM
ORDER_EVENTS_RETENTION_7D
DEV_ORDER_EVENTS
ORDER_CONSUMER_STREAM
```

### Core Principle

> **A Stream name identifies what persisted workload the Stream represents-not how the workload happens to be implemented.**

---

## 2. Consumer

A **Consumer** represents a **logical consumption relationship with a Stream** and maintains the state and configuration required for message delivery.

Multiple Consumers can independently consume messages from the same Stream.

```bash
ORDER_EVENTS
|
+-- fulfillment
+-- analytics
\-- audit-replay
```

A Consumer's identity is distinct from its delivery configuration. Delivery mode, acknowledgement policy, redelivery policy, replay behavior, and similar settings describe **how the Consumer operates**, not what logical Consumer it represents.

### Design Principle

> **Consumer identity represents the logical consumption purpose or role; Consumer configuration defines how that consumption is performed.**

A change to Consumer configuration does not inherently require a new Consumer. A separate Consumer SHOULD be created when an independent consumption state, lifecycle, or logical consumption purpose is required.

### Enterprise Standard for Naming

Consumer names MUST identify the **logical consumption purpose or role**.

| Principle | Standard |
|---|---|
| **Semantic identity** | Identify the logical consumption purpose or role. |
| **Stability** | Names SHOULD remain stable across application deployments and configuration changes. |
| **Instance independence** | Do not identify application instances, pods, hosts, or other ephemeral infrastructure. |
| **Delivery independence** | Do not encode Pull/Push delivery mode. |
| **Policy independence** | Do not encode ACK policy, `AckWait`, `MaxDeliver`, replay policy, or similar configuration. |
| **Version independence** | Do not encode application or deployment version. |
| **Parent-resource independence** | Do not unnecessarily repeat Account or Stream identity. |

### Naming Format

Consumer names MUST use:

```bash
lowercase-alphanumeric-with-hyphens
```

Examples:

```bash
fulfillment
analytics
notification
audit-replay
```

Avoid:

```bash
fulfillment-pull
fulfillment-explicit-ack
fulfillment-30s-5-retries
fulfillment-v2
fulfillment-pod-7f8d
```

### Core Principle

> **A Consumer name identifies who or what logically consumes a Stream-not how that consumption is configured or where it happens to run.**

---

## 3. Account

A NATS Account is an isolated messaging and security boundary within NATS. It provides the boundary within which an application and its associated NATS resources operate.
For the enterprise platform, one application identity maps to one NATS Account. This mapping remains consistent across environments, allowing the same application identity to be promoted across DEV, STAGE, and PROD without changing its NATS identity.

The Account represents the application's NATS boundary; individual services within the application operate within the same Account and are granted access through NATS permissions.

### Design Principle

> **An Account identifies the logical enterprise boundary to which NATS resources and permissions belong.**

An application SHOULD have a single Account. Individual services within an application SHOULD NOT require separate Accounts unless they represent an independently managed application boundary with distinct ownership or isolation requirements.

### Enterprise Standard for Naming

Account names MUST represent the application identity.

| Principle | Standard |
|---|---|
| **Application identity** | The Account name MUST correspond to the canonical application identity. |
| **Environment independence** | Account names MUST remain the same across environments. |
| **Infrastructure independence** | Do not include Cluster, Server, region, or deployment information. |
| **Service independence** | Do not create Account names based on individual microservices within an application. |
| **Configuration independence** | Do not encode permissions, limits, quotas, or other Account configuration. |
| **Stability** | Account names SHOULD remain stable throughout the application's lifecycle. |
| **Uniqueness** | Application identities MUST map to a unique Account within the applicable NATS namespace. |

### Naming Format

```bash
lowercase-alphanumeric-with-hyphens
```

Examples:

```bash
trade
payment
inventory
document-processing
customer-profile
```

Avoid:

```bash
trade-prod
trade-dev
trade-hyd
trade-cluster-01
trade-extraction
trade-service
trade-account
```

### Core Principle

> **The Account name is the application's stable NATS identity. It MUST remain independent of environment, infrastructure, services, and configuration.**

---

## 4. Cluster

A **NATS Cluster** is a logical group of NATS servers that cooperate as a single NATS deployment.

For enterprise infrastructure, the NATS Cluster is deployed within an existing organizational infrastructure environment. Its identity MUST inherit the organization's established infrastructure naming convention rather than introducing a separate NATS-specific hierarchy.

The organizational naming convention is:

```bash
<org>-<cloud>-<region>-<environment>-<platform>-<resource>
```

For example:

```bash
acme-gcp-mum-dev-gke-cluster
```

The NATS Cluster extends this established identity with the NATS workload designation:

```bash
<org>-<cloud>-<region>-<environment>-<platform>-<resource>-nats
```

Example:

```bash
acme-gcp-mum-dev-gke-cluster-nats
```

### Design Principle

> **A NATS Cluster name represents the NATS deployment within an organizational infrastructure environment and MUST inherit the organization's established infrastructure identity.**

The NATS Cluster name MUST remain aligned with the underlying infrastructure context, including region and environment.

### Enterprise Standard for Naming

| Principle | Standard |
|---|---|
| **Organizational standard** | NATS Cluster names MUST adopt the organization's established infrastructure naming convention. |
| **Infrastructure context** | Cloud provider, region, environment, and platform MUST follow the existing organizational naming order. |
| **NATS identity** | The Cluster name MUST include `nats` to identify the NATS deployment. |
| **Environment** | Environment MUST be represented according to the organizational convention. |
| **Region** | Region MUST be represented according to the organizational convention. |
| **Stability** | Cluster names SHOULD remain stable for the lifetime of the NATS deployment. |
| **Configuration independence** | Do not encode NATS configuration such as replica count, JetStream settings, or retention policies. |

### Naming Format

```bash
<org>-<cloud>-<region>-<environment>-<platform>-<resource>-nats
```

Example:

```bash
acme-gcp-mum-dev-gke-cluster-nats
```

Avoid:

```bash
nats-dev-mum
dev-nats-cluster
acme-nats-prod
acme-gcp-mum-dev-nats-3replicas
```

### Core Principle

> **NATS Cluster naming extends the existing organizational infrastructure identity; it does not replace or redefine it.**

---

## 5. Server

A **NATS Server** is an individual NATS runtime node participating in a NATS Cluster.

The Server inherits its infrastructure and deployment context from the parent NATS Cluster. Its name therefore only needs to provide a stable identifier for the individual server within that Cluster.

For example:

```bash
acme-gcp-mum-dev-gke-cluster-nats
|
+-- acme-gcp-mum-dev-gke-cluster-nats-01
+-- acme-gcp-mum-dev-gke-cluster-nats-02
\-- acme-gcp-mum-dev-gke-cluster-nats-03
```

### Design Principle

> **A NATS Server name identifies an individual NATS runtime node within a NATS Cluster and MUST inherit its identity from the parent Cluster.**

### Enterprise Standard for Naming

| Principle | Standard |
|---|---|
| **Cluster inheritance** | Server names MUST be derived from the parent NATS Cluster name. |
| **Stable identity** | Server identifiers SHOULD remain stable for the lifetime of the server's NATS membership. |
| **Unique identity** | Each Server MUST have a unique identifier within its NATS Cluster. |
| **Sequential identity** | Server identifiers SHOULD use a deterministic numeric sequence such as `01`, `02`, `03`. |
| **Infrastructure inheritance** | Do not repeat region, environment, cloud, or platform information independently. |
| **Runtime independence** | Do not use ephemeral Pod names, generated container IDs, IP addresses, or hostnames as the NATS Server identity. |
| **Configuration independence** | Do not encode JetStream configuration, replica count, or other server configuration. |

### Naming Format

```bash
<nats-cluster-name>-<server-id>
```

Example:

```bash
acme-gcp-mum-dev-gke-cluster-nats-01
acme-gcp-mum-dev-gke-cluster-nats-02
acme-gcp-mum-dev-gke-cluster-nats-03
```

### Core Principle

> **A Server name identifies the NATS runtime node; all broader infrastructure identity is inherited from the parent NATS Cluster.**

---

## 6. Cross-Resource Naming Principles

These principles apply to the naming of NATS resources.

### 6.1 Name the Resource's Semantic Identity

A resource name SHOULD describe **what the resource represents**, not its current implementation.

```bash
ORDER_EVENTS
```

rather than:

```bash
ORDER_EVENTS_7D_3REPLICAS
```

### 6.2 Separate Identity from Configuration

Resource names MUST NOT encode configuration that can change independently of the resource's identity.

Examples include:

```bash
retention
storage type
replica count
ACK policy
delivery mode
timeouts
retry limits
```

Configuration belongs to the resource definition, not its name.

### 6.3 Avoid Runtime Identity

Names MUST NOT depend on ephemeral runtime characteristics such as:

```bash
pod
hostname
IP address
instance number
deployment version
```

unless the resource itself is explicitly intended to represent that runtime entity.

### 6.4 Avoid Unnecessary Duplication

A resource SHOULD NOT repeat semantic information already represented by its parent or surrounding NATS resource hierarchy.

For example:

```bash
Account:  order
Stream:   ORDER_EVENTS
Consumer: fulfillment
```

is preferable to:

```bash
Account:  order
Stream:   ORDER_SERVICE_ORDER_EVENTS
Consumer: ORDER_SERVICE_ORDER_EVENTS_FULFILLMENT_PULL
```

### 6.5 Preserve Resource Independence

Changing one resource's configuration or lifecycle SHOULD NOT unnecessarily require renaming another resource.

The resulting model is:

```bash
Account
   |
   v
Ownership / Isolation

Stream
   |
   v
Persistence / Operational Boundary

Consumer
   |
   v
Consumption / Delivery State

Cluster
   |
   v
Infrastructure / Deployment Boundary

Server
   |
   v
Runtime Node
```

### Core Principle

> **Each NATS resource MUST be named according to the semantic responsibility it represents. Resource names MUST NOT become containers for configuration, runtime state, or information that belongs to another resource.**