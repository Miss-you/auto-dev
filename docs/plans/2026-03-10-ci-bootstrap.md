# CI Bootstrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a repository CI baseline that enforces formatting, linting, tests, and compilation on GitHub Actions for pull requests and pushes to `main`.

**Architecture:** Reuse the repository's existing `Makefile` command surface so local development and CI stay aligned. Add one non-mutating formatting check target, one checked-in linter config, and one GitHub Actions workflow that executes the quality gates in order.

**Tech Stack:** Go 1.24.2, GitHub Actions, golangci-lint, gofmt, goimports, GNU Make

---

### Task 1: Add a CI-safe formatting gate

**Files:**
- Modify: `Makefile`

**Step 1: Write the failing test**

Run:

```bash
make fmt-check
```

Expected: FAIL with `No rule to make target 'fmt-check'`.

**Step 2: Run test to verify it fails**

Run:

```bash
make fmt-check
```

Expected: FAIL because the target does not exist yet.

**Step 3: Write minimal implementation**

Add a `fmt-check` target that runs `gofmt -w .`, runs `goimports -w .`, and exits non-zero when the working tree changes.

**Step 4: Run test to verify it passes**

Run:

```bash
make fmt-check
```

Expected: PASS when the tree is already formatted.

**Step 5: Commit**

```bash
git add Makefile
git commit -m "chore: add fmt check target"
```

### Task 2: Check in lint configuration

**Files:**
- Modify: `.golangci.yml`

**Step 1: Write the failing test**

Run:

```bash
golangci-lint run ./...
```

Expected: either FAIL because the tool is missing locally or run without a checked-in baseline config.

**Step 2: Run test to verify it fails**

Run:

```bash
golangci-lint run ./...
```

Expected: current repository state is not yet anchored by project config.

**Step 3: Write minimal implementation**

Create `.golangci.yml` that enables the linters documented in `CLAUDE.md`.

**Step 4: Run test to verify it passes**

Run:

```bash
make lint
```

Expected: PASS with the checked-in config.

**Step 5: Commit**

```bash
git add .golangci.yml
git commit -m "chore: add golangci-lint config"
```

### Task 3: Add GitHub Actions CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

**Step 1: Write the failing test**

Run:

```bash
test -f .github/workflows/ci.yml
```

Expected: FAIL because the workflow does not exist yet.

**Step 2: Run test to verify it fails**

Run:

```bash
test -f .github/workflows/ci.yml
```

Expected: non-zero exit code.

**Step 3: Write minimal implementation**

Create a workflow that triggers on `pull_request` and `push` to `main`, installs Go, `golangci-lint`, and `goimports`, and runs `make fmt-check`, `make lint`, `make test`, and `make build`.

**Step 4: Run test to verify it passes**

Run:

```bash
test -f .github/workflows/ci.yml
```

Expected: PASS.

**Step 5: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add github actions workflow"
```

### Task 4: Verify and review

**Files:**
- Modify as needed: `Makefile`, `.golangci.yml`, `.github/workflows/ci.yml`

**Step 1: Run repository verification**

Run:

```bash
make fmt-check
make lint
make test
make build
```

Expected: all commands pass.

**Step 2: Review the diff**

Check the final diff for:

- CI-only commands accidentally mutating unrelated files
- missing tool installation in workflow
- lint config drift from `CLAUDE.md`

**Step 3: Apply minimal fixes**

Adjust only the files above if verification or review exposes problems.

**Step 4: Re-run verification**

Run:

```bash
make fmt-check
make lint
make test
make build
```

Expected: all commands pass.

**Step 5: Commit**

```bash
git add Makefile .golangci.yml .github/workflows/ci.yml docs/plans/2026-03-10-ci-bootstrap-design.md docs/plans/2026-03-10-ci-bootstrap.md
git commit -m "ci: initialize repository checks"
```
