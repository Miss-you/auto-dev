package tasksource

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// GitHubConfig configures the GitHub Issues provider.
type GitHubConfig struct {
	Token   string   // PAT or GitHub App token
	Owner   string   // repo owner
	Repo    string   // repo name
	Labels  []string // include labels (GitHub API AND filter)
	State   string   // "open", "closed", "all" (default: "open")
	PerPage int      // items per page (default 100, max 100)
}

// GitHubProvider implements Provider for GitHub Issues.
type GitHubProvider struct {
	client  *github.Client
	owner   string
	repo    string
	labels  []string
	state   string
	perPage int
}

// NewGitHubProvider creates a new GitHub Issues provider.
func NewGitHubProvider(cfg GitHubConfig) (*GitHubProvider, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("github token is required: %w", ErrAuthFailure)
	}
	if cfg.Owner == "" || cfg.Repo == "" {
		return nil, fmt.Errorf("github owner and repo are required")
	}

	state := cfg.State
	if state == "" {
		state = "open"
	}
	perPage := cfg.PerPage
	if perPage <= 0 || perPage > 100 {
		perPage = 100
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &GitHubProvider{
		client:  client,
		owner:   cfg.Owner,
		repo:    cfg.Repo,
		labels:  cfg.Labels,
		state:   state,
		perPage: perPage,
	}, nil
}

// newGitHubProviderWithClient creates a provider with a custom github.Client (for testing).
func newGitHubProviderWithClient(client *github.Client, cfg GitHubConfig) *GitHubProvider {
	state := cfg.State
	if state == "" {
		state = "open"
	}
	perPage := cfg.PerPage
	if perPage <= 0 || perPage > 100 {
		perPage = 100
	}
	return &GitHubProvider{
		client:  client,
		owner:   cfg.Owner,
		repo:    cfg.Repo,
		labels:  cfg.Labels,
		state:   state,
		perPage: perPage,
	}
}

func (p *GitHubProvider) FetchCandidateTasks(ctx context.Context) ([]NormalizedTask, error) {
	opts := &github.IssueListByRepoOptions{
		State:  p.state,
		Labels: p.labels,
		ListOptions: github.ListOptions{
			PerPage: p.perPage,
		},
	}

	var allTasks []NormalizedTask
	for {
		issues, resp, err := p.client.Issues.ListByRepo(ctx, p.owner, p.repo, opts)
		if err != nil {
			return nil, p.wrapError(err)
		}

		for _, issue := range issues {
			// Skip pull requests (GitHub API returns PRs as issues)
			if issue.IsPullRequest() {
				continue
			}
			allTasks = append(allTasks, p.normalize(issue))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allTasks, nil
}

func (p *GitHubProvider) PostComment(ctx context.Context, externalID string, body string) error {
	num, err := strconv.Atoi(externalID)
	if err != nil {
		return fmt.Errorf("invalid external ID %q: %w", externalID, err)
	}
	comment := &github.IssueComment{Body: github.Ptr(body)}
	_, _, err = p.client.Issues.CreateComment(ctx, p.owner, p.repo, num, comment)
	if err != nil {
		return p.wrapError(err)
	}
	return nil
}

func (p *GitHubProvider) AddLabels(ctx context.Context, externalID string, labels []string) error {
	num, err := strconv.Atoi(externalID)
	if err != nil {
		return fmt.Errorf("invalid external ID %q: %w", externalID, err)
	}
	_, _, err = p.client.Issues.AddLabelsToIssue(ctx, p.owner, p.repo, num, labels)
	if err != nil {
		return p.wrapError(err)
	}
	return nil
}

func (p *GitHubProvider) RemoveLabel(ctx context.Context, externalID string, label string) error {
	num, err := strconv.Atoi(externalID)
	if err != nil {
		return fmt.Errorf("invalid external ID %q: %w", externalID, err)
	}
	_, err = p.client.Issues.RemoveLabelForIssue(ctx, p.owner, p.repo, num, label)
	if err != nil {
		return p.wrapError(err)
	}
	return nil
}

func (p *GitHubProvider) normalize(issue *github.Issue) NormalizedTask {
	labels := make([]string, 0, len(issue.Labels))
	for _, l := range issue.Labels {
		labels = append(labels, l.GetName())
	}

	metadata := map[string]string{
		"state": issue.GetState(),
	}

	task := NormalizedTask{
		ExternalID:  strconv.Itoa(issue.GetNumber()),
		ExternalKey: fmt.Sprintf("%s/%s#%d", p.owner, p.repo, issue.GetNumber()),
		Title:       issue.GetTitle(),
		Body:        issue.GetBody(),
		Labels:      labels,
		SourceType:  "github_issue",
		SourceURL:   issue.GetHTMLURL(),
		Metadata:    metadata,
	}

	if issue.CreatedAt != nil {
		task.CreatedAt = issue.CreatedAt.Time
	}
	if issue.UpdatedAt != nil {
		task.UpdatedAt = issue.UpdatedAt.Time
	}

	return task
}

// wrapError converts go-github errors into tasksource error types.
// Check order: AbuseRateLimitError -> RateLimitError -> HTTP status.
func (p *GitHubProvider) wrapError(err error) error {
	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		retryAfter := time.Now().Add(abuseErr.GetRetryAfter())
		slog.Warn("github secondary rate limit hit", "retry_after", retryAfter)
		return &RateLimitError{RetryAfter: retryAfter}
	}

	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		retryAfter := rateLimitErr.Rate.Reset.Time
		slog.Warn("github primary rate limit hit", "retry_after", retryAfter)
		return &RateLimitError{RetryAfter: retryAfter}
	}

	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) {
		switch errResp.Response.StatusCode {
		case 401, 403:
			slog.Error("github authentication failure", "status", errResp.Response.StatusCode)
			return fmt.Errorf("%w: %v", ErrAuthFailure, err)
		}
	}

	return err
}
