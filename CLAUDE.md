# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

auto-dev is a spec-driven automated development orchestration system. It uses a **Dispatcher → Supervisor → Worker** pattern where:
- **Dispatcher** polls GitHub Issues, manages concurrency, and controls lifecycle
- **Supervisor** (Claude Code) manages tmux worker sessions, monitors progress, handles intervention
- **Workers** (Claude Code/Codex agents) run in tmux sessions, execute development tasks, push code and create PRs

The project follows a **local-first, spec-driven** philosophy — core development runs locally to minimize token cost, and OpenSpec specifications govern all automation behavior.

## Build & Development Commands

```bash
make build    # go build ./...
make test     # go test ./... -v -race
make lint     # golangci-lint run ./...
make fmt      # gofmt -w . && goimports -w .
make fmt-check # verify formatting without mutating the worktree
make clean    # go clean ./...
```

Single test: `go test ./internal/runlifecycle/... -v -run TestName`

Go version: 1.24.2. Zero external dependencies (stdlib only); testify is used for test assertions.

## Architecture

### Package Structure

- `cmd/autodev/` — CLI entry point (placeholder, will expand to dispatcher/supervisor/worker commands)
- `internal/runlifecycle/` — Run state machine: 11 states across 3 phases, actor-based transition authorization, complete audit history
- `openspec/` — Specification framework: canonical specs (`specs/`) and change tracking (`changes/`)
- `doc/` — Architecture docs: `plan.md` (system design & roadmap), `ref.md` (design decisions & references)

### Run State Machine (`internal/runlifecycle`)

The core module. A Run progresses through phases:
- **Phase A (Spec):** `spec_drafting` → `spec_in_review` → `spec_approved`
- **Phase B (Impl):** `impl_queued` → `implementing` → `code_in_review` → `fixing_review` → `verified`
- **Terminal:** `completed`, `failed`, `aborted`

Three actors (`dispatcher`, `supervisor`, `operator`) are authorized for different transitions. Transition rules are defined in a map-based table in `transition.go`.

Key types: `Run`, `RunState`, `Actor`, `TransitionRecord`. Uses functional options pattern (`RunOption`, `TransitionOption`).

### OpenSpec Workflow

Specs define domain requirements (9 specs covering run lifecycle, dispatch, supervision, worker execution, review feedback, etc.). Changes follow a proposal → design → delta-specs → tasks flow, tracked in `openspec/changes/` and archived when complete.

Use `/opsx:` commands (e.g., `/opsx:new`, `/opsx:continue`, `/opsx:apply`, `/opsx:verify`, `/opsx:archive`) to work with the OpenSpec workflow.

## Code Conventions

- **Errors:** Sentinel errors (`var ErrX = errors.New(...)`) with `fmt.Errorf("... : %w", err)` wrapping. Callers use `errors.Is()`.
- **Configuration:** Functional options pattern for constructors (`NewRun(id, WithReviewFixLimit(3))`).
- **State types:** String-based enums for JSON serialization (`RunState`, `Actor`).
- **Testing:** Table-driven tests, testify/assert, always run with `-race`.
- **Linting:** golangci-lint with govet, errcheck, staticcheck, ineffassign, unused enabled.
- **Formatting:** `make fmt` rewrites files with `gofmt` and `goimports`; `make fmt-check` is the CI-safe verification target.

## Implementation Roadmap

Phase 1 (Infrastructure) → Phase 2 (Single-task closure) → Phase 3 (Automation/concurrency) → Phase 4 (Stability/observability). Currently in Phase 1, implementing the run-lifecycle state machine.
