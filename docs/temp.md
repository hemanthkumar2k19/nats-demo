
# NATS Developer Guide — Checklist

## 1. NATS Fundamentals for Application Developers

- [ ] **Required:** Explain Core NATS vs. JetStream and their delivery semantics.
- [ ] **Required:** Explain publishers, subscribers, subjects, streams, and consumers at the application level.
- [ ] **Required:** Explain the difference between transient messages and persisted messages.
- [ ] **Recommended:** Explain how NATS decouples services through subject-based communication.

## 2. Subject Namespace and Naming Design

- [ ] **Required:** Define a standard subject hierarchy and naming convention.
- [ ] **Required:** Establish namespace ownership by domain, application, and event or operation.
- [ ] **Required:** Define conventions for commands, events, requests, and responses.
- [ ] **Required:** Define rules for token ordering, casing, delimiters, and identifiers.
- [ ] **Required:** Define wildcard subscription usage and restrictions.
- [ ] **Required:** Avoid embedding sensitive data, large payloads, or rapidly changing values in subjects.
- [ ] **Recommended:** Document examples of valid and invalid subject names.
- [ ] **Recommended:** Define how subject names relate to platform-managed stream configurations.

## 3. Messaging Pattern Selection

- [ ] **Required:** Define criteria for choosing publish-subscribe, request-reply, queue groups, and JetStream-based messaging.
- [ ] **Required:** Document when Core NATS is appropriate and when persistence through JetStream is needed.
- [ ] **Required:** Define when competing consumers and work distribution are appropriate.
- [ ] **Recommended:** Document fan-out, service-to-service communication, event notification, and asynchronous workflow examples.

## 4. Message Contract and Payload Design

- [ ] **Required:** Define supported serialization formats and content types.
- [ ] **Required:** Establish message envelope conventions, including event type, event ID, timestamp, producer, and schema version where applicable.
- [ ] **Required:** Define schema ownership, compatibility, and evolution rules.
- [ ] **Required:** Define required versus optional fields and validation expectations.
- [ ] **Required:** Establish payload size guidance and handling for large payloads.
- [ ] **Recommended:** Document canonical examples for each message category.
- [ ] **Recommended:** Define how to handle unknown fields and unsupported schema versions.

## 5. Publishing Messages

- [ ] **Required:** Use approved NATS client libraries and organization-supported wrappers where provided.
- [ ] **Required:** Define publish acknowledgment expectations for Core NATS and JetStream.
- [ ] **Required:** Define error handling for failed publishes, timeouts, and unavailable services.
- [ ] **Required:** Define retry behavior, including bounded retries and backoff.
- [ ] **Required:** Define message identity and deduplication practices for operations that may be retried.
- [ ] **Recommended:** Document synchronous and asynchronous publishing patterns and their trade-offs.

## 6. Subscribing and Consuming Messages

- [ ] **Required:** Define subscription types and selection criteria.
- [ ] **Required:** Define consumer ownership, naming, and reuse conventions.
- [ ] **Required:** Explain durable versus ephemeral consumption from the developer's perspective.
- [ ] **Required:** Define acknowledgment behavior and when a message may be redelivered.
- [ ] **Required:** Define processing concurrency, backpressure, and in-flight message considerations.
- [ ] **Required:** Define graceful shutdown and subscription cleanup behavior.
- [ ] **Recommended:** Document pull versus push consumer selection guidance.
- [ ] **Recommended:** Explain replay and recovery expectations for persisted messages.

## 7. Delivery Guarantees and Idempotent Processing

- [ ] **Required:** Explain at-most-once behavior in Core NATS and the relevant JetStream delivery guarantees.
- [ ] **Required:** Require idempotent handlers where redelivery or retries can repeat processing.
- [ ] **Required:** Define acknowledgment timing relative to successful business processing.
- [ ] **Required:** Document duplicate detection and safe retry strategies.
- [ ] **Recommended:** Provide examples of idempotency keys and deduplication at the application boundary.

## 8. Failure Handling and Recovery

- [ ] **Required:** Define handling for processing errors, negative acknowledgments, and message redelivery.
- [ ] **Required:** Define retry limits and escalation behavior in coordination with platform capabilities.
- [ ] **Required:** Define how poison messages and repeatedly failing messages are handled.
- [ ] **Required:** Document timeout handling for request-reply interactions.
- [ ] **Required:** Define behavior when a downstream dependency is unavailable.
- [ ] **Recommended:** Document replay-based recovery and operational escalation paths.

## 9. Application Security and Access

- [ ] **Required:** Use platform-issued credentials and approved connection configuration.
- [ ] **Required:** Explain the NATS authentication mechanisms supported by the platform and their intended usage.
- [ ] **Required:** Document connection authentication options, including username/password, token, NKEY, JWT credentials, and TLS client certificates, where supported.
- [ ] **Required:** Define the platform-approved authentication mechanism for each application deployment environment.
- [ ] **Required:** Provide connection examples using approved credentials and supported client libraries.
- [ ] **Required:** Explain the distinction between authentication (identity verification), authorization (subject-level permissions), and TLS (transport security).
- [ ] **Required:** Require TLS for connections wherever mandated by platform security policy.
- [ ] **Required:** Define secure credential storage, retrieval, rotation, and revocation expectations for application teams.
- [ ] **Required:** Establish least-privilege publishing and subscribing permissions, including restrictions on wildcard subscriptions.
- [ ] **Required:** Explain how applications should handle authentication failures, expired credentials, and authorization errors.
- [ ] **Required:** Prohibit credentials, secrets, and sensitive information in message payloads or subjects unless explicitly approved.
- [ ] **Required:** Define handling requirements for sensitive data and encryption expectations.
- [ ] **Recommended:** Provide language-specific examples for connecting to NATS using the approved authentication method.
- [ ] **Recommended:** Explain the distinction between NATS user credentials and platform-managed application identity.

