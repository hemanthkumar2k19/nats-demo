# NATS Documentation 

1. Understanding NATS
2. Requirement of Env
    1. Local Setup - Platform
    2. Request for Dev Environment Access - Platform
        1. Cluster Management - Platform
        2. Account Management - Platform
        3. Identity and Access Management - Platform
3. NATS Client Guide
    1. Authentication Guide. - Platform
    2. Connection Parameters Guide - Platform
4. NATS Subject Design
5. NATS Core Guide
    1. Subject Lifecycle Management - Platform
6. NATS Jetstream Guide
    1. Stream Lifecycle Management - Platform
        1. Stream Design
    2. Consumer Lifecycle Management - Platform
        1. Consumer Design
7. Observability
    1. Application Enablement Guide - Platform
    2. Request for Observability Enablement - Platform
        1. Observability Architecture


| Step | Document Brief/Purpose | Document Link |
|---|---|---|
| Step 1: Understanding NATS | Overview of NATS architecture, core capabilities, dual messaging models (Core NATS vs. JetStream), and Solace translation concepts. | |
| Step 2: Local Environment Setup | Guide for running single-node and 3-node clustered NATS instances locally using Podman, podman-compose, and the NATS CLI. | |
| Step 3: NATS Client SDK & Messaging Fundamentals | Foundational messaging concepts (subjects, messages), Go and Java SDK setup, connection lifecycle management, and resilience best practices. | |
| Step 4: Subject Design & Standards | Enterprise subject naming conventions, token hierarchy structure, wildcard semantics, and routing/access boundaries. | |
| Step 5: Core NATS Messaging Patterns | Implementation guide for in-memory, at-most-once messaging: Publish-Subscribe, Request-Reply, Queue Groups, and pattern best practices. | |
| Step 6: JetStream Persistent Messaging | Implementation guide for at-least-once persistent messaging: stream publishing (sync, async, deduplication, TTL) and pull/push consumer patterns. | |

