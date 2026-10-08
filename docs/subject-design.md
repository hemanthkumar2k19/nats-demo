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

| Good Example | Bad Example | Reason / Rule Violated |
| :--- | :--- | :--- |
| `prod.order.event.order.created` | `PROD.ORDER.EVENT.ORDER.CREATED` | Bad example uses uppercase letters. Subjects MUST be strictly lowercase. |
| `prod.order.event.order.created` | `prod.order.event.orderCreated` | Bad example uses camelCase. Subject tokens MUST follow dot-separated hierarchy. |
| `prod.order.event.order.created` | `prod.order.event.sql.postgres.insert` | Bad example encodes database implementation details rather than business domain semantics. |


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

| Good Example | Bad Example | Reason / Rule Violated |
| :--- | :--- | :--- |
| `order-processing` | `order_processing` | Bad example uses an underscore (`_`). Word separators MUST be hyphens (`-`). |
| `customer-id` | `CUSTOMER-ID` | Bad example uses uppercase letters. Tokens MUST contain lowercase alphanumeric characters only. |
| `prod.order.event.order.created` | `created.order.event.order.prod` | Bad example reverses token order. Hierarchy MUST progress from broader domain to narrower operation. |


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

| Good Example | Bad Example | Reason / Rule Violated |
| :--- | :--- | :--- |
| `nc.Publish("prod.order.event.order.created", payload)` | `nc.Publish("prod.order.event.order.*", payload)` | Bad example publishes to a wildcard pattern. Publishers MUST publish to concrete subjects. |
| `nc.Subscribe("prod.order.event.order.processing.*")` | `nc.Subscribe(">")` | Bad example uses global wildcard (`>`). Consumers SHOULD subscribe using the narrowest required scope. |
| `nc.Subscribe("prod.order.event.order.processing.completed")` | `nc.Subscribe("prod.order.>")` | Bad example uses a broad multi-token wildcard when consuming a single specific operation. |


## Subject Constraints

NATS supports hierarchical subjects with multiple tokens. Subject length and depth should be kept reasonable to maintain readability and operational manageability.

### Enterprise Standard

- Anything past 8-10 tokens is usually a sign that you are encoding data into the subject.
- Total subject length **MUST NOT** exceed **256 characters**.
- Total number of tokens **MUST NOT** exceed **16**.
- Subject depth **SHOULD** be limited to meaningful architectural or business boundaries.

| Good Example | Bad Example | Reason / Rule Violated |
| :--- | :--- | :--- |
| `prod.order.event.order.processing.completed` (6 tokens) | `prod.order.event.order.step1.step2.step3.step4.step5.step6.step7.step8.step9.step10` (13 tokens) | Bad example exceeds recommended depth (> 10 tokens), indicating data is encoded into subject tokens. |
| `prod.order.event.order.created` (31 chars) | `prod.order.event.order.processing.completed.with.a.very.long.descriptive.sentence.that.exceeds.two.hundred.fifty.six.characters.total.length.in.a.single.subject.string.identifier...` (> 256 chars) | Bad example exceeds maximum subject length limit of 256 characters. |
| `prod.order.event.order.created` | `prod.order.event.eu-west-1.rack-3.node-12.order.created` | Bad example exposes physical infrastructure depth instead of business domain boundaries. |


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

| Good Example | Bad Example | Reason / Rule Violated |
| :--- | :--- | :--- |
| `prod.order.event.order.processing.completed` | `prod.order.event.order.processing.completed.ORD-12345` | Bad example includes entity identifier (`ORD-12345`). Identifiers MUST be carried in payload/headers. |
| `prod.order.event.order.processing.completed` | `prod.order.telemetry.order.processing.completed` | Bad example uses unapproved category (`telemetry`). Category MUST be `event` or `audit`. |
| `prod.order.event.order.processing.completed` | `order.processing.completed` | Bad example omits required `environment`, `tenant`, and `category` tokens. |
| `prod.order.event.order.processing.completed` | `prod.us-east-1.order.event.order.processing.completed` | Bad example exposes platform topology (`us-east-1`) in application subject string. |

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

| Good Example | Bad Example | Reason / Rule Violated |
| :--- | :--- | :--- |
| Stream A: `prod.order.event.>` (7-day retention)<br/>Stream B: `prod.order.audit.>` (7-year retention) | Single Stream bound to `prod.order.>` storing both operational events (7-day) and audit logs (7-year) | Bad example combines audit logs and operational events with conflicting retention policies into one stream. |
| Consumer Filter: `prod.order.event.order.processing.completed` | Creating 50 separate micro-streams for every individual action token | Bad example over-fragmentates streams instead of using unified streams with subject filters. |
| Stable Subject: `prod.order.event.order.processing.completed` | Mutating subject to `prod.order.event.streamA.order.processing.completed` | Bad example changes application subject contract solely to satisfy internal stream configuration naming. |


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

| Good Example | Bad Example | Reason / Rule Violated |
| :--- | :--- | :--- |
| `prod.order.event.order.created.v2` | `prod.order.event.order.created.v1.2.3` | Bad example uses semver (`v1.2.3`). Version format MUST be major-only `v<major>` (e.g. `v2`). |
| `prod.order.event.order.created.v2` | `prod.order.v2.event.order.created` | Bad example places version token in middle of subject string. Version MUST be the final token. |
| Maintain `prod.order.event.order.created.v1` when adding an optional `discount_code` payload field | Incrementing to `v2` subject when adding a non-breaking optional payload field | Bad example versions subject for backward-compatible additions. Non-breaking changes MUST remain on existing subject. |


## References

- [Synadia - Designing NATS Subject Hierarchies](https://www.synadia.com/blog/designing-nats-subject-hierarchies)
- [NATS Documentation - Core NATS Deep Dive](https://docs.nats.io/learn/core-nats/)
- [NATS Documentation - Authorization](https://docs.nats.io/learn/security/authorization)
- [NATS Documentation - JetStream Deep Dive](https://docs.nats.io/learn/jetstream/)
- [NATS Documentation - JetStream Reference](https://docs.nats.io/reference/2.12/jetstream)