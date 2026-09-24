# Changelog

All notable changes to the NATS reference evaluation platform will be documented in this file.

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

- **Change:** Created `docs/developer-docs/usage-patterns/core-nats.md` structured into 3 distinct parts: 1) Publishing Patterns, 2) Subscribing Patterns & Lifecycle, and 3) Architectural Messaging Patterns.
- **Reason:** Organize each NATS SDK capability into an independent, scannable guide with point-wise generic descriptions and Go SDK examples.
- **Affected Area:** `docs/developer-docs/usage-patterns/core-nats.md`







