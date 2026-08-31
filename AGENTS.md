# AGENTS.md

## 1. Purpose

This repository is a small, demo-level implementation for evaluating and demonstrating NATS platform capabilities.

The primary goal is to keep the code:

* Simple
* Easy to understand
* Easy to modify
* Easy to demonstrate
* Easy to extend
* Suitable for a normal developer to maintain

This is not intended to be production infrastructure.

Prefer clear and direct implementations over sophisticated or highly abstract designs.

---

# 2. Agent Role

The coding agent is responsible for implementing and maintaining application code and documentation within this repository.

The agent should behave like a careful developer working on an existing codebase.

Before making changes:

1. Understand the existing implementation.
2. Check the relevant documentation.
3. Create an implementation plan.
4. Make the smallest reasonable change.
5. Review the resulting code.
6. Update the change record.

Do not make unrelated improvements while implementing a requested change.

---

# 3. Implementation Plan Is Mandatory

Every code change must start with an implementation plan.

Before modifying files, create or update a plan that clearly describes:

* What needs to change
* Why it needs to change
* Which files are expected to change
* The implementation approach
* Any important considerations

For small changes, the plan can be short.

Example:

```text
Implementation Plan

1. Add the jobs.submitted subject constant.
2. Add a publisher method for submitting jobs.
3. Update POST /jobs to publish the event.
4. Add a basic processor subscription.
5. Update developer documentation.
```

Do not begin implementation before the plan is established.

---

# 4. Change Record Is Mandatory

Maintain a simple record of implemented changes.

Use:

```text
docs/CHANGELOG.md
```

Each meaningful change should record:

* Date
* Change
* Reason
* Affected area

Example:

```text
## 2026-08-31

### Added
- Added NATS Request/Reply support for job validation.

### Changed
- Added POST /jobs/validate API.

### Documentation
- Updated developer guide with validation flow.
```

The change record should describe meaningful implementation changes, not every minor edit.

---

# 5. Simplicity First

Use the simplest implementation that satisfies the requirement.

Prefer:

* Direct code
* Small functions
* Small packages
* Explicit control flow
* Standard library functionality where practical
* Straightforward data structures
* Minimal dependencies

Avoid introducing abstractions merely because they may be useful in the future.

Do not introduce:

* Generic frameworks
* Complex dependency injection
* Excessive interfaces
* Factory patterns
* Repository patterns
* Event frameworks
* Generic messaging frameworks
* Complex configuration systems

unless there is a demonstrated need.

A small amount of duplication is acceptable if removing it would introduce unnecessary abstraction.

---

# 6. Abstractions

Introduce an abstraction only when at least one of the following is true:

* It removes meaningful complexity.
* It isolates a genuinely changing concern.
* It is required by the existing design.
* It makes testing substantially easier.
* It is required to support an identified future capability.

Do not create abstractions simply to make the code look architecturally sophisticated.

For this demo, the NATS Go client can be used directly where doing so keeps the code easier to understand.

---

# 7. Code Readability

Code should be understandable by a normal backend developer who is unfamiliar with the project.

Prefer:

```text
clear name
clear function
clear flow
```

over:

```text
generic abstraction
multiple layers
indirection
implicit behavior
```

Functions should have a focused responsibility.

Avoid unnecessarily long functions, but do not split every few lines into separate functions.

Use descriptive names rather than relying on comments.

---

# 8. Comments and Documentation

Comments are required where they add understanding.

Add comments when:

* NATS behavior is not obvious.
* A capability has an important semantic distinction.
* A workaround is required.
* A non-obvious configuration exists.
* Failure or retry behavior requires explanation.
* A design decision may otherwise be misunderstood.

Do not add comments for obvious code.

Avoid comments such as:

```go
// Create a connection
conn := ...
```

Prefer comments that explain why:

```go
// Use a durable consumer so messages remain available when
// the processor service is restarted.
```

The code should explain **what** it is doing.

Comments should explain **why**, when the reason is not obvious.

---

# 9. ASCII-Only Rule

Use ASCII characters only in:

* Source code
* Comments
* Log messages
* Error messages
* CLI output
* Configuration examples
* Test data
* Documentation examples

