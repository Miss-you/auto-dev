## 1. 新增主规格

- [ ] 1.1 创建 `openspec/specs/change-lifecycle/spec.md`，包含 Purpose + 5 个 Requirements（Two-Phase Gate Model、Spec Phase Self-Check Loop、Deferred Spec Finalization、Spec Phase Entry And Exit Criteria、Implementation Phase Entry And Exit Criteria）
- [ ] 1.2 创建 `openspec/specs/run-lifecycle/spec.md`，包含 Purpose + 3 个 Requirements（Canonical Task States、Allowed State Transitions、Terminal State Semantics）

## 2. 退役 workflow-contract

- [ ] 2.1 删除 `openspec/specs/workflow-contract/` 目录

## 3. 更新关联 spec 引用

- [ ] 3.1 更新 `openspec/specs/dispatch-lifecycle/spec.md`：Dispatch Orchestration 和 Persistent Lifecycle State 引用 run-lifecycle 状态名和 change-lifecycle 门禁
- [ ] 3.2 更新 `openspec/specs/supervisor-control-loop/spec.md`：Single-Task Supervision 和 Run Acceptance And Cleanup 引用 run-lifecycle 状态转移规则
- [ ] 3.3 更新 `openspec/specs/worker-execution/spec.md`：Predictable Status Reporting 引用 run-lifecycle 状态上报格式
- [ ] 3.4 更新 `openspec/specs/review-feedback-loop/spec.md`：Review Intake And Routing 引用 change-lifecycle Phase B 和 run-lifecycle fixing_review 状态

## 4. 验证

- [ ] 4.1 运行 `openspec validate --specs --strict` 确认所有主规格格式和语义正确
- [ ] 4.2 确认 workflow-contract 已从主规格中完全移除
