# NATS Subject Design

Subject design is a foundational architectural decision in NATS. Publishers and subscribers encode subject names and patterns into their application logic; changes to established subjects can therefore require coordinated changes across services that use the subject.

Subjects should be designed as stable messaging contracts and should account for routing, access control, and message persistence requirements from the beginning.

## Subject

- A NATS subject is a **case-sensitive, dot-separated string of tokens** that forms a hierarchical namespace.
- Publishers send messages to a **fully specified subject**.
- Subscribers express interest using either a **fully specified subject** or a **wildcard pattern**.
- The subject hierarchy can serve as:
  - **Routing key**
  - **Authorization boundary**
  - **Storage filter** for JetStream

### Enterprise Standard

- Subjects **MUST** use lowercase characters.
- Subject names **MUST** follow a consistent hierarchical structure.
- Subjects **MUST** be treated as stable messaging contracts.
- Subject names **MUST NOT** unnecessarily encode implementation-specific details.

## Token

- Each `.`-separated segment is a token.
- Tokens should use lowercase alphanumeric characters.
- Hyphens (`-`) can be used as word separators.
- Subject hierarchy should generally progress from **broader concepts on the left to narrower concepts on the right**.

Example:

```text
orders.customer.created
```

Here, `orders` represents the broader domain, while `customer` and `created` progressively narrow the meaning.

### Enterprise Standard

- Tokens **MUST** use lowercase characters.
- Tokens **MUST** contain only alphanumeric characters and hyphens.
- Hyphens (`-`) **MUST** be used as word separators.
- Underscores (`_`) **MUST NOT** be used.
- Token ordering **MUST** follow the enterprise subject hierarchy defined in this standard.

## Wildcards

- `*` matches exactly one token at its position.
- `>` matches one or more tokens and **MUST** be the final token in the pattern.
- Wildcards are used for subscriptions and applicable server-side subject mappings.
- Wildcards are not valid publish destinations.

```text
orders.*.created     # matches orders.customer.created
                      # does not match orders.customer.line.created

orders.>             # matches orders.customer.created
                      # matches orders.customer.line.created

orders.*.>           # matches subjects two or more tokens deep under orders
```

Publishers must send messages to fully specified subjects.

### Enterprise Standard

- Publishers **MUST** publish only to fully specified subjects.
- Application consumers **SHOULD** use the narrowest wildcard scope required.
- Broad wildcard subscriptions such as `>` **SHOULD NOT** be used unless there is a justified requirement.

## Subject Constraints

NATS supports hierarchical subjects with multiple tokens. Subject length and depth should be kept reasonable to maintain readability and operational manageability.

### Enterprise Standard
- Anything past 8–10 tokens is usually a sign that you’re encoding data into the subject
- Total subject length **MUST NOT** exceed **256 characters**.
- Total number of tokens **MUST NOT** exceed **16**.
- Subject depth **SHOULD** be limited to meaningful architectural or business boundaries.


## Subject Structure

The enterprise NATS subject namespace follows a consistent hierarchical structure designed to support environment isolation, tenant identification, message classification, and business semantics.

The standard structure is:

```text
<environment>.<tenant>.<category>.<domain>.<activity>.<operation>
```

Example:

```text
prod.order.event.order.processing.completed
```

### Structure

| Token | Purpose | Example |
|---|---|---|
| `environment` | Identifies the application environment | `prod` |
| `tenant` | Identifies the enterprise isolation boundary | `order` |
| `category` | Identifies the message category | `event` / `audit` |
| `domain` | Identifies the business domain or entity concerned | `order` |
| `activity` | Identifies the business activity or capability | `processing` |
| `operation` | Identifies the specific action, outcome, or state | `completed` |

### Enterprise Standard

- The subject **MUST** follow the enterprise hierarchical naming structure.
- `environment`, `tenant`, and `category` **MUST** be present.
- `category` **MUST** be one of the approved enterprise categories:
  - `event`
  - `audit`
- `tenant` **MUST** represent the enterprise-defined isolation boundary established during application onboarding.
- `domain`, `activity`, and `operation` **MUST** represent meaningful business or application semantics when applicable.
- Subject hierarchy **MUST** expose dimensions that consumers are expected to filter on.
- Identifiers such as `order-id`, `transaction-id`, `saga-id`, and `correlation-id` **MUST NOT** be included in subjects. Such identifiers **MUST** be carried in message payloads or headers.
- Application teams **MUST NOT** introduce additional top-level categories without platform approval.
- Platform deployment details such as physical region, primary/DR role, or infrastructure topology **MUST NOT** be exposed in application subjects.

### Examples

**Business Event**

```text
prod.order.event.order.processing.completed
```

**Audit Event**

```text
prod.order.audit.order.processing.completed
```

**Document Processing**

```text
prod.document.event.document.extraction.completed
```

The semantic hierarchy is intentionally limited to dimensions required for **routing, authorization, subscription filtering, and message classification**. Metadata that does not participate in these concerns should remain in the message payload or headers.

## Stream

Subjects define the namespace from which JetStream Streams select messages for persistence, while Consumers define how persisted messages are delivered to applications. Subject hierarchy should therefore expose the filtering boundaries required by the expected persistence and consumption patterns.

### Enterprise Standard

- Streams **MUST** align with meaningful persistence, retention, and operational boundaries.
- Streams **SHOULD** group subjects that share common persistence and retention requirements.
- Consumers **MUST** use the narrowest applicable subject filter for their consumption requirement.
- Subject hierarchy **MUST** expose dimensions required for expected Stream and Consumer filtering.
- Stream and Consumer topology **MUST NOT** drive unnecessary changes to established subject contracts.
- Application teams **MUST NOT** introduce subject levels solely to satisfy a current Stream or Consumer configuration.
- Highly fragmented Streams and Consumers **SHOULD NOT** be created when a common subject hierarchy provides the required isolation and filtering.
- Audit subjects **SHOULD** be separated from business event subjects when their persistence, retention, or operational requirements differ.


## Subject Versioning

Subject versioning provides a contract boundary when a messaging contract changes in a way that cannot safely be introduced to existing consumers.

For example:

```text
prod.order.event.order.created.v1
prod.order.event.order.created.v2
```

The versions allow old and new consumers to coexist and migrate independently.

### When Required

Subject versioning is appropriate for:
- **Breaking contract changes** that existing consumers cannot safely consume.
- **Material semantic changes** where the meaning of an existing event changes.
- **Contract migrations** where old and new consumers must coexist.

### Enterprise Standard

- Backward-compatible payload changes **MUST** remain on the existing subject by default.
- Subject versioning **MUST NOT** be used for metadata, tracing, per-message identifiers, or implementation changes.
- When required, the version **MUST** be the **final subject token**.
- Version format **MUST** be `v<major>` using a positive integer, for example `v1`, `v2`.
- Each version **MUST** represent a distinct messaging contract; a version **MUST NOT** be reused for a different contract.
- Breaking contract changes **MUST NOT** be published to an existing subject without an approved migration strategy.
- Old and new versions **MUST** remain available during the approved migration period and **SHOULD** be retired after migration completes.