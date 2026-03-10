## MODIFIED Requirements

### Requirement: Single-Task Supervision
The system SHALL define supervisor behavior for starting one worker, monitoring its progress, interpreting its status, and deciding whether to wait, intervene, accept, or terminate the run. The supervisor triggers run-lifecycle state transitions as defined by the allowed transition rules.

#### Scenario: Supervisor monitors an active worker
- **WHEN** a worker session is running for a task
- **THEN** the supervisor periodically inspects runtime output and maps the observed state to a run-lifecycle state transition decision

#### Scenario: Supervisor triggers state transition
- **WHEN** the supervisor observes an event warranting a state change (e.g., spec PR created, code PR created, verify passed)
- **THEN** it requests the corresponding run-lifecycle transition

### Requirement: Run Acceptance And Cleanup
The supervisor SHALL own task-level acceptance and runtime cleanup after a worker reaches a terminal state, using run-lifecycle terminal state semantics.

#### Scenario: Worker reaches terminal state
- **WHEN** a worker reports success, failure, or abort
- **THEN** the supervisor records the result using run-lifecycle terminal semantics, performs cleanup actions, and notifies the dispatcher
