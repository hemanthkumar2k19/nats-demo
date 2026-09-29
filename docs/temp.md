# Application Onboarding Journey

## Pre Dev
1. Prerequistes - NATS Knowledge
2. Local Setup
3. Usage Patterns
4. Migration Guide

## Dev
1. Cluster Provisioning 
2. Connectivity – endpoint and connection configuration
3. Account Provisioning – identify or provision the required account
4. Subject Naming and Validation
5. Message Contract and Schema
6. Identity and Access Provisioning – authentication and authorization
7. Stream Provisioning – where JetStream is required
8. Connectivity and Messaging Validation
9. Observability Enablement
10. Application Onboarding Complete



Reference:
1. Prerequisites
1. Local Setup
2. Connectivity Guide
3. Usage Patterns
4. Message Envelope - Schema Management
5. Dev Enablement
6. Observability Enablement
7. Secuirty Enablement


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


|  # | Document / Artifact                        | Primary Audience                    | Purpose                                                |
| -: | ------------------------------------------ | ----------------------------------- | ------------------------------------------------------ |
|  1 | **NATS Platform Architecture**             | SRE                                 | Enterprise NATS architecture and platform model        |
|  2 | **NATS Deployment Architecture**           | SRE                                 | Deployment topology and architecture                   |
|  3 | **Observability Guide**                    | SRE                                 | Platform monitoring and observability                  |
|  4 | **Security Guide**                         | SRE                                 | NATS security model and SRE security practices         |
|  5 | **Connectivity and Troubleshooting Guide** | SRE                                 | Platform connectivity and troubleshooting              |
|  6 | **NATS Local Setup Guide**                 | Developers                          | Local NATS development setup                           |
|  7 | **NATS Usage Patterns**                    | Developers                          | Supported application messaging patterns               |
|  8 | **Application Onboarding Guide**           | Application Team / Developers / SRE | Overall onboarding journey and responsibilities        |
|  9 | **Subject Naming Guide**                   | Developers / SRE                    | Subject naming and namespace expectations              |
| 10 | **Cluster and Account Management Guide**   | SRE                                 | Cluster and account provisioning/management            |
| 11 | **Identity and Access Management Guide**   | SRE                                 | Application identity, authentication and authorization |
| 12 | **Subject Management Guidelines**          | SRE                                 | Subject validation, creation and management            |
| 13 | **Stream Management Guidelines**           | SRE                                 | Persistence, stream creation and stream management     |
| 14 | **SDK Connectivity and Testing Guide**     | Developers                          | SDK-based connectivity and application testing         |
| 15 | **Message Envelope and Schema Management** | Developers                          | Message envelope, schema and contract management       |
| 16 | **Security Enablement Guide**              | Developers                          | Application-side security enablement                   |
| 17 | **Observability Enablement Guide**         | Developers                          | Application-side observability enablement              |


