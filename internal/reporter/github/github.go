package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	gh "github.com/google/go-github/v68/github"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/reporter"
)

func init() {
	reporter.Register("github", func() reporter.Reporter {
		return &GitHub{}
	})
}

type GitHub struct {
	client  *gh.Client
	owner   string
	repo    string
	labels  []string
	mu      sync.RWMutex
	healthy bool
}

func (g *GitHub) Type() string { return "github" }

func (g *GitHub) Init(ctx context.Context, settings models.JSONMap) error {
	slog.Info("github: initializing reporter")

	token := settings["token"]
	if token == "" {
		return fmt.Errorf("github: token is required")
	}
	slog.Info("github: token present", "length", len(token))

	repoFull := settings["repo"]
	if repoFull == "" {
		return fmt.Errorf("github: repo is required (format: owner/repo)")
	}

	parts := strings.SplitN(repoFull, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("github: repo must be in format owner/repo")
	}
	g.owner = parts[0]
	g.repo = parts[1]

	if labels, ok := settings["labels"]; ok && labels != "" {
		g.labels = strings.Split(labels, ",")
		for i := range g.labels {
			g.labels[i] = strings.TrimSpace(g.labels[i])
		}
		slog.Info("github: labels configured", "labels", g.labels)
	}

	g.client = gh.NewClient(nil).WithAuthToken(token)

	g.mu.Lock()
	g.healthy = true
	g.mu.Unlock()

	slog.Info("github: reporter initialized", "repo", repoFull, "owner", g.owner, "repo_name", g.repo)
	return nil
}

func (g *GitHub) FileReport(ctx context.Context, issue models.Issue) (*models.Report, error) {
	slog.Info("github: filing issue", "title", issue.Title, "severity", issue.Severity, "category", issue.Category)

	body := issue.Description
	if issue.Severity != "" {
		body = fmt.Sprintf("**Severity:** %s\n**Category:** %s\n\n%s", issue.Severity, issue.Category, body)
	}

	req := &gh.IssueRequest{
		Title:  gh.Ptr(issue.Title),
		Body:   gh.Ptr(body),
		Labels: &g.labels,
	}

	slog.Info("github: creating issue via API", "owner", g.owner, "repo", g.repo)
	ghIssue, _, err := g.client.Issues.Create(ctx, g.owner, g.repo, req)
	if err != nil {
		slog.Error("github: API create issue failed", "owner", g.owner, "repo", g.repo, "error", err)
		return nil, fmt.Errorf("github: creating issue: %w", err)
	}

	report := &models.Report{
		IssueID:      issue.ID,
		ReporterType: "github",
		ExternalID:   fmt.Sprintf("%d", ghIssue.GetNumber()),
		ExternalURL:  ghIssue.GetHTMLURL(),
		Status:       "filed",
	}

	slog.Info("github: issue created", "number", ghIssue.GetNumber(), "url", ghIssue.GetHTMLURL())
	return report, nil
}

func (g *GitHub) UpdateReport(ctx context.Context, externalID string, issue models.Issue) error {
	slog.Debug("github: UpdateReport not implemented", "external_id", externalID)
	return nil
}

func (g *GitHub) FindDuplicate(ctx context.Context, issue models.Issue) (string, error) {
	slog.Info("github: searching for duplicate", "title", issue.Title, "owner", g.owner, "repo", g.repo)

	// Basic title-match dedup: search for open issues with similar titles
	query := fmt.Sprintf("repo:%s/%s is:open in:title %s", g.owner, g.repo, issue.Title)
	slog.Debug("github: search query", "query", query)

	result, _, err := g.client.Search.Issues(ctx, query, &gh.SearchOptions{
		ListOptions: gh.ListOptions{PerPage: 5},
	})
	if err != nil {
		slog.Error("github: search API failed", "error", err)
		return "", fmt.Errorf("github: searching issues: %w", err)
	}

	slog.Info("github: search results", "total_count", result.GetTotal(), "returned", len(result.Issues))

	for _, existing := range result.Issues {
		if strings.EqualFold(existing.GetTitle(), issue.Title) {
			slog.Info("github: exact duplicate found", "number", existing.GetNumber(), "title", existing.GetTitle())
			return fmt.Sprintf("%d", existing.GetNumber()), nil
		}
	}

	slog.Info("github: no duplicate found", "title", issue.Title)
	return "", nil
}

func (g *GitHub) Close(ctx context.Context) error {
	slog.Info("github: closing reporter", "owner", g.owner, "repo", g.repo)
	g.mu.Lock()
	g.healthy = false
	g.mu.Unlock()
	slog.Info("github: reporter closed")
	return nil
}

func (g *GitHub) Healthy(ctx context.Context) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.healthy
}
