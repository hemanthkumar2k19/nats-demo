# NATS Messaging Resource Request

## 1. Request Details

| Field | Details |
|---|---|
| Request ID | [System-generated / assigned by SRE] |
| Request Type | [New Resources / Modify Existing Resources / Decommission Resources] |
| Application Name | |
| Application Identity | |
| Requesting Team | |
| Technical Contact | |
| Target Environment | [Dev / Test / Staging / Production] |
| NATS Account Reference | [From Environment Access] |
| Business Purpose | [Brief description of the messaging use case] |
| Related Environment Access Request ID | |
| Required By | |

## 2. Subject Registration and Ownership

The App Team declares the subjects it intends to use, their purpose, ownership, and expected messaging behavior. SRE validates and registers the approved subject metadata and configures the corresponding authorization as required.

### 2.1 Subject Inventory

| Field | Details |
|---|---|
| Subject Name | [e.g., dev.orders-app.event.order.created] |
| Business Purpose | |
| Category | [audit / event] |
| Resource | [e.g., order] |
| Operation | [e.g., created] |
| Owning Application / Team | |
| Publishing Logical Client(s) | |
| Subscribing Logical Client(s) | |
| Persistence Required? | [Yes / No] |
| Expected Message Rate | [Optional, if different from Environment Access estimates] |
| Expected Payload Size | [Optional, if different from Environment Access estimates] |
| Additional Notes | |

*Repeat for each subject.*

### 2.2 Subject Naming and Ownership Confirmation

- [ ] Subject names comply with the enterprise naming standard: `<environment>.<account>.<category>.<resource>.<operation>`.
- [ ] Tokens are lowercase, with hyphens for word separation.
- [ ] Subject names do not contain individual record IDs or entity IDs.
- [ ] Each subject has a defined business purpose and owning team.
- [ ] Intended publishing and subscribing clients are identified.
- [ ] Existing subjects are referenced where applicable rather than duplicated.

## 3. Persistence and Retention Requirements

Complete this section for each subject or group of subjects that requires persistence. SRE will use these requirements to determine the appropriate Stream design.

| Field | Details |
|---|---|
| Subject(s) Requiring Persistence | |
| Persistence Justification | |
| Required Retention Duration | [e.g., 24 hours / 7 days / 30 days] |
| Expected Retained Data Volume | [If known] |
| Data Durability / Recovery Requirements | [If applicable] |
| Existing Stream Reference | [Optional, if known] |
| Functional Constraints on Stream Reuse | [If applicable; explain the requirement, not the infrastructure solution] |
| Additional Requirements | |

### Ownership and Decision Rules

- The App Team defines persistence, retention, and business durability requirements.
- SRE evaluates whether an existing Stream can satisfy those requirements.
- SRE determines whether to reuse an existing Stream, modify it where permitted, or create a new Stream.
- A Stream must not be reused or modified if doing so would violate another application's requirements, ownership expectations, or platform policy.
- If no safe, policy-compliant solution exists, SRE returns the conflict for resolution or escalates it through the approved exception process.
- SRE owns the final subject-to-Stream mapping and records the approved mapping in the Messaging Resource Registry.

*The App Team is not required to propose Stream names, choose Stream retention policies, or decide whether a new Stream must be created.*

## 4. Consumer Requirements

Complete this section for each JetStream Consumer required by the application. Each Consumer must reference a subject or set of subjects that will be captured by an existing or proposed Stream.

### 4.1 Consumer Inventory

| Field | Details |
|---|---|
| Consumer Name / Proposed Name | |
| Consumer Purpose | |
| Owning Application / Team | |
| Logical Client Using the Consumer | |
| Existing Stream Reference | [If known] |
| Required Subject Filter(s) | [Subject names or patterns] |
| Consumer Type | [Durable / Ephemeral / To Be Assessed] |
| Delivery Model | [Push / Pull / To Be Assessed] |
| Acknowledgement Requirement | [Describe processing and acknowledgement needs] |
| Replay / Delivery Requirement | [If applicable] |
| Expected Consumption Rate | [Optional, if known] |
| Additional Requirements | |

