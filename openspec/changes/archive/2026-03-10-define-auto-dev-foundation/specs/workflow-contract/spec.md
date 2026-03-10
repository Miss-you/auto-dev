## ADDED Requirements

### Requirement: Unified Workflow Contract
The system SHALL define a single workflow contract that governs task intake, execution, review handling, and terminal outcomes across dispatcher, supervisor, and worker actors.

#### Scenario: Shared lifecycle vocabulary
- **WHEN** different runtime components record or exchange task state
- **THEN** they use the same lifecycle names and terminal outcome semantics defined by the workflow contract

### Requirement: Project Guidance Injection
The system SHALL require project-specific workflow guidance to be injected into every worker startup context and supervisor intervention before repository actions are taken.

#### Scenario: Worker starts with project guidance
- **WHEN** a worker session is launched for a task
- **THEN** the startup context includes repository workflow guidance, protected constraints, and completion expectations before implementation begins

### Requirement: Repository Action Policy
The workflow contract SHALL define branch naming, test gating, commit, push, and pull request expectations that must be satisfied before a task is reported as complete.

#### Scenario: Completion requires repository policy compliance
- **WHEN** a worker reports a task as ready for completion
- **THEN** the result includes evidence that the required repository actions were attempted under the shared policy
