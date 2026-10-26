## Purpose

Use this form to request the creation, modification, or retirement of a NATS Subject. The Application Team defines the Subject's intended usage, ownership, and requirements. Platform/SRE validates the request, maintains the Subject's governance record, and configures any required authorization or related JetStream resources.

**Important:** Subjects can be used for Core NATS messaging without persistence. When persistence is required, Platform/SRE determines the appropriate Stream mapping and whether an existing Stream can satisfy the requirements.

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
| Business Purpose | [Brief description of the messaging use case] |
| Related Request ID | [If applicable] |
| Required By | |

## 2. Subject Details

Complete the subsection corresponding to the requested operation.

### 2.1 Create a Subject

| Field | Details |
|---|---|
| Subject Name | [Proposed Subject name, e.g., `dev.orders-app.event.order.created`] |
| Business Purpose | [What business event or message does this Subject represent?] |
| Owning Application / Team | |
| Persistence Required? | [Yes / No] |
| Persistence Justification | [Required if persistence is Yes] |
| Expected Message Rate | [Optional; provide if materially different from Environment Access estimates] |
| Expected Payload Size | [Optional; provide if materially different from Environment Access estimates] |
| Additional Notes | [Optional] |

*Repeat for each Subject requested.*

### 2.2 Modify an Existing Subject

| Field | Details |
|---|---|
| Existing Subject Name | |
| Existing Subject Reference | [If available] |
| Current Owner | [If known] |
| Requested Change | [Purpose / Ownership / Persistence Requirement / Other] |
| Current Value | |
| Requested Value | |
| Reason for Change | |
| Expected Application Impact | [If applicable] |
| Migration / Cutover Requirements | [If applicable] |
| Required By | |

*If the request involves renaming a Subject, identify the proposed replacement Subject and provide a migration or cutover plan. Treat a rename as a migration rather than a simple metadata edit.*

### 2.3 Retire an Existing Subject

| Field | Details |
|---|---|
| Existing Subject Name | |
| Existing Subject Reference | [If available] |
| Owning Application / Team | |
| Reason for Retirement | |
| Replacement Subject | [If applicable] |
| Migration / Cutover Plan | [If applicable] |
| Dependency Removal Plan | [Describe how active dependencies will be removed or migrated] |
| Persistence / Retention Considerations | [If applicable] |
| Requested Retirement Date | |
| Additional Notes | [Optional] |

*Retirement is subject to dependency assessment and approval. Retiring a Subject does not automatically delete a Stream or its stored messages.*

## 3. Subject Naming and Design Confirmation

Complete the applicable confirmations for the requested operation.

- [ ] Subject names comply with the enterprise naming standard: `<environment>.<account>.<category>.<resource>.<operation>`.
- [ ] Tokens are lowercase, with hyphens for word separation.
- [ ] Subject names do not contain individual record IDs or entity IDs.
- [ ] Each Subject has a defined business purpose and owning application/team.
- [ ] Existing Subjects are referenced where applicable rather than duplicated.
- [ ] Changes to existing Subjects have been assessed for application dependencies.
- [ ] Retirement requests identify relevant dependencies and required migration or cutover activities.

## 4. Persistence Requirements

Complete this section when persistence is required or when modifying or retiring a Subject with existing persistence dependencies.

| Field | Details |
|---|---|
| Subject(s) Requiring Persistence | |
| Persistence Justification | |
| Required Retention Duration | [e.g., 24 hours / 7 days / 30 days] |
| Expected Retained Data Volume | [If known] |
| Data Durability / Recovery Requirements | [If applicable] |
| Existing Stream Reference | [Optional, if known] |
| Functional Constraints on Stream Reuse | [If applicable; explain the requirement, not the infrastructure solution] |
| Additional Requirements | [Optional] |

### Ownership and Decision Rules

- The Application Team defines persistence, retention, and business durability requirements.
- Platform/SRE evaluates whether an existing Stream can satisfy those requirements.
- Platform/SRE determines whether to reuse a Stream, modify it where permitted, or create a new Stream.
- The Application Team is not required to propose Stream names or choose Stream configuration.
- A Stream must not be reused or modified if doing so would violate another application's requirements, ownership expectations, or platform policy.
- Platform/SRE records the approved Subject-to-Stream mapping in the Messaging Resource Registry.

## 5. Application Team Confirmation

- [ ] The request type and target environment are correct.
- [ ] Subject names, business purposes, and ownership details are complete.
- [ ] Persistence requirements are documented where applicable.
- [ ] Changes and retirements identify affected dependencies and required migration steps.
- [ ] Workload estimates are consistent with the Environment Access Request, or differences are explained.
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
| Naming and Design Validation | [Pass / Fail / Not Applicable] |
| Ownership and Dependency Validation | [Pass / Fail / Not Applicable] |
| Authorization Impact Reviewed | [Yes / No / Not Applicable] |
| Persistence / Stream Mapping Decision | [If applicable] |
| Exception Reference | [If applicable] |
| Implementation / Registry Reference | |
| Reviewer | |
| Review Date | |
| Implementation Status | [Pending / In Progress / Completed / Failed] |
| Verification Notes | |
| Completion Date | |

Platform/SRE records the approved Subject metadata, ownership, relationships, and lifecycle status in the appropriate governance records. Any required authorization or JetStream configuration changes are implemented and verified through the applicable platform process.