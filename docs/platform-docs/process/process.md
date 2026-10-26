# Process Definition — NATS Developer Journey

## 1. Purpose

This document describes the steps an application team follows to obtain NATS environment access, request messaging resources, configure authorisation, integrate NATS into the application, and deploy the application.

It focuses on developer responsibilities, required requests, relevant guides, and expected outcomes. Internal platform provisioning and operational procedures are covered by the separate Platform Version and SRE Operational Journey.

## 2. Developer Journey Overview

1. **Environment Access** — Request access and verify NATS connectivity.
2. **Messaging Resource Request** — Review the relevant guides and request the required messaging resources.
3. **Authorisation and Access Management** — Request the required Client–Subject bindings and permissions.
4. **Application Development and Integration** — Implement and validate NATS messaging.
5. **Application Deployment** — Deploy the application and verify its messaging integration.
6. **Operational Handover** — Follow the separate SRE Operational Journey.

## 3. Environment Access

### Steps

1. Read the **Development Environment Access Guide**.
2. Complete and submit the Development Environment Access Request.
3. Receive NATS connection details, credentials or identity setup instructions, and access guidance from the SRE Team.
4. Verify local prerequisites and connectivity using the provided instructions.

**Expected outcome:** The application team has the required environment access and can connect to NATS.

## 4. Messaging Resource Request

### Steps

1. Read the relevant guides:
   - Subject Design and Naming Guide
   - Consumer Design Guide, where JetStream Consumers are required
2. Identify the messaging requirements for the application.
3. Complete and submit the **Messaging Resource Request** according to the applicable guidelines and request template.
4. Review any validation findings provided by SRE, correct the request, and resubmit if necessary.
5. Wait for confirmation that the requested messaging resources are ready.

**Expected outcome:** The requested messaging resources are approved and ready for application integration.

**Important:** Follow the relevant guides and specify application requirements in the request. Internal platform configuration and provisioning are managed by the platform team.

## 5. Authorisation and Access Management

### Steps

1. Identify the logical clients within the application.
2. Determine the required client-to-Subject bindings.
3. Read the **Authorisation and Access Management Guide**.
4. Complete and submit the relevant **Client–Subject Binding Request** through the established access management process.
5. Specify the required publish/subscribe access for each logical client.
6. Address any clarification or correction requests from the responsible team.
7. Wait for confirmation that the required identity bindings and permissions have been configured.
8. Validate that the application can perform the required messaging operations within its assigned permissions.

**Expected outcome:** The required Client–Subject bindings and permissions are configured and validated.

**Important:** Client–Subject binding and authorisation are handled through the separate access management process, not through the Messaging Resource Request. Approval of a messaging resource request does not, by itself, grant a client permission to access that resource.

## 6. Application Development and Integration

### Steps

1. Read the **NATS Client Guide** and the relevant Core NATS or JetStream integration guides.
2. Integrate the application using the approved messaging resources and authorised access.
3. Use platform-standard configuration and environment variables.
4. Externalise environment-specific settings and do not hardcode credentials.
5. Validate application connectivity, messaging behavior, and persistence behavior where applicable.
6. Verify that the application uses the approved messaging resources and assigned permissions.

**Expected outcome:** The application is integrated with NATS and passes the required messaging validation checks.

## 7. Application Deployment

### Steps

1. Verify deployment prerequisites and required environment configuration.
2. Confirm that the required messaging resources and permissions are ready for the target environment.
3. Deploy the application using approved resources and platform-standard configuration.
4. Validate application connectivity and messaging behavior in the target environment.
5. Address issues identified during validation before declaring the application ready.

**Expected outcome:** The application is deployed and its NATS integration is validated in the target environment.

## 8. Operational Handover

Post-deployment operational readiness, baseline checks, and handover follow the separate **SRE Operational Journey**.

Refer to that process for the operational checks and handover requirements applicable to the deployed application.

## 9. Related Documentation

- Development Environment Access Guide
- Development Environment Access Request
- Subject Design and Naming Guide
- Consumer Design Guide
- Messaging Resource Request
- Authorisation and Access Management Guide
- Client–Subject Binding Request
- NATS Client Guide
- Core NATS Integration Guide
- JetStream Integration Guide
- SRE Operational Journey

```plaintext
    title Process Definition - NATS Developer Journey

    participant "Application Team" as app
    participant "SRE Team" as sre

    == 1. Environment Access ==

    app->>app: Read Development Environment Access Guide
    app->>sre: Submit Development Environment Access Request
    sre-->>app: Provide Connection Details and Access Instructions
    app->>app: Verify Prerequisites and NATS Connectivity

    == 2. Messaging Resource Request ==

    app->>app: Read Subject Design and Naming Guide
    app->>app: Read Consumer Design Guide, if JetStream Consumers are required
    app->>app: Identify Application Messaging Requirements
    app->>sre: Submit Messaging Resource Request

    alt Request Requires Correction
        sre-->>app: Provide Validation Findings
        app->>app: Correct Request
        app->>sre: Resubmit Messaging Resource Request
    else Request Approved
        sre-->>app: Confirm Messaging Resources Are Ready
    end

    == 3. Authorisation and Access Management ==

    app->>app: Identify Logical Clients and Required Client-to-Subject Bindings
    app->>app: Read Authorisation and Access Management Guide
    app->>sre: Submit Client-Subject Binding Request

    alt Request Requires Correction
        sre-->>app: Request Clarification or Corrections
        app->>app: Update Request
        app->>sre: Resubmit Client-Subject Binding Request
    else Access Configured
        sre-->>app: Confirm Identity Bindings and Permissions
    end

    app->>app: Validate Required Publish/Subscribe Access

    == 4. Application Development and Integration ==

    app->>app: Read NATS Client and Relevant Integration Guides
    app->>app: Implement NATS Integration Using Approved Resources
    app->>app: Apply Platform-Standard Configuration
    app->>app: Validate Connectivity and Messaging Behavior

    opt Validation Issues
        app->>app: Troubleshoot and Correct Integration
        app->>app: Revalidate Messaging Behavior
    end

    == 5. Application Deployment ==

    app->>app: Verify Deployment Prerequisites
    app->>app: Confirm Resources and Permissions Are Ready
    app->>app: Deploy Application
    app->>app: Validate NATS Integration in Target Environment

    opt Deployment Validation Issues
        app->>app: Address Issues and Revalidate
    end

    == 6. Operational Handover ==

    app->>sre: Follow Separate SRE Operational Journey

```