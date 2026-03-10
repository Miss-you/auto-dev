## ADDED Requirements

### Requirement: Two-Phase Gate Model
The change lifecycle SHALL divide every task into a Spec phase (Phase A) and an Implementation phase (Phase B), separated by an explicit approval gate. The system MUST NOT start implementation work before the spec is approved by a human reviewer.

#### Scenario: Spec 尚未批准时阻止实现
- **WHEN** 任务的 spec 产物已生成，但 spec review 尚未通过
- **THEN** 系统不允许开始实现工作（openspec apply）

#### Scenario: Spec 批准后启动实现
- **WHEN** 人工审查者批准了 spec PR
- **THEN** 该任务进入可派发实现的状态

### Requirement: Spec Phase Self-Check Loop
The Spec phase SHALL include an AI self-check loop before submitting to human review. After generating spec artifacts, the AI MUST run validation (e.g., openspec validate); only after self-check passes SHALL the spec PR be created for human review.

#### Scenario: 自检发现格式或语义问题
- **WHEN** AI 生成 spec 产物后执行自检，发现格式错误或语义缺漏
- **THEN** AI 在同一阶段内修复并重新自检，直到通过

#### Scenario: 自检通过后提交审查
- **WHEN** AI 自检通过
- **THEN** 创建 spec PR 并进入人工审查环节

### Requirement: Deferred Spec Finalization
The change lifecycle SHALL keep the OpenSpec change in active state throughout both phases. The system MUST NOT sync delta specs to main specs or archive the change until implementation verification succeeds.

#### Scenario: Spec 已批准但实现未完成
- **WHEN** spec review 通过但代码尚未 verify
- **THEN** OpenSpec change 保持 active；delta specs 不同步到主规格

#### Scenario: 实现已验证后执行归档
- **WHEN** 代码 review 收敛后 openspec verify 通过
- **THEN** 系统执行 sync 将 delta specs 同步到主规格，然后 archive

### Requirement: Spec Phase Entry And Exit Criteria
The Spec phase SHALL define explicit entry and exit criteria. Entry: task is assigned. Exit (success): spec PR is approved by human reviewer. Exit (rejection): reviewer requests changes or task is aborted. The system MUST keep the task in Phase A when review changes are requested.

#### Scenario: Spec PR 收到修改请求
- **WHEN** 审查者对 spec PR 提出修改意见
- **THEN** 任务在 Phase A 内回到起草状态，不进入 Phase B

#### Scenario: Spec 被人工批准
- **WHEN** 审查者批准 spec PR
- **THEN** 任务通过审批门禁，可进入实现阶段

### Requirement: Implementation Phase Entry And Exit Criteria
The Implementation phase SHALL define explicit entry and exit criteria. Entry: Phase A spec is approved. Exit (success): verify passes → sync → archive → completed. Exit (failure): unrecoverable implementation failure or manual abort. The system MUST keep the task in Phase B's review-fix loop when code review requests changes.

#### Scenario: 代码 review 触发返工
- **WHEN** 代码审查者要求修改
- **THEN** 任务留在 Phase B 的 review-fix 循环中，不回到 Phase A

#### Scenario: 实现过程中发现 spec 缺口
- **WHEN** 实现过程中需要修改 spec
- **THEN** 可直接修改 active change 中的 delta specs；如变更较大，须对 spec 部分重新审查
