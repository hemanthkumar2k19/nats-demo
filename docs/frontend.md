# Frontend Guide

## 1. Purpose

The frontend is a simple developer-oriented UI for demonstrating and observing the NATS platform demo.

The UI should be inspired by the overall usability and visual approach of the MCP Inspector:

* Developer tool rather than a business application
* Clean and compact layout
* Clear status information
* Interactive controls
* Real-time visibility into activity
* Minimal navigation
* Easy to understand without prior frontend knowledge

The frontend should remain simple enough for a backend developer to understand and modify.

---

# 2. Technology

Use a simple React SPA.

Keep the frontend stack minimal.

Preferred approach:

* React
* TypeScript
* Simple CSS or a lightweight styling approach
* A simple icon library
* Browser APIs where sufficient

Avoid introducing complex frontend frameworks or state-management libraries unless there is a clear need.

The frontend communicates only with the Demo Service HTTP API.

It should not communicate directly with NATS.

```text
+----------------+
|   React SPA    |
+-------+--------+
        |
        | HTTP
        v
+----------------+
| Demo Service   |
+-------+--------+
        |
        | NATS
        v
+----------------+
|     NATS       |
+----------------+
```

---

# 3. UI Goals

The UI has two primary goals.

### 1. Show platform status

The developer should immediately be able to see:

* Whether NATS is available
* Which application services are active
* Which Processor Service instances are active
* Basic connection/health status

### 2. Show job processing

When a job is submitted, the developer should be able to see the activity as it happens.

Example:

```text
Job Submitted
     |
     v
NATS
     |
     v
Processor
     |
     v
Processing
     |
     v
Completed
```

The UI should make this flow visible without requiring the developer to inspect logs.

---

# 4. Main UI Layout

Use a single primary screen initially.

```text
+=======================================================================================================================================+
| [Header]                                                                                                                              |
|  NATS Platform Demo Console                                                                10:30:00    [ NATS CONNECTED ]            |
+=======================================================================================================================================+
| [Dynamic Alert Banner - success / error / warning]                                                                                   |
+=======================================================================================================================================+
| PLATFORM STATUS                                                                                                                       |
+---------------------------------------------------------------------------------------------------------------------------------------+
| NATS Server       [ Connected ]     Demo Service      [ Active ]     Processor Service   [ Active ]     Processing [ ON ]           |
| JetStream         [ Available ]     Stream: JOBS     Pending: 0     Workers: 1            Consumer: Active                           |
+=======================================================================================================================================+
| LEFT: DEMO ACTIONS                                      | RIGHT: LIVE OBSERVABILITY                                                   |
+---------------------------------------------------------+-----------------------------------------------------------------------------+
| [JobPanel]                                              | [ActivityPanel]                                                             |
| +-----------------------------------------------------+ | +-------------------------------------------------------------------------+ |
| | Submit Job                                          | | Activity Log                                                [Refresh] | |
| |                                                     | | Time     Job ID    Mode       Event        Subject      Worker  Dlv  | |
| | Delivery Mode: (o) Core NATS  ( ) JetStream         | | ----------------------------------------------------------------------- | |
| | Job ID:    [ job-101                 ]              | | 10:30:01 job-101   CORE       PUBLISHED    jobs.sub...  -       1    | |
| | Job Type:  [ image-processing        v ]            | | 10:30:01 job-101   CORE       RECEIVED     jobs.rec...  proc-1  1    | |
| | Payload:   { "file": "img.jpg"       }              | | 10:30:02 job-101   CORE       COMPLETED    jobs.com...  proc-1  1    | |
| | [ Submit Job ]                    [ Validate ]      | +-------------------------------------------------------------------------+ |
| +-----------------------------------------------------+                                                                               |
|                                                         | [JobInspectorPanel] (When Job ID is clicked)                                |
| [RequestReplyPanel]                                     | +-------------------------------------------------------------------------+ |
| +-----------------------------------------------------+ | | Job Details                                      [Close]              | |
| | Request / Reply                                    | | | Job ID: job-101        Status: COMPLETED                              | |
| | Subject: jobs.validate                             | | | Correlation ID: abc-123    Delivery Count: 1                          | |
| | Job ID:    [ job-val-101 ]                          | | | Status History: SUBMITTED -> PROCESSING -> COMPLETED                  | |
| | [ Send Request ]                                    | | | Raw Payload: { ... }                                                  | |
| | Status: [ SUCCESS / TIMEOUT ]                       | | +-------------------------------------------------------------------------+ |
| +-----------------------------------------------------+                                                                               |
|                                                         | [AddressingPanel]                                                          |
| [ReplayPanel]                                           | +-------------------------------------------------------------------------+ |
| +-----------------------------------------------------+ | | NATS Subject Addressing                                                 | |
| | JetStream Replay                                    | | | Exact, Single (*), Multi (>) active subscriptions                     | |
| | From Sequence: [ 100 ]   To Sequence: [ 120 ]       | | | Subject Routing Activity matrix                                       | |
| | [ Start Replay ]                                     | | +-------------------------------------------------------------------------+ |
| +-----------------------------------------------------+                                                                               |
+=======================================================================================================================================+
```

