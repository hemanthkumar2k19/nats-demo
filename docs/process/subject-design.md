# NATS Subject Design

## 1. Overview

Subject design is a foundational architectural decision in NATS. Publishers and subscribers encode subject names and patterns into their application logic; changes to established subjects can therefore require coordinated changes across services that use them.

Subjects should be designed as stable messaging contracts that account for routing, access control, subscription filtering, and message persistence requirements from the beginning.

A NATS subject is a **case-sensitive, dot-separated string of tokens** that forms a hierarchical namespace.

- Publishers send messages to a fully specified subject.
- Subscribers express interest using either a fully specified subject or a wildcard pattern.
- Subject hierarchy can serve as a routing key, support authorization boundaries, and provide filtering dimensions for JetStream Streams and Consumers.

This document defines the enterprise standards and recommended patterns for designing NATS subjects.

## 2. Scope

This document covers:

- Enterprise subject namespace and naming conventions.
- Token format, ordering, and subject length and depth constraints.
- Wildcard usage for subscriptions.
- Subject hierarchy requirements for routing, authorization, and JetStream filtering.
- Subject stability and versioning.
- Recommended design patterns and anti-patterns.

Resource registration, approval, provisioning, ownership governance, and retirement processes are covered by the NATS Resource Lifecycle Management document. Stream and Consumer configuration requirements are defined in their respective design documents.

## 3. Design Principles

Subject designs must follow these principles:

1. **Stable messaging contracts:** Subject names should remain stable as application implementations evolve.
2. **Business-oriented semantics:** Tokens should describe meaningful business or application concepts rather than implementation details.
3. **Consistent hierarchy:** Subjects must follow the approved enterprise namespace structure.
4. **Filterable dimensions:** The hierarchy must expose dimensions that subscribers and persistence configurations are expected to filter on.
5. **Minimal complexity:** Subjects should not encode unnecessary metadata, identifiers, or excessive levels of hierarchy.
6. **Separation of concerns:** Message-specific data belongs in payloads or headers; subjects represent stable routing and classification dimensions.
7. **Persistence-aware design:** Subject hierarchy should support expected Stream and Consumer filtering patterns without coupling application contracts to internal resource configurations.

## 4. Naming Conventions

### 4.1 Enterprise Subject Structure

The enterprise subject namespace follows a consistent hierarchy to support environment identification, Account isolation, message classification, and business semantics.

The standard structure is:

```text
<environment>.<account>.<category>.<resource>.<operation>[.v<major>]
```

The version token is **optional** and must be included as the final token when subject versioning is required.

Example without versioning:

```text
prod.order.event.order-processing.completed
```

Example with versioning:

```text
prod.order.event.order.created.v2
```

| Token | Purpose | Example |
|---|---|---|
| `environment` | Identifies the application environment | `prod` |
| `account` | Identifies the NATS Account associated with the application | `order` |
| `category` | Identifies the approved message category | `event`, `audit` |
| `resource` | Identifies the relevant business domain, entity, or application resource | `order`, `order-processing` |
| `operation` | Identifies the action, outcome, or state | `created`, `completed` |
| `v<major>` | Optional contract version, included only when required | `v1`, `v2` |

The `resource` and `operation` tokens must communicate meaningful application semantics. The standard structure must not be extended with additional tokens unless the enterprise standard is formally revised.

### 4.2 Token Conventions

Each dot-separated segment is a token. The hierarchy progresses from broader classification on the left to more specific meaning on the right.

**Enterprise standards:**

- Subject names **MUST** use lowercase characters.
- Tokens **MUST** contain only lowercase alphanumeric characters and hyphens (`-`).
- Hyphens **MUST** be used as word separators.
- Underscores (`_`) **MUST NOT** be used.
- Token ordering **MUST** follow the approved enterprise structure.
- Subject names **MUST NOT** unnecessarily encode implementation-specific details.

| Compliant | Non-compliant | Reason |
|---|---|---|
| `order-processing` | `order_processing` | Uses an underscore instead of a hyphen. |
| `customer-id` | `CUSTOMER-ID` | Uses uppercase characters. |
| `prod.order.event.order-processing.completed` | `prod.order.event.completed.order-processing` | Does not follow the required token order. |
| `prod.order.event.order-processing.completed` | `prod.order.event.sql-postgres.insert` | Encodes database implementation details instead of business semantics. |

### 4.3 Subject Length and Depth

Subject length and depth must remain manageable to preserve readability and avoid encoding unnecessary data into the namespace.

**Enterprise standards:**

- Total subject length **MUST NOT** exceed 256 characters.
- Total subject depth **MUST NOT** exceed 16 tokens.
- Subject depth **SHOULD** generally remain within 8–10 tokens or fewer.
- Each token **MUST** represent a meaningful classification or business dimension.
- Physical infrastructure details, such as node identifiers and rack locations, **MUST NOT** be included in application subjects.

