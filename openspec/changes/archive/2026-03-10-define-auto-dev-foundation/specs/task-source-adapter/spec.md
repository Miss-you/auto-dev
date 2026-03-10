## ADDED Requirements

### Requirement: Task Source Normalization
The system SHALL normalize incoming tasks from an external source into a shared task model before dispatch decisions are made.

#### Scenario: External issue becomes normalized task
- **WHEN** the system reads a task from an enabled source provider
- **THEN** it stores a normalized task record containing the external identifier, title, body, routing metadata, and source type

### Requirement: Source-Side Filtering And Deduplication
The task source adapter SHALL support provider-specific filtering rules and deduplication before a task enters the dispatch queue.

#### Scenario: Unqualified issue is excluded
- **WHEN** an incoming task does not match the configured source filters or is already active in local state
- **THEN** the system excludes it from dispatch and records why it was ignored

### Requirement: Provider Abstraction Boundary
The system SHALL isolate provider-specific API details behind the task source adapter so downstream lifecycle logic depends only on the normalized task model.

#### Scenario: Dispatcher consumes normalized tasks only
- **WHEN** the dispatcher requests pending work
- **THEN** it receives normalized task data without needing provider-specific API fields
