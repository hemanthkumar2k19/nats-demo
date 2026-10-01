# Changelog

All notable changes to the NATS reference evaluation platform will be documented in this file.

## [2026-10-01] Create NATS Core Subscription Guide

- **Change:** Created [docs/nats-subscription-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-subscription-guide.md) defining application-level Core NATS subscription practices. Converted section headings and descriptions to be language-agnostic for future multi-language SDK additions (Java, Python). Integrated Request-Reply Responder (`msg.Respond` / `msg.RespondMsg`) directly under Asynchronous Callback Subscriptions, and covered Buffer Options (`SetPendingLimits`), Failure & Error Semantics (`ErrSlowConsumer`, DLQ pattern, exception recovery), and Graceful Shutdown (`sub.Drain()`, `nc.Drain()`).
- **Reason:** Standardize developer documentation with language-agnostic conceptual headings while embedding specific Go SDK code blocks and comments for multi-language extensibility.
- **Affected Area:** [docs/nats-subscription-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-subscription-guide.md)

## [2026-10-01] Wire Subscription Lifecycle through Service Layer

- **Change:** Refactored [services/service/service.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/service/service.go) to wrap subscription lifecycle management (`CreateSubscription`, `ListSubscriptions`, `Unsubscribe`, `DrainSubscription`, `DrainAllSubscriptions`) using `subscription.Manager` and `messaging.Subscribe` with `messaging.OrderHandler`. Updated [services/api/subscription_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/subscription_handler.go) to delegate all subscription calls strictly to the `service.Service` layer.
- **Reason:** Enforce clean architectural layering (`SubscriptionHandler` -> `Service` -> `Subscription Manager` & `Messaging`).
- **Affected Area:** [services/api/subscription_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/subscription_handler.go), [services/service/service.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/service/service.go), [services/cmd/processing-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/processing-service/main.go), [services/cmd/order-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/order-service/main.go)

## [2026-10-01] Convert API Servers and Handlers to Gin Framework

- **Change:** Converted HTTP API server handlers in [services/api/publisher_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/publisher_handler.go) and [services/api/subscription_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/subscription_handler.go) to use the Gin Web Framework (`github.com/gin-gonic/gin`). Updated [services/cmd/order-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/order-service/main.go) and [services/cmd/processing-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/processing-service/main.go) to initialize Gin router engines and bind route groups. Added Gin dependency to [services/go.mod](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/go.mod).
- **Reason:** Migrate REST layer to Gin framework for structured routing and JSON binding capabilities.
- **Affected Area:** [services/api/publisher_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/publisher_handler.go), [services/api/subscription_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/subscription_handler.go), [services/cmd/order-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/order-service/main.go), [services/cmd/processing-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/processing-service/main.go), [services/go.mod](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/go.mod)

## [2026-10-01] Split API Handlers and Segregate Publisher and Processing Services

- **Change:** Refactored package `api` into modular domain handlers: [services/api/publisher_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/publisher_handler.go) for publishing endpoints (`/api/v1/publish`, `/api/v1/publish/request`) and [services/api/subscription_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/subscription_handler.go) for subscription management (`/api/v1/subscriptions`, `/api/v1/subscriptions/unsubscribe`, `/api/v1/subscriptions/drain`). Configured [services/cmd/order-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/order-service/main.go) as the dedicated publishing service and [services/cmd/processing-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/processing-service/main.go) as the consuming/subscription management service.
- **Reason:** Achieve clean separation of concerns between publishing side and consuming side of the platform while modularizing API routes.
- **Affected Area:** [services/api/publisher_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/publisher_handler.go), [services/api/subscription_handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/subscription_handler.go), [services/cmd/order-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/order-service/main.go), [services/cmd/processing-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/processing-service/main.go)

## [2026-09-30] Implement NATS Subscription Lifecycle Management APIs

- **Change:** Created [services/subscription/manager.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/subscription/manager.go) to maintain an in-memory thread-safe registry of `*nats.Subscription` handles. Updated [services/messaging/nats.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/messaging/nats.go) to return subscription pointers. Added REST APIs in [services/api/handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/handler.go) for creating (`POST /api/v1/subscriptions`), listing (`GET /api/v1/subscriptions`), unsubscribing (`POST /api/v1/subscriptions/unsubscribe`), and draining (`POST /api/v1/subscriptions/drain`) subscriptions. Integrated graceful subscription drain in [services/cmd/order-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/order-service/main.go).
- **Reason:** Provide full REST lifecycle control (create, list, unsubscribe, drain) over background NATS subscriptions.
- **Affected Area:** [services/subscription/manager.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/subscription/manager.go), [services/api/handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/handler.go), [services/messaging/nats.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/messaging/nats.go), [services/cmd/order-service/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/cmd/order-service/main.go)

