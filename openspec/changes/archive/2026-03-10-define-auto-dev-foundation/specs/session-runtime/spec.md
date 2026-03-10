## ADDED Requirements

### Requirement: Managed Worker Sessions
The system SHALL create and manage named local worker sessions through a runtime abstraction instead of issuing ad hoc terminal commands from higher-level orchestration logic.

#### Scenario: Supervisor requests a worker session
- **WHEN** supervisor logic starts work for a task
- **THEN** it requests a named worker session through the session runtime and receives a stable session identifier

### Requirement: Runtime Control Operations
The session runtime SHALL expose standard operations for session creation, command injection, output capture, liveness checks, listing, and cleanup.

#### Scenario: Supervisor captures recent output
- **WHEN** supervisor logic asks for the latest worker activity
- **THEN** the runtime returns captured output for the requested session using a standard capture operation

### Requirement: Session Naming And Isolation
The session runtime SHALL enforce deterministic naming and isolation rules so concurrent worker sessions do not collide.

#### Scenario: Two tasks run concurrently
- **WHEN** two different tasks are active at the same time
- **THEN** the runtime assigns distinct session identities and keeps their command streams and captured output separate
