## 1. 基础类型（模型、接口、错误）

- [x] 1.1 在 `internal/tasksource/model.go` 中定义 `NormalizedTask` 结构体（ExternalID, ExternalKey, Title, Body, Labels, Priority, SourceType, SourceURL, Metadata, CreatedAt, UpdatedAt + json tag）
- [x] 1.2 在 `internal/tasksource/provider.go` 中定义 `Provider` 接口（FetchCandidateTasks, PostComment, AddLabels, RemoveLabel —— 均接受 context.Context）
- [x] 1.3 在 `internal/tasksource/errors.go` 中定义 sentinel errors：`ErrAuthFailure`；定义 `RateLimitError` 结构体（RetryAfter time.Time），实现 `error` 接口
- [x] 1.4 在 `internal/tasksource/filter.go` 中定义 `FilterConfig` 结构体（IncludeLabels, ExcludeLabels, States）和 `Apply(tasks []NormalizedTask) []NormalizedTask` 方法
- [x] 1.5 编写 `model_test.go`：验证 NormalizedTask 的 JSON 序列化往返正确性、零值默认值语义
- [x] 1.6 编写 `filter_test.go`：表驱动测试覆盖 label include/exclude、state 过滤、空配置透传、组合过滤

## 2. Memory Provider（测试替身）

- [x] 2.1 在 `internal/tasksource/memory_provider.go` 中实现 `MemoryProvider`，满足 `Provider` 接口
- [x] 2.2 `MemoryProvider` 支持预置任务列表、记录 PostComment/AddLabels/RemoveLabel 调用历史、可配置返回错误
- [x] 2.3 编写 `memory_provider_test.go`：验证接口契约（fetch 返回预置任务、写操作调用被记录、错误注入工作）

## 3. GitHub Provider

- [x] 3.1 运行 `go get github.com/google/go-github/v68`（使用实现时最新稳定版本）添加依赖
- [x] 3.2 在 `internal/tasksource/github_provider.go` 中定义 `GitHubConfig` 结构体和 `NewGitHubProvider(cfg GitHubConfig) (*GitHubProvider, error)` 构造函数
- [x] 3.3 实现 `FetchCandidateTasks`：调用 `IssuesService.ListByRepo`，每页 100 条，通过 `Response.NextPage` 循环处理所有分页，按 labels/state 过滤，归一化为 `NormalizedTask`（issue number 通过 `strconv.Itoa` 转为 ExternalID）
- [x] 3.4 实现 `PostComment`：`strconv.Atoi(externalID)` 转换后调用 `IssuesService.CreateComment`
- [x] 3.5 实现 `AddLabels`：调用 `IssuesService.AddLabelsToIssue`；实现 `RemoveLabel`：调用 `IssuesService.RemoveLabelForIssue`
- [x] 3.6 实现错误处理：先检查 `*github.AbuseRateLimitError`（取 `GetRetryAfter()`），再检查 `*github.RateLimitError`（从 `err.Rate.Reset` 计算等待时间），最后检查 HTTP 401/403 返回 `ErrAuthFailure`
- [x] 3.7 编写 `github_provider_test.go`：使用 `httptest.NewServer` mock GitHub API，测试 fetch 归一化（含分页）、comment 发送、label add/remove、primary rate limit 处理、secondary rate limit (abuse) 处理、auth 失败处理、ExternalID 转换错误处理

## 4. 轮询引擎

- [x] 4.1 在 `internal/tasksource/poller.go` 中定义 `PollerConfig`（Interval, Provider, Filter, OnNewTasks callback, SeenMapResetInterval）和 `Poller` 结构体
- [x] 4.2 实现 `NewPoller(cfg PollerConfig) *Poller` 和 `(*Poller).Run(ctx context.Context) error`（阻塞直到 ctx 取消，ctx 取消后返回 nil 而非 context.Canceled）
- [x] 4.3 实现会话级去重逻辑：`seen` map (ExternalID → UpdatedAt)，只 yield 新任务或 UpdatedAt 有变化的任务；每 N 个周期（默认 100）清空 seen map 防止无限增长
- [x] 4.4 实现 rate limit 退避：当 provider 返回 `RateLimitError` 时，使用 `RetryAfter` 动态调整下次 poll 延迟
- [x] 4.5 实现 auth failure 停止：当 provider 返回 `ErrAuthFailure` 时，停止轮询并返回错误
- [x] 4.6 编写 `poller_test.go`：使用 MemoryProvider 测试正常轮询、去重、UpdatedAt 变更检测、seen map 周期清空、rate limit 退避、auth 停止、context 取消优雅退出（确认回调不在 ctx 取消后调用）

## 5. 集成验证

- [x] 5.1 编写 E2E 测试 `poller_e2e_test.go`：MemoryProvider + Poller + Filter 全链路，验证 "外部 issue → 过滤 → 去重 → 回调" 完整数据流
- [x] 5.2 编写 E2E 测试：验证 Poller 在 provider transient error 后自动恢复继续轮询
- [x] 5.3 编写 E2E 测试：验证新 Poller 实例重新 yield 所有合格 issue（模拟重启后无 seen 状态）
- [x] 5.4 验证 `go build ./...`、`go test -race ./internal/tasksource/...`、`go vet ./...` 全部通过
