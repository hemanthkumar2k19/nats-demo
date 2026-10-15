# Process Definiton

```text
title Process Definition - NATS Developer Journey

participant "Application Team" as app
participant "SRE Team" as sre
participant "Messaging Resource Registry" as registry

== 1. Environment Access ==

app->app: Read Process to Request Dev Environment
app->sre: Request Dev Environment Access (Filled Request Template)

group Internal SRE Process
    sre->sre: Identify or Provision NATS Cluster
    sre->sre: Identify or Provision NATS Account
    sre->sre: Configure Identity and Access
end

sre->app: Provide Connection Details and Access Instructions
app->app: Verify Prerequisites and Connectivity


== 2. Messaging Resource Design, Registration and Provisioning ==

app->app: Read Subject, Stream, Consumer and Client Binding Guides
app->app: Define Logical Clients and Required Messaging Access
app->app: Propose Subjects, Persistence Requirements, Consumers, Ownership and Client Bindings
app->sre: Submit Complete Messaging Resource Request

group 2.1 Request Validation
    sre->sre: Validate Subjects, Streams, Consumers and Ownership
    sre->sre: Validate Client Bindings Against Established NATS Account and Identity
    sre->sre: Validate Publish/Subscribe ACLs and Resource Dependencies

    alt Request Contains Violations
        sre->app: Return Consolidated Validation Findings
        app->app: Correct Request and Resubmit
    else Complete Request Valid
        sre->sre: Approve Complete Messaging Resource Request

        group 2.2 Client Binding and Authorization
            sre->sre: Resolve or Provision NATS Identity for Each Logical Client
            sre->sre: Configure Approved Publish/Subscribe Permissions
            sre->registry: Register Client Bindings, Identity References and ACL Metadata
        end

        group 2.3 Subject Configuration and Registration
            sre->sre: Configure Required Subject-Level Authorization
            sre->registry: Register Approved Subjects and Ownership Metadata
        end

        group 2.4 Stream Management
            alt Persistent Messaging Required
                sre->sre: Evaluate Existing Stream Against Approved Requirements

                alt Existing Stream Suitable
                    sre->sre: Apply Required Stream and Subject Mapping Changes
                    sre->registry: Update Stream and Subject Mapping
                else New Stream Required
                    sre->sre: Provision New Stream in NATS
                    sre->registry: Register Stream, Ownership and Subject Mappings
                end

                opt JetStream Consumers Required
                    sre->sre: Validate Consumer Configuration Against Stream
                    sre->sre: Provision Consumers in NATS
                    sre->registry: Register Consumers and Resource Dependencies
                end
            else Non-Persistent Messaging
                sre->sre: No Stream Required for Core NATS Messaging
            end
        end

        sre->sre: Verify Provisioning Results, Permissions and Registry Records
        sre->app: Confirm Messaging Resource Readiness
    end
end

== 3. Application Development and Integration ==

app->app: Read NATS Client Guide and Relevant Core NATS / JetStream Guides
app->app: Implement NATS Integration Using Approved Resources and Platform-Standard Configuration
app->app: Apply Standard Environment Variables and Externalize Environment-Specific Settings
app->app: Validate Messaging, Persistence and Application Integration

== 4. Observability Enablement ==

app->sre: Request Application Observability Enablement

group Internal SRE Process
    sre->sre: Configure Observability Pipeline and Access
    sre->sre: Provision Application Team Namespace, If Required
    sre->sre: Enable NATS MELT Flow to Observability Stack
end

sre->app: Provide OTLP Collector Endpoint, Credentials and Connection Details

app->app: Read Application Observability Guide
app->app: Implement Application MELT Instrumentation and OTLP Export
app->app: Configure and Validate Trace Propagation
app->app: Verify Telemetry Flow to the Observability Stack

alt Observability Validation Failed
    app->sre: Report Observability Integration Issue
    sre->sre: Investigate Platform Configuration
    sre->app: Communicate Findings or Resolution
    app->app: Revalidate Telemetry Flow
else Observability Validation Successful
    app->app: Confirm Observability Integration Readiness
end

== 5. Application Deployment ==

app->app: Verify Deployment Prerequisites and Standard Configuration
app->app: Deploy Application Using Approved Resources and Platform-Standard Configuration
app->app: Validate Application Messaging and Telemetry in Target Environment

note over app,sre: Post-Deployment Operational Readiness, Baseline Checks and Handover Are Covered by the Separate SRE Operational Journey
```

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