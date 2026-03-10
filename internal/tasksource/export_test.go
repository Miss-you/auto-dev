package tasksource

import "github.com/google/go-github/v68/github"

// ExportNewGitHubProviderWithClient exposes newGitHubProviderWithClient for external tests.
var ExportNewGitHubProviderWithClient = newGitHubProviderWithClient

// Ensure the export has the correct type at compile time.
var _ func(*github.Client, GitHubConfig) *GitHubProvider = ExportNewGitHubProviderWithClient
