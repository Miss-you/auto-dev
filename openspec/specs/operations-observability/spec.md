# operations-observability Specification

## Purpose

该规格定义 auto-dev 的运行可观测性和运维控制基线，让操作者能够看清当前系统状态、发现异常健康信号，并在统一生命周期约束下执行恢复类动作。

## Requirements

### Requirement: Runtime Visibility
The system SHALL expose current task, session, and orchestration status in a form that allows an operator to understand what the local system is doing.

#### Scenario: Operator checks current system state
- **WHEN** an operator inspects the local auto-dev environment
- **THEN** the system can present active tasks, their lifecycle stages, recent activity timestamps, and recent failure summaries

### Requirement: Operational Health Signals
The operations and observability capability SHALL surface health signals needed to detect stalled sessions, failed reconciliations, runtime errors, and abnormal retry behavior.

#### Scenario: Health signal shows degraded run
- **WHEN** a task experiences repeated runtime or orchestration problems
- **THEN** the system records health information that identifies the degraded condition and the affected run

### Requirement: Recovery-Oriented Controls
The system SHALL define operational controls that allow later changes to add restart, abort, retry, and cleanup strategies without redefining core lifecycle semantics.

#### Scenario: Recovery action is required
- **WHEN** an operator or automated policy selects a recovery action for a failed or stalled run
- **THEN** the chosen action is evaluated against the shared lifecycle and health model before execution
