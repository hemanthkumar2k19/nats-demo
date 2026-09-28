# Changelog

All notable changes to the NATS reference evaluation platform will be documented in this file.

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













