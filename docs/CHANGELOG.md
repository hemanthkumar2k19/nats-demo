# Changelog

All notable changes to the NATS reference evaluation platform will be documented in this file.

## [2026-09-30] Create NATS Publisher Guide

- **Change:** Refined `docs/nats-publisher-guide.md` into a language-agnostic reference document with Go code blocks (`github.com/nats-io/nats.go` and `github.com/nats-io/nats.go/jetstream`). Standardized document heading hierarchy (H1 title, H2 main sections, H3 subsections). Added explicit code snippets demonstrating how optional reply subjects (`msg.Reply`) are populated on structured messages and used in `PublishMsg` and `PublishRequest`. Strictly enforced repository ASCII-only standards across diagrams and tables.
- **Reason:** Clarify optional reply subject usage in message construction and request-reply publishing.
- **Affected Area:** `docs/nats-publisher-guide.md`

## [2026-09-30] Create NATS Client & SDK Developer Guide

- **Change:** Completed `docs/developer-guide/client-guide.md` providing production Go SDK (`github.com/nats-io/nats.go`) and modern JetStream package (`github.com/nats-io/nats.go/jetstream`) reference code and best practices. Covers 8 core developer sections: Client Initialization, Connection Lifecycle, Connection Configuration, Connection Validation, JetStream API Context, Client Resource Management, Client Error Handling, and Client Testing.
- **Reason:** Provide developer reference documentation detailing Go SDK connectivity, connection pooling/reuse patterns, exponential backoff, and modern JetStream API context initialization.
- **Affected Area:** `docs/developer-guide/client-guide.md`

## [2026-09-30] Create SRE NATS Connectivity & Troubleshooting Guide

- **Change:** Refined `docs/nats-connectivity-guide.md` specifically for SRE teams. Formatted all document headings with hierarchical numerical notation (e.g., `1. Server Connectivity Check`, `1.1 DNS Resolution`, `5.3.1 Step 1 - Network`). Removed deployment startup instructions (designated as a prerequisite) and local internal document cross-references. Cleaned ChatGPT UTM tracking parameters from all official NATS reference URLs. Updated operational workflow using strict ASCII-only flow diagrams.
- **Reason:** Standardize document section hierarchy for improved readability, align SRE guide with post-deployment troubleshooting responsibilities, clean external links, and enforce repository ASCII standards.
- **Affected Area:** `docs/nats-connectivity-guide.md`

## [2026-09-29] Convert Guides List to Matrix Table

- **Change:** Converted the Guides list in `docs/temp.md` into a single consolidated Markdown table formatted with `Category`, `Guide`, `Status`, and `Owner` columns.
- **Reason:** Improve document readability and tracking for NATS onboarding guides.
- **Affected Area:** `docs/temp.md`

## [2026-09-29] Add Distributed Tracing REST API and Service Layer

- **Change:** Implemented W3C distributed trace propagation across HTTP REST API Layer (`services/api`), Publisher Service Layer (`services/publisher`), Consumer Subscriber (`services/consumer`), and core messaging operations (`services/messaging`). Encapsulated `InstrumentOutboundMessage` inside `messaging.PublishMsg` and `InstrumentInboundMessage` inside `messaging.Subscribe`. Configured OpenTelemetry `resource.WithAttributes` for `service.name` (`SERVICE_NAME` / `nats-tracing-demo`) in `services/telemetry/tracer.go`, and modularized runtime environment management into `services/config`.
- **Reason:** Provide reference implementation demonstrating end-to-end distributed tracing across HTTP and Core NATS messaging layers with modular environment configuration.
- **Affected Area:** `services/api`, `services/publisher`, `services/consumer`, `services/messaging`, `services/telemetry`, `services/config`, `services/main.go`, `services/go.mod`

## [2026-09-29] Add NATS Security Overview Guide

- **Change:** Refined `docs/developer-guide/security/overview.md` with architectural foundations covering the three core capabilities of NATS Security: Authentication (AuthN), Authorization (AuthZ), and Encryption. Grouped AuthN methods (Token, Username/Password, mTLS, NKey, JWT, Auth Callout), positioned Account Isolation under AuthZ alongside subject permissions, imports/exports, resource limits, and JetStream API permissions, and simplified Encryption to core mental boundaries (TLS/mTLS for data in transit and JetStream storage encryption for data at rest).
- **Reason:** Provide clear, high-level reference documentation for evaluating NATS security capabilities without implementation clutter.
- **Affected Area:** `docs/developer-guide/security/overview.md`

