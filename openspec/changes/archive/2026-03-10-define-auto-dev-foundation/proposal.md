## Why

`doc/plan.md` 已经给出了 auto-dev 系统的整体方案，但当前仓库还没有把这些设计意图沉淀为可演进的 OpenSpec 基线。需要先把叙述式方案转换成 capability 级别的正式规格，作为后续分批实现、并行推进和归档追踪的共同约束。

## What Changes

- 将 `doc/plan.md` 和 `doc/ref.md` 中的整体方案整理为一组正式的 OpenSpec 基线能力。
- 为后续实现建立八个 capability：`workflow-contract`、`task-source-adapter`、`session-runtime`、`worker-execution`、`supervisor-control-loop`、`dispatch-lifecycle`、`review-feedback-loop`、`operations-observability`。
- 明确 capability 之间的前后依赖、可并行关系和推荐推进波次。
- 产出一份规划型 `design.md`，说明为什么这样拆分，以及后续 change 应如何引用这套基线规格。

## Capabilities

### New Capabilities
- `workflow-contract`: 统一任务生命周期、仓库操作约束和项目 spec 注入规则。
- `task-source-adapter`: 统一外部任务源接入、过滤、去重和任务归一化能力。
- `session-runtime`: 统一本地 worker session 的创建、操控、抓取和清理能力。
- `worker-execution`: 统一 worker 的输入输出、执行边界、完成标准和结果回报。
- `supervisor-control-loop`: 统一单任务 supervisor 的启动、轮询、干预、验收和清理行为。
- `dispatch-lifecycle`: 统一 dispatcher 的派发、状态持久化和任务生命周期编排。
- `review-feedback-loop`: 统一 PR review 反馈回流、二次修复和终止策略。
- `operations-observability`: 统一并发控制、运行时观测、故障识别和运维可见性。

### Modified Capabilities
- None.

## Impact

- 影响 `openspec/changes/define-auto-dev-foundation/` 下的 proposal、design、specs、tasks 产物。
- 将在 sync 阶段创建 `openspec/specs/*` 主规格文件，作为后续 change 的引用基线。
- 当前变更只定义规划和规格，不实现任何业务代码或运行脚本。
- 主要来源文档为 `doc/plan.md`，背景补充为 `doc/ref.md`。
