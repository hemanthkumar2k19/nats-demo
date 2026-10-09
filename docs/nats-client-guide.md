# NATS Client & SDK Developer Guide

This guide provides application developers with practical reference patterns for initializing, configuring, and managing NATS client connections and JetStream API access.

The concepts and descriptions are language-agnostic. Code examples feature the official NATS Go and Java SDKs.

## Prerequisites

### Go NATS Client SDK

- **Go Version:** Go 1.22+
- **SDK Installation:**

  ```bash
  go get github.com/nats-io/nats.go
  ```

### Java NATS Client SDK

- **Java Version:** Java 17+
- **Dependency Management (Maven):**

  ```xml
  <dependency>
      <groupId>io.nats</groupId>
      <artifactId>jnats</artifactId>
      <version>2.20.0</version>
  </dependency>
  ```

- **Dependency Management (Gradle):**

  ```groovy
  implementation 'io.nats:jnats:2.20.0'
  ```

## SDK Usage and Reference Guide

Use the following table to find the official SDK documentation and examples for each development task. Follow platform-provided connection, authentication, TLS, and configuration requirements.

| Development Task                | Description                                                                                   | Go SDK Reference                                                                                                 | Java SDK Reference                                                                                                                    |
| :------------------------------ | :-------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------ |
| **Installation**                | Install the official Go or Java client SDK.                                                   | [Installation](https://github.com/nats-io/nats.go#installation)                                                  | [Installation](https://github.com/nats-io/nats.java#installation)                                                                    |
| **Connection initialization**   | Establish a connection using a server URL or list of server URLs.                             | [Basic Usage](https://github.com/nats-io/nats.go#basic-usage)                                                    | [Connecting](https://github.com/nats-io/nats.java#connecting)                                                                         |
| **Cluster connectivity**        | Connect using multiple server URLs and understand server discovery and reconnection.          | [nats.Connect API](https://pkg.go.dev/github.com/nats-io/nats.go#Connect)                                         | [Clusters & Reconnecting](https://github.com/nats-io/nats.java#clusters--reconnecting)                                               |
| **Connection states**           | Inspect the current connection status and related APIs.                                       | [Status API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Status)                                          | [Connection.Status Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.Status.html)                      |
| **Lifecycle events**            | Register handlers for connection, disconnection, reconnection, closure, and server discovery. | [ConnHandler API](https://pkg.go.dev/github.com/nats-io/nats.go#ConnHandler)                                     | [ConnectionListener Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/ConnectionListener.html)                         |
| **Connection options**          | Configure server URLs, client name, timeout, reconnection, and event handlers.                | [Option API](https://pkg.go.dev/github.com/nats-io/nats.go#Option)                                               | [Connection Options](https://github.com/nats-io/nats.java#connection-options)                                                         |
| **Authentication and TLS**      | Configure authentication and secure transport according to platform requirements.             | [New Authentication](https://github.com/nats-io/nats.go#new-authentication-nkeys-and-user-credentials)          | [Connection Security](https://github.com/nats-io/nats.java#connection-security)                                                       |
| **Graceful shutdown**           | Close or drain connections during application termination.                                    | [Conn.Drain API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Drain)                                       | [Connection.drain Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html#drain-java.time.Duration-) |
| **JetStream initialization**    | Initialize the JetStream client API from an established connection.                           | [JetStream README](https://github.com/nats-io/nats.go#jetstream)                                                 | [JetStream README](https://github.com/nats-io/nats.java#jetstream)                                                                     |
| **Client verification & testing** | Explore runnable examples and verify API usage.                                               | [NATS by Example](https://natsbyexample.com/)                                                                    | [Java NATS Examples Repo](https://github.com/nats-io/java-nats-examples)                                                              |

Note: The Java Javadocs using `latest` may describe a newer SDK version than the one your application uses. Verify the APIs against your approved dependency version before adopting them.

## Connection Usage Guidelines

- **Reuse connections:** Establish a long-lived connection during application startup and share it across application components. Do not open a connection for every message or request.
- **Use SDK reconnection:** Prefer the client's built-in reconnection behavior instead of implementing a competing reconnect loop.
- **Handle lifecycle events:** Use callbacks for connection monitoring and state inspection for point-in-time checks. Handle messaging-operation errors separately.
- **Use platform-approved configuration:** Avoid overriding SDK defaults unless an application requirement justifies it.
- **Drain when required:** Use graceful draining during controlled shutdown when pending work must be handled before the connection closes.
- **Validate permissions separately:** Establishing a connection does not guarantee access to every subject, stream, consumer, or JetStream API operation.

## Platform Configuration

Before connecting to an environment, obtain the following from the platform onboarding documentation:

- Approved NATS endpoint or server list.
- Authentication mechanism and credential delivery method.
- TLS requirements and certificate configuration.
- Required subject permissions and JetStream resource access.
- Platform recommendations for client naming, connection timeouts, reconnection, and graceful shutdown.

## Next Steps