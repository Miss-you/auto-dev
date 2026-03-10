# task-source-adapter Specification

## Purpose

该规格定义外部任务源进入 auto-dev 编排系统时的统一边界，确保不同 provider 的输入先被过滤、去重并归一化，再进入后续派发和执行流程。adapter 同时负责向外部源回写状态（评论、标签），使 provider-specific API 细节完全隔离在此边界内。

## Requirements

### Requirement: Normalized Task Model
The system SHALL define a canonical normalized task model with the following fields: ExternalID (provider-internal stable ID), ExternalKey (human-readable key like "owner/repo#123"), Title, Body, Labels (list of strings for routing/filtering), Priority (0=unset, 1=highest), SourceType (e.g. "github_issue"), SourceURL, Metadata (extensible key-value map), CreatedAt, UpdatedAt. All downstream components (dispatcher, supervisor) SHALL depend only on this model, never on provider-specific API types.

#### Scenario: External issue becomes normalized task
- **WHEN** the system reads a task from an enabled source provider
- **THEN** it produces a normalized task record with all canonical fields populated, mapping provider-specific fields to the shared model

#### Scenario: Unknown or missing fields get safe defaults
- **WHEN** a provider does not supply a field (e.g. priority is absent)
- **THEN** the normalized task uses the defined zero value (0 for priority, empty string for optional fields) without error

### Requirement: Polling-Based Task Ingestion
The task source adapter SHALL poll the external source on a configurable fixed cadence (default 30 seconds) to discover new or updated tasks. Webhook-based ingestion is out of scope for the initial implementation. The adapter MUST be safe to restart at any time; after restart it resumes polling from the current state without requiring persistent storage beyond what the provider API offers (e.g. issue list endpoint is idempotent).

#### Scenario: Adapter polls on cadence
- **WHEN** the configured poll interval elapses
- **THEN** the adapter fetches the current candidate task list from the provider and produces normalized tasks for any new or updated items

#### Scenario: Adapter restarts mid-cycle
- **WHEN** the adapter process restarts
- **THEN** it resumes polling from scratch on the next tick; no persistent local state is required because the provider API is the source of truth

### Requirement: Source-Side Filtering And Deduplication
The task source adapter SHALL support provider-specific filtering rules (e.g. label inclusion/exclusion, state whitelist) and perform initial deduplication before handing tasks to the dispatcher. Filtering rules are configured per-provider. Deduplication at the adapter level is issue-level: the adapter tracks which external IDs were already yielded in the current session and suppresses duplicates within a poll cycle. The dispatcher performs the authoritative run-level deduplication (whether an issue already has an active run); the adapter's dedup is a performance optimization, not a correctness guarantee.

#### Scenario: Unqualified issue is excluded by filter
- **WHEN** an incoming task does not match the configured source filters (e.g. missing required label, wrong issue state)
- **THEN** the adapter excludes it and records the exclusion reason in structured logs

#### Scenario: Duplicate issue suppressed within session
- **WHEN** the adapter polls and finds an issue whose ExternalID was already yielded to the dispatcher in a previous poll cycle of the current session
- **THEN** it suppresses the duplicate unless the issue's UpdatedAt has changed (indicating the issue was modified)

### Requirement: Provider Abstraction Boundary
The system SHALL isolate all provider-specific API details behind a Go interface so downstream lifecycle logic depends only on the normalized task model and a small set of adapter operations. The interface SHALL support at minimum: FetchCandidateTasks (read), PostComment (write), AddLabels (write), and RemoveLabel (write). State writeback (comments, labels) lives in the adapter because the adapter is the only component that knows how to talk to the provider API. Label management MUST be non-destructive: the adapter adds or removes only the labels it owns rather than replacing the issue's full label set.

#### Scenario: Dispatcher consumes normalized tasks only
- **WHEN** the dispatcher requests pending work
- **THEN** it receives normalized task data without needing provider-specific API fields

#### Scenario: Dispatcher requests status writeback
- **WHEN** the dispatcher or supervisor needs to post a comment or manage labels on the external issue
- **THEN** it calls the adapter's writeback methods using only ExternalID and payload; the adapter translates to provider-specific API calls

### Requirement: Error Resilience
The task source adapter SHALL handle transient provider errors (network failures, API rate limits, authentication expiry) gracefully. On transient failure, the adapter logs the error and retries on the next poll cycle without crashing. On rate limit (HTTP 429 or equivalent), the adapter respects the provider's retry-after header and backs off accordingly. On persistent authentication failure, the adapter logs an error and stops polling until the issue is resolved, rather than hammering the API.

#### Scenario: Transient network error during poll
- **WHEN** a poll attempt fails due to a transient network error
- **THEN** the adapter logs the error at warn level and retries on the next scheduled poll tick

#### Scenario: GitHub API rate limit hit
- **WHEN** the GitHub API returns HTTP 429 or rate limit headers indicate exhaustion
- **THEN** the adapter respects the retry-after / rate-limit-reset header and delays the next poll accordingly

#### Scenario: Authentication token expired
- **WHEN** the provider returns HTTP 401 consistently
- **THEN** the adapter logs an error indicating token renewal is needed and stops polling
