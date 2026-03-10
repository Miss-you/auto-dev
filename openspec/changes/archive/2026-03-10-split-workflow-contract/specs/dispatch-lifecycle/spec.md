## MODIFIED Requirements

### Requirement: Dispatch Orchestration
The system SHALL orchestrate the movement of normalized tasks through the change-lifecycle phases and run-lifecycle states, from queueing through active supervision and terminal state transitions.

#### Scenario: Eligible task is dispatched
- **WHEN** a normalized task is eligible under the current concurrency and routing policy
- **THEN** the dispatcher creates a tracked run, sets its run-lifecycle state to spec_drafting, and hands control to supervisor logic

#### Scenario: Spec approved triggers implementation dispatch
- **WHEN** a task reaches spec_approved in the run-lifecycle
- **THEN** the dispatcher transitions it to impl_queued per the change-lifecycle gate model

### Requirement: Persistent Lifecycle State
The dispatch lifecycle SHALL persist the minimum task state needed to recover the local orchestrator view across restarts, using run-lifecycle canonical state names.

#### Scenario: Dispatcher restarts mid-run
- **WHEN** the local dispatcher process restarts while tasks are already active
- **THEN** it reloads persisted task state (expressed in run-lifecycle states) and reconciles running sessions before making new dispatch decisions