## [2026-09-28] Add NATS Metrics Architecture & Strategy Guide

- **Change:** Created `docs/developer-guide/observability/metrics.md` establishing the 3-tier enterprise metrics selection mental model (visualized via Mermaid flowchart: Native HTTP Endpoints = Master Catalogue -> NATS Surveyor = Primary Cluster Metrics -> Prometheus NATS Exporter = Per-Node Coverage Fallback), illustrating infrastructure and application metrics telemetry flow via a topological Mermaid diagram (distinguishing per-node Prometheus Exporters from the centralized per-cluster NATS Surveyor), and providing language-agnostic conceptual guidelines alongside Go OpenTelemetry metrics code examples.
- **Reason:** Provide developer documentation for evaluating NATS metrics collection strategies and OpenTelemetry application metrics instrumentation.
- **Affected Area:** `docs/developer-guide/observability/metrics.md`

## [2026-09-28] Add NATS Event Observability & Advisory Guide

- **Change:** Created `docs/developer-guide/observability/events.md` detailing event observability concepts, differentiating application domain events from NATS advisories (`$SYS.>` and `$JS.EVENT.ADVISORY.>`), illustrating telemetry flow via FluentBit `nats` input plugin subscribing to `$JS.EVENT.ADVISORY.STREAM.>` on port 4222 (including a link to official [Fluent Bit NATS Input Plugin Documentation](https://docs.fluentbit.io/manual/pipeline/inputs/nats)), and providing language-agnostic guidelines and Go SDK code examples for ingesting NATS advisories.
- **Reason:** Provide developer documentation for evaluating NATS event observability and JetStream system advisories.
- **Affected Area:** `docs/developer-guide/observability/events.md`

## [2026-09-28] Refactor NATS Logging Guide & Section Layout

- **Change:** Updated `docs/developer-guide/observability/logs.md` to relocate NATS Server Logging Configuration (`nats.conf`) to the start of Section 4 ("Establishing Correlation Between NATS Logs and Trace Context"), establishing a logical progression from general server log settings to header-based protocol tracing mechanics.
- **Reason:** Align server logging configuration with log-to-trace correlation setup.
- **Affected Area:** `docs/developer-guide/observability/logs.md`

## [2026-09-28] Add NATS Logging Architecture & Strategy Guide

- **Change:** Created `docs/developer-guide/observability/logs.md` detailing NATS Server log export options (stdout/stderr container log harvesting, dedicated file log rotation, native syslog output), evaluating the enterprise recommended stdout/filelog pattern, detailing application-side OTLP log instrumentation, and demonstrating trace-to-log correlation via Go `slog` OpenTelemetry context extraction.
- **Reason:** Provide reference documentation for NATS server logging strategies and application trace-log correlation.
- **Affected Area:** `docs/developer-guide/observability/logs.md`

## [2026-09-28] Add Unified Tracing Execution Lifecycle to Tracing Guide

- **Change:** Refactored `docs/developer-guide/observability/tracing.md` to center around the **Two Universal Tracing Operations** (`InstrumentOutboundMessage` for outbound context injection on publisher side and `InstrumentInboundMessage` for inbound context extraction on subscriber side), eliminating redundant code blocks across individual Core NATS and JetStream transport patterns while maintaining concise workflow applications and refining Section 7 matrix to map patterns directly to universal operations and OpenTelemetry `SpanKind` definitions.
- **Reason:** Simplify developer documentation by establishing a single best-fit operational pattern for all NATS publishing and subscribing scenarios.
- **Affected Area:** `docs/developer-guide/observability/tracing.md`

## [2026-09-28] Reorganize JetStream Usage Patterns Guide

- **Change:** Consolidated raw publish, structured headers, and optional `Nats-Msg-Id` deduplication under `Synchronous Publish` (1.1), and expanded `Consumer Lifecycle Management` (2.1.3) with code examples for both time-based automatic pause/resume and application logic-driven manual `PauseConsumer`/`ResumeConsumer` calls in `docs/developer-guide/usage-patterns/jetstream.md`.
- **Reason:** Clarify optional deduplication header behavior and document code-driven consumer pause/resume patterns.
- **Affected Area:** `docs/developer-guide/usage-patterns/jetstream.md`

## [2026-09-28] Reorganize Core NATS Usage Patterns Guide

- **Change:** Consolidated simple and structured publishing under `Publish` (1.1), consolidated normal and structured requests under `Request (Synchronous)` (1.3), and reordered subscribing sections into `Subscription Lifecycle & Flow Control` (2.1), `Synchronous Pull Subscription` (2.2), `Asynchronous Callback Subscription` (2.3), and `Channel Subscription` (2.4) in `docs/developer-guide/usage-patterns/core-nats.md`.
- **Reason:** Improve document presentation and group functionally equivalent SDK methods together.
- **Affected Area:** `docs/developer-guide/usage-patterns/core-nats.md`

## [2026-09-28] Add NATS Observability Setup & Deployment Guide

- **Change:** Created `docs/developer-guide/observability/setup.md` featuring local testing guidance using `grafana/otel-lgtm` container (`https://github.com/grafana/docker-otel-lgtm`), Mermaid architecture diagram, standalone Podman/Docker CLI commands, `docker-compose.yaml` snippets, and step-by-step instructions for checking Metrics (Prometheus/Mimir), Traces (Tempo), Logs (Loki), and Trace-to-Log correlation in Grafana UI (`http://localhost:3000`).
- **Reason:** Provide practical all-in-one local setup guidance for NATS observability testing and verification.
- **Affected Area:** `docs/developer-guide/observability/setup.md`

## [2026-09-28] Add NATS Distributed Tracing Developer Guide

- **Change:** Refactored `docs/developer-guide/observability/tracing.md` into a standalone, publication-ready guide: removed internal doc file links, rephrased messaging tracing concepts neutrally, converted W3C header definitions and pattern mapping matrices into clean language-agnostic tables, and structured pattern sections into language-agnostic concepts with Go SDK code sub-blocks.
- **Reason:** Standardize documentation for publication and multi-language SDK extensibility.
- **Affected Area:** `docs/developer-guide/observability/tracing.md`

## [2026-09-28] Add NATS Observability Architecture Overview & Mermaid Diagram

- **Change:** Created `docs/developer-guide/observability/overview.md` with a vendor-neutral architectural overview, Mermaid diagram, and detailed breakdown of the 4 telemetry pillars (Metrics, Traces, Logs, Events) routing through OpenTelemetry Collector to an Enterprise Observability Platform.
- **Reason:** Focus on architectural intent ("what to do") rather than execution tool implementations ("how to do"), decoupling design from backend vendors.
- **Affected Area:** `docs/developer-guide/observability/overview.md`

## [2026-09-28] Add NATS HTTP Monitoring Endpoints Reference Guide

- **Change:** Expanded `docs/developer-guide/observability.md` with structured analysis tables for all 12 NATS server HTTP monitoring endpoints (`/varz`, `/jsz`, `/connz`, `/accountz`, `/accstatz`, `/subsz`, `/routez`, `/leafz`, `/gatewayz`, `/raftz`, `/healthz`, `/debug/vars`), categorizing metrics group types and descriptions.
- **Reason:** Provide developer documentation for evaluating NATS observability metrics across standalone and clustered server nodes.
- **Affected Area:** `docs/developer-guide/observability.md`

## [2026-09-28] Fix JetStream PauseConsumer and ResumeConsumer Return Types

- **Change:** Corrected return types of `PauseConsumer` and `ResumeConsumer` to `(*jetstream.ConsumerPauseResponse, error)` in `services/messaging/js.go`.
- **Reason:** Return types defined in wrapper functions did not match the signature of methods provided by `jetstream.JetStream` interface in `github.com/nats-io/nats.go/jetstream`.
- **Affected Area:** `services/messaging/js.go`

## [2026-09-28] Fix JetStream PauseConsumer Signature and Add ResumeConsumer

- **Change:** Corrected `PauseConsumer` wrapper in `services/messaging/js.go` to invoke `js.PauseConsumer(ctx, stream, consumer, until)` on `jetstream.JetStream` instead of calling `cons.Pause(...)`. Added `ResumeConsumer` wrapper and updated code examples in `docs/developer-guide/usage-patterns/jetstream.md`.
- **Reason:** In NATS Go JetStream SDK (`github.com/nats-io/nats.go/jetstream`), consumer pause and resume are managed via `jetstream.JetStream` context methods, not methods on `jetstream.Consumer`.
- **Affected Area:** `services/messaging/js.go`, `docs/developer-guide/usage-patterns/jetstream.md`

## [2026-09-28] Add Single-Feature JetStream Operation Functions

- **Change:** Added explicit, comment-free demo wrapper functions for JetStream single-feature operations (`PublishJS`, `PublishMsgJS`, `PublishAsyncJS`, `PublishAsyncCompleteJS`, `CreateOrUpdateConsumer`, `OrderedConsumer`, `ConsumerInfo`, `PauseConsumer`, `DeleteConsumer`, `Stream`, `GetMsg`, `GetLastMsgForSubject`, `Consume`, `Fetch`, `FetchNoWait`, `NextMsgJS`, `AckMsg`, `NakMsg`, `NakMsgWithDelay`, `InProgressMsg`, `TermMsg`) matching `docs/developer-guide/usage-patterns/jetstream.md`.
- **Reason:** Provide concise single-feature helper functions in `services/messaging/js.go` for instructional and demonstration purposes.
- **Affected Area:** `services/messaging/js.go`

## [2026-09-28] Add Single-Feature Core NATS Operation Functions

- **Change:** Added explicit, comment-free demo wrapper functions for Core NATS single-feature operations (`Publish`, `PublishRequest`, `PublishMsg`, `Request`, `RequestMsg`, `FlushTimeout`, `Subscribe`, `SubscribeSync`, `NextMsg`, `ChanSubscribe`, `QueueSubscribe`, `AutoUnsubscribe`, `SetPendingLimits`, `DrainSubscription`, `Unsubscribe`, `Respond`, `NewRespInbox`) matching `docs/developer-guide/usage-patterns/core-nats.md`.
- **Reason:** Provide concise single-feature helper functions in `services/messaging/nats.go` for instructional and demonstration purposes.
- **Affected Area:** `services/messaging/nats.go`

## [2026-09-24] Fix NATS ConnectedServerJetStream Multi-Value Context Error

- **Change:** Unpacked `nc.ConnectedServerJetStream()` into `jsEnabled, _` before passing to `log.Printf` in `PrintConnectionDetails`.
- **Reason:** `nc.ConnectedServerJetStream()` returns `(bool, int)`. Passing a multi-value function directly as a variadic parameter in `log.Printf` causes a Go compiler error (`multiple-value in single-value context`).
- **Affected Area:** `services/natsclient/client.go`

## [2026-09-24] Add JetStream Client Options Reference Implementation

- **Change:** Updated `NewJSClient` to accept `opts ...jetstream.JetStreamOpt` and added `NewJSClientWithOptions` reference constructor in `services/natsclient/js.go`.
- **Reason:** Provide configurable JetStream initialization covering domain routing, API prefix overrides, async publish limits, error callbacks, and client API tracing.
- **Affected Area:** `services/natsclient/js.go`

## [2026-09-24] Compact Connectivity Guide Documentation

- **Change:** Updated `docs/developer-docs/connectivity.md` to use concise language-agnostic descriptions and simplified JetStream connection setup (removed options section).
- **Reason:** Provide a clean, compact, developer-friendly guide focusing on essential connectivity patterns across NATS Conn and JetStream.
- **Affected Area:** `docs/developer-docs/connectivity.md`

## [2026-09-24] Add Core NATS Usage Patterns Guide

- **Change:** Created `docs/developer-docs/usage-patterns/core-nats.md` structured into 3 distinct parts with language-agnostic headings and descriptions alongside Go SDK code implementations.
- **Reason:** Maintain consistent language-agnostic documentation standards across all usage pattern guides.
- **Affected Area:** `docs/developer-docs/usage-patterns/core-nats.md`


## [2026-09-25] Add Advanced Key-Value and Object Store Features to Store Guide

- **Change:** Updated `docs/developer-docs/usage-patterns/store.md` with Key Expiration (TTL), Tombstone Cleanup, and Object Symbolic Links with language-agnostic headings/descriptions and Go SDK code examples.
- **Reason:** Expand JetStream Store developer guide to cover transient state TTL, tombstone storage reclamation, and zero-byte object aliases.
- **Affected Area:** `docs/developer-docs/usage-patterns/store.md`

## [2026-09-25] Update NATS Server Token Authentication Syntax

- **Change:** Updated `deploy/local-nats/nats.conf` authorization block to use standard top-level token configuration syntax `token: "<token_string>"`.
- **Reason:** Ensure clean token authentication handshake without requiring user map wrapping, resolving `EOF` disconnect errors on client auth.
- **Affected Area:** `deploy/local-nats/nats.conf`













