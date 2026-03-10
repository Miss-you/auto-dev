# Auto-Dev 本地自动化开发系统方案

来源：
- Claude 分享链接：<https://claude.ai/share/ec110a1b-4b17-48c5-9129-252272c5082f>
- 本文件基于前述讨论中的最终方案整理，作为当前实现基线。

## 1. 目标

构建一个本地版的 Symphony：

- 以 GitHub Issues 作为主要任务源
- 以本地 `tmux` 会话作为 worker 运行容器
- 以 Claude Code / Codex / OpenCode 等本地 agent 作为执行引擎
- 以项目 spec 作为统一行为约束
- 形成 issue -> 开发 -> 测试 -> PR -> review -> 修复 的闭环

核心原则：

- 本地优先，尽量降低云端 token 成本
- workflow 可配置，必须能编排进自己的 spec
- 支持并发，但先保证单任务闭环可跑通
- 监工与工人职责分离

## 2. 系统整体架构

```text
GitHub Issues（任务源）
       ↓ 轮询 / Webhook
┌──────────────────────────┐
│ Dispatcher（Python）     │
│ - 拉取 issue             │
│ - 去重与分配             │
│ - 控制并发               │
│ - 管理生命周期           │
└──────────┬───────────────┘
           ↓ 启动
┌──────────────────────────┐
│ Supervisor（Claude Code）│
│ - 通过 tmux 管理工人      │
│ - capture-pane 判断进度   │
│ - send-keys 干预         │
│ - 验收与收尾             │
└──────────┬───────────────┘
           ↓ 管理 N 个
┌─────────┐ ┌─────────┐ ┌─────────┐
│ Worker 1│ │ Worker 2│ │ Worker 3│
│ (Agent) │ │ (Agent) │ │ (Agent) │
└────┬────┘ └────┬────┘ └────┬────┘
     ↓           ↓           ↓
   git push → 创建 PR → GitHub Actions / Review
                              ↓
                        Review 结果回来
                              ↓
                     Supervisor 派工人修复
```

## 3. 模块拆分

### 模块 1：项目 Spec 配置文件

这是整个系统的灵魂，建议放在项目根目录，用 `WORKFLOW.md`、`CLAUDE.md` 或二者组合维护。

至少应包含：

- 开发流程定义：拿到 issue -> 分析 -> 分支 -> 编码 -> 测试 -> 提交 -> PR
- 分支命名规范：例如 `feat/issue-42-xxx`
- commit message 规范
- 代码风格要求
- 测试要求
- PR 模板和描述要求
- 目录边界、禁改区域、架构约束

要求：

- 所有 worker 启动时默认读取
- supervisor 派工时默认引用
- review 修复阶段仍然遵守同一份 spec

### 模块 2：Tmux 操控 Skill

建议放在 `.claude/commands/tmux-ops.md`，给 supervisor 使用，定义标准操作。

核心动作：

- 创建 session：`tmux new-session -d -s worker-{issue_id}`
- 发送指令：`tmux send-keys -t worker-{issue_id} '...' Enter`
- 抓取输出：`tmux capture-pane -t worker-{issue_id} -p -S -{lines}`
- 检查存活：`tmux has-session -t worker-{issue_id}`
- 杀掉 session：`tmux kill-session -t worker-{issue_id}`
- 列出 session：`tmux list-sessions`

要求：

- supervisor 只能通过统一动作操控 worker，避免散乱命令
- capture 策略要支持“最近 N 行”和“扩大抓取范围”
- 要有明确的 session 命名规则

### 模块 3：Supervisor Skill

建议放在 `.claude/commands/supervisor.md`，是整个系统的核心编排逻辑。

职责：

1. 接收 dispatcher 分派的 issue
2. 为 issue 创建 tmux worker session
3. 在 session 内启动 Claude Code / Codex / OpenCode
4. 发送包含 issue 内容和 spec 约束的启动 prompt
5. 周期性抓取输出，判断状态
6. 在异常、阻塞、测试失败、review 反馈等场景下追加 prompt 干预
7. 验收任务结果
8. 清理 session 并更新状态

worker 启动 prompt 模板示例：

```text
你正在处理 GitHub Issue #{id}
标题：{title}
描述：{body}

请严格按照项目 CLAUDE.md / WORKFLOW.md 中的规范完成开发。
完成后请执行：
1. 运行必要测试
2. git add / commit / push
3. 创建 PR
```

状态判断规则：

- 正在分析或编码：继续等待
- 正在跑测试：继续等待
- 测试失败：发 prompt 要求定位并修复
- 明显报错且停滞：抓更多输出，再引导
- 等待权限确认：按预设策略发送确认输入
- 长时间无输出：判定卡死，进入重试或人工介入

### 模块 4：Worker Skill

建议放在 `.claude/commands/worker.md`，定义工人处理 issue 的标准动作。

标准流程：

1. 读取 issue 内容并理解需求
2. 新建分支：`git checkout -b feat/issue-{id}-{slug}`
3. 分析代码库与改动范围
4. 编码实现
5. 写或更新测试
6. 运行项目测试命令
7. 按规范提交：`git add` + `git commit`
8. 推送分支：`git push origin ...`
9. 用 `gh pr create` 创建 PR
10. 收到 review 修复任务后重复开发闭环

