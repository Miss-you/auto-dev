## ADDED Requirements

### Requirement: Review Intake And Routing
The system SHALL collect review feedback associated with a task run and route it back to the corresponding worker context or replacement repair run.

#### Scenario: Review feedback arrives for an open run
- **WHEN** review comments or automated review findings are detected for a tracked pull request
- **THEN** the system associates that feedback with the originating task run and prepares a repair action

### Requirement: Structured Repair Context
The review feedback loop SHALL transform raw review feedback into a structured repair context before sending it to worker execution.

#### Scenario: Worker receives review repair task
- **WHEN** supervisor logic initiates a repair cycle
- **THEN** the worker receives a structured summary of the feedback, linked task context, and expected completion criteria

### Requirement: Bounded Rework Attempts
The system SHALL enforce a maximum number of automated repair attempts before handing the run back to manual intervention.

#### Scenario: Repair attempts exceed threshold
- **WHEN** the configured review repair limit has been reached without a successful outcome
- **THEN** the system marks the run for manual handling instead of continuing automated retries
