# review-feedback-loop Specification

## Purpose

该规格定义 pull request review 反馈如何回流到 auto-dev 执行闭环中，包括反馈采集、结构化修复上下文、自动修复轮次限制以及人工兜底边界。

## Requirements

### Requirement: Review Intake And Routing
The system SHALL collect review feedback associated with a task run during change-lifecycle Phase B and route it back to the corresponding worker context or replacement repair run, triggering run-lifecycle transitions between code_in_review and fixing_review.

#### Scenario: Review feedback arrives for an open run
- **WHEN** review comments or automated review findings are detected for a tracked pull request in Phase B
- **THEN** the system associates that feedback with the originating task run, triggers a transition to fixing_review, and prepares a repair action

#### Scenario: Repair cycle completes
- **WHEN** a worker finishes addressing review feedback and pushes the fix
- **THEN** the system transitions the task back to code_in_review for re-evaluation

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
