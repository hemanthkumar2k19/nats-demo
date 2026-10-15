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

app->app: Read Subject, Stream and Consumer Design Guides
app->app: Propose Subjects, Persistence Requirements, Consumer Configurations and Ownership
app->sre: Submit Complete Messaging Resource Request

group 2.1 Request Validation
    sre->sre: Validate All Subjects, Streams and Consumers
    sre->sre: Validate Ownership, Authorization and Resource Dependencies

    alt Request Contains Violations
        sre->app: Return Consolidated Validation Findings
        app->app: Correct Request and Resubmit
    else Complete Request Valid
        sre->sre: Approve Complete Messaging Resource Request

        group 2.2 Subject Configuration and Registration
            sre->sre: Configure Required Subject-Level Authorization
            sre->registry: Register Approved Subjects and Ownership Metadata
        end

        group 2.3 Stream Management
            alt Persistent Messaging Required
                sre->sre: Evaluate Existing Stream Against Approved Requirements

                alt Existing Stream Suitable
                    sre->sre: Approve and Apply Required Subject Mapping or Configuration Changes
                    sre->registry: Register or Update Stream and Subject Mapping
                else New Stream Required
                    sre->sre: Approve Stream Configuration
                    sre->sre: Provision New Stream in NATS
                    sre->registry: Register Stream, Ownership and Subject Mappings
                end

                group 2.4 Consumer Management
                    opt JetStream Consumers Required
                        sre->sre: Validate Approved Consumer Configuration Against Stream
                        sre->sre: Provision Consumers in NATS
                        sre->registry: Register Consumers, Ownership and Resource Dependencies
                    end
                end

            else Non-Persistent Messaging
                sre->sre: No Stream Required for Core NATS Messaging
            end
        end

        sre->sre: Verify Provisioning Results and Registry Records
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