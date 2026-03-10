## Why

当前 `workflow-contract` spec 把三类关注点混在一起：OpenSpec 变更流转、运行时状态跟踪、编码约束注入。它描述了"应该有一个契约"，但没有把契约本身写实——缺少具体的状态名、转移规则、阶段门禁和终态语义。后续 dispatch-lifecycle、supervisor-control-loop、worker-execution 等 spec 在引用它时无法得到明确约束，会各自解释一套。

需要将其拆成两个边界清晰的 spec：一个管"流程怎么走"（两阶段门禁），一个管"状态怎么转"（运行时状态机），然后退役原来的 workflow-contract。

## What Changes

- 新增 `change-lifecycle` capability：定义基于 OpenSpec 的两阶段开发生命周期（Spec 阶段 → 审批门禁 → 实现阶段），包括自检修复循环、延迟规格定稿、入口/出口条件。
- 新增 `run-lifecycle` capability：定义任务运行实例的规范状态集（spec_drafting / spec_in_review / spec_approved / impl_queued / implementing / code_in_review / fixing_review / verified / completed / failed / aborted）、允许的状态转移、触发者和终态语义。
- **BREAKING** 删除 `workflow-contract` capability：其职责完全被以上两个 spec 替代。
- 更新 dispatch-lifecycle、supervisor-control-loop、worker-execution、review-feedback-loop 的引用，从依赖 workflow-contract 改为依赖 change-lifecycle 和 run-lifecycle。

## Capabilities

### New Capabilities
- `change-lifecycle`: 基于 OpenSpec 的两阶段开发生命周期编排（阶段、门禁、sync/archive 时机）
- `run-lifecycle`: 任务运行实例的规范状态机（状态名、转移规则、终态语义）

### Modified Capabilities
- `workflow-contract`: **BREAKING** 退役删除，职责由 change-lifecycle + run-lifecycle 替代
- `dispatch-lifecycle`: 引用从 workflow-contract 改为 run-lifecycle（状态名）+ change-lifecycle（阶段门禁）
- `supervisor-control-loop`: 引用从 workflow-contract 改为 run-lifecycle（状态转移触发规则）
- `worker-execution`: 引用从 workflow-contract 改为 run-lifecycle（状态上报格式）
- `review-feedback-loop`: 引用从 workflow-contract 改为 change-lifecycle（Phase B）+ run-lifecycle（fixing_review 状态）

## Impact

- `openspec/specs/workflow-contract/` 将被删除
- `openspec/specs/change-lifecycle/` 和 `openspec/specs/run-lifecycle/` 将被新增
- 其余 4 个 spec（dispatch-lifecycle、supervisor-control-loop、worker-execution、review-feedback-loop）的 requirement 描述需更新引用
- 无代码影响（当前仓库尚无实现代码）
