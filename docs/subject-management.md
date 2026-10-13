## Subject Management

Subject Management defines the enterprise process for governing NATS subject usage across the internal messaging platform. It establishes a consistent approach to subject ownership, registration, approval, provisioning, change management, and retirement.

### Subject Management Scope

| # | Domain | Scope |
|---|---|---|
| 1 | **Standards and Governance** | Define and enforce subject naming, namespace, access, and persistence standards. |
| 2 | **Ownership and Accountability** | Establish subject ownership and responsibilities across application and Platform/SRE teams. |
| 3 | **Registry and Discovery** | Maintain a central registry of approved subjects, ownership, metadata, and lifecycle status. |
| 4 | **Request, Approval and Provisioning** | Define the request, review, maker-checker approval, registration, and associated NATS configuration process. |
| 5 | **Operation and Change Management** | Manage changes to subject usage, ownership, dependencies, permissions, and Stream mappings. |
| 6 | **Deprecation and Retirement** | Define dependency assessment, migration, retirement approval, and associated configuration cleanup. |

### Operating Model: Phase 1 and Future Platformisation

| Capability | Phase 1: Direct NATS | Future: Platformised NATS |
|---|---|---|
| Request | Ticket or request template | Self-service portal/API |
| Validation and approval | Manual review and maker-checker process | Automated validation and workflow-driven approval |
| Registry | Central document or version-controlled repository | Authoritative, searchable registry |
| Provisioning | Manual configuration of permissions and Streams, where required | Automated provisioning and policy enforcement |
| Change management | Tickets, dependency assessment, and version history | Integrated change workflows and automated validation |
| Retirement | Manual dependency checks and configuration cleanup | Workflow-driven retirement and controlled cleanup |
| Auditability | Tickets, approvals, and repository history | Integrated audit trail and lifecycle history |