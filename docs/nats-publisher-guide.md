# NATS Publisher Guide

## 1. Purpose

This guide provides application developers with a unified reference for publishing messages to NATS using Core NATS and JetStream.

It introduces the publishing patterns, capabilities, and operational considerations developers need when implementing publishers in Go or Java.

The guide follows a reference-oriented approach: each topic provides a brief description and links to the relevant SDK documentation or examples.

## 2. NATS Subjects

A NATS subject identifies the destination to which a message is published. Subjects consist of dot-separated tokens and provide the namespace used by publishers, subscribers, and JetStream.

Publishers must use fully specified subjects. Wildcards are used in subscription patterns and applicable JetStream subject filters, not to select a group of destinations when publishing.

Subject hierarchy can influence message routing, authorization, and JetStream stream filtering.

### 2.1 Learn the Fundamentals

| Topic | Description | Reference |
|---|---|---|
| Subject concepts | Understand subject syntax, tokens, hierarchy, and wildcard fundamentals. | [NATS Subjects](https://docs.nats.io/concepts/subjects) |
| Subjects and wildcards | Learn how subject patterns work for publishing and subscribing, with practical examples. | [Subjects and Wildcards](https://docs.nats.io/learn/core-nats/subjects-and-wildcards) |

### 2.2 Developer Considerations

Before publishing messages, developers should understand the following:

- **Concrete subjects:** Publish to a fully specified subject rather than a wildcard pattern.
- **Subject stability:** Treat established subjects as messaging contracts shared between publishers and consumers.
- **Message identifiers:** Keep per-message identifiers, such as order IDs and correlation IDs, in the payload or headers unless the application has an explicit design requirement otherwise.
- **Authorization:** Ensure the application's credentials permit publishing to the intended subject.
- **JetStream:** When publishing through JetStream, ensure the subject is covered by the intended stream configuration.

Detailed subject design and enterprise naming conventions are outside the scope of this guide and will be maintained separately.

## 3. Message Construction

A NATS message contains a destination subject and payload, with optional headers and a reply subject.

Developers should select the appropriate message representation and serialization format based on the application contract.

| Topic | Description | Go Reference | Java Reference |
|---|---|---|---|
| Basic message | Construct a message using a subject and payload. | [Pub-Sub by Example](https://natsbyexample.com/examples/messaging/pub-sub/go) / [nats-pub sample](https://github.com/nats-io/nats.go/blob/main/examples/nats-pub/main.go) | [Pub-Sub by Example](https://natsbyexample.com/examples/messaging/pub-sub/java) / [NatsPub sample](https://github.com/nats-io/nats.java/blob/main/src/examples/java/io/nats/examples/NatsPub.java) |
| Payload serialization | Serialize application objects into bytes before publishing. | [JSON by Example](https://natsbyexample.com/examples/messaging/json/go) / [Protobuf by Example](https://natsbyexample.com/examples/messaging/protobuf/go) | [JSON by Example](https://natsbyexample.com/examples/messaging/json/java) |
| Message headers | Attach metadata such as content type, correlation identifiers, and JetStream deduplication IDs. | [Header Documentation](https://docs.nats.io/learn/core-nats/headers) / [Header Type API](https://pkg.go.dev/github.com/nats-io/nats.go#Header) | [Header Documentation](https://docs.nats.io/learn/core-nats/headers) / [Headers Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/impl/Headers.html) |
| Reply subject | Specify where a response can be sent when using a request-reply interaction. | [Request-Reply by Example](https://natsbyexample.com/examples/messaging/request-reply/go) / [Conn.PublishMsg API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.PublishMsg) | [Request-Reply by Example](https://natsbyexample.com/examples/messaging/request-reply/java) / [Connection Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html) |

## 4. Core NATS Publishing

Core NATS supports lightweight, low-latency messaging. A publish operation does not wait for a server-side publish acknowledgement or confirmation that a subscriber processed the message.

Use Core NATS when its delivery semantics meet the application's requirements.

### 4.1 Publishing Patterns

| Publishing pattern | Description | Go Reference | Java Reference |
|---|---|---|---|
| Basic publish | Publish a payload to a concrete subject. | [Pub-Sub by Example](https://natsbyexample.com/examples/messaging/pub-sub/go) / [Conn.Publish API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Publish) | [Pub-Sub by Example](https://natsbyexample.com/examples/messaging/pub-sub/java) / [NatsPub sample](https://github.com/nats-io/nats.java/blob/main/src/examples/java/io/nats/examples/NatsPub.java) |
| Structured message publish | Publish a message with headers and, optionally, a reply subject. | [Header Documentation](https://docs.nats.io/learn/core-nats/headers) / [Conn.PublishMsg API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.PublishMsg) | [Header Documentation](https://docs.nats.io/learn/core-nats/headers) / [Connection Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html) |
| Request-reply | Send a request and wait for a response within a defined timeout. | [Request-Reply by Example](https://natsbyexample.com/examples/messaging/request-reply/go) / [Conn.Request API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Request) | [Request-Reply by Example](https://natsbyexample.com/examples/messaging/request-reply/java) / [NatsReq sample](https://github.com/nats-io/nats.java/blob/main/src/examples/java/io/nats/examples/NatsReq.java) |
| Flush | Wait for the server to process pending client operations. This does not confirm subscriber processing or durable storage. | [Conn.Flush API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Flush) / [Conn.FlushTimeout](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.FlushTimeout) | [Connection Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html) |
| Concurrent publishing | Publish from concurrent routines or threads using a shared connection where appropriate. | [Go Client Basic Usage](https://github.com/nats-io/nats.go#basic-usage) | [Java Client Basic Usage](https://github.com/nats-io/nats.java#basic-usage) |

### 4.2 Important Considerations

- Check client-side errors returned by publishing operations.
- Do not interpret a successful Core NATS publish call as proof that a subscriber received or processed the message.
- Use request-reply when an application requires a response rather than merely attaching a reply subject.
- Reuse established connections instead of creating a new connection for each message.
- Use JetStream when server-side persistence and publish acknowledgements are required.

## 5. JetStream Publishing

JetStream extends NATS messaging with persistence and server-side publish acknowledgements, subject to the configured stream and storage policies.

Applications should select synchronous or asynchronous publishing according to their latency, throughput, acknowledgement, and shutdown requirements.

### 5.1 Publishing Patterns

| Publishing pattern | Description | Go Reference | Java Reference |
|---|---|---|---|
| Synchronous publish | Publish a message and wait for the JetStream acknowledgement. | [JetStream README](https://github.com/nats-io/nats.go/blob/main/jetstream/README.md#basic-usage) / [Publisher.Publish API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Publisher.Publish) | [JetStream README](https://github.com/nats-io/nats.java#jetstream) / [NatsJsPub sample](https://github.com/nats-io/nats.java/blob/main/src/examples/java/io/nats/examples/NatsJsPub.java) |
| Asynchronous publish | Initiate publishing without blocking for each individual acknowledgement. | [Publisher.PublishAsync API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Publisher.PublishAsync) | [NatsJsPubAsync sample](https://github.com/nats-io/nats.java/blob/main/src/examples/java/io/nats/examples/NatsJsPubAsync.java) / [JetStream Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStream.html) |
| Publish acknowledgement | Inspect acknowledgement information, including the stream and message sequence. | [PubAck Type](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#PubAck) | [PublishAck Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/api/PublishAck.html) |
| Pending publish management | Track outstanding asynchronous publishes and account for them before shutdown. | [Publisher.PublishAsyncComplete API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Publisher.PublishAsyncComplete) | [JetStream Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStream.html) |

### 5.2 Deduplication

JetStream supports duplicate publish detection using the `Nats-Msg-Id` message header.

| Topic | Description | Go Reference | Java Reference |
|---|---|---|---|
| Message ID | Assign a stable identifier to a logical message when using JetStream deduplication. | [WithMsgID Option](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithMsgID) | [PublishOptions Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/PublishOptions.html) |
| Duplicate acknowledgement | Understand how JetStream indicates that a message was recognized as a duplicate. | [PubAck Type](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#PubAck) | [PublishAck Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/api/PublishAck.html) |

When retrying the same logical message, retain the same message ID. Deduplication applies within the configured duplicate-detection window; it is not a general guarantee of end-to-end exactly-once processing.

### 5.3 Publish Expectations

JetStream publish expectations allow publishers to assert conditions about stream state before a message is accepted. They can support optimistic concurrency control when multiple publishers update shared stream state.

| Topic | Description | Go Reference | Java Reference |
|---|---|---|---|
| Expected stream | Assert that a publish targets the intended stream. | [WithExpectStream Option](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithExpectStream) | [PublishOptions Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/PublishOptions.html) |
| Expected sequence | Assert a condition on the stream's last sequence or the relevant subject sequence. | [WithExpectLastSequence](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithExpectLastSequence) | [PublishOptions Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/PublishOptions.html) |
| Expectation failures | Handle rejected publishes when an expected condition no longer holds. | [APIError Struct](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#APIError) | [JetStreamApiException](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStreamApiException.html) |

Use publish expectations only when the application requires conditional writes or concurrency control. Verify the supported expectation options and error handling against the SDK version in use.

## 6. Failure Handling and Retry

Publish failures can arise from client-side validation, connection interruptions, authorization failures, JetStream configuration, resource limits, or acknowledgement timeouts.

The application must distinguish a known rejection from an outcome that remains unknown.

### 6.1 Failure Scenarios

| Scenario | Description and consideration | Go Reference | Java Reference |
|---|---|---|---|
| Client-side publish error | Handle errors reported directly by the client API. | [Conn.Publish API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Publish) | [Connection Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html) |
| Request timeout | A request-reply operation did not receive a response within its deadline. | [ErrTimeout Definition](https://pkg.go.dev/github.com/nats-io/nats.go#ErrTimeout) | [Connection Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html) |
| JetStream rejection | A publish was rejected because of an API error, configuration issue, or failed expectation. | [APIError Struct](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#APIError) | [JetStreamApiException](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStreamApiException.html) |
| Publish acknowledgement timeout | The acknowledgement was not received in time; the message may already have been stored. | [Publisher.Publish API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Publisher.Publish) | [JetStream Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStream.html) |
| Authorization failure | Verify that the application's identity has permission to publish to the intended subject. | [NATS Authorization Guide](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization) | [NATS Authorization Guide](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization) |

### 6.2 Retry Considerations

- **Use SDK reconnection:** Let the client SDK handle supported transport reconnection behavior.
- **Separate reconnection from message retry:** Re-establishing a connection does not guarantee that an earlier publish was unsuccessful or that it is safe to repeat.
- **Classify errors:** Avoid retrying permanent configuration or authorization failures as if they were transient network failures.
- **Handle unknown outcomes:** A JetStream acknowledgement timeout does not necessarily mean the server failed to store the message.
- **Use deduplication where appropriate:** Reuse the same `Nats-Msg-Id` for retries of the same logical message.
- **Bound retry behavior:** Apply appropriate deadlines, backoff, and error reporting to avoid uncontrolled retry loops.

### 6.3 SDK Resilience References

| Topic | Description | Go Reference | Java Reference |
|---|---|---|---|
| Initial connection retry | Configure behavior when the initial connection attempt fails. | [RetryOnFailedConnect](https://pkg.go.dev/github.com/nats-io/nats.go#RetryOnFailedConnect) | [Connecting Guide](https://github.com/nats-io/nats.java#connecting) |
| Reconnection | Configure reconnect attempts, delay, and related connection options. | [MaxReconnects Option](https://pkg.go.dev/github.com/nats-io/nats.go#MaxReconnects) / [ReconnectWait](https://pkg.go.dev/github.com/nats-io/nats.go#ReconnectWait) | [Clusters & Reconnecting](https://github.com/nats-io/nats.java#clusters--reconnecting) |
| Asynchronous publish errors | Handle errors from asynchronous JetStream publishing. | [WithPublishAsyncErrHandler](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithPublishAsyncErrHandler) | [ErrorListener Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/ErrorListener.html) |

## 7. Graceful Shutdown

Publishers should stop accepting new work before shutting down their NATS connection. Outstanding operations should be completed or explicitly accounted for within a bounded shutdown period.

A typical shutdown sequence is:

1. Stop accepting new publishing work.
2. Complete or account for application-level pending work.
3. Wait for outstanding JetStream asynchronous publishes where applicable.
4. Flush pending Core NATS client operations where applicable.
5. Drain the connection when graceful shutdown is required.
6. Close remaining resources and complete shutdown.

| Topic | Description | Go Reference | Java Reference |
|---|---|---|---|
| Async publish completion | Track outstanding JetStream publishes before terminating the application. | [Publisher.PublishAsyncComplete API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Publisher.PublishAsyncComplete) | [JetStream Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStream.html) |
| Flush | Allow pending client operations to be processed by the server before shutdown. | [Conn.Flush API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Flush) | [Connection Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html) |
| Connection drain | Gracefully process pending client work before closing the connection. | [Conn.Drain API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Drain) | [Connection Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html) |

Drain and flush do not independently guarantee message durability or subscriber processing. Applications should use bounded timeouts and handle shutdown failures explicitly.

## 8. Publisher Capability Summary

| Capability | Core NATS | JetStream |
|---|---|---|
| Publish to a subject | Supported | Supported |
| Message headers | Supported | Supported |
| Reply subject | Supported | Supported, with API-specific behavior |
| Server-side publish acknowledgement | Not provided | Provided |
| Server-side persistence | Not provided by Core NATS | Supported according to stream configuration |
| Asynchronous publishing | Client-side publish operations | Asynchronous publishing with acknowledgement tracking |
| Server-side duplicate detection | Not provided by Core NATS | Supported through message IDs and the configured duplicate-detection window |
| Conditional publishing | Not provided by Core NATS | Supported through publish expectations |
| Flush and graceful connection drain | Supported | Supported |

Choose the messaging model based on the application's delivery and persistence requirements rather than treating JetStream as a drop-in equivalent for every Core NATS operation.

## 9. Official References

### NATS Concepts and Platform Documentation

- [NATS Subjects](https://docs.nats.io/concepts/subjects)
- [NATS Subjects and Wildcards](https://docs.nats.io/learn/core-nats/subjects-and-wildcards)
- [NATS Core NATS Documentation](https://docs.nats.io/learn/core-nats/)
- [NATS JetStream Documentation](https://docs.nats.io/learn/jetstream/)
- [NATS Authorization](https://docs.nats.io/learn/security/authorization)

### Go SDK

- [Official NATS Go Client Repository](https://github.com/nats-io/nats.go)
- [NATS Go Client Examples](https://github.com/nats-io/nats.go/tree/main/examples)
- [NATS Go Client API](https://pkg.go.dev/github.com/nats-io/nats.go)
- [NATS Go JetStream API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)

### Java SDK

- [Official NATS Java Client Repository](https://github.com/nats-io/nats.java)
- [Official Java Client Examples](https://github.com/nats-io/nats.java/tree/main/src/examples/java/io/nats/examples)
- [Java NATS Examples Repository](https://github.com/nats-io/java-nats-examples)
- [NATS Java Client Javadocs](https://javadoc.io/doc/io.nats/jnats/latest/index.html)

### Cross-Client Reference Examples

- [NATS by Example](https://natsbyexample.com/)

Use the API documentation corresponding to the SDK version approved for your application. The latest documentation may describe capabilities not present in an older pinned dependency.