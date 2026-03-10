## ADDED Requirements

### Requirement: Canonical Task States
The run lifecycle SHALL define the complete set of canonical task states shared by all runtime components. Phase A states: spec_drafting, spec_in_review, spec_approved. Phase B states: impl_queued, implementing, code_in_review, fixing_review, verified. Terminal states: completed, failed, aborted. All components MUST use only these canonical state names when recording or exchanging task state.

#### Scenario: 各组件使用共享状态名
- **WHEN** dispatcher、supervisor 或 worker 记录任务状态变更
- **THEN** 只使用本 spec 定义的规范状态名

#### Scenario: spec_drafting 内部包含自检循环
- **WHEN** AI 在 spec_drafting 阶段进行自检修复
- **THEN** 从外部状态机视角，状态保持在 spec_drafting 不发生转移

### Requirement: Allowed State Transitions
The run lifecycle SHALL define the set of valid state transitions and the authorized trigger for each. The system MUST reject any transition not in the allowed set. Allowed transitions: (new) → spec_drafting (dispatcher); spec_drafting → spec_in_review (supervisor, spec PR created and self-check passed); spec_in_review → spec_drafting (supervisor, reviewer requests changes); spec_in_review → spec_approved (supervisor, human approves); spec_approved → impl_queued (dispatcher); impl_queued → implementing (supervisor, worker started); implementing → code_in_review (supervisor, code PR created); code_in_review → fixing_review (supervisor, review feedback received); fixing_review → code_in_review (supervisor, fix pushed); code_in_review → verified (supervisor, verify passed); verified → completed (dispatcher, sync + archive done); any non-terminal → failed (supervisor or dispatcher); any non-terminal → aborted (operator manual).

#### Scenario: 合法转移被执行
- **WHEN** 某组件按允许的转移规则请求状态变更
- **THEN** 状态转移成功并记录

#### Scenario: 非法转移被拒绝
- **WHEN** 某组件尝试执行不在允许集中的状态转移
- **THEN** 转移被拒绝，当前状态不变

### Requirement: Terminal State Semantics
The run lifecycle SHALL define the semantics and required evidence for each terminal state. completed: verify passed, delta specs synced to main specs, change archived, all PRs merged or closed. failed: unrecoverable error requiring human investigation before retry. aborted: operator manually cancelled with no further automated action. The system MUST NOT allow any automated state transition after a task reaches a terminal state.

#### Scenario: 任务到达终态
- **WHEN** 任务转移到 completed、failed 或 aborted
- **THEN** 该运行实例不再允许任何自动化状态转移

#### Scenario: completed 须满足全部证据
- **WHEN** 任务尝试转移到 completed
- **THEN** 须提供 verify 通过、specs 已同步、change 已归档的证据
