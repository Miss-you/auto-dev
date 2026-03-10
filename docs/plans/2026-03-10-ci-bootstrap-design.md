# CI Bootstrap Design

**Date:** 2026-03-10

## Goal

Initialize a repository-level CI baseline for this Go project that validates formatting, linting, tests, and compilation on GitHub Actions for new pull requests, pull request merges, and pushes to `main`.

## Context

The repository already exposes the core developer commands through `Makefile` targets:

- `make build`
- `make test`
- `make lint`
- `make fmt`

The missing pieces are:

- a non-mutating formatting check suitable for CI
- a checked-in linter configuration
- a GitHub Actions workflow wired to the project defaults

## Options Considered

### Option 1: Single GitHub Actions workflow

Use one `ci.yml` workflow with sequential steps for setup, format check, lint, test, and build.

Pros:
- Matches current repository size
- Keeps operational overhead low
- Makes failures easy to inspect in one place

Cons:
- Less granular than separate workflows

### Option 2: Split quality and verification workflows

Use one workflow for format/lint and one for test/build.

Pros:
- More targeted workflow names
- Potentially clearer status signals

Cons:
- More files and maintenance for little current benefit

### Option 3: Introduce extra tooling

Adopt additional orchestration tools for CI abstraction.

Pros:
- More future flexibility

Cons:
- Unnecessary complexity for the current repository

## Decision

Use Option 1.

## Design

### Workflow triggers

- `pull_request`
- `push` to `main`

This covers:

- new pull requests
- merge commits landing on `main`
- direct pushes to `main`

### Verification stages

1. `fmt-check`
2. `lint`
3. `test`
4. `build`

### Formatting strategy

Keep `make fmt` as the mutating local formatter, but add `make fmt-check` for CI. The new target will:

- copy the current worktree to a temporary directory
- run `gofmt -w .`
- run `goimports -w .`
- fail when the formatted temporary copy differs from the current worktree

This reuses the exact same formatting tools as local development while keeping the original worktree untouched during verification.

### Lint strategy

Check in `.golangci.yml` with a small, explicit baseline aligned with the repository guidance in `CLAUDE.md`.

### CI strategy

GitHub Actions will:

- set up the pinned Go version from the repo
- install `golangci-lint`
- install `goimports`
- run the four verification commands

## Non-Goals

- release automation
- multi-version Go test matrix
- caching optimizations beyond standard setup actions
- branch protection configuration via API