## 10. Application Configuration and Connectivity

- [ ] **Required:** Use the platform-approved connection endpoints and configuration mechanism.
- [ ] **Required:** Define how applications load credentials and environment-specific settings.
- [ ] **Required:** Document connection timeout, reconnect, and shutdown expectations.
- [ ] **Required:** Define supported client versions and dependency management practices.
- [ ] **Recommended:** Provide standard configuration examples for local development, testing, and deployed environments.

## 11. Application-Level Observability

- [ ] **Required:** Define useful application metrics, including publish failures, processing errors, latency, throughput, and redelivery indicators.
- [ ] **Required:** Define structured logging conventions and correlation identifiers.
- [ ] **Required:** Define trace propagation and correlation across asynchronous message boundaries.
- [ ] **Required:** Define how distributed trace context is propagated through NATS messages.
- [ ] **Required:** Establish the standard trace propagation format, such as W3C Trace Context, subject to organizational standards.
- [ ] **Required:** Define how producers inject trace context into message headers.
- [ ] **Required:** Define how consumers extract trace context and create or continue processing spans.
- [ ] **Required:** Define tracing behavior for request-reply, publish-subscribe, queue groups, and JetStream-based asynchronous processing.
- [ ] **Required:** Explain how to preserve trace context across retries, redelivery, and message replay.
- [ ] **Required:** Define how message correlation IDs and business identifiers should be used alongside trace IDs.
- [ ] **Required:** Establish rules for avoiding sensitive information in trace attributes, baggage, and logs.
- [ ] **Required:** Establish dashboards or instrumentation expectations for application-owned behavior.
- [ ] **Required:** Define which failures require application-team action versus platform-team escalation.
- [ ] **Recommended:** Provide OpenTelemetry instrumentation examples for supported NATS client libraries.
- [ ] **Recommended:** Provide an end-to-end example showing trace propagation from an HTTP request, through a NATS publisher, to a consumer and downstream service.
- [ ] **Recommended:** Explain how developers can troubleshoot missing spans, broken trace relationships, and incomplete traces.
- [ ] **Recommended:** Explain when to use parent-child span relationships versus span links, particularly for asynchronous processing, fan-in, and message replay.

## 12. Performance and Resource-Aware Design

- [ ] **Required:** Define payload size and message rate guidance for application teams.
- [ ] **Required:** Explain batching, concurrency, and flow-control considerations where relevant.
- [ ] **Required:** Define how applications should avoid unbounded buffering and uncontrolled retries.
- [ ] **Recommended:** Provide workload-based guidance for choosing pull batch sizes and processing concurrency.
- [ ] **Recommended:** Explain how subject proliferation and unnecessary subscriptions can affect system design.

## 13. Testing and Local Development

- [ ] **Required:** Define how to run application tests without depending on production NATS resources.
- [ ] **Required:** Define unit, integration, and contract testing expectations.
- [ ] **Required:** Include tests for duplicate delivery, retries, timeouts, and consumer restarts where applicable.
- [ ] **Required:** Define how test subjects and test resources are isolated and cleaned up.
- [ ] **Recommended:** Provide reusable test fixtures, sample messages, and local NATS/JetStream setup guidance.

## 14. Application Onboarding and Delivery

- [ ] **Required:** Define the application onboarding workflow and information developers must supply.
- [ ] **Required:** Document how to request subject access and required platform-managed messaging resources.
- [ ] **Required:** Define the application team's responsibilities for configuration, testing, and deployment readiness.
- [ ] **Required:** Document escalation routes and the information to include in a support request.
- [ ] **Recommended:** Provide a pre-production readiness checklist and example onboarding request.

## 15. Reference Implementations and Anti-Patterns

- [ ] **Required:** Provide approved examples for publish-subscribe, request-reply, queue groups, and JetStream consumption.
- [ ] **Required:** Document common mistakes, such as assuming Core NATS persists messages, acknowledging before processing succeeds, and implementing unsafe retries.
- [ ] **Required:** Document prohibited or discouraged patterns specific to organizational standards.
- [ ] **Recommended:** Provide reference implementations in the organization's supported languages.
- [ ] **Recommended:** Include troubleshooting scenarios and frequently asked questions.

---

## Developer Guide Review Checklist

- [ ] The guide clearly separates developer responsibilities from platform-managed responsibilities.
- [ ] All mandatory organizational standards are explicitly identified.
- [ ] Recommendations are distinguished from mandatory requirements.
- [ ] Authentication examples use approved mechanisms and do not expose real credentials.
- [ ] Subject naming conventions include valid and invalid examples.
- [ ] Messaging pattern selection includes practical decision criteria.
- [ ] Delivery guarantees, acknowledgments, retries, and idempotency are explained.
- [ ] Trace propagation guidance covers both producers and consumers.
- [ ] Application-level observability requirements are documented.
- [ ] Reference implementations align with the supported client libraries and platform interfaces.
- [ ] Platform-specific operational procedures are referenced rather than duplicated.
- [ ] The guide has an owner, review process, and versioning convention.
