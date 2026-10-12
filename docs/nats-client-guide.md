# NATS Messaging & Client SDK Developer Guide

## 1. Overview

This guide is the entry point for application developers learning how to use NATS messaging and manage client connections.

It introduces foundational messaging concepts (subjects, message structure) shared across Core NATS and JetStream, provides practical reference patterns for initializing and managing NATS client connections in Go and Java, and guides developers through production resilience best practices and enterprise platform connectivity requirements.

The guide is an implementation and navigation reference. It does not replace the official NATS documentation or define enterprise-specific security policies.

## 2. Subjects

A subject is the address to which a NATS message is published. Subscribers use subject patterns to identify the messages they want to receive.

NATS subjects consist of tokens separated by periods (`.`). Wildcards allow subscriptions to match multiple subjects without requiring a separate subscription for every subject.

Developers should understand:

- Subject token hierarchy and separators.
- Exact subject matching.
- Single-token (`*`) and multi-token (`>`) wildcards.
- How subscription patterns determine which messages are received.

**Official reference:** [Core NATS - Subjects and Wildcards](https://docs.nats.io/learn/core-nats/subjects-and-wildcards)

For enterprise subject naming conventions and namespace design, refer to the [Subject Design Guide](subject-design.md).

## 3. Message Structure

A NATS message carries application data and the metadata needed to route or process it.

The main elements are:

- **Subject:** Identifies the message's destination or routing address.
- **Payload:** Contains the application data. Its format and schema are defined by the application.
- **Headers (optional):** Carry metadata associated with the message.
- **Reply subject (optional):** Provides a reply destination, commonly used in request-reply messaging.

Headers and payload serve different purposes. Headers carry message metadata, while the payload contains the application data being exchanged. NATS does not automatically define an application's business payload schema.

**Official reference:** [Core NATS - Message Headers](https://docs.nats.io/learn/core-nats/headers)

For enterprise requirements concerning message envelopes, payload schemas, and metadata conventions, refer to Message Envelope and Schema Management.

## 4. Prerequisites and SDK Setup

### 4.1 Go NATS Client SDK

**Prerequisites:** Go 1.22+

Install the official Go client:

```bash
go get github.com/nats-io/nats.go
```

Reference: [NATS Go client - Installation and usage](https://github.com/nats-io/nats.go#installation)

### 4.2 Java NATS Client SDK

**Prerequisites:** Java 17+

**Maven**

```xml
<dependency>
    <groupId>io.nats</groupId>
    <artifactId>jnats</artifactId>
    <version>2.20.0</version>
</dependency>
```

**Gradle**

```groovy
implementation 'io.nats:jnats:2.20.0'
```

Reference: [NATS Java client - Installation and usage](https://github.com/nats-io/nats.java#installation)

Use the approved dependency version for your application. When consulting Javadocs hosted under `latest`, verify that the documented API is available in the version you use.

## 5. Connect to NATS

Start with the official [Core NATS - Connecting guide](https://docs.nats.io/learn/core-nats/connecting).

This is the entry point for understanding how a client connects to a NATS server and establishes a connection for publishing and subscribing.

The client library handles the NATS protocol handshake. Application code should use the SDK APIs rather than implement the protocol directly.

For production connection configuration, including multiple server URLs and connection timeouts, continue to the resilient-client learning path below.

## 6. Build a Resilient NATS Client

Once basic connectivity is understood, follow the official [Resilient Clients Deep Dive](https://docs.nats.io/learn/resilient-clients/).

This is the primary learning path for production connection behavior. It builds on a working connection and introduces resilience mechanisms progressively rather than treating each connection option as an isolated feature.

### 6.1 Recommended Reading Path

| Order | Official reference | What developers should learn |
|---|---|---|
| 1 | [Connecting](https://docs.nats.io/learn/resilient-clients/connecting) | Connection naming, server pools, connect timeouts, and the connection handshake. |
| 2 | [Reconnection](https://docs.nats.io/learn/resilient-clients/reconnection) | Automatic reconnection, retry behavior, backoff, and client behavior when a server becomes unavailable. |
| 3 | [Connection Events](https://docs.nats.io/learn/resilient-clients/connection-events) | Connection lifecycle callbacks, state inspection, and application responses to disconnects, reconnects, and closure. |
| 4 | [Drain & Shutdown](https://docs.nats.io/learn/resilient-clients/drain-and-shutdown) | Graceful shutdown, draining pending work, and handling connection termination. |

## 7. Go and Java SDK Reference

Use this section as a quick lookup when implementing concepts covered in Sections 5 and 6.

| Development task | Go reference | Java reference |
|---|---|---|
| SDK installation and basic usage | [NATS Go client](https://github.com/nats-io/nats.go) | [NATS Java client](https://github.com/nats-io/nats.java) |
| Connection initialization and options | [Go Connect API](https://pkg.go.dev/github.com/nats-io/nats.go#Connect) | [Java client - Connecting](https://github.com/nats-io/nats.java#connecting) |
| Connection status and lifecycle handling | [Go Status API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Status) | [Java client - Clusters & Reconnecting](https://github.com/nats-io/nats.java#clusters--reconnecting) |
| Authentication and TLS configuration | [Go client - Authentication](https://github.com/nats-io/nats.go#new-authentication-nkeys-and-user-credentials) | [Java client - Connection Security](https://github.com/nats-io/nats.java#connection-security) |
| Graceful shutdown and draining | [Go `Conn.Drain` API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Drain) | [Java `Connection.drain` API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html#drain-java.time.Duration-) |
| JetStream API initialization | [Go JetStream documentation](https://github.com/nats-io/nats.go#jetstream) | [Java JetStream documentation](https://github.com/nats-io/nats.java#jetstream) |
| Runnable examples | [NATS by Example](https://natsbyexample.com/) | [Java NATS examples](https://github.com/nats-io/java-nats-examples) |

**Reference guidance**

- Use the official learning pages to understand concepts and recommended behavior.
- Use SDK documentation to look up language-specific APIs, option names, and version-specific details.
- Use runnable examples to validate an implementation pattern.
- Confirm the API against the dependency version approved for your application, particularly when a reference points to a repository's current default branch or the latest Javadocs.

The API reference table is intentionally secondary to the learning path. Developers should not need to read every SDK API page before building a basic client.

## 8. Key Connection and Resilience Recommendations

The following principles consolidate foundational connection practices and client-side resilience into a unified implementation checklist:

- **Connection reuse:** Open a single, long-lived client connection during application startup and share it across application components instead of creating connections per request or message.
- **Connection naming:** Assign a meaningful service or application name to each connection to improve operational visibility in cluster logs and monitoring tools.
- **Prefer built-in reconnection:** Rely on the SDK's native reconnection logic, backoff, jitter, and server-pool rotation rather than implementing custom reconnect loops.
- **Handle lifecycle events:** Register event callbacks or listeners for disconnect, reconnect, and closed events to inform application health checks and alerting; use state inspection for point-in-time readiness checks.
- **Graceful shutdown and draining:** Use SDK drain methods (`Drain()`) during controlled termination (such as on SIGTERM) to process buffered messages and complete in-flight work before connection closure.
- **Isolate connection and messaging errors:** A connected client can still encounter errors on individual publishes, subscriptions, subject permissions, or JetStream operations. Handle and log connection and operation failures distinctly.
- **Bound resource consumption:** Apply appropriate pending-message limits on subscriptions to catch slow consumers early and prevent unbounded memory growth.
- **Design retries intentionally:** A timeout does not confirm whether an operation was executed by a receiver. Evaluate message idempotency and duplicate side effects before configuring automatic retries.
- **Decouple authentication and transport security:** Treat authentication (verifying client identity via tokens or credentials) and TLS (securing transport and validating server certificates) as separate concerns per platform security policies.

## 9. Platform Connection Requirements

Before connecting to a shared or enterprise NATS environment, consult the relevant platform onboarding documentation.

The platform documentation is the source of truth for environment-specific configuration, including:

- **Endpoint and connectivity:** Approved NATS endpoint or server list, network access requirements, and environment-specific connection details.
- **Authentication:** Approved authentication mechanism and how credentials are issued, delivered, loaded, and rotated.
- **TLS:** Required transport security, trusted certificate authorities, and any client certificate requirements.
- **Authorization:** Permitted subjects and access to the required streams, consumers, and JetStream API operations.
- **Operational configuration:** Platform guidance for client naming, timeouts, reconnection, graceful shutdown, and observability.

Do not infer production endpoints, credentials, certificate paths, or authorization rules from local examples in the official NATS documentation.

The official [NATS Security documentation](https://docs.nats.io/learn/security) explains the underlying security mechanisms. Platform onboarding documentation defines how those mechanisms are applied in your environment.

## 10. JetStream Client Access

Applications that use JetStream must initialize the JetStream API through the established client connection using the relevant SDK.

Start with the SDK-specific references:

- [Go JetStream documentation](https://github.com/nats-io/nats.go#jetstream)
- [Java JetStream documentation](https://github.com/nats-io/nats.java#jetstream)

This guide covers only the client-connection prerequisite and API initialization entry point. For streams, consumers, persistence, acknowledgments, publishing guarantees, and consumption patterns, follow the dedicated JetStream and application messaging guides.

JetStream API initialization does not itself guarantee that an application is authorized to access a particular stream, consumer, or operation. Verify the permissions and resource onboarding requirements for the target environment.

## 11. Next Steps

Choose the appropriate messaging model based on your application's delivery, persistence, and processing requirements:

- [Core NATS Developer Guide](developer-guide/usage-patterns/core-nats.md) - Implementation of Core NATS messaging (ephemeral publish/subscribe, request-reply, and load-balanced queue groups).
- [JetStream Developer Guide](developer-guide/usage-patterns/jetstream.md) - Implementation of persistent messaging with JetStream (streams, consumers, deduplication, and acknowledgements).