These are **enterprise-defined limits**, not native NATS subject length or depth limits. The optional version token counts toward the total token depth.

## 5. Design Standards

### 5.1 Required Namespace Dimensions

- Subjects **MUST** follow the approved enterprise structure: `environment.account.category.resource.operation`, with an optional final version token.
- `environment`, `account`, and `category` **MUST** be present.
- `category` **MUST** be one of the approved enterprise categories:
  - `event`
  - `audit`
- `account` **MUST** identify the enterprise-defined NATS Account associated with the application.
- `resource` and `operation` **MUST** express meaningful business or application semantics.
- Subject hierarchy **MUST** expose dimensions that consumers are expected to filter on.
- Application teams **MUST NOT** introduce additional categories without Platform approval.
- Application subjects **MUST NOT** expose platform deployment details such as physical region, primary/DR role, or infrastructure topology.

### 5.2 Identifiers and Metadata

Unique identifiers and per-message metadata must not be embedded in subject names.

- Identifiers such as `order-id`, `transaction-id`, `saga-id`, and `correlation-id` **MUST NOT** be included as per-message subject tokens.
- Such identifiers **MUST** be carried in message payloads or headers.
- Subject tokens **MUST NOT** encode arbitrary runtime values or data that does not form part of the stable routing and filtering contract.

| Compliant | Non-compliant | Reason |
|---|---|---|
| `prod.order.event.order.completed` | `prod.order.event.order.completed.ord-12345` | Includes a per-message identifier. |
| `prod.order.event.order.completed` | `prod.order.telemetry.order.completed` | Uses an unapproved category. |
| `prod.order.event.order.completed` | `order.event.order.completed` | Omits required namespace dimensions. |
| `prod.order.event.order.completed` | `prod.us-east-1.event.order.completed` | Adds a physical region token and omits the required Account token. |

### 5.3 Wildcard Usage

NATS supports wildcards in subscription patterns and applicable server-side subject mappings.

- `*` matches exactly one token.
- `>` matches one or more tokens and **MUST** be the final token in the pattern.
- Wildcards **MUST NOT** be used as publish destinations.
- Publishers **MUST** publish to fully specified subjects.
- Application subscribers **SHOULD** use the narrowest wildcard scope required.
- Broad wildcard subscriptions such as `>` **SHOULD NOT** be used unless there is a justified requirement.

Examples:

```text
orders.*.created
```

Matches `orders.customer.created`, but not `orders.customer.line.created`.

```text
orders.>
```

Matches `orders.customer.created` and `orders.customer.line.created`.

```text
orders.*.>
```

Matches subjects with at least two tokens following `orders`, such as `orders.customer.created`.

| Compliant | Non-compliant | Reason |
|---|---|---|
| `nc.Publish("prod.order.event.order.completed", payload)` | `nc.Publish("prod.order.event.order.*", payload)` | Publishers must use fully specified subjects. |
| `nc.Subscribe("prod.order.event.order.*")` | `nc.Subscribe(">")` | The global wildcard is unnecessarily broad unless explicitly justified. |
| `nc.Subscribe("prod.order.event.order.completed")` | `nc.Subscribe("prod.order.>")` | The subscription is broader than the single-operation requirement. |

### 5.4 Subject and JetStream Alignment

Subject design influences how Streams capture messages and how Consumers filter them. Subject hierarchy must support expected persistence and consumption patterns without coupling application contracts to internal Stream or Consumer configuration.

**Enterprise standards:**

- Subject hierarchy **MUST** expose dimensions required for expected Stream and Consumer filtering.
- Stream and Consumer topology **MUST NOT** drive unnecessary changes to established subject contracts.
- Application teams **MUST NOT** introduce subject levels solely to satisfy a current Stream or Consumer configuration.
- Streams **SHOULD** group subjects that share compatible persistence, retention, and operational requirements.
- Excessive Stream and Consumer fragmentation **SHOULD** be avoided when a shared subject hierarchy meets the required filtering and isolation needs.
- Audit subjects **SHOULD** be separated from business event subjects when their persistence, retention, access, or operational requirements differ.
- Consumer filters **MUST** be compatible with the subject scope captured by the associated Stream.

| Recommended | Avoid | Reason |
|---|---|---|
| Separate Streams for operational events and audit events when their retention requirements differ | Combining short-retention events and long-retention audit records in one Stream | Avoids conflicting retention requirements. |
| Use subject filters to select required messages | Creating excessive Streams for every individual operation | Avoids unnecessary resource fragmentation. |
| Keep `prod.order.event.order.completed` stable | Renaming it to include an internal Stream name | Prevents application contracts from depending on internal Stream configuration. |

Detailed Stream separation, retention, and Consumer configuration policies are defined in the Stream Design and Consumer Design documents.

