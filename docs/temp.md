
# NATS Developer Usage and Enablement Guide

## 1. Connectivity and Setting Up

### Usage
- [ ] Document the supported connection methods and client libraries.
- [ ] Define the standard connection configuration expected for applications.
- [ ] Document environment-specific connectivity requirements (Development, Test, Production).
- [ ] Define application responsibilities for connection management, reconnection, and graceful shutdown.

### Process to Enable
- [ ] Document the process for requesting NATS connectivity.
- [ ] Define the information developers must provide (Application ID, environment, use case, etc.).
- [ ] Document how connection endpoints and configuration details are provided.
- [ ] Define the onboarding and connectivity validation process.

## 2. Usage Patterns

### Usage
- [ ] Define recommended patterns for synchronous and asynchronous communication.
- [ ] Document when to use Core NATS versus JetStream-enabled messaging.
- [ ] Define guidance for Request/Reply, Publish/Subscribe, and Queue Groups.
- [ ] Document recommended patterns for persistent messaging and independent consumer processing.
- [ ] Define application-level expectations for acknowledgments, retries, and message processing.
- [ ] Provide reference examples for the supported messaging patterns.

### Process to Enable
- [ ] Define how developers specify their messaging requirements during onboarding.
- [ ] Document how to request JetStream-enabled capabilities and associated resources.
- [ ] Define the information required to evaluate the use case (delivery requirements, retention needs, consumers, etc.).
- [ ] Document how approved messaging capabilities and configurations are communicated to the application team.

## 3. Local Setup

### Usage
- [ ] Document the supported approaches for running NATS locally for development and testing.
- [ ] Define how developers can configure their applications to connect to a local NATS instance.
- [ ] Document local testing approaches for Core NATS and JetStream-based use cases.
- [ ] Define guidelines for isolating test subjects, data, and messaging resources.
- [ ] Provide sample configurations and reference implementations.

### Process to Enable
- [ ] Document the prerequisites and setup instructions for local development.
- [ ] Define how developers obtain approved local setup resources, tools, and configurations.
- [ ] Explain how local environments should differ from shared or production environments.
- [ ] Define the process for validating application behavior before requesting shared-environment access.

## 4. Security — Usage Patterns and Enablement

### Usage
- [ ] Document the authentication mechanisms supported by the platform, such as username/password, token, NKEY, JWT credentials, and TLS client certificates, where applicable.
- [ ] Define the approved authentication mechanism for each application environment.
- [ ] Document how applications should establish authenticated and TLS-protected connections.
- [ ] Define subject-level authorization expectations for publishing and subscribing.
- [ ] Establish guidelines for secure credential handling, rotation, and avoiding exposure of secrets.
- [ ] Provide approved connection and authentication examples for supported client libraries.

### Process to Enable
- [ ] Document the process for requesting application identity and NATS access.
- [ ] Define the information required for access provisioning, including application identity, environment, and subject permissions.
- [ ] Document how credentials and connection configuration are securely delivered to application teams.
- [ ] Define the process for requesting permission changes, credential rotation, and revocation.
- [ ] Identify the application-team and platform-team responsibilities for access management.

## 5. Observability — Usage Patterns and Enablement

### Usage
- [ ] Define the application-level metrics developers are expected to expose for NATS interactions.
- [ ] Establish structured logging and message correlation ID conventions.
- [ ] Define the standard approach for distributed trace propagation through NATS.
- [ ] Document how producers inject trace context into messages and consumers extract it.
- [ ] Define tracing guidance for Request/Reply, Publish/Subscribe, and JetStream-based asynchronous processing.
- [ ] Document how trace context should be handled across retries, redelivery, and message replay.
- [ ] Provide OpenTelemetry instrumentation examples for supported client libraries.

### Process to Enable
- [ ] Document the prerequisites for integrating applications with the organization's observability platform.
- [ ] Define how developers enable or configure approved NATS instrumentation.
- [ ] Document the process for accessing application-level dashboards and traces.
- [ ] Define the information developers should provide when requesting observability support.
- [ ] Clarify which telemetry capabilities are provided by the platform and which must be instrumented by the application team.

## 6. Subject Naming — Standards and Enablement Process

### Usage
- [ ] Define the organization-wide subject namespace and naming conventions.
- [ ] Establish standards for domain, application, event, command, and operation naming.
- [ ] Define conventions for token ordering, casing, delimiters, and identifiers.
- [ ] Establish guidelines for wildcard subscriptions and subject ownership.
- [ ] Document naming considerations for Core NATS and JetStream-enabled use cases.
- [ ] Provide valid and invalid subject naming examples.

### Process to Enable
- [ ] Document how developers propose and register new subjects or namespaces.
- [ ] Define the information required when requesting subject access or a new namespace.
- [ ] Establish the review and approval process for subject naming and ownership.
- [ ] Document how subject permissions are mapped to application identities.
- [ ] Define the process for changing existing subject names and coordinating changes with producers and consumers.

## 7. Message Contract and Schema

### Usage
- [ ] Define the supported message serialization formats and content types.
- [ ] Establish standard message envelope conventions, including event type, message ID, timestamp, producer, and schema version where applicable.
- [ ] Define schema ownership and responsibility for maintaining message contracts.
- [ ] Establish schema compatibility and evolution guidelines.
- [ ] Define conventions for required fields, optional fields, validation, and unsupported schema versions.
- [ ] Document payload size guidance and the approach for handling large payloads.
- [ ] Provide canonical message examples and reference implementations.

### Process to Enable
- [ ] Document how developers define and share message contracts before integration.
- [ ] Define the review or registration process for new schemas and contract changes.
- [ ] Establish how producers and consumers coordinate schema changes.
- [ ] Document how developers discover existing message contracts and identify their owners.
- [ ] Define the process for communicating breaking changes and coordinating application migration.

---

## Document-Level Review Checklist

- [ ] The guide focuses on organizational standards, supported usage patterns, and developer enablement.
- [ ] Each section distinguishes usage guidance from the process to enable the capability.
- [ ] Platform-managed infrastructure and operational responsibilities are abstracted from developers.
- [ ] Mandatory standards are distinguished from recommendations.
- [ ] Each section provides relevant examples, references, or links to supporting documentation.
- [ ] Application-team and platform-team responsibilities are clearly defined.
- [ ] The guide identifies ownership and a process for reviewing and updating standards.
