## MODIFIED Requirements

### Requirement: Review Intake And Routing
The system SHALL collect review feedback associated with a task run during change-lifecycle Phase B and route it back to the corresponding worker context or replacement repair run, triggering run-lifecycle transitions between code_in_review and fixing_review.

#### Scenario: Review feedback arrives for an open run
- **WHEN** review comments or automated review findings are detected for a tracked pull request in Phase B
- **THEN** the system associates that feedback with the originating task run, triggers a transition to fixing_review, and prepares a repair action

#### Scenario: Repair cycle completes
- **WHEN** a worker finishes addressing review feedback and pushes the fix
- **THEN** the system transitions the task back to code_in_review for re-evaluation
