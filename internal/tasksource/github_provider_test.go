package tasksource_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v68/github"
	"github.com/lihui/auto-dev/internal/tasksource"
)

// Compile-time check: GitHubProvider must satisfy the Provider interface.
var _ tasksource.Provider = (*tasksource.GitHubProvider)(nil)

// newTestProvider creates a GitHubProvider wired to a test HTTP server.
func newTestProvider(t *testing.T, handler http.Handler) (*tasksource.GitHubProvider, *httptest.Server) {
	return newTestProviderWithConfig(t, handler, tasksource.GitHubConfig{
		Owner: "owner",
		Repo:  "repo",
	})
}

func newTestProviderWithConfig(t *testing.T, handler http.Handler, cfg tasksource.GitHubConfig) (*tasksource.GitHubProvider, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	client := github.NewClient(nil)
	baseURL := ts.URL + "/"
	var err error
	client.BaseURL, err = client.BaseURL.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}

	return tasksource.ExportNewGitHubProviderWithClient(client, cfg), ts
}

// writeJSON is a test helper that encodes v as JSON to w, failing the test on error.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode JSON response: %v", err)
	}
}

func TestGitHubProviderFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		issues := []map[string]any{
			{
				"number":   1,
				"title":    "Fix bug",
				"body":     "There is a bug",
				"state":    "open",
				"html_url": "https://github.com/owner/repo/issues/1",
				"labels": []map[string]any{
					{"name": "bug"},
				},
				"created_at": "2025-01-01T00:00:00Z",
				"updated_at": "2025-01-02T00:00:00Z",
			},
			{
				"number":   2,
				"title":    "Add feature PR",
				"body":     "This is a PR",
				"state":    "open",
				"html_url": "https://github.com/owner/repo/pull/2",
				"labels":   []map[string]any{},
				"pull_request": map[string]any{
					"url": "https://api.github.com/repos/owner/repo/pulls/2",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, issues)
	})

	provider, _ := newTestProvider(t, mux)
	tasks, err := provider.FetchCandidateTasks(context.Background())
	if err != nil {
		t.Fatalf("FetchCandidateTasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task (PR should be skipped), got %d", len(tasks))
	}

	task := tasks[0]
	if task.ExternalID != "1" {
		t.Errorf("ExternalID = %q, want %q", task.ExternalID, "1")
	}
	if want := "owner/repo#1"; task.ExternalKey != want {
		t.Errorf("ExternalKey = %q, want %q", task.ExternalKey, want)
	}
	if task.Title != "Fix bug" {
		t.Errorf("Title = %q, want %q", task.Title, "Fix bug")
	}
	if task.Body != "There is a bug" {
		t.Errorf("Body = %q, want %q", task.Body, "There is a bug")
	}
	if len(task.Labels) != 1 || task.Labels[0] != "bug" {
		t.Errorf("Labels = %v, want [bug]", task.Labels)
	}
	if task.SourceType != "github_issue" {
		t.Errorf("SourceType = %q, want %q", task.SourceType, "github_issue")
	}
	if task.SourceURL != "https://github.com/owner/repo/issues/1" {
		t.Errorf("SourceURL = %q, want %q", task.SourceURL, "https://github.com/owner/repo/issues/1")
	}
	if task.Metadata["state"] != "open" {
		t.Errorf("Metadata[state] = %q, want %q", task.Metadata["state"], "open")
	}
}

func TestGitHubProviderFetchPagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch {
		case page == "" || page == "1":
			// First page: set Link header pointing to page 2
			linkURL := fmt.Sprintf("<%s/repos/owner/repo/issues?page=2>; rel=\"next\"", "http://"+r.Host)
			w.Header().Set("Link", linkURL)
			w.Header().Set("Content-Type", "application/json")
			issues := []map[string]any{
				{
					"number":   1,
					"title":    "Issue 1",
					"body":     "Body 1",
					"state":    "open",
					"html_url": "https://github.com/owner/repo/issues/1",
					"labels":   []map[string]any{},
				},
			}
			writeJSON(t, w, issues)
		case page == "2":
			// Second page: no Link header (last page)
			w.Header().Set("Content-Type", "application/json")
			issues := []map[string]any{
				{
					"number":   2,
					"title":    "Issue 2",
					"body":     "Body 2",
					"state":    "open",
					"html_url": "https://github.com/owner/repo/issues/2",
					"labels":   []map[string]any{},
				},
			}
			writeJSON(t, w, issues)
		}
	})

	provider, _ := newTestProvider(t, mux)
	tasks, err := provider.FetchCandidateTasks(context.Background())
	if err != nil {
		t.Fatalf("FetchCandidateTasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks across pages, got %d", len(tasks))
	}
	if tasks[0].Title != "Issue 1" {
		t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Issue 1")
	}
	if tasks[1].Title != "Issue 2" {
		t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Issue 2")
	}
}

