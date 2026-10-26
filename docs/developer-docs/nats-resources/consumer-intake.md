# NATS Consumer Intake Form

## Purpose

Use this form to request the creation, modification, or retirement of a platform-managed NATS JetStream Consumer. The Application Team defines its message consumption requirements and intended processing behavior. Platform/SRE validates the request, determines the appropriate Consumer configuration, provisions or modifies the Consumer, and maintains its governance record.

**Important:**
- A JetStream Consumer is associated with exactly one Stream. The Application Team specifies the required subject filters; Platform/SRE determines the appropriate Stream based on the approved Subject-to-Stream mapping.
- Core NATS subscriptions do not require JetStream Consumers.
- Ephemeral Consumers created dynamically by applications do not require a separate intake request by default, subject to platform policy and applicable access requirements.

## 1. Request Details

| Field | Details |
|---|---|
| Request ID | [System-generated / assigned by SRE] |
| Request Type | [Create / Modify / Retire] |
| Application Name | |
| Requesting Team | |
| Technical Contact | |
| Target Environment | [Dev / Test / Staging / Production] |
| NATS Account Reference | [From Environment Access] |
| Business Purpose | [Brief description of the message consumption use case] |
| Related Request ID | [If applicable] |
| Required By | |

## 2. Consumer Details

Complete the subsection corresponding to the requested operation.

### 2.1 Create a Consumer

| Field | Details |
|---|---|
| Consumer Name / Proposed Name | [Proposed name, if known] |
| Consumer Purpose | [What processing or business function does this Consumer support?] |
| Owning Application / Team | |
| Required Subject Filter(s) | [Subject names or patterns] |
| Delivery Model | [Push / Pull / To Be Assessed] |
| Deliver Subject | [Required for Push; proposed value if known; Not Applicable for Pull] |
| Acknowledgement Requirement | [Describe when processing is considered successful and when a message can be acknowledged] |
| Replay / Delivery Requirement | [e.g., deliver new messages only or replay retained messages, if applicable] |
| Expected Consumption Rate | [Optional, if known] |
| Processing Duration / Long-running Work | [Optional; describe if relevant to acknowledgement or redelivery behavior] |
| Additional Requirements | [Optional] |

*Repeat for each Consumer requested.*

### 2.2 Modify an Existing Consumer

| Field | Details |
|---|---|
| Existing Consumer Name | |
| Existing Consumer Reference | [If available] |
| Requested Change | [Subject Filters / Delivery Model / Deliver Subject / Acknowledgement Behavior / Replay Behavior / Ownership / Other] |
| Current Configuration or Behavior | |
| Requested Configuration or Behavior | |
| Reason for Change | |
| Expected Application Impact | [If applicable] |
| Migration / Cutover Requirements | [If applicable] |
| Required By | |

*Describe the intended processing behavior rather than prescribing low-level configuration parameters unless a specific parameter is an explicit application requirement. Platform/SRE validates whether the requested change is supported and assesses its impact on message delivery, redelivery, and processing continuity.*

### 2.3 Retire an Existing Consumer

| Field | Details |
|---|---|
| Existing Consumer Name | |
| Existing Consumer Reference | [If available] |
| Owning Application / Team | |
| Reason for Retirement | |
| Application or Processing Dependencies | |
| Replacement Consumer | [If applicable] |
| Migration / Cutover Plan | [If applicable] |
| Pending / Unprocessed Message Considerations | [Describe any relevant processing or recovery requirements] |
| Requested Retirement Date | |
| Additional Notes | [Optional] |

*Retiring a Consumer does not automatically retire its associated Stream or delete the Stream's stored messages. Platform/SRE assesses dependencies, pending work, and applicable retention requirements before implementing retirement.*

## 3. Consumer Design Confirmation

Complete the applicable confirmations for the requested operation.

- [ ] The Consumer has a defined purpose and owning application/team.
- [ ] Required subject filters are specified.
- [ ] The Consumer type and delivery model are identified, or Platform/SRE is asked to assess them.
- [ ] A Deliver Subject is provided or proposed for Push Consumers, where applicable.
- [ ] Acknowledgement and failure-handling requirements are described.
- [ ] Replay or delivery requirements are documented where applicable.
- [ ] Changes to existing Consumers have been assessed for application compatibility.
- [ ] Retirement requests identify relevant dependencies and pending-work considerations.

## 4. Processing and Delivery Requirements

Complete this section when additional explanation is needed for acknowledgement, failure handling, replay, ordering, or processing continuity.

| Field | Details |
|---|---|
| Processing Success Criteria | [What must complete before processing is considered successful?] |
| Failure Handling Requirements | [Describe expected behavior when processing fails] |
| Redelivery Considerations | [If applicable] |
| Ordering Requirements | [If applicable; describe the ordering needed by the application] |
| Replay Requirements | [If applicable] |
| Processing Continuity Requirements | [If applicable, especially for Consumer changes or migration] |
| Additional Constraints | [Optional] |

### Ownership and Decision Rules

- The Application Team defines its message-processing and delivery requirements.
- Platform/SRE determines the appropriate Consumer configuration in accordance with platform standards.
- The requested delivery model, acknowledgement behavior, replay behavior, Deliver Subject, and subject filters must be compatible with the associated Stream and the application's requirements.
- Platform/SRE validates the Stream relationship and determines the final Consumer configuration.
- The Application Team is not required to select a Stream or prescribe low-level Consumer configuration parameters unless they represent an explicit application requirement.

## 5. Application Team Confirmation

- [ ] The request type and target environment are correct.
- [ ] Consumer purpose and ownership are identified.
- [ ] Required subject filters are provided.
- [ ] Delivery, acknowledgement, and failure-handling requirements are documented.
- [ ] Deliver Subject requirements are addressed for Push Consumers.
- [ ] Replay, ordering, and processing-continuity requirements are documented where applicable.
- [ ] Changes and retirements identify affected dependencies and required migration steps.
- [ ] No credentials, tokens, private keys, or other secret material are included.
- [ ] The Application Team understands that Platform/SRE approval is required before governed changes are implemented.

**Submitted By:**

**Submission Date:**

**Architecture / Supporting References:**

**Additional Comments:**

## 6. Platform/SRE Review and Outcome

*For Platform/SRE completion.*

| Field | Details |
|---|---|
| Review Outcome | [Approved / Changes Required / Rejected] |
| Consumer Design Validation | [Pass / Fail / Not Applicable] |
| Subject Filter and Stream Compatibility | [Pass / Fail / Not Applicable] |
| Delivery and Acknowledgement Requirements Reviewed | [Yes / No] |
| Deliver Subject Validated | [Yes / No / Not Applicable] |
| Application Dependencies Reviewed | [Yes / No / Not Applicable] |
| Approved Consumer Configuration / Reference | |
| Exception Reference | [If applicable] |
| Implementation / Registry Reference | |
| Reviewer | |
| Review Date | |
| Implementation Status | [Pending / In Progress / Completed / Failed] |
| Verification Notes | |
| Completion Date | |

Platform/SRE verifies that the associated Stream exists or is approved for creation, captures the required subjects, and supports the requested Consumer behavior. For Push Consumers, Platform/SRE also validates the Deliver Subject and any required authorization. The approved Consumer configuration, ownership, relationships, and lifecycle status are recorded in the appropriate governance records.