## [2026-09-30] Convert NATS Client Event Handlers to Zerolog Logging

- **Change:** Refactored all connection event handlers (`ConnectHandler`, `DisconnectErrHandler`, `ReconnectHandler`, `ClosedHandler`, `DiscoveredServersHandler`, `ErrorHandler`) in [services/natsclient/client.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/natsclient/client.go) from `log.Printf` to Zerolog structured logging methods (`log.Info()`, `log.Warn()`, `log.Error()`).
- **Reason:** Standardize connection lifecycle and failover logging across the service using Zerolog structured logs.
- **Affected Area:** [services/natsclient/client.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/natsclient/client.go)

## [2026-09-30] Configure Zerolog Logger with Colorful Console Output

- **Change:** Created [services/logger/logger.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/logger/logger.go) to configure Zerolog (`github.com/rs/zerolog`) supporting both raw JSON output and human-friendly colorful console output via `zerolog.ConsoleWriter`. Updated [services/messaging/nats.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/messaging/nats.go) and [services/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/main.go) to use Zerolog structured logging.
- **Reason:** Provide colorful, high-performance structured JSON console logs for developer inspection and operations.
- **Affected Area:** [services/logger/logger.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/logger/logger.go), [services/messaging/nats.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/messaging/nats.go), [services/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/main.go), [services/go.mod](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/go.mod)

## [2026-09-30] Refine NATS Publish Log Message Formatting

