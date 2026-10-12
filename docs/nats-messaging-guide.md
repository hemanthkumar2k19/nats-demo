# NATS Messaging

## 1. Overview

This page is the entry point for application developers learning how to use NATS messaging. It introduces the foundational concepts shared by Core NATS and JetStream and guides developers to the relevant implementation documentation.

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

## 4. Next Steps

To begin building applications with NATS, continue to the client connection and SDK setup guide:

- [NATS Client SDK Developer Guide](nats-client-guide.md) - SDK setup, connection lifecycle, reconnection, connection events, and graceful shutdown.