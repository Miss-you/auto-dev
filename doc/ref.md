# Auto-Dev 方案头脑风暴参考

来源：
- Claude 分享链接：<https://claude.ai/share/ec110a1b-4b17-48c5-9129-252272c5082f>
- 当前文件基于你补充贴出的完整讨论内容整理，作为后续设计参考。

## 1. 起点问题

这轮讨论的核心目标，是评估是否可以构建一个类似 Symphony 的自动化开发系统，并且尽量满足以下诉求：

- 任务来源可以接 GitHub，也可以扩展到公司内部系统，比如 TAPD。
- 真正执行开发工作的 agent 要尽量跑在本地。
- 要能接入自己的 spec，而不是只能按工具默认流程工作。
- 最好能形成从 issue 到 PR，再到 review 修复的闭环。
- 成本要低，尤其是避免把大量开发 token 消耗在云端 agent 上。

## 2. Symphony 是什么

Symphony 被理解为一个 AI 编码 agent 的编排器/调度器。它不是单个写代码的 agent，而是一套把任务系统、开发环境、执行 agent、验收与清理串起来的自动化工作流。

讨论里对它的定位总结为：

- 它会从任务系统中拉取 issue。
- 它会为每个任务创建独立工作空间。
- 它会启动独立 agent 去完成任务。
- 完成后会附带 CI、review、复杂度等证明材料。
- 在任务被关闭或取消后，会自动停止相关 agent 并清理资源。

对于“并发是不是它最大的点”这个问题，讨论结论是：

- 并发确实是重要能力。
- 但更核心的价值不是单纯“多开几个 agent”，而是“工作流编排”。
- `max_concurrent_agents` 这种配置说明它天然支持并行处理多个任务。
- 并发是编排能力的结果，不是唯一卖点。

## 3. 相关技术名词梳理

### Linear

可以类比 TAPD、Jira、飞书项目这类 issue/看板系统。Symphony 从 Linear 拉任务，本质上只是一个任务追踪器适配层问题。

### Elixir

Elixir 是运行在 Erlang 虚拟机上的编程语言，语法相对友好，适合写高并发、高可靠的后端服务。

### Erlang / BEAM / OTP

这三个词在讨论里被拆成一套运行时体系：

- Erlang：一门为高可靠场景设计的语言。
- BEAM：Erlang/Elixir 的虚拟机。
- OTP：一套成熟的并发与监督模型，适合管理长时间运行的进程。

之所以会出现在 Symphony 里，是因为“同时管理很多长期运行的 agent”这个问题，天然适合 Erlang/OTP 的进程模型。

### mise

`mise` 是多语言版本管理工具，用来安装和固定 Elixir、Erlang、Node.js、Python 等运行时版本。它在 Symphony 的参考实现里承担环境准备工作。

## 4. 是否可以自己做

讨论中的答案是明确的：可以，而且完全值得做。

### 方向 A：对接 TAPD

如果面向公司内部团队协作，可以把任务追踪器那层替换成 TAPD：

- 拉取待处理任务
- 认领任务
- 更新状态
- 写评论或回填结果

核心编排流程不需要变，只是把 Linear 的适配层换成 TAPD 的 API 调用。

### 方向 B：先做 GitHub + 本地开发

这是讨论里更推荐的起步路线，因为更容易落地：

- 用 GitHub Issues 或 GitHub Projects 作为任务源。
- 用本地脚本轮询或接 Webhook。
- 每个 issue 建独立分支或工作空间。
- 调本地 agent 开发。
- 自动提交、推送、创建 PR。
- 再根据 review 结果做修复。

这个方向的优势是：

- 不依赖 Linear。
- 不依赖 Elixir。
- 不依赖官方 Codex 编排体系。
- 更适合个人或小团队先做原型。

## 5. 相关替代项目调研结论

讨论里区分了两类东西：一类是“云端全自动编排产品”，另一类是“本地优先的 agent 编排/执行方案”。

### 更接近成品的平台

#### GitHub Copilot Coding Agent

优点：

- 在 GitHub 上直接工作，issue 到 PR 流程成熟。
- 会自动建分支、写代码、跑 CI、做 review。
- 能根据 review 反馈继续修。

缺点：

- 更偏云端托管。
- 本地执行与成本可控性不符合当前最核心诉求。

#### OpenCode

优点：

- 直接对接 GitHub Issues / PR。
- 可以自动 review、自动分类 issue。
- 模型选择更灵活。

缺点：

- 主要依赖 GitHub Actions 等自动化流程。
- 本地优先程度不如“自己编排 + 本地 agent”方案。

#### CodeRabbit

适合补足 review 环节，不是完整的 issue -> PR 编排器。

#### Cline

是本地编码 agent，模型无关，但默认不提供完整的 GitHub issue 编排闭环。

