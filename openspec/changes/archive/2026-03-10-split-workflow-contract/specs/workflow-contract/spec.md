## REMOVED Requirements

### Requirement: Unified Workflow Contract
**Reason**: 职责已拆分到 change-lifecycle（两阶段门禁模型）和 run-lifecycle（规范状态机）。
**Migration**: 阶段和门禁相关约束参见 change-lifecycle spec；状态名和转移规则参见 run-lifecycle spec。

### Requirement: Project Guidance Injection
**Reason**: worker 读 CLAUDE.md/AGENTS.md 是 coding CLI 的隐含行为；supervisor 给 worker 的约束通过 prompt 注入完成。不需要在顶层 spec 中定义。
**Migration**: worker 启动上下文的定义参见 worker-execution spec；supervisor 的干预策略参见 supervisor-control-loop spec。

### Requirement: Repository Action Policy
**Reason**: 分支、测试、提交、PR 等仓库操作的完成标准属于 worker-execution 的职责，不应由顶层契约管理。
**Migration**: 仓库操作完成标准参见 worker-execution spec 的 Worker Completion Gate requirement。
