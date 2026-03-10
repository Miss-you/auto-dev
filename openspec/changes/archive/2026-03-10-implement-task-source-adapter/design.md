## Context

auto-dev 需要从 GitHub Issues 拉取任务，归一化后供 dispatcher 使用。task-source-adapter spec 定义了 5 个 requirement：归一化模型、轮询机制、过滤去重、provider 抽象、错误弹性。本设计决定如何将这些 requirement 映射到 Go 代码结构。

已实现的 `internal/runlifecycle/` 包提供了 run 状态机。task-source-adapter 是 dispatcher 的输入端，两者通过 `NormalizedTask` 模型解耦。

参考了 [OpenAI Symphony](https://github.com/openai/symphony) 的 Tracker 架构：Elixir behaviour 接口 + Linear adapter + Memory test double。

## Goals / Non-Goals

**Goals:**
- 定义 `NormalizedTask` 模型作为系统中任务的唯一规范表示
- 实现 `Provider` Go 接口，隔离所有 provider-specific 细节
- 实现 GitHub Issues provider（基于 `google/go-github`）
- 实现可配置间隔的轮询引擎
- 实现 source-side 过滤（label/state）和会话级去重
- 实现状态回写（comment、label add/remove）
- 实现错误弹性（transient retry、rate limit backoff、auth failure stop）
- 提供 Memory provider 测试替身

**Non-Goals:**
- Webhook 接收（未来扩展）
- 持久化本地状态（provider API 是 source of truth）
- 多 provider 同时轮询（首版仅支持单 provider）
- dispatcher 集成（单独的 change）

## Decisions

### Decision 1: 包结构

```
internal/tasksource/
├── model.go           # NormalizedTask 模型
├── provider.go        # Provider 接口定义
├── errors.go          # ErrAuthFailure, RateLimitError
├── filter.go          # FilterConfig + 过滤逻辑
├── poller.go          # Poller 轮询引擎
├── github_provider.go # GitHub Issues 实现
└── memory_provider.go # 测试替身
```

**为什么单包不拆子包？** 这是一个内聚的 adapter 层，类型之间紧密关联（NormalizedTask 被所有文件引用），拆子包只增加 import 复杂度。与 `internal/runlifecycle/` 保持一致的组织风格。

### Decision 2: Provider 接口设计

```go
type Provider interface {
    // FetchCandidateTasks 从外部源拉取所有候选任务（自动处理分页）。
    FetchCandidateTasks(ctx context.Context) ([]NormalizedTask, error)

    // PostComment 在外部源的指定任务上发表评论。
    PostComment(ctx context.Context, externalID string, body string) error

    // AddLabels 在外部源的指定任务上添加标签（不影响已有标签）。
    AddLabels(ctx context.Context, externalID string, labels []string) error

    // RemoveLabel 从外部源的指定任务上移除单个标签。
    RemoveLabel(ctx context.Context, externalID string, label string) error
}
```

**为什么 4 个方法？**
- `FetchCandidateTasks` 合并了 Symphony 的 `fetch_candidate_issues` 和 `fetch_issues_by_states`——过滤逻辑内聚在 provider 实现中，自动处理分页
- `PostComment` 对应 Symphony 的 `create_comment`
- `AddLabels` / `RemoveLabel` 替代原设计的 `UpdateLabels`——避免 `ReplaceLabelsForIssue` 的破坏性全量替换，只管理 auto-dev 拥有的标签，不影响用户手动添加的标签
- 去掉了 `fetch_issue_states_by_ids`——reconciliation 是 dispatcher 的职责，不在 adapter scope

**ExternalID 类型约定**：`NormalizedTask.ExternalID` 是 string 类型（适配不同 provider），GitHub provider 内部使用 `strconv.Atoi(externalID)` 将其转回 issue number。ExternalID 存储的是 GitHub issue number 的字符串形式（如 "42"），不是 node ID。

**替代方案**：把读和写拆成 `TaskReader` + `TaskWriter` 两个接口。但当前只有一个 provider（GitHub），接口分离带来的灵活性不值得额外复杂度。

### Decision 3: 轮询引擎架构

```
                      Poller.Run(ctx) loop
                      ┌──────────────────────────────────────┐
                      │                                      │
  time.Ticker  ──tick─▶  Provider.FetchCandidateTasks(ctx)  │
                      │         │ (handles pagination)       │
                      │         ▼                            │
                      │    FilterConfig.Apply()              │
                      │         │                            │
                      │         ▼                            │
                      │    Dedup (seen map)                  │
                      │         │                            │
                      │         ▼                            │
                      │    OnNewTasks([]NormalizedTask)       │
                      │    (synchronous callback)            │
                      └──────────────────────────────────────┘
```

Poller 持有一个 `time.Ticker`，每个 tick：
1. 调用 `Provider.FetchCandidateTasks(ctx)`（provider 内部自动处理分页，返回所有匹配 issue）
2. 通过 `FilterConfig` 过滤（label include/exclude、state whitelist）
3. 会话级去重（`seen` map by ExternalID + UpdatedAt）
4. 将新任务通过同步回调函数发送给消费者

**为什么用回调 `func([]NormalizedTask)` 而不是 channel？** 回调更容易测试（直接传 mock function），且消费者（dispatcher）的处理逻辑是同步的。如果未来需要异步，可以在回调内部发送到 channel。

**分页处理**：`FetchCandidateTasks` 内部使用 go-github 的 `ListByRepo` 方法，每页 100 条，通过 `Response.NextPage` 迭代直到 `NextPage == 0`。确保不遗漏任何匹配 issue。

**GitHub API 参数映射**：`IssueListByRepoOptions.Labels` 使用 AND 语义（issue 必须包含所有指定 label）。`IssueListByRepoOptions.State` 接受 "open"/"closed"/"all"。首版不使用 `Since` 参数（增量轮询），依赖全量拉取 + 会话级去重。

**优雅关闭**：当 ctx 被取消时，当前 in-flight 的 poll 调用会通过 ctx 传播取消。`Run` 在 ctx 取消后返回 `nil`（不返回 `context.Canceled`），且不会在返回后再调用回调。

**并发不变量**：`seen` map 仅在 Poller 的 `Run` goroutine 中访问，回调是同步调用的，因此无需 mutex。如果未来回调变为异步，`seen` map 需要加锁保护。

### Decision 4: GitHub Provider 认证

使用 `golang.org/x/oauth2` + Personal Access Token（PAT）。

```go
func NewGitHubProvider(cfg GitHubConfig) (*GitHubProvider, error)

type GitHubConfig struct {
    Token      string   // PAT or GitHub App token
    Owner      string   // repo owner
    Repo       string   // repo name
    Labels     []string // include labels (GitHub API AND filter)
    State      string   // "open", "closed", "all" (default: "open")
    PerPage    int      // items per page, default 100 (max)
}
```

**为什么 PAT 而不是 GitHub App？** PAT 更简单，适合单机本地部署场景。GitHub App 认证可以未来添加（换一个 `http.Client` 即可，go-github 的设计支持这种替换）。

### Decision 5: 过滤和去重策略

**过滤**（两层）：
1. Provider 层：GitHub API 原生过滤（labels、state 参数直接传给 API，减少网络传输）
2. Adapter 层：`FilterConfig` 做二次过滤（label exclude、自定义规则）——因为 GitHub API 不支持 label 排除

**去重**（adapter 层，非权威）：
- `seen` map: `map[string]time.Time` (key=ExternalID, value=UpdatedAt)
- 如果 ExternalID 已在 map 中且 UpdatedAt 未变，跳过
- 如果 UpdatedAt 变了，重新 yield（说明 issue 有更新）
- 每 100 个 poll 周期（约 50 分钟 @30s 间隔）清空 `seen` map，防止无限增长。清空只会导致重新 yield 已知 issue，由 dispatcher 做 run-level 去重保证正确性
- 这是性能优化，不是正确性保证——权威去重在 dispatcher

### Decision 6: 错误处理策略

go-github 有两种 rate limit 错误类型，需要分别处理：

| 错误类型 | 行为 |
|---------|------|
| 网络 transient (timeout, DNS) | warn log，下次 tick 重试 |
| `*github.RateLimitError` (primary, 5000/hr) | 从 `err.Rate.Reset` 计算等待时间，动态调整下次 poll 延迟 |
| `*github.AbuseRateLimitError` (secondary) | 从 `err.GetRetryAfter()` 获取等待时间，动态调整下次 poll 延迟 |
| GitHub 401 Unauthorized | error log，停止轮询，返回 `ErrAuthFailure` |
| GitHub 403 Forbidden (非 rate limit) | error log，停止轮询，返回 `ErrAuthFailure` |
| 其他 4xx/5xx | warn log，下次 tick 重试 |

**注意**：GitHub 的 secondary rate limit 也返回 403，但 go-github 会将其解析为 `*github.AbuseRateLimitError`。代码中先检查 `AbuseRateLimitError`，再检查 `RateLimitError`，最后检查 HTTP 状态码。

**为什么不用指数退避？** 我们已有固定间隔轮询（30s），transient 错误在下一个 tick 自然重试。只有 rate limit 需要特殊处理（可能需要等待数分钟）。

### Decision 7: 依赖管理

| 依赖 | 用途 | 版本策略 |
|------|------|---------|
| `github.com/google/go-github/v68` | GitHub REST API client | 实现时使用最新稳定大版本 |
| `golang.org/x/oauth2` | Token 认证 | go-github 传递依赖 |
| `log/slog` | 结构化日志 | Go 标准库，无外部依赖 |

**为什么不用 `github.com/shurcooL/githubv4` (GraphQL)?** REST API 对 Issues 操作足够，go-github 更成熟，团队更熟悉。GraphQL 的优势（字段选择、减少请求数）在 issue 列表场景下不显著。

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| go-github 大版本更新频繁 | 用 Go module 版本管理，更新时只需改 import path；tasks 不硬编码版本号 |
| PAT token 泄漏 | 从环境变量读取，不硬编码；.gitignore 排除 .env 文件 |
| GitHub API rate limit (5000/hr authenticated) | 30s 间隔 = 120 req/hr（含分页则更多），远低于上限 |
| 单 provider 设计可能限制未来扩展 | Provider 接口已抽象，新增 provider 只需实现接口 |
| 轮询延迟最大 30s | 对 Issue 场景可接受；未来可降低间隔或加 webhook |
| 会话级去重在重启后丢失 | By design：provider API 是 source of truth，重启后全量重新拉取，由 dispatcher 做 run-level 去重 |
| seen map 长期运行增长 | 每 100 个周期清空一次，非正确性依赖 |
| ExternalID string↔int 转换 | GitHub provider 内部处理，转换失败返回明确错误 |