#### Graphite Agent

适合 GitHub + stacked PR 工作流，但比较偏团队工作流体系，不是当前最佳切入点。

### 更贴近“本地执行”需求的方案

#### CCPM（Claude Code PM）

讨论里认为它非常接近“GitHub Issues 作为任务库，本地启动多个 Claude Code 实例并行处理”的方向。

#### 1Code

可以从 GitHub、Linear、Slack、git 事件触发 agent，并支持本地运行或云端运行，属于“本地编排层”这一类更值得重点参考的项目。

## 6. 真正的核心诉求：GitHub 监听 + 本地 agent 执行

后续讨论逐步收敛出真正的关键点：

- 不是单纯要“自动写代码”。
- 而是要“监听 GitHub 事件，然后在本地拉起 Codex / Claude Code / OpenCode 去执行开发”。
- 本地执行非常重要，因为 token 成本是主要约束。

这个诉求带来的架构方向是：

- GitHub 负责任务源和 PR/review 数据源。
- 本地常驻调度器负责监听与派发。
- 本地 agent 负责真实代码修改。
- 云端只在必要时承担少量 review 或集成动作。

## 7. spec 必须可编排

这是整个方案最重要的约束之一。

讨论里明确指出，真正想要的并不是一个“自动写代码工具”，而是一个“能遵守自己团队 spec 的自动化开发系统”。因此，spec 需要是一级对象。

spec 可注入的层次被拆成三层：

- 项目级规范文件，比如 `CLAUDE.md` 或 `WORKFLOW.md`
- 自定义 commands / skills
- 每次派发给 worker 的任务 prompt 模板

换句话说：

- spec 不只是“提示词补充”
- 而是整个 worker 行为、分支规范、测试规范、提交流程、PR 规范的来源
- 后续的 dispatcher、supervisor、worker 都要围绕 spec 执行

## 8. 为什么从 `claude -p` 转向 `tmux`

讨论中一个重要转折点，是把“用 `claude -p` 单次执行任务”切换为“在 `tmux` 中启动交互式 Claude Code/Codex 会话”。

原因主要有四个：

- `claude -p` 是一次性调用，过程不可持续观察。
- `tmux` 可以保留完整交互式会话，更像一个持续工作的工人进程。
- 可以用 `capture-pane` 抓取输出，便于 supervisor 监督。
- 断线后任务仍然继续，便于长时间运行和多任务并发。

讨论里对 `tmux` 的理解可以概括为：

- 它是终端复用器。
- 能启动多个后台 session。
- 每个 session 可以跑一个独立 worker。
- supervisor 可以通过 `send-keys` 发指令，通过 `capture-pane` 看进度。

## 9. 监工 + 工人模型

这是整个方案最终成型前最关键的脑暴结论。

模型被描述为：

- 你当前交互的 Claude Code 充当“监工”。
- 后台多个 `tmux` session 里的 Claude Code/Codex 充当“工人”。
- 监工不亲自写业务代码，而是负责派工、巡检、催办、纠偏、收尾。

监工的职责：

- 拉任务
- 创建 session
- 启动 worker
- 抓取输出判断状态
- 在错误或卡住时追加 prompt
- 在信息不足时继续抓更多输出
- 完成后验收结果，推进 PR / review / cleanup

工人的职责：

- 理解 issue
- 修改代码
- 跑测试
- 修复失败
- 提交代码
- 推送分支
- 创建 PR

这个模型的好处是：

- 逻辑上清晰
- 天然支持并发
- 本地执行成本更低
- 管理动作和开发动作分离

## 10. 最终收敛出的设计原则

这轮脑暴最终收敛出几个后续设计必须坚持的原则：

- `local-first`：核心开发尽量在本地执行。
- `spec-driven`：你的 spec 必须贯穿任务派发、执行、验收全过程。
- `supervisor/worker`：采用监工 + 工人的多 agent 结构。
- `tmux-based execution`：用 `tmux` 承载可监督、可恢复、可并发的 worker 会话。
- `GitHub-centric`：优先接 GitHub Issues、PR、review、CI 形成闭环。
- `progressive build`：先做单任务 MVP，再扩展并发、review 闭环和面板。

## 11. 对后续实现最有价值的直接结论

如果基于这轮讨论直接往下做，最重要的不是立刻追求完整平台，而是先做一个最小可运行版本：

1. GitHub issue 监听。
2. 本地 `tmux` session 启动 worker。
3. worker 读取项目 spec 开发。
4. supervisor 用 `capture-pane` 监控并干预。
5. worker 自动提交并创建 PR。
6. 后续再补 review 自动修复和并发控制。

`ref.md` 的作用，就是保留这些从调研、理解、转向到收敛的思考路径，避免后面直接进入实现时丢掉设计意图。