*Repeat for each Consumer.*

### 4.2 Consumer Dependencies

- [ ] Every requested Consumer has an identified purpose and owning team.
- [ ] The required subject filters are specified.
- [ ] The intended logical client is identified.
- [ ] Required delivery, acknowledgement, and replay behavior is described.
- [ ] Existing Stream references are provided where known.
- [ ] Any dependency on a new Stream is identified through the persistence requirements.

*SRE validates that the associated Stream exists or is approved for creation, covers the required subjects, and supports the requested Consumer behavior. Core NATS subscriptions do not require JetStream Consumers.*

## 5. Client Binding and Authorization

Declare which logical application clients need to publish messages, subscribe to messages, or consume from JetStream. SRE validates these requirements and configures the corresponding permissions using the platform-approved security model.

### 5.1 Logical Client Inventory

| Field | Details |
|---|---|
| Logical Client Name | [Application-defined name for a distinct messaging use case or component] |
| Purpose | |
| Owning Application / Team | |
| Existing NATS Identity Reference | [If known] |
| New NATS Identity Required? | [Yes / No / To Be Assessed] |
| Required Publish Subjects | [Subject names or patterns] |
| Required Subscribe Subjects | [Subject names or patterns] |
| Required JetStream Consumers | [If applicable] |
| Additional Authorization Requirements | |

*Repeat for each logical client. Do not include passwords, tokens, private keys, or other secret material.*

### 5.2 Authorization Matrix

| Logical Client | NATS Identity Reference | Publish Permissions | Subscribe Permissions | JetStream Access |
|---|---|---|---|---|
| | | [Subjects / Patterns] | [Subjects / Patterns] | [Required Streams / Consumers] |

### Client Binding Confirmation

- [ ] Each logical client has an identified purpose and owner.
- [ ] Publish and subscribe requirements are declared separately.
- [ ] Permissions are limited to the required subjects and operations.
- [ ] Broad wildcards are avoided unless explicitly justified.
- [ ] Clients with materially different privilege requirements are identified separately.
- [ ] Existing NATS identity references are supplied where known.
- [ ] New identity provisioning is requested where required.
- [ ] No secrets are included in the request.

*The registry records logical clients, identity references, ownership, approved permissions, and relationships to subjects, Streams, and Consumers. Registry records do not enforce authorization by themselves; SRE must apply and verify the actual NATS permissions.*

## 6. Existing Resources and Change Scope

Complete this section when modifying or decommissioning resources.

| Field | Details |
|---|---|
| Existing Resource Type | [Subject / Stream / Consumer / Client Binding / Identity] |
| Existing Resource Name / Reference | |
| Current Owner | |
| Requested Change | [Add / Modify / Remove / Transfer Ownership] |
| Reason for Change | |
| Affected Subjects / Streams / Consumers / Clients | |
| Expected Application Impact | |
| Migration / Cutover Requirements | |
| Data Retention or Decommissioning Requirements | |

*Repeat for each affected resource or relationship.*

## 7. App Team Confirmation

- [ ] Subject names, business purposes, ownership, and intended clients are provided.
- [ ] Persistence and retention requirements are documented for applicable subjects.
- [ ] Consumer requirements and dependencies are identified.
- [ ] Logical client bindings and publish/subscribe permissions are declared.
- [ ] Existing resources are referenced where known.
- [ ] Workload estimates are consistent with the Environment Access Request, or differences are explained.
- [ ] No credentials or private key material are included.
- [ ] The App Team understands that SRE approval is required before resources or permissions are provisioned or modified.

**Submitted By:**  
**Submission Date:**  
**Architecture / Supporting References:**  
**Additional Comments:**