要求：

- 工人的行为尽量一致、可预测
- 对外只暴露清晰的状态与结果
- 不在未通过必要测试时直接宣告完成

### 模块 5：Dispatcher（Python）

这是系统入口，建议先做成一个轻量常驻脚本。

职责：

- 轮询 GitHub API 或接收 Webhook
- 拉取符合条件的 issue，例如带 `auto-dev` label
- 做去重，避免重复分派
- 控制最大并发 worker 数
- 启动或唤起 supervisor
- 记录状态、日志、开始时间、结束时间
- 做基础健康检查

建议初版能力：

- 单仓库支持
- 单 supervisor 进程
- 本地 JSON / SQLite 存状态
- 定时轮询优先，Webhook 后补

### 模块 6：Review 反馈回路

用于处理 PR 创建后的自动 review 和回修闭环。

可选组合：

- Claude Code GitHub Actions 做自动 review
- CodeRabbit 做 PR review
- GitHub 原生 review comments 作为输入源

流程：

1. PR 创建后触发 review
2. dispatcher / supervisor 拉回 review comments
3. supervisor 把反馈整理后发给对应 worker
4. worker 修复并重新 push
5. 重复直到通过或达到重试上限

要求：

- review 信息要可回溯
- 需要有最大修复轮次限制
- 失败要能回到人工处理

### 模块 7：状态面板（可选但推荐）

作用是让你快速知道系统现在在做什么。

可以先做成终端 dashboard，后续再升级成简单本地 Web 页面。

建议展示：

- 当前运行中的 worker 数
- 每个 issue 当前阶段
- 最近一次输出时间
- 最近错误
- 成功 / 失败统计
- 总处理耗时

## 4. 数据与状态模型

建议为每个 issue 维护最小状态机：

- `queued`
- `dispatching`
- `running`
- `waiting_review`
- `fixing_review`
- `completed`
- `failed`
- `aborted`

每条任务至少记录：

- issue id
- title
- 当前分支
- tmux session 名称
- 当前 worker agent 类型
- 当前状态
- 最近心跳时间
- 最近一次错误摘要
- PR 链接

## 5. MVP 范围

第一阶段只做最小闭环，不追求一步到位：

1. 从 GitHub 拉取指定 label 的 issue
2. 启动单个 worker 的 tmux session
3. 让 worker 按 spec 开发
4. supervisor 能查看输出并做简单干预
5. worker 能提交代码并创建 PR

暂不强求：

- 多仓库
- 复杂权限系统
- 图形化控制台
- 完整自愈策略
- 高级调度策略

## 6. 推荐构建顺序

### 第一阶段：基础设施

1. 写好项目 spec：`CLAUDE.md` / `WORKFLOW.md`
2. 写 tmux 操控 skill
3. 手动测试 tmux 控制链路

验收标准：

- 能手动创建 session
- 能手动启动 agent
- 能稳定 `send-keys`
- 能稳定 `capture-pane`

### 第二阶段：单任务闭环

4. 写 worker skill
5. 写 supervisor skill
6. 手动指定一个 issue 跑完整流程

验收标准：

- 能完成一次 issue -> commit -> PR
- supervisor 能识别常见卡点
- 基础清理逻辑可用

### 第三阶段：自动化与并发

7. 写 dispatcher 脚本
8. 接入 GitHub 轮询或 Webhook
9. 支持多个 worker 并发
10. 接 review 反馈回路

验收标准：

- 多 issue 不互相污染
- 并发上限可控
- review 修复至少能跑通一轮

### 第四阶段：稳定性与体验

11. 加异常处理
12. 加状态面板
13. 调优 capture 策略、轮询频率、重试策略

验收标准：

- 能识别卡死、网络失败、git 冲突
- 能恢复或失败退出
- 你可以快速看到整体运行状态

## 7. 关键设计取舍

### 为什么本地优先

- 控制 token 成本
- 更容易利用本地已登录的 Claude Code / Codex 环境
- 更容易接入项目私有上下文和本地工具链

### 为什么用 tmux

- 长任务不断线
- 可以后台并发多个 worker
- 可以通过 `capture-pane` 做监督
- 比一次性 `claude -p` 更适合持续迭代

### 为什么分 supervisor 和 worker

- 让“管理任务”和“写代码”两类行为分离
- 更容易做状态判断、重试、异常恢复
- 方便后续横向扩展多个 worker

## 8. 当前方案的最大价值

这套方案最大的价值，不只是“让多个 agent 并发”，而是：

- 把你自己的 spec 编入自动化流程
- 让 GitHub 成为任务和验收闭环的中心
- 让实际开发尽量在本地完成
- 让成本、可控性和扩展性都更适合个人或小团队

如果后续进入实现阶段，建议直接从以下三个文件开始：

- `CLAUDE.md` 或 `WORKFLOW.md`
- `.claude/commands/tmux-ops.md`
- `.claude/commands/supervisor.md`