Do not introduce Unicode characters intentionally.

Avoid:

* Chinese characters
* Japanese characters
* Korean characters
* Emoji
* Unicode arrows
* Smart quotes
* Special bullets
* Decorative Unicode symbols

Use ASCII equivalents such as:

```text
-> 
<-
+
|
-
```

instead of Unicode equivalents.

This rule exists to keep the demo portable and avoid accidental non-ASCII content.

---

# 10. Error Handling

Handle errors explicitly.

Do not silently ignore errors unless there is a clear reason.

Errors should:

* Provide useful context.
* Preserve the original error where appropriate.
* Be understandable during the demo.

Prefer:

```text
failed to publish job event: <original error>
```

over:

```text
operation failed
```

Do not over-engineer error handling for this demo.

---

# 11. Logging

Logs should help demonstrate the NATS behavior.

Important operations should be visible, such as:

* Connection established
* Message published
* Request sent
* Reply received
* Message received
* Message processed
* ACK
* NACK
* Redelivery
* Consumer started
* Consumer stopped

Do not produce excessive logs for trivial operations.

Log messages must remain ASCII-only.

---

# 12. Configuration

Keep configuration simple.

Prefer environment variables for runtime configuration.

Do not introduce a complex configuration framework unless necessary.

Configuration should be easy to understand and documented in the deployment guide.

---

# 13. NATS Demonstration Principle

Every NATS capability implemented in the project should be demonstrable.

When adding a capability, consider:

1. What NATS feature is being demonstrated?
2. What is the simplest application scenario?
3. How can a developer observe that it works?
4. What failure or edge case is useful to demonstrate?
5. Is the behavior documented?

Do not add NATS features merely for completeness.

The implementation should help explain NATS behavior.

---

# 14. Backend Guidelines

Backend implementation should prioritize:

* Go standard conventions
* Clear HTTP handlers
* Simple domain models
* Explicit NATS interactions
* Small services
* Clear subject names
* Easy-to-follow message flows

Do not introduce a database unless explicitly required.

In-memory state is acceptable for demo purposes.

Do not implement production concerns unless they are part of the capability being evaluated.

---

# 15. Frontend Guidelines

The frontend is a demonstration UI, not a production application.

Use:

* React SPA
* A simple and well-known component structure
* Minimal frontend dependencies
* Straightforward state management
* Simple API integration

The frontend should be easy for a backend developer to understand.

Avoid:

* Complex state-management frameworks
* Complex frontend architecture
* Excessive component nesting
* Over-engineered design systems
* Unnecessary abstractions

Keep the component structure shallow and obvious.

---

# 16. Frontend Visual Quality

Although the frontend architecture should remain simple, the UI should have a polished and premium appearance.

Prioritize:

* Clean spacing
* Strong typography
* Consistent layout
* Clear hierarchy
* Responsive design
* Useful empty/loading/error states
* Professional cards and tables
* Clear status indicators
* Appropriate icons

Use a simple, consistent icon library rather than creating custom icons.

Visual quality should come from good layout, typography, spacing, and consistency rather than complicated frontend code.

Do not sacrifice maintainability for visual effects.

---

# 17. Backend and Frontend Separation

Keep backend and frontend concerns clearly separated.

The backend should expose APIs.

The frontend should consume those APIs.

Do not:

* Put business logic into React components unnecessarily.
* Make the frontend understand NATS internals.
* Expose NATS credentials to the frontend.
* Couple UI components directly to NATS.

The UI should interact with the backend API.

---

# 18. Documentation

Maintain two developer-focused guides:

```text
docs/
├── DEVELOPER_GUIDE.md
└── DEPLOYMENT_GUIDE.md
```

Both backend and frontend documentation should be clearly separated.

### Developer Guide

Explain:

* Project purpose
* Repository structure
* Backend structure
* Frontend structure
* Service responsibilities
* API overview
* NATS capability mapping
* Important implementation concepts
* How to make common code changes

Keep it concise and practical.

### Deployment Guide

Explain:

* Required runtime components
* Backend deployment
* Frontend deployment
* NATS deployment
* Required configuration
* Service connectivity
* Deployment topology
* Basic verification steps

