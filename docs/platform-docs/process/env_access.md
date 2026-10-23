# Environment Access Request

## 1. Application and Request Details

| Field | Details |
|---|---|
| Request ID | [System-generated / assigned by SRE] |
| Request Type | [New Access / Modify Existing Access / Revoke Access] |
| Application Name | |
| Application Identity | |
| Requesting Team | |
| Technical Contact | [Name / Email / Team Channel] |
| Target Environment | [Dev / Test / Staging / Production] |
| Business Purpose | [Brief description of the application and why NATS is required] |
| Required By | [Date] |

## 2. Set A — Workload and Capacity Profile

Provide estimates for the expected workload in the target environment. Mark each value as **Estimated**, **Measured**, or **Unknown**, where applicable.

| Requirement | Application Estimate |
|---|---|
| Messaging Use Case | [Audit Logging / Inter-service Events / Saga / Other] |
| Average Message Rate | [Messages/sec] |
| Peak Message Rate | [Messages/sec] |
| Average Payload Size | [Bytes or KB] |
| Maximum Expected Payload Size | [Bytes or KB] |
| Expected Concurrent NATS Connections | |
| Expected Subscribers / Fan-out | [Typical and peak, if known] |
| Traffic Pattern | [Steady / Bursty / Periodic / Other] |
| Expected Workload Growth | [Estimated growth over 6–12 months, if known] |
| Estimate Basis | [Forecast / Load Test / Existing Production Metrics / Initial Estimate] |

## 3. Set A — Persistence and Data Profile

| Requirement | Application Estimate |
|---|---|
| Message Persistence Required? | [Yes / No / Undecided] |
| Approximate Retention Horizon | [e.g., 24 hours / 7 days / 30 days / Unknown] |
| Expected Retained Data Volume | [If known; otherwise Unknown] |
| Data Growth Pattern | [If known] |
| Additional Data Durability Requirements | [If applicable] |

*Detailed subject-to-Stream mappings, exact retention policies, Stream limits, and Consumer configuration will be submitted through Messaging Resource Management.*

## 4. Set A — Reliability Requirements

| Requirement | Application Need |
|---|---|
| Business Criticality | [Low / Medium / High / Critical, according to platform definitions] |
| Required Availability Tier | [Platform-defined tier / To be assessed] |
| Recovery Time Objective (RTO) | [If defined / Not defined] |
| Recovery Point Objective (RPO) | [If defined / Not defined] |
| Burst Handling / Degradation Tolerance | [Describe, if relevant] |
| Workload Isolation Requirement | [Shared cluster acceptable / Isolation required / Unknown] |
| Disaster Recovery Requirement | [Required / Not required / To be assessed] |
| Additional Reliability Constraints | [If applicable] |

*Platform policies define supported availability, recovery, and isolation options. SRE will assess feasibility against those policies.*

## 5. Connectivity Source Details

| Field | Details |
|---|---|
| Source IP Address(es) / CIDR(s) | |
| Source Network / Egress Details | [If known] |
| Source Environment / Deployment Location | [e.g., corporate network, cloud environment] |
| Additional Connectivity Context | [Existing network restrictions or relevant dependencies] |

*The App Team provides known source connectivity details. SRE coordinates endpoint assignment and required network/firewall changes.*

## 6. Existing Access (If Applicable)

| Field | Details |
|---|---|
| Existing NATS Account Reference | [If known] |
| Existing NATS Identity Reference | [If known] |
| Existing Cluster / Environment Reference | [If known] |
| Requested Change | [Describe the change or additional access required] |

## 7. App Team Confirmation

- [ ] Workload and reliability figures are identified as measured, estimated, or unknown where applicable.
- [ ] Known source IP/CIDR and connectivity details have been provided.
- [ ] No passwords, private keys, tokens, or other secrets are included in this request.
- [ ] Detailed messaging resources will be submitted separately through the approved Messaging Resource Management process.

**Submitted By:**  
**Submission Date:**  

# Environment Access Response

## 1. Request Outcome

| Field | Details |
|---|---|
| Request ID | |
| Application Name | |
| Application Identity | |
| Target Environment | |
| Request Status | [Approved / Approved with Actions / Pending Information / Rejected / Completed] |
| Assigned SRE Owner | |
| Assessment / Response Date | |

