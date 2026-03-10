## ADDED Requirements

### Requirement: Worker Execution Contract
The system SHALL define a standard worker contract that includes task context, project guidance, allowed actions, expected outputs, and terminal result reporting.

#### Scenario: Worker receives executable context
- **WHEN** a worker is started for a normalized task
- **THEN** it receives the task context, project workflow guidance, and expected result format before beginning repository work

### Requirement: Worker Completion Gate
The worker execution capability SHALL require a worker to complete the mandatory verification and repository steps defined by the workflow contract before reporting success.

#### Scenario: Worker finishes with gated result
- **WHEN** a worker believes implementation is done
- **THEN** it reports success only after the required verification and repository steps have been attempted

### Requirement: Predictable Status Reporting
The system SHALL require workers to surface progress, blockers, and terminal outcomes in a format that supervisor logic can interpret consistently.

#### Scenario: Worker reports a blocker
- **WHEN** a worker cannot proceed because of a test failure, permission issue, or missing context
- **THEN** it emits a status update that identifies the blocker category and requested next action
