## 1. Go 项目初始化

- [x] 1.1 运行 `go mod init` 创建 `go.mod`（模块路径 `github.com/lihui/auto-dev`）
- [x] 1.2 创建目录结构：`cmd/autodev/`、`internal/runlifecycle/`
- [x] 1.3 创建 `cmd/autodev/main.go` 占位入口（仅 `package main` + 空 `main()`）
- [x] 1.4 创建 `Makefile`，包含 `build`、`test`、`lint`、`fmt` 目标
- [x] 1.5 创建 `.golangci.yml`，配置基础 linter 规则（govet、errcheck、staticcheck、gofmt）
- [x] 1.6 运行 `go mod tidy` 并验证 `go build ./...` 和 `go test ./...` 通过

## 2. 状态与 Actor 类型

- [x] 2.1 在 `internal/runlifecycle/state.go` 中定义 `RunState` 类型和 11 个状态常量
- [x] 2.2 定义 `Actor` 类型和 3 个 Actor 常量（dispatcher、supervisor、operator）
- [x] 2.3 定义 Phase 分组变量（`PhaseAStates`、`PhaseBStates`、`TerminalStates`、`AllStates`）
- [x] 2.4 提供辅助函数 `IsTerminal(RunState) bool` 和 `PhaseOf(RunState) string`
- [x] 2.5 编写 `state_test.go`：验证状态常量完整性（11 个）、Phase 分组互斥且并集覆盖全部状态、IsTerminal 正确性

## 3. 转移规则与校验

- [x] 3.1 在 `internal/runlifecycle/transition.go` 中定义 `allowedTransitions` map
- [x] 3.2 实现 `ValidateTransition(from, to RunState, actor Actor) error`，合法返回 nil，非法返回 wrapped ErrIllegalTransition
- [x] 3.3 转移校验包含终态锁定检查：终态出发的任何转移直接返回 ErrTerminalState
- [x] 3.4 在 `internal/runlifecycle/errors.go` 中定义 `ErrIllegalTransition` 和 `ErrTerminalState` sentinel error
- [x] 3.5 编写 `transition_test.go`：表驱动测试覆盖所有合法转移（逐条验证）、非法转移拒绝（随机非法对）、终态锁定（3 个终态各自尝试出发）、actor 不匹配拒绝

## 4. Run 模型

- [x] 4.1 在 `internal/runlifecycle/run.go` 中定义 `TransitionRecord` 结构体（From, To, Actor, Reason, Timestamp + json tag）
- [x] 4.2 定义 `Run` 结构体：ID, State, History, ReviewFixCount, ReviewFixLimit, FailureReason + json tag
- [x] 4.3 实现 `NewRun(id string, opts ...RunOption) *Run` 构造函数（functional options 设置 ReviewFixLimit 等）
- [x] 4.4 实现 `(*Run).Transition(to RunState, actor Actor, opts ...TransitionOption) error`：校验 → 特殊规则 → 记录历史 → 更新状态
- [x] 4.5 实现 failure reason 自动标注：从 verified 转移到 failed 时设置 FailureReason 为 `post_verification_cleanup_failed`
- [x] 4.6 实现 review 修复轮次计数：`code_in_review → fixing_review` 时递增 ReviewFixCount，达到 ReviewFixLimit 时自动转 failed
- [x] 4.7 验证 `encoding/json` 对 Run 的 Marshal/Unmarshal 往返正确性
- [x] 4.8 编写 `run_test.go`：覆盖完整 happy path（spec_drafting → completed 全路径）、failure reason 自动标注、轮次超限自动 failed、终态后拒绝转移、JSON 序列化往返、aborted 终止场景