### 5.5 Subject Versioning

Subject versioning provides a contract boundary when a messaging contract changes in a way that cannot safely be introduced to existing consumers.

Examples:

```text
prod.order.event.order.created.v1
prod.order.event.order.created.v2
```

Versioning allows old and new consumers to coexist and migrate independently.

**When to version:**

- Breaking contract changes that existing consumers cannot safely consume.
- Material semantic changes where the meaning of an existing event changes.
- Contract migrations where old and new consumers must coexist.

**Enterprise standards:**

- Backward-compatible payload changes **MUST** remain on the existing subject by default.
- Subject versioning **MUST NOT** be used for metadata, tracing, per-message identifiers, or implementation changes.
- When required, the version **MUST** be the final subject token.
- The version format **MUST** be `v<major>`, using a positive integer such as `v1` or `v2`.
- Each version **MUST** represent a distinct messaging contract; a version **MUST NOT** be reused for a different contract.
- Breaking contract changes **MUST NOT** be published to an existing subject without an approved migration strategy.
- Old and new versions **MUST** remain available during the approved migration period and **SHOULD** be retired after migration completes.

| Compliant | Non-compliant | Reason |
|---|---|---|
| `prod.order.event.order.created.v2` | `prod.order.event.order.created.v1.2.3` | Uses semantic versioning rather than the required major-only format. |
| `prod.order.event.order.created.v2` | `prod.order.v2.event.order.created` | Places the version in the middle of the subject. |
| Keep the existing subject when adding a backward-compatible optional payload field | Create a new subject version for every optional payload field | Creates unnecessary contract fragmentation. |

## 6. Design Patterns

### 6.1 Business Events

Use the `event` category for messages representing business or application events.

```text
prod.order.event.order.created
prod.order.event.order-processing.completed
```

### 6.2 Audit Events

Use the `audit` category for audit records. Audit subjects should remain distinguishable from business events, particularly when their persistence, retention, access, or operational requirements differ.

```text
prod.order.audit.order.updated
```

Audit logging is distinct from telemetry and must not be represented using an unapproved `telemetry` category.

### 6.3 Other Application Resources

Apply the same namespace structure to other application domains.

```text
prod.order.event.document-extraction.completed
```

The resource and operation tokens should reflect the actual business or application semantics.

### 6.4 Wildcard Subscriptions

Use wildcard subscriptions when an application needs to consume a group of related subjects.

For example:

```text
prod.order.event.order.*
```

This matches subjects with exactly one token after `order`. Use `>` only when a multi-level subscription is required and justified.

## 7. Anti-patterns

| Anti-pattern | Example | Why to avoid it |
|---|---|---|
| Encoding unique identifiers | `prod.order.event.order.created.ord-12345` | Embeds per-message data in the namespace and creates unnecessary subject variants. |
| Encoding implementation details | `prod.order.event.sql-postgres.insert` | Couples the messaging contract to a specific implementation. |
| Exposing infrastructure topology | `prod.us-east-1.event.order.created` | Couples application contracts to deployment topology and omits the required Account token. |
| Inconsistent casing or separators | `prod.order.event.orderCreated` | Violates enterprise naming conventions. |
| Unjustifiably broad subscriptions | `>` | Subscribes to an unnecessarily broad message scope. |
| Designing subjects around Stream names | `prod.order.event.stream-a.order.created` | Couples the subject contract to internal persistence configuration. |
| Excessive hierarchy | `prod.order.event.order.step1.step2.step3.step4.step5.step6.step7` | Adds complexity without meaningful routing or filtering value. |
| Versioning every payload change | Adding `.v2` for a backward-compatible optional field | Creates unnecessary contract fragmentation. |

## 8. Examples

| Use case | Subject | Purpose |
|---|---|---|
| Business event | `prod.order.event.order.created` | An order has been created. |
| Business event | `prod.order.event.order-processing.completed` | Order processing has completed. |
| Audit event | `prod.order.audit.order.updated` | An order-related audit event. |
| Document processing | `prod.order.event.document-extraction.completed` | Document extraction has completed. |
| Versioned contract | `prod.order.event.order.created.v2` | A new major version of the order-created contract. |

All examples must comply with the approved namespace structure and token rules. Resource and operation semantics should be selected according to the application domain.

## 9. References

- [Synadia — Designing NATS Subject Hierarchies](https://www.synadia.com/blog/designing-nats-subject-hierarchies)
- [NATS Documentation — Core NATS Deep Dive](https://docs.nats.io/learn/core-nats/)
- [NATS Documentation — Authorization](https://docs.nats.io/learn/security/authorization)
- [NATS Documentation — JetStream Deep Dive](https://docs.nats.io/learn/jetstream/)
- [NATS Documentation — JetStream Reference](https://docs.nats.io/reference/2.12/jetstream)