Do not turn either guide into a low-level design document.

---

# 19. Documentation Maintenance

Documentation is part of the implementation.

When a change affects:

* APIs
* Models
* NATS subjects
* Service behavior
* Configuration
* Deployment
* Frontend behavior

update the relevant documentation as part of the same change.

Do not leave documentation updates for a future task when the required information is already known.

---

# 20. File and Directory Exploration

The agent may use commands to:

* List files
* Inspect directories
* Read files
* Search source code
* Search documentation
* Inspect configuration
* Review existing implementation

General workspace exploration is allowed.

---

# 21. Command Restrictions

The agent must not execute commands related to language or package management.

Do not execute commands involving:

* Go
* Git
* npm
* pnpm
* yarn
* bun
* Other package managers
* Dependency installation
* Dependency upgrades
* Dependency removal
* Repository operations

Examples of commands that are out of scope:

```text
go build
go test
go run
go mod tidy
git status
git diff
git commit
npm install
npm run
npm test
```

The agent's scope is **code-level implementation and documentation**.

Do not modify the environment or repository state through package managers or version-control commands.

If validation requires such a command, document the command for the developer instead of executing it.

---

# 22. Validation

Within the allowed scope, inspect the resulting code carefully after changes.

Check:

* Imports
* Naming
* Interfaces
* Error handling
* API contracts
* NATS subjects
* Message structures
* Configuration usage
* Documentation consistency
* Unintended changes

If a validation step requires an out-of-scope command, do not execute it.

Instead, clearly state what the developer should run manually.

---

# 23. Scope Discipline

Only implement what is requested or required by the current capability.

Do not make unrelated changes to:

* Architecture
* Dependencies
* Styling
* APIs
* Folder structure
* Naming conventions
* Infrastructure
* Deployment

unless the current change requires them.

If a broader improvement is identified, mention it separately rather than silently implementing it.

---

# 24. Backward Compatibility

When changing an existing API, subject, message model, or behavior:

1. Check existing usages.
2. Understand the impact.
3. Prefer backward-compatible changes when practical.
4. Update documentation.
5. Record the change.

Do not silently break existing demo flows.

---

# 25. Demo-Oriented Design

This repository is intended to be demonstrated.

Code should therefore make important behavior easy to observe.

Prefer explicit behavior such as:

```text
Publish job
Receive job
Process job
ACK job
```

over hidden framework behavior.

When demonstrating a NATS capability, make the relevant behavior visible through:

* Logs
* API responses
* Clear status values
* Simple test scenarios
* Eventually, the UI

---

# 26. Extendability

The implementation should be easy to extend without trying to solve future problems prematurely.

Good extension points include:

* Additional NATS subjects
* Additional consumers
* Additional job types
* Additional lifecycle events
* Additional demo APIs
* Observer functionality
* UI views

Do not implement these capabilities before they are needed.

The guiding principle is:

```text
Simple now.
Easy to extend later.
```

---

# 27. Agent Working Process

For every requested implementation:

```text
1. Inspect
   |
   v
2. Understand existing code
   |
   v
3. Create implementation plan
   |
   v
4. Implement smallest reasonable change
   |
   v
5. Review changed code
   |
   v
6. Update documentation
   |
   v
7. Update CHANGELOG.md
   |
   v
8. Report what changed
```

The final response should briefly contain:

* What was implemented
* Files changed
* Important design decisions
* Validation performed
* Any manual commands the developer should run
* Any known limitations

---

# 28. Decision Principles

When multiple implementation approaches are possible, use this order of preference:

1. Simplest solution
2. Clearest solution
3. Existing project conventions
4. Standard Go/React approach
5. Minimal dependency approach
6. Extensible solution
7. More sophisticated architecture only when necessary

Do not optimize for theoretical scalability at the expense of demo clarity.

---

# 29. Final Principle

This project should feel like code written by a good developer for other developers to understand.

It should not feel like generated framework code.

Prefer:

```text
Readable
Explicit
Small
Practical
Documented
Observable
```

over:

```text
Abstract
Generic
Highly configurable
Over-engineered
```

The purpose of the agent is to help build a clear NATS capability demonstration that can be understood, maintained, and extended by normal developers.