- **Change:** Added `formatMsg(msg *model.Message) string` in [services/messaging/nats.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/messaging/nats.go#L14) to format published message fields and headers as clean JSON strings in log output instead of raw Go struct dumps (`%v`). Updated request-reply logs to format the response payload string cleanly (`string(resp.Data)`).
- **Reason:** Improve log readability and presentation when inspecting published message details and headers.
- **Affected Area:** [services/messaging/nats.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/messaging/nats.go)

## [2026-09-30] Add Domain-Based Route Registration and Dedicated Cleanup

- **Change:** Added `RegisterV1Routes(mux *http.ServeMux)` in [services/api/handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/handler.go) for domain-level route grouping. Refactored server shutdown logic in [services/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/main.go) into a dedicated `cleanup(httpServer *http.Server, nc *nats.Conn)` function.
- **Reason:** Improve modularity for scaling future API endpoints and encapsulate server resource teardown cleanly.
- **Affected Area:** [services/api/handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/handler.go), [services/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/main.go)

## [2026-09-30] Refactor Publisher APIs and Remove Tracing Code

- **Change:** Refactored [services/api/handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/handler.go) and [services/publisher/service.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/publisher/service.go) to provide clean Publisher APIs (`/api/v1/publish` and `/api/v1/publish/request`) backed by `messaging.PublishMsg`, `messaging.PublishMsgWithReply`, and `messaging.PublishRequest`. Removed OpenTelemetry trace propagation and extra endpoint handlers. Updated route registration in [services/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/main.go).
- **Reason:** Simplify publisher interface, align with core messaging functions in `services/messaging/nats.go`, and remove observability/tracing overhead.
- **Affected Area:** [services/api/handler.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/api/handler.go), [services/publisher/service.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/publisher/service.go), [services/messaging/nats.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/messaging/nats.go), [services/main.go](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/services/main.go)

## [2026-09-30] Expand Common NATS Error Reference Table in Publisher Guide

- **Change:** Removed the verbose Go error inspection code snippet from section `5.1 Publish Errors` in [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md). Expanded `#### Common NATS Error Reference` into a rich, comprehensive table covering Core NATS errors (`ErrNoResponders`, `ErrTimeout`, `ErrMaxPayload`, `ErrAuthorization`, `ErrConnectionClosed`, `ErrReconnectBufExceeded`) and JetStream API error codes (`10005` `ErrStreamNotFound`, `10054` `ErrStreamLimits`, `10071` `ErrStreamWrongLastSequence`, `10072` `ErrStreamWrongLastSubjectSequence`, `10073` `ErrStreamWrongLastMsgID`, `10075` `ErrStreamWrongStream`, `10077` `ErrDuplicate`, `10014` `ErrNoStreamResponse`, `10023` `ErrClusterUnavail`), complete with HTTP status codes, classifications, and remediation steps.
- **Reason:** Provide exhaustive, accurate reference documentation for NATS publish errors while eliminating redundant code blocks.
- **Affected Area:** [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md)

## [2026-09-30] Refine Section 5.2 (SDK Retry Policy & Client Options) in Publisher Guide

- **Change:** Refactored Section `5.2 Retry Policy` in [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md) to document native NATS SDK and Client options for retries, backoff, and buffering instead of custom application-level loop logic. Documented connection options (`RetryOnFailedConnect`, `MaxReconnects`, `ReconnectWait`, `CustomReconnectDelay`, `ReconnectJitter`, `ReconnectBufSize`), JetStream options (`WithPublishAsyncMaxPending`, `WithPublishAsyncErrHandler`, `PublishAsyncComplete`), and provided a Go code snippet showing how to configure these driver options.
- **Reason:** Align developer guide with NATS SDK driver capabilities for connection reconnects, backoff strategies, and async publish error callbacks.
- **Affected Area:** [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md)

## [2026-09-30] Refine Section 5 (Publish Errors & Retry Policy) in Publisher Guide

- **Change:** Added section `## 5. Failure Handling & Retry` to [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md) structured into `### 5.1 Publish Errors` and `### 5.2 Retry Policy`. Guided developers on error manifestation across Core NATS and JetStream, explained error code ambiguity/inspection methods (`jetstream.JetStreamError`, `APIError().ErrorCode`), detailed built-in SDK resilience vs developer-owned retry responsibilities, and provided a production Go retry pattern implementation (`BoundedPublishRetry`).
- **Reason:** Provide practical developer guidance on diagnosing NATS publish error codes and structuring production-grade SDK retry loops.
- **Affected Area:** [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md)

## [2026-09-30] Consolidate Failure Handling & Retry Sections in Publisher Guide

- **Change:** Restructured Sections 5 & 6 in [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md) into a single consolidated section `## 5. Failure Handling & Retry` with 5 standardized subsections (`5.1 Classify the Failure`, `5.2 Retryable Failures`, `5.3 Non-Retryable Failures`, `5.4 Unknown Publish Outcome`, `5.5 Retry Strategy`). Renumbered subsequent top-level sections: Graceful Shutdown (`6`), Publisher Capability Summary (`7`), and Official References (`8`).
- **Reason:** Simplify document navigation, eliminate redundant sections, and present a unified failure handling framework for NATS publishers.
- **Affected Area:** [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md)

## [2026-09-30] Expand JetStream Publish Expectations Section

- **Change:** Expanded section `4.5 Publish Expectations` in [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md). Added a comprehensive breakdown of Optimistic Concurrency Control (OCC), an ASCII sequence diagram showing evaluation workflow, a breakdown of expectation options (`WithExpectStream`, `WithExpectLastSequence`, `WithExpectLastSubjectSequence`, `WithExpectLastMsgID`), a Go SDK `jetstream` example asserting stream sequence and catching expectation mismatch error code 10071, and practical use cases.
- **Reason:** Provide clear conceptual and practical understanding of how JetStream prevents concurrent race conditions and out-of-order writes without distributed locks.
- **Affected Area:** [docs/nats-publisher-guide.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/docs/nats-publisher-guide.md)

## [2026-09-30] Create NATS Publisher Guide

- **Change:** Refined `docs/nats-publisher-guide.md` into a language-agnostic reference document with Go code blocks (`github.com/nats-io/nats.go` and `github.com/nats-io/nats.go/jetstream`). Standardized Section 2 with `###` subsections (`2.1 Subject`, `2.2 Payload`, `2.3 Headers`, `2.4 Reply Subject`) matching Section 1, 3, and 4. Fixed JetStream async publish channel handling in Go (`select { case ack := <-future.Ok(): ... case err := <-future.Err(): ... }`), and differentiated `PublishRequest` from synchronous `Request` / `RequestMsg` queries. Strictly enforced repository ASCII-only standards across diagrams and tables.
- **Reason:** Resolve logical code bugs and presentation inconsistencies across section headings.
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













