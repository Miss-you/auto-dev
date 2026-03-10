## Context

当前仓库仍处于方案沉淀阶段：`doc/plan.md` 给出了本地优先、spec-driven、supervisor/worker、tmux-based execution、GitHub-centric 的整体方向，`doc/ref.md` 则保留了从调研到收敛的设计背景。`openspec/` 目前尚未形成任何主规格，因此后续每个实现 change 如果直接从方案文档起步，会造成边界不稳、命名漂移和依赖关系重复讨论。

这个 planning change 的目标不是实现系统，而是把整体方案翻译成一套稳定的 capability contract，让后续变更优先引用主规格，再逐步补实现。由于当前没有代码包袱，最合适的做法是一次性建立 capability 基线，然后把 change 归档为“初始化规划”。

## Goals / Non-Goals

**Goals:**
- 把 `doc/plan.md` 中的整体方案落成可演进的 OpenSpec 主规格基线。
- 定义一组边界清晰、前后依赖明确、可并行推进的 capability。
- 约束未来 change 优先围绕 capability 修改规格，而不是重复改写方案文档。
- 保留足够抽象层级，避免把 GitHub、tmux、Claude Code、Codex 等实现选择过早硬编码为不可替换的架构事实。

**Non-Goals:**
- 不实现 dispatcher、supervisor、worker、dashboard 或任何运行时代码。
- 不冻结所有实现细节，例如最终默认存储使用 JSON 还是 SQLite、默认采用轮询还是 Webhook。
- 不在本变更中定义完整的测试矩阵、CI 流程或 UI 交互设计。
- 不把未来所有实现 change 一次性展开为独立 proposal；这里只建立主规格和推荐拆分方式。

## Decisions

### Decision: 先创建一个规划型 foundation change，再 sync 为主规格

- Why: 当前仓库还没有任何主规格，直接逐个实现 change 会导致每个 change 都重复解释能力边界。先用一个 planning change 形成基线，再归档，可以让后续 change 直接在 `openspec/specs/` 上演进。
- Alternative considered: 直接创建多个 implementation-first change。
- Why not: 那样会把“能力边界定义”和“实现选择”耦合在一起，降低并行推进时的一致性。

### Decision: capability 按“系统职责边界”拆分，而不是按 `doc/plan.md` 模块标题一比一拆分

- Why: `doc/plan.md` 里的“Tmux 操控 Skill”“Supervisor Skill”“Worker Skill”更接近实现载体，不是稳定的长期 capability 边界。长期稳定的边界应围绕 contract、source、runtime、control、dispatch、review、ops 来定义。
- Alternative considered: 按模块标题直接落 7 个 capability。
- Why not: 这会让 runtime、worker、supervisor 三者的职责重叠，后续很难清楚描述谁负责约束、谁负责执行、谁负责监管。

### Decision: `workflow-contract` 作为根 capability

- Why: 它承载统一状态机、仓库操作约束、项目 spec 注入和完成标准，是 dispatcher、supervisor、worker 共享语义的源头。没有这层 contract，其他 capability 的 spec 很容易出现命名不一致和完成定义不一致。
- Alternative considered: 让每个 capability 各自定义自己的状态和完成条件。
- Why not: 会造成跨模块协作时的语义冲突，特别是在 review 回流和异常恢复场景里。

### Decision: 将 `task-source-adapter`、`session-runtime`、`worker-execution` 设计成彼此解耦的早期并行能力

- Why: 任务源适配、session 承载和 worker 协议分别属于输入、运行容器和执行契约，是天然可以并行定义和后续并行实现的三块。
- Alternative considered: 合并成一个“大而全的 worker runtime” capability。
- Why not: 会让 GitHub 适配、tmux 语义和 worker 完成标准互相纠缠，降低未来替换 provider 或 runtime 的弹性。

### Decision: `supervisor-control-loop` 依赖 `session-runtime` 与 `worker-execution`，`dispatch-lifecycle` 再依赖 supervisor 与 task source

- Why: supervisor 需要知道如何操控 session，也需要知道 worker 完成什么算成功；dispatcher 则是在此之上负责选择任务、触发运行和保存状态。
- Alternative considered: 让 dispatcher 直接操控 worker session。
- Why not: 会把编排入口和运行中控制耦合，增加异常处理和多任务扩展的复杂度。

### Decision: `review-feedback-loop` 与 `operations-observability` 在 dispatcher MVP 之后并行扩展

- Why: review 回流和运行可观测性都依赖最小闭环先跑通，但它们之间不必互为前置，适合并行推进。
- Alternative considered: 把 review 和 observability 作为 dispatcher 的一部分。
- Why not: 会让 dispatcher MVP 过大，阻碍最小闭环尽快稳定。

### Decision: 本 change 的 `tasks.md` 记录“规划交付任务”，而不是未来所有实现任务

- Why: 这个 change 的工作内容是建立规格基线、完成自检、同步主规格和归档；未来实现需要各自独立的 change 和任务列表。
- Alternative considered: 在本 change 中写入未来全部实现任务。
- Why not: 那会让本 change 永远无法真实完成，也不符合后续按 capability 拆 change 的目标。

## Risks / Trade-offs

- [规格先于代码，可能过度抽象] -> 通过只约束 contract 和边界、不提前冻结细节实现来降低风险。
- [capability 仍可能存在边界重叠] -> 通过在主规格中强调输入、运行、控制、派发、反馈、运维的职责边界来降低重叠。
- [规划 change 归档后，后续实现可能偏离基线] -> 要求未来实现 change 优先修改主规格，再开展实现。
- [当前没有真实运行验证，部分细节会在实现时调整] -> 在后续 change 中使用 `MODIFIED Requirements` 演进规格，而不是回改方案文档。

## Migration Plan

1. 在本 change 中创建 proposal、design、delta specs 与 planning tasks。
2. 对 change 执行一次严格自检并修复格式或内容问题。
3. 将 delta specs sync 到 `openspec/specs/` 作为主规格基线。
4. 对主规格执行一次自检并修复问题。
5. 归档本 planning change，后续实现从主规格继续推进。

## Open Questions

- dispatcher MVP 默认状态存储优先采用本地 JSON 还是 SQLite。
- GitHub 接入的默认策略优先使用轮询、Webhook，还是二者混合。
- `operations-observability` 是否在后续实现阶段再细分为观测能力与恢复能力两个独立 change。
