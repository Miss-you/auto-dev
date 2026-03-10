# run-lifecycle Specification

## Purpose

定义 auto-dev 中任务运行实例的规范状态集合、允许的状态转移、触发条件和终态语义，供 dispatcher、supervisor、worker 和 review loop 使用统一的状态词汇。

## Requirements

### Requirement: Canonical Task States
The run lifecycle SHALL define the complete set of canonical task states shared by all runtime components. Phase A states: spec_drafting, spec_in_review, spec_approved. Phase B states: impl_queued, implementing, code_in_review, fixing_review, verified. Terminal states: completed, failed, aborted. All components MUST use only these canonical state names when recording or exchanging task state. The run lifecycle begins when the dispatcher creates a run instance in spec_drafting; pre-dispatch queuing is managed internally by the dispatcher and is not a run-lifecycle state. The spec_approved state doubles as a holding state when concurrency limits prevent immediate Phase B dispatch. No backward transition from Phase B states to Phase A states is defined; if implementation reveals a significant spec gap, the delta specs within the active OpenSpec change may be edited in place per change-lifecycle rules without regressing the run-lifecycle state.

#### Scenario: 各组件使用共享状态名
- **WHEN** dispatcher、supervisor 或 worker 记录任务状态变更
- **THEN** 只使用本 spec 定义的规范状态名

#### Scenario: spec_drafting 内部包含自检循环
- **WHEN** AI 在 spec_drafting 阶段进行自检修复
- **THEN** 从外部状态机视角，状态保持在 spec_drafting 不发生转移

#### Scenario: spec_approved 兼任实现排队
- **WHEN** spec 已批准但并发限制阻止立即进入 Phase B
- **THEN** 任务在 spec_approved 状态等待，直到 dispatcher 将其转移到 impl_queued

### Requirement: Allowed State Transitions
The run lifecycle SHALL define the set of valid state transitions and the authorized trigger for each. The system MUST reject any transition not in the allowed set. Allowed transitions: (new) → spec_drafting (dispatcher); spec_drafting → spec_in_review (supervisor, spec PR created and self-check passed); spec_in_review → spec_drafting (supervisor, reviewer requests changes); spec_in_review → spec_approved (supervisor, human approves); spec_approved → impl_queued (dispatcher); impl_queued → implementing (supervisor, worker started); implementing → code_in_review (supervisor, code PR created); code_in_review → fixing_review (supervisor, review feedback received); fixing_review → code_in_review (supervisor, fix pushed); code_in_review → verified (supervisor, verify passed); verified → completed (dispatcher, sync + archive done); any non-terminal → failed (supervisor or dispatcher); any non-terminal → aborted (operator manual). The fixing_review ↔ code_in_review cycle count is bounded by the review-feedback-loop capability; when the configured limit is reached, the run transitions to failed. There is no transition from spec_approved back to spec_drafting; if the spec must be reworked after approval, the operator aborts the current run and creates a new one.

#### Scenario: 合法转移被执行
- **WHEN** 某组件按允许的转移规则请求状态变更
- **THEN** 状态转移成功并记录

#### Scenario: 非法转移被拒绝
- **WHEN** 某组件尝试执行不在允许集中的状态转移
- **THEN** 转移被拒绝，当前状态不变

#### Scenario: verified 后 sync/archive 失败
- **WHEN** 任务在 verified 状态下执行 sync 或 archive 操作失败
- **THEN** 任务转移到 failed，failure reason 标注为 post_verification_cleanup_failed，区别于代码质量导致的失败

#### Scenario: review 修复轮次超限
- **WHEN** fixing_review ↔ code_in_review 循环次数达到 review-feedback-loop 配置的上限
- **THEN** 任务转移到 failed，不再继续自动修复

### Requirement: Terminal State Semantics
The run lifecycle SHALL define the semantics and required evidence for each terminal state. completed: verify passed, delta specs synced to main specs, change archived, all PRs merged or closed. failed: unrecoverable error requiring human investigation before retry; when a run transitions to failed from the verified state, it indicates a post-verification cleanup failure (sync or archive) rather than a code quality issue, and the failure reason MUST be recorded as post_verification_cleanup_failed. aborted: operator manually cancelled with no further automated action; this is the only terminal state that can be triggered exclusively by an operator, and it halts all automated processing for the run including any in-flight worker sessions. The system MUST NOT allow any automated state transition after a task reaches a terminal state.

#### Scenario: 任务到达终态
- **WHEN** 任务转移到 completed、failed 或 aborted
- **THEN** 该运行实例不再允许任何自动化状态转移

#### Scenario: completed 须满足全部证据
- **WHEN** 任务尝试转移到 completed
- **THEN** 须提供 verify 通过、specs 已同步、change 已归档、所有 PR 已合并或关闭的证据

#### Scenario: failed 从 verified 状态到达时须标注原因
- **WHEN** 任务从 verified 状态转移到 failed
- **THEN** failure reason 须标注为 post_verification_cleanup_failed，以区分代码实现失败

#### Scenario: aborted 由操作者触发并终止一切自动化
- **WHEN** 操作者手动将任务转移到 aborted
- **THEN** 系统停止该运行实例的一切自动化处理，包括终止正在运行的 worker session
