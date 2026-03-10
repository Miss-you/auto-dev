## Context

auto-dev 项目当前只有 spec 文档，无任何实现代码。run-lifecycle spec 定义了 11 个规范状态、允许的转移规则、触发者授权和终态语义。本设计将其落地为 Go 包，作为后续 dispatcher、supervisor 等组件的共享基础设施。

参考文档：
- `openspec/specs/run-lifecycle/spec.md`（需求规范）
- `doc/plan.md`（整体方案）

## Goals / Non-Goals

**Goals:**
- 按 Go 社区最佳实践初始化项目结构（cmd/ + internal/ 布局、go mod、Makefile、lint 配置）
- 提供 `internal/runlifecycle` 包，包含状态常量、转移校验、Run 数据模型
- 100% 覆盖 spec 中定义的场景（合法转移、非法拒绝、终态锁定、failure reason、轮次上限、aborted 清理）
- 保持零外部依赖（仅用标准库 + testify 用于测试断言）
- 提供清晰的 API 供上游组件调用

**Non-Goals:**
- 不实现状态持久化（属于 dispatch-lifecycle 职责，后续实现）
- 不实现 worker session 管理（属于 session-runtime 职责）
- 不实现 review 反馈采集（属于 review-feedback-loop 职责）
- 不实现并发控制（属于 dispatch-lifecycle 职责）
- 不提供 CLI 或 API 端点，纯内部包

## Decisions

### Decision: Go 项目结构遵循 standard layout

```
auto-dev/
├── cmd/
│   └── autodev/              # 主程序入口（当前仅占位）
│       └── main.go
├── internal/
│   └── runlifecycle/         # run-lifecycle 状态机包
│       ├── state.go          # RunState / Actor 类型、Phase 分组
│       ├── state_test.go
│       ├── transition.go     # 转移表、校验函数
│       ├── transition_test.go
│       ├── run.go            # Run 模型
│       ├── run_test.go
│       └── errors.go         # 错误类型
├── go.mod
├── go.sum
├── Makefile                  # build / test / lint 快捷命令
└── .golangci.yml             # linter 配置
```

- Why: `cmd/` + `internal/` 是 Go 社区广泛采用的项目布局。`internal/` 确保包不会被外部项目意外引用，保持 API 边界清晰。
- Alternative considered: 平铺在根目录（`pkg/runlifecycle/`）。
- Why not: `pkg/` 暗示可供外部导入，但 run-lifecycle 是 auto-dev 的内部模块，不应对外暴露。

### Decision: 状态用 string 类型 + iota 常量

```go
type RunState string

const (
    StateSpecDrafting  RunState = "spec_drafting"
    StateSpecInReview  RunState = "spec_in_review"
    StateSpecApproved  RunState = "spec_approved"
    StateImplQueued    RunState = "impl_queued"
    StateImplementing  RunState = "implementing"
    StateCodeInReview  RunState = "code_in_review"
    StateFixingReview  RunState = "fixing_review"
    StateVerified      RunState = "verified"
    StateCompleted     RunState = "completed"
    StateFailed        RunState = "failed"
    StateAborted       RunState = "aborted"
)
```

- Why: `string` 底层类型可直接 JSON 序列化/反序列化，日志友好，与 spec 中的规范状态名完全对应。
- Alternative considered: `int` + iota。
- Why not: 序列化后不可读，调试时需要反查映射表。

### Decision: Actor 同样用 string 类型

```go
type Actor string

const (
    ActorDispatcher Actor = "dispatcher"
    ActorSupervisor Actor = "supervisor"
    ActorOperator   Actor = "operator"
)
```

- Why: 与 RunState 保持一致的风格，JSON 友好。

### Decision: 转移表用 map 声明

```go
type transitionKey struct {
    From RunState
    To   RunState
}

var allowedTransitions = map[transitionKey]Actor{
    {StateSpecDrafting, StateSpecInReview}: ActorSupervisor,
    // ...
}
```

- Why: O(1) 查找，声明式定义与 spec 转移列表一一对应。struct key 比 string 拼接更类型安全。
- Alternative considered: `map[string]Actor`，key 为 `"from->to"` 字符串。
- Why not: 字符串拼接容易拼错，编译器无法检查。

### Decision: 转移校验作为 Run 方法

```go
func (r *Run) Transition(to RunState, actor Actor, opts ...TransitionOption) error {
    // 校验 → 特殊规则 → 记录历史 → 更新状态
}
```

- Why: 校验与状态变更原子化，避免调用方忘记校验。用 functional options 模式传递可选参数（如 reason）。
- Alternative considered: 独立 `Validate()` + `Apply()` 两步。
- Why not: 增加误用风险。

### Decision: Review 修复轮次由 Run 模型计数

Run 内维护 `ReviewFixCount int`，每次 `code_in_review → fixing_review` 时递增。上限通过 `ReviewFixLimit int` 字段设定（构造时传入，默认 3）。达到上限时 `Transition()` 自动转 failed。

- Why: 计数是 Run 实例级状态，放在 Run 中最自然。上限作为字段而非全局变量，允许不同 run 有不同策略。
- Alternative considered: 由 supervisor 在外部计数。
- Why not: 散落在调用方的计数逻辑容易遗漏。

### Decision: 转移历史用结构体切片

```go
type TransitionRecord struct {
    From      RunState  `json:"from"`
    To        RunState  `json:"to"`
    Actor     Actor     `json:"actor"`
    Reason    string    `json:"reason,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}
```

每次 transition 成功后 append 到 `Run.History []TransitionRecord`。

- Why: 提供审计追踪，终态证据校验时可回溯。
- Alternative considered: 不记录历史。
- Why not: 无法回答"任务经过了哪些状态"。

### Decision: 错误类型用 sentinel + 结构体

```go
var (
    ErrIllegalTransition = errors.New("illegal state transition")
    ErrTerminalState     = errors.New("run is in terminal state")
)
```

具体错误通过 `fmt.Errorf("... : %w", ErrIllegalTransition)` 包装，调用方用 `errors.Is()` 判断。

- Why: Go 惯用的错误处理模式，支持 `errors.Is()` 和 `errors.As()` 判断。
- Alternative considered: 自定义 error struct。
- Why not: 当前错误类型简单，sentinel 就够了。后续需要更多上下文时再升级为 struct。

### Decision: 序列化用 JSON struct tag

Run 和 TransitionRecord 都带 `json:"..."` tag，提供 `MarshalJSON` / `UnmarshalJSON` 或直接依赖标准库 `encoding/json`。

- Why: 后续持久化（JSON 文件或 SQLite JSON 列）可直接使用，零额外代码。

## Risks / Trade-offs

- [无持久化] → 当前 Run 对象仅存在于内存中。dispatcher 实现时须将 Run 序列化到 JSON/SQLite。JSON tag 已预留。
- [上限值外部传入] → review 修复轮次上限依赖构造时传入，默认值 3 兜底。
- [并发安全] → 当前不加锁。后续如果多 goroutine 访问同一 Run，可加 `sync.Mutex`。当前单 goroutine-per-run 的设计不需要。
- [testify 依赖] → 测试引入 `github.com/stretchr/testify`。这是 Go 生态最广泛使用的测试辅助库，风险极低。如果追求零依赖测试，可回退到标准库 `testing` 包，但断言代码会更冗长。
