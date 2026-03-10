## Context

当前 `openspec/specs/workflow-contract/spec.md` 包含 3 个 requirement（Unified Workflow Contract、Project Guidance Injection、Repository Action Policy），但均停留在"应该有"的抽象层面，没有定义具体的状态名、转移规则、阶段门禁和终态语义。

在前期探索中已确认：
- 系统实际上存在两个状态机：OpenSpec 变更生命周期（阶段和门禁）和任务运行状态机（状态和转移）。
- 开发流程分两阶段：先生成并审批 spec（Phase A），再基于已批准 spec 实现代码（Phase B），两者之间有显式审批门禁。
- Spec 阶段包含 AI 自检修复循环，只有自检通过才能提交人工审查。
- OpenSpec change 在两个阶段中都保持 active，sync + archive 延迟到实现 verify 之后。
- worker 读 CLAUDE.md/AGENTS.md 是 coding CLI 的隐含行为；supervisor 给 worker 的约束通过 prompt 注入完成——这些不需要单独建 requirement。

参考文档：`doc/plan.md`（整体方案）、`doc/ref.md`（设计背景）。

## Goals / Non-Goals

**Goals:**
- 将 workflow-contract 拆成 change-lifecycle 和 run-lifecycle 两个边界清晰的 spec。
- change-lifecycle 只管"流程怎么走"：两阶段、门禁、自检循环、sync/archive 时机。
- run-lifecycle 只管"状态怎么转"：规范状态名、允许转移、触发者、终态语义。
- 退役 workflow-contract，更新其余 4 个 spec 的引用关系。

**Non-Goals:**
- 不修改 dispatch-lifecycle、supervisor-control-loop、worker-execution、review-feedback-loop 的核心 requirement 逻辑，只更新它们的引用描述。
- 不定义 CLAUDE.md / AGENTS.md / WORKFLOW.md 的具体内容细则。
- 不定义状态持久化的实现方式（属于 dispatch-lifecycle）。
- 不定义 worker 怎么写代码/测试/提 PR 的细节（属于 worker-execution）。

## Decisions

### Decision: 拆成 2 个 spec 而不是 4 个

- Why: 当前还在定"主编排骨架"，不是在细化实现细节。4 个会产生很多"很薄但互相引用"的 spec，review 成本更高。
- Alternative considered: 拆成 4 个（change-lifecycle、run-lifecycle、guidance-resolution、publish-gates）。
- Why not: guidance-resolution 单独成 spec 太薄；publish-gates 和 worker-execution 高度重叠。等后面真复杂了再从 change-lifecycle 里拆出子 spec。

### Decision: spec_drafting 内部包含 AI 自检修复循环

- Why: 生成 spec 产物后 AI 需要自检并修复，这是 spec 阶段的关键质量保证。从外部状态机视角看，自检未通过时状态保持在 spec_drafting，不发生状态转移。
- Alternative considered: 增加 spec_validating 状态。
- Why not: 自检修复是 spec_drafting 的内部行为，不需要暴露给 dispatcher 或 supervisor 做调度决策。

### Decision: Guidance resolution 不单独建 requirement

- Why: worker 读 CLAUDE.md/AGENTS.md 是 coding CLI 的隐含规则；supervisor 给 worker 的约束是 prompt 注入（类似 tmux send-keys 发消息时拼入需要遵守的规范）。这些是实现层的事，不需要在 change-lifecycle 或 run-lifecycle 中定义。
- Alternative considered: 在 change-lifecycle 中加 Role-Specific Guidance Resolution requirement。
- Why not: 会让 change-lifecycle 越界管"谁读什么文件"，偏离"流程怎么走"的职责。

### Decision: 延迟 sync + archive 到 verify 之后

- Why: 如果 spec review 一通过就 sync + archive，后面实现阶段发现 spec 要调就得 reopen。保持 change active 到 verify 之后再归档更稳。
- Alternative considered: spec 批准后立即 sync，实现阶段开新 change。
- Why not: 增加 change 数量，且 spec 和实现的关联会断开。

### Decision: workflow-contract 直接退役删除

- Why: 其职责完全被 change-lifecycle + run-lifecycle 覆盖，保留为"伞 spec"只增加维护成本。
- Alternative considered: 保留 workflow-contract 作为伞 spec 只写引用。
- Why not: 增加一层间接引用，没有额外价值。

## Risks / Trade-offs

- [两个 spec 之间存在耦合] → change-lifecycle 定义阶段，run-lifecycle 定义状态，状态与阶段有映射关系。通过在 run-lifecycle 中显式标注"Phase A 状态"和"Phase B 状态"来管理这个耦合，而不是让两边各自隐含。
- [退役 workflow-contract 是 BREAKING 变更] → 当前没有实现代码依赖它，影响范围仅限于 spec 文档引用。
- [状态粒度可能在实现时调整] → 状态集是可演进的，后续可通过新 change 增减状态。当前粒度基于两阶段流程和 review 循环的实际需要。
