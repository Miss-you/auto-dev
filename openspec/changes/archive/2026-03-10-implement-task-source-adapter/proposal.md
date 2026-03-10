## Why

auto-dev 需要从外部任务源（首先是 GitHub Issues）获取待处理任务，归一化后交给 dispatcher 编排执行。当前系统缺少统一的任务源适配层，dispatcher 无法获取外部任务。这是整条 "Issue → Run → PR" 数据通路的第一环。

## What Changes

- 新增 `internal/tasksource/` 包，定义归一化任务模型 `NormalizedTask` 和 provider 接口 `Provider`
- 实现 `Provider` 接口的 GitHub Issues 适配器 `github_provider`，基于 `google/go-github` 库
- 实现轮询引擎 `Poller`，按可配置间隔（默认 30s）调用 provider 拉取候选任务
- 实现 source-side 过滤（label 白名单/黑名单、issue state 白名单）和会话级去重
- 实现状态回写接口（PostComment、UpdateLabels），供 dispatcher/supervisor 通过 adapter 回写外部源
- 实现错误弹性：transient 重试、rate limit 退避、auth 失败停止轮询
- 提供 `memory_provider` 测试替身，用于单测和集成测试

## Capabilities

### New Capabilities

_无新增 spec capability。本次变更是对已有 `task-source-adapter` spec 的实现。_

### Modified Capabilities

_无 spec 级别变更。spec 已在实现前更新完毕。_

## Impact

- 新增 Go 依赖：`github.com/google/go-github/v68`、`golang.org/x/oauth2`
- 新增包：`internal/tasksource/`（model、interface、poller、github provider、memory provider）
- dispatcher（未来实现）将依赖此包的 `Provider` 接口和 `NormalizedTask` 模型
- 需要 GitHub Personal Access Token 或 GitHub App 认证配置
