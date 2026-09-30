# NATS Application Onboarding Flow

## SRE Track

- Learn NATS Fundamentals
- Understand NATS Platform Architecture
- Understand SRE Guides & Best Practices
  - Deployment Architecture
  - Observability Guide
  - Security Guide
  - Connectivity and Troubleshooting Guide
- Ready for Enablement

## Application Onboarding

### Developer Track

- Learn NATS Fundamentals
- Understand Local Setup
- Understand Usage Patterns
- Define Application Messaging Requirements

### Enablement Track

- App Team Requests Onboarding
  - Application Ownership Details
  - Environment requirements
  - Subject requirements
  - Persistence requirement
  - Capacity, Resource and Resilency Requirements
- SRE Reviews Application Requirements
- Provision / Allocate NATS Resources
  - Cluster
  - Account - One Per Application Identity
- Configure Application Identity & Access
  - Application credentials/identity
  - Authentication
  - Authorization
  - Encryption
- Review Subject Requirements
  - Validate Subject Naming
  - Check persistance requirement per subject
- Enable Observability
- Provide NATS Connectivity Details
  - NATS endpoint
  - Account/application identity
  - Authentication/Authorization Credentials
  - Enabled Permissions
- Ready for Usage

### Application Track

- Understand Developer Guides & Best Practices
  - Connectivity Guide
  - Usage Patterns
  - Message Envelope - Schema Management
  - Security Enablement
  - Observability Enablement
- Develop
- Deploy
- Test
- Promote


## New List of Documents

|  # | Document / Artifact                             | Primary Audience                    | Covers / Data Points                                                                                | Priority |
| -: | ----------------------------------------------- | ----------------------------------- | --------------------------------------------------------------------------------------------------- | :------: |
|  1 | **NATS Solution Architecture**                  | SRE / Platform                      | Overall solution architecture and NATS role                                                         |    P1    |
|  2 | **NATS Platform Architecture**                  | SRE / Platform                      | NATS platform architecture, platform components and boundaries                                      |    P1    |
|  3 | **NATS Deployment Architecture**                | SRE                                 | Local Setup, Edge, HA                                                                               |    P1    |
|  4 | **NATS Connectivity Guide**                     | Developers / SRE                    | Edge connectivity, Server connectivity                                                              |    P1    |
|  5 | **NATS Publisher Guide**                        | Developers                          | Registration, Pooling, Retry Policies, Buffer Management, Trace Propagation                         |    P1    |
|  6 | **NATS Consumer Guide**                         | Developers                          | Registration, Pooling, At-least-Once Delivery, Cursor Management, Retry Policies, Trace Propagation |    P1    |
|  7 | **NATS Usage Patterns**                         | Developers / Application Teams      | Saga, Inter-Service Calls, supported usage patterns                                                 |    P1    |
|  8 | **NATS Security Guide**                         | SRE / Developers                    | TLS, Server RBAC, Encryption at Rest, Encryption in Transit, Client RBAC                            |    P1    |
|  9 | **NATS Semantics & Naming Guidelines**          | SRE / Developers                    | Cluster & Node Naming, Account Naming, Stream Naming, Subject Naming                                |    P1    |
| 10 | **NATS Local Setup Guide**                      | Developers                          | Local NATS setup and development environment                                                        |    P1    |
| 11 | **NATS Application Onboarding Guide**           | Application Team / Developers / SRE | End-to-end onboarding journey, responsibilities and process                                         |    P1    |
| 12 | **Application Messaging Requirements Template** | Application Team / Developers       | Use case, usage pattern, volume, persistence, capacity, resiliency and subject requirements         |    P1    |
| 13 | **Message Envelope & Schema Management**        | Developers                          | Message envelope, schema, contract, versioning and compatibility                                    |    P1    |
| 14 | **NATS Security Enablement Guide**              | Developers                          | Application-side security enablement and required configuration                                     |    P1    |
| 15 | **NATS Scale & Capacity Guide**                 | SRE / Platform                      | Server, Storage, Network scaling and capacity considerations                                        |    P2    |
| 16 | **NATS Governance Guide**                       | SRE / Platform                      | Server, Account, Policies, Streams, Subjects                                                        |    P2    |
| 17 | **NATS Observability Guide**                    | SRE / Developers                    | System MELT, Application MELT                                                                       |    P2    |
| 18 | **NATS Message Flow Tracing Guide**             | SRE / Developers                    | Message flow tracing, trace propagation                                                             |    P2    |
| 19 | **NATS Gating Guidelines**                      | Platform / SRE / Application Owners | Use case, Usage, Volume                                                                             |    P2    |
| 20 | **NATS Disaster Recovery Guide**                | SRE / Platform                      | DR strategy, recovery considerations                                                                |    P2    |
| 21 | **NATS Observability Enablement Guide**         | Developers                          | Application-side observability and instrumentation expectations                                     |    P2    |