Keep the initial UI to one main screen.

Do not introduce multiple pages unless a later capability requires it.

---

# 5. Status Section

The status section should provide a quick view of the running components.

Minimum components:

```text
NATS Server
Demo Service
Processor Service
```

If multiple Processor Service instances are running, show them individually.

Example:

```text
SERVICES

NATS Server          ● Connected
Demo Service         ● Active
Processor-1          ● Active
Processor-2          ● Active
Processor-3          ● Active
```

Use simple visual indicators:

```text
● Active
● Connected
● Disconnected
● Unknown
```

Use icons and styling to make these states easy to distinguish.

Do not rely only on color.

---

# 6. Job Submission

Provide a simple form for creating a demo job.

Minimum fields:

```text
Job ID
Job Type
Payload
```

Example:

```text
Job ID

[ job-101 ]

Job Type

[ image-processing ]

Payload

{
  "file": "image-101.jpg"
}

[ Submit Job ]
```

The form should call:

```http
POST /jobs
```

The response should immediately update the UI.

---

# 7. Job Validation

Provide a simple validation action.

Example:

```text
[ Validate Job ]
```

The frontend calls:

```http
POST /jobs/validate
```

Show the result clearly:

```text
Validation Result

Valid
Job is ready for processing.
```

or:

```text
Validation Result

Invalid
<validation message>
```

The UI should make the Request/Reply demonstration obvious.

---

# 8. Activity View

The Activity section is the most important part of the UI.

It should show what is happening during a job's lifecycle.

Example:

```text
ACTIVITY

Time      Job       Mode       Event          Seq   Subject              Worker

14:32:01  job-101   CORE       PUBLISHED      -     jobs.submitted       -
14:32:01  job-101   CORE       RECEIVED       -     jobs.received        processor-1
14:32:02  job-101   CORE       COMPLETED      -     jobs.completed       processor-1
14:33:01  job-102   JETSTREAM  PUBLISHED      -     jobs.submitted       -
14:33:01  job-102   JETSTREAM  STORED         #1    jobs.stored          -
14:33:05  job-102   JETSTREAM  DELIVERED      #1    jobs.delivered       processor-1
14:33:06  job-102   JETSTREAM  ACKED          #1    jobs.acked           processor-1
```

The activity view supports events such as:

```text
PUBLISHED
STORED
RECEIVED
DELIVERED
PROCESSING
COMPLETED
FAILED
ACKED
NO CONSUMER
```

Keep the event representation simple.

---

# 9. Real-Time Updates

The UI should eventually receive activity updates without requiring manual refresh.

The initial implementation may use a simple polling mechanism if that is the easiest approach.

A real-time mechanism can be introduced later if required.

Do not introduce WebSockets or Server-Sent Events solely for architectural completeness.

The simplest mechanism that provides a good demo experience should be preferred.

---

# 10. NATS Capability Visibility

The UI should expose enough information to understand the NATS behavior.

For example:

```text
Job: job-101

Status: COMPLETED

NATS Activity

Subject:
jobs.submitted

Consumer:
job-processor

Worker:
processor-2

Delivery Count:
1

Message:
<JSON payload>
```

This makes the NATS behavior visible without requiring the user to inspect the source code.

Detailed NATS exploration screens can be added later.

---

# 11. Failure Demonstration

The UI should eventually provide a simple way to demonstrate failure behavior.

Example:

```text
[ Submit Job ]

Processing:
    ● Received
    ● Processing
    X Failed
    ↻ Redelivered
    ● Completed
```

The UI should clearly show when a message is redelivered.

Example:

```text
job-101

Delivery Count: 2

1. Processing failed
2. Message redelivered
3. Processing completed
```

This is particularly useful for demonstrating JetStream delivery semantics.

---

# 12. Competing Consumer Visibility

