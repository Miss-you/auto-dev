## MODIFIED Requirements

### Requirement: Predictable Status Reporting
The system SHALL require workers to surface progress, blockers, and terminal outcomes using run-lifecycle canonical state names, in a format that supervisor logic can interpret consistently.

#### Scenario: Worker reports a blocker
- **WHEN** a worker cannot proceed because of a test failure, permission issue, or missing context
- **THEN** it emits a status update using run-lifecycle state semantics that identifies the blocker category and requested next action

#### Scenario: Worker reports implementation complete
- **WHEN** a worker believes implementation and self-verification are done
- **THEN** it signals readiness for code review, enabling the supervisor to transition the task to code_in_review
