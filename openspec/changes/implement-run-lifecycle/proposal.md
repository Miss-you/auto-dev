## Why

run-lifecycle spec 已定义完整的状态机规范（规范状态名、允许转移、触发者、终态语义），但当前项目无任何实现代码。dispatcher、supervisor、worker 和 review-feedback-loop 都需要一个共享的状态机模块来记录和校验任务状态。这是整个 auto-dev 系统的基础骨架，必须先于其他组件实现。

## What Changes

- 初始化工程化的 Go 项目结构（go mod、目录布局、Makefile、CI lint/test 基础配置）
- 新增 Go 状态机包 `internal/runlifecycle`，包含：
  - 规范状态常量（Phase A / Phase B / Terminal）
  - 允许转移表及触发者校验
  - 非法转移拒绝逻辑
  - 终态锁定（到达终态后禁止自动化转移）
  - failure reason 标注（区分 post_verification_cleanup_failed 与一般失败）
  - review 修复轮次计数与上限校验
- 新增 Run 数据模型，记录单个任务运行实例的状态、历史和元数据
- 新增单元测试，覆盖合法转移、非法转移拒绝、终态锁定、failure reason 标注等场景

## Capabilities

### New Capabilities

（无。run-lifecycle spec 已存在，本 change 是对其的代码实现。）

### Modified Capabilities

（无。spec 层面的需求未变更。）

## Impact

- 新增 `internal/runlifecycle/` 包（Go），成为 dispatcher、supervisor 等组件的共享依赖
- 新增项目工程基础设施（go.mod、Makefile、golangci-lint 配置）
- 无外部 API 或系统影响，纯内部模块
- 后续组件（dispatch-lifecycle、supervisor-control-loop 等）将导入此包进行状态转移操作