When multiple Processor Service instances are active, the Activity view should show which worker processed each job.

Example:

```text
Job        Worker

job-101    processor-1
job-102    processor-2
job-103    processor-1
job-104    processor-3
```

This provides a simple visual demonstration of competing consumers.

---

# 13. Replay Visibility

When replay functionality is added, provide a simple replay control.

Example:

```text
REPLAY

From Sequence: [ 100 ]
To Sequence:   [ 120 ]

Consumer:      [ replay-consumer ]

[ Start Replay ]
```

Show replay activity in the same Activity view.

Do not create a complex event-management interface initially.

---

# 14. Visual Design

The UI should have a polished developer-tool appearance.

Use:

* Dark or neutral developer-tool styling
* Clear typography
* Compact spacing
* Consistent cards/panels
* Subtle borders
* Clear status indicators
* Professional icons
* Monospace font for subjects, IDs, JSON, and technical values

The visual design should feel closer to a developer console than a business dashboard.

Avoid:

* Large marketing-style sections
* Excessive animations
* Large decorative graphics
* Unnecessary charts
* Excessive colors
* Complex navigation

The interface should prioritize information density and clarity.

---

# 15. Component Structure

Keep the component hierarchy simple.

Suggested structure:

```text
src/
|
+-- components/
|   +-- Header
|   +-- StatusPanel
|   +-- JobPanel
|   +-- RequestReplyPanel
|   +-- ReplayPanel
|   +-- JobInspectorPanel
|   +-- AddressingPanel
|   +-- ActivityPanel
|   +-- ActivityItem
|   +-- JsonViewer
|   +-- StatusIndicator
|
+-- api/
|   +-- demoApi
|
+-- App
+-- main
```

Do not introduce additional layers unless the frontend grows enough to require them.

---

# 16. Frontend Data Models

Keep frontend models aligned with backend API models.

Example:

```text
Job

job_id
type
payload
status
```

Activity:

```text
Activity

timestamp
job_id
event
subject
worker
delivery_count
```

Service status:

```text
ServiceStatus

name
status
details
```

Avoid duplicating backend business logic in frontend models.

---

# 17. API Integration

Keep HTTP communication in a small API module.

For example:

```text
api/
  demoApi
```

The React components should call simple functions such as:

```text
submitJob()
validateJob()
getJobStatus()
getServiceStatus()
getActivity()
startReplay()
```

Components should not contain raw HTTP implementation details.

---

# 18. UI State

Keep state management simple.

Local React state is sufficient initially.

Use shared state only where multiple components genuinely need the same data.

Do not introduce Redux, Zustand, or another state-management library unless the application grows enough to justify it.

---

# 19. Frontend Scope

The initial frontend should include only:

```text
1. Service status
2. NATS status
3. Job submission
4. Job validation
5. Job activity
6. Processing status
7. Worker visibility
```

Later capabilities can extend the same UI:

```text
8. Consumer information
9. Stream information
10. Message inspection
11. Redelivery information
12. Replay controls
```

---

# 20. Design Principle

The frontend should answer two questions immediately:

```text
Is the platform running?

What is happening to my job?
```

Everything else is secondary.

The UI should remain a simple observability and demonstration layer over the backend rather than becoming another complex application.

# 21. Extensibility

The frontend should be designed so that new NATS capabilities can be added incrementally.

A new capability may introduce:

- A new backend API
- A new NATS operation
- A new UI action/button
- A new activity event
- Additional status or metadata

Adding a capability should not require restructuring the existing application.

For example:

| Capability | Backend API | UI Action |
|---|---|---|
| Submit | `POST /jobs` | Submit Job |
| Request/Reply | `POST /jobs/validate` | Validate Job |
| Failure/Redelivery | Existing job API | Simulate Failure |
| Replay | `POST /jobs/replay` | Replay |
| Consumer inspection | Future API | View Consumers |
| Stream inspection | Future API | View Stream |
| Message inspection | Future API | Inspect Message |
| Ordering test | Future API | Test Ordering |

The UI should provide a natural place for new actions without introducing a complex navigation structure.

Prefer extending existing panels or adding a small focused panel over redesigning the entire UI.

The backend API and frontend should remain loosely coupled through clear HTTP contracts.

The frontend must not assume that the currently defined NATS capabilities are the complete set of capabilities.

The project should support the following evolution:

```text
Current Capability
       |
       +--> Backend API
       |       |
       |       +--> NATS Operation
       |
       +--> UI Action
               |
               +--> Activity / Result