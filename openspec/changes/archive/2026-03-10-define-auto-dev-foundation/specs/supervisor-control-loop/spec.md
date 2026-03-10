## ADDED Requirements

### Requirement: Single-Task Supervision
The system SHALL define supervisor behavior for starting one worker, monitoring its progress, interpreting its status, and deciding whether to wait, intervene, accept, or terminate the run.

#### Scenario: Supervisor monitors an active worker
- **WHEN** a worker session is running for a task
- **THEN** the supervisor periodically inspects runtime output and maps the observed state to an explicit supervision decision

### Requirement: Guided Intervention Policy
The supervisor control loop SHALL support targeted interventions for common execution problems such as test failures, stalled output, missing confirmations, or review-driven rework.

#### Scenario: Worker stalls during execution
- **WHEN** the supervisor detects that output has stopped beyond the allowed idle threshold
- **THEN** it chooses a defined intervention path such as recapturing more output, prompting the worker, retrying, or escalating

### Requirement: Run Acceptance And Cleanup
The supervisor SHALL own task-level acceptance and runtime cleanup after a worker reaches a terminal state.

#### Scenario: Worker reaches terminal state
- **WHEN** a worker reports success, failure, or abort
- **THEN** the supervisor records the result, performs cleanup actions, and transitions the task to the next lifecycle state
