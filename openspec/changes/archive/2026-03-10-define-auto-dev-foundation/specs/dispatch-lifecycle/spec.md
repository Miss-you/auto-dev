## ADDED Requirements

### Requirement: Dispatch Orchestration
The system SHALL orchestrate the movement of normalized tasks from queueing through active supervision and terminal state transitions.

#### Scenario: Eligible task is dispatched
- **WHEN** a normalized task is eligible under the current concurrency and routing policy
- **THEN** the dispatcher creates a tracked run and hands control to supervisor logic

### Requirement: Persistent Lifecycle State
The dispatch lifecycle SHALL persist the minimum task state needed to recover the local orchestrator view across restarts.

#### Scenario: Dispatcher restarts mid-run
- **WHEN** the local dispatcher process restarts while tasks are already active
- **THEN** it reloads persisted task state and reconciles running sessions before making new dispatch decisions

### Requirement: Bounded Concurrency Policy
The system SHALL enforce a configurable upper bound on concurrently active tasks.

#### Scenario: Concurrency limit reached
- **WHEN** the number of active tasks reaches the configured maximum
- **THEN** additional eligible tasks remain queued until capacity is available