func TestGitHubProviderFetchUsesConfiguredQueryParams(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("state"), "closed"; got != want {
			t.Errorf("state query = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("labels"), "auto-dev,bug"; got != want {
			t.Errorf("labels query = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("per_page"), "50"; got != want {
			t.Errorf("per_page query = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, []map[string]any{})
	})

	provider, _ := newTestProviderWithConfig(t, mux, tasksource.GitHubConfig{
		Owner:   "owner",
		Repo:    "repo",
		State:   "closed",
		Labels:  []string{"auto-dev", "bug"},
		PerPage: 50,
	})

	tasks, err := provider.FetchCandidateTasks(context.Background())
	if err != nil {
		t.Fatalf("FetchCandidateTasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestGitHubProviderPostComment(t *testing.T) {
	var capturedBody string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/42/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		capturedBody = string(bodyBytes)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		writeJSON(t, w, map[string]any{
			"id":   1,
			"body": "working on it",
		})
	})

	provider, _ := newTestProvider(t, mux)
	err := provider.PostComment(context.Background(), "42", "working on it")
	if err != nil {
		t.Fatalf("PostComment: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(capturedBody), &parsed); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if parsed["body"] != "working on it" {
		t.Errorf("comment body = %q, want %q", parsed["body"], "working on it")
	}
}

func TestGitHubProviderAddLabels(t *testing.T) {
	var capturedBody string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/42/labels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		capturedBody = string(bodyBytes)

		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, []map[string]any{
			{"name": "enhancement"},
			{"name": "help wanted"},
		})
	})

	provider, _ := newTestProvider(t, mux)
	err := provider.AddLabels(context.Background(), "42", []string{"enhancement", "help wanted"})
	if err != nil {
		t.Fatalf("AddLabels: %v", err)
	}

	var parsed []string
	if err := json.Unmarshal([]byte(capturedBody), &parsed); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if len(parsed) != 2 || parsed[0] != "enhancement" || parsed[1] != "help wanted" {
		t.Errorf("labels = %v, want [enhancement, help wanted]", parsed)
	}
}

func TestGitHubProviderRemoveLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/42/labels/bug", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	provider, _ := newTestProvider(t, mux)
	err := provider.RemoveLabel(context.Background(), "42", "bug")
	if err != nil {
		t.Fatalf("RemoveLabel: %v", err)
	}
}

func TestGitHubProviderRateLimitError(t *testing.T) {
	mux := http.NewServeMux()
	resetTime := time.Now().Add(1 * time.Hour).Unix()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
		w.WriteHeader(http.StatusForbidden)
		writeJSON(t, w, map[string]any{
			"message":           "API rate limit exceeded",
			"documentation_url": "https://docs.github.com/rest/overview/resources-in-the-rest-api#rate-limiting",
		})
	})

	provider, _ := newTestProvider(t, mux)
	_, err := provider.FetchCandidateTasks(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rateLimitErr *tasksource.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}

	if rateLimitErr.RetryAfter.IsZero() {
		t.Error("RetryAfter should not be zero")
	}
}

func TestGitHubProviderAuthFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(t, w, map[string]any{
			"message":           "Bad credentials",
			"documentation_url": "https://docs.github.com/rest",
		})
	})

	provider, _ := newTestProvider(t, mux)
	_, err := provider.FetchCandidateTasks(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, tasksource.ErrAuthFailure) {
		t.Fatalf("expected ErrAuthFailure, got: %v", err)
	}
}

func TestGitHubProviderForbiddenAuthFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		writeJSON(t, w, map[string]any{
			"message":           "Resource not accessible by integration",
			"documentation_url": "https://docs.github.com/rest",
		})
	})

	provider, _ := newTestProvider(t, mux)
	_, err := provider.FetchCandidateTasks(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, tasksource.ErrAuthFailure) {
		t.Fatalf("expected ErrAuthFailure, got: %v", err)
	}
}

func TestGitHubProviderInvalidExternalID(t *testing.T) {
	mux := http.NewServeMux()
	provider, _ := newTestProvider(t, mux)

	err := provider.PostComment(context.Background(), "abc", "hello")
	if err == nil {
		t.Fatal("expected error for non-numeric external ID, got nil")
	}
	if want := `invalid external ID "abc"`; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestGitHubProviderAbuseRateLimitError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusForbidden)
		writeJSON(t, w, map[string]any{
			"message":           "You have exceeded a secondary rate limit. Please wait a few minutes before you try again.",
			"documentation_url": "https://docs.github.com/rest/overview/resources-in-the-rest-api#secondary-rate-limits",
		})
	})

	provider, _ := newTestProvider(t, mux)
	_, err := provider.FetchCandidateTasks(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rateLimitErr *tasksource.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected *RateLimitError from abuse rate limit, got %T: %v", err, err)
	}

	if rateLimitErr.RetryAfter.IsZero() {
		t.Error("RetryAfter should not be zero")
	}
	// RetryAfter should be ~60s from now.
	if time.Until(rateLimitErr.RetryAfter) < 30*time.Second {
		t.Errorf("RetryAfter too soon: %v", rateLimitErr.RetryAfter)
	}
}