## 2. Capacity and Reliability Assessment

| Assessment | SRE Outcome |
|---|---|
| Workload Profile Reviewed | [Complete / Additional Information Required] |
| Target Cluster Capacity | [Sufficient / Capacity Change Required / Further Assessment Required] |
| CPU / Memory Impact | [Assessment or reference] |
| Network / Throughput Impact | [Assessment or reference] |
| JetStream Storage Impact | [Assessment or reference / Not Applicable] |
| Availability / Recovery Feasibility | [Supported / Conditional / Not Supported / Pending] |
| Workload Isolation Assessment | [Shared Cluster / Dedicated Cluster / Further Assessment] |
| Placement Decision | [Existing Cluster / Capacity Expansion / New or Dedicated Cluster / Pending] |
| Assumptions / Outstanding Actions | [Details, owner, and due date] |

*Capacity estimates and placement decisions are based on the supplied workload profile, current platform capacity, benchmarks, and platform policies. A new cluster or capacity expansion is subject to the relevant approval and provisioning process.*

## 3. NATS Environment and Account Details

| Field | Details |
|---|---|
| Target Cluster Name / Reference | |
| NATS Account Name / Reference | |
| Application Identity-to-Account Mapping | [Confirmed / Pending] |
| Account Provisioning Status | [Existing / Provisioned / Pending] |
| NATS Connection Endpoint(s) | [SRE-provided endpoint or approved configuration reference] |
| Network / Firewall Status | [Not Required / Pending / Completed] |
| Network Change / Firewall Ticket | [Reference, if applicable] |

*Application Identity remains consistent across environments. The corresponding NATS Account exists within the relevant environment's cluster.*

## 4. Authentication and Security Configuration

| Field | Details |
|---|---|
| Platform-Defined Authentication Method | [Configured method] |
| NATS Identity Reference | [Approved identity reference, if applicable] |
| Identity Provisioning Status | [Existing / Provisioned / Pending / Not Applicable] |
| Credential Acquisition Instructions | [Approved secret-management or delivery reference] |
| TLS / Certificate Requirements | [Platform-defined requirements] |
| Security Dependencies | [Outstanding actions, if any] |

*Authentication, authorization, TLS, and credential-handling standards are defined by the Platform Team. Credentials and private key material must not be included directly in this response.*

## 5. Connection Configuration and Developer References

| Item | Details |
|---|---|
| Approved Connection Endpoint | [Endpoint / configuration reference] |
| Required Configuration Variables | [Platform-standard variable names and configuration reference] |
| Credential / Certificate Reference | [Secure retrieval instructions] |
| Client / Connection Guide | [Link] |
| Local Setup Guide | [Link] |
| Security / Identity & Access Guide | [Link] |
| Support / Troubleshooting Guide | [Link] |

## 6. Validation and Handover

### SRE Validation

- [ ] Target cluster and account assignment recorded.
- [ ] Required account and identity configuration completed.
- [ ] Network/firewall dependencies completed or explicitly tracked.
- [ ] Approved connection details and credential retrieval instructions provided.
- [ ] Capacity assessment and outstanding actions recorded.

### App Team Validation

- [ ] Application can reach the provided endpoint.
- [ ] TLS and authentication succeed using the approved configuration.
- [ ] Connection remains healthy under an initial connectivity test.
- [ ] Any connection failures have been shared with SRE for investigation.

**Connectivity validation result:** [Pending / Successful / Failed]

**Validation date and evidence/reference:**  

*Successful connection validation confirms basic connectivity and authentication. It does not imply permission to publish or subscribe to all subjects, or that Streams and Consumers have been provisioned.*

## 7. Closure and Next Steps

| Field | Details |
|---|---|
| Environment Access Status | [Ready for Messaging Resource Management / Pending Actions / Blocked] |
| Outstanding Actions | [Action / Owner / Due Date] |
| Next Process | Messaging Resource Design, Registration and Provisioning |
| SRE Contact | |
| Closure Date | |

**SRE Confirmation:**  
[Summarize the environment access outcome, any remaining dependencies, and the conditions for proceeding to messaging resource onboarding.]