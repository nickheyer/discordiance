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

func (g *GitHub) Init(ctx context.Context, cfg models.Reporter) error {
	slog.Info("github: initializing reporter")

	if cfg.Token == "" {
		return fmt.Errorf("github: token is required")
	}
	slog.Info("github: token present", "length", len(cfg.Token))

	if cfg.Repo == "" {
		return fmt.Errorf("github: repo is required (format: owner/repo)")
	}

	parts := strings.SplitN(cfg.Repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("github: repo must be in format owner/repo")
	}
	g.owner = parts[0]
	g.repo = parts[1]

	if cfg.Labels != "" {
		g.labels = strings.Split(cfg.Labels, ",")
		for i := range g.labels {
			g.labels[i] = strings.TrimSpace(g.labels[i])
		}
		slog.Info("github: labels configured", "labels", g.labels)
	}

	g.client = gh.NewClient(nil).WithAuthToken(cfg.Token)

	g.mu.Lock()
	g.healthy = true
	g.mu.Unlock()

	slog.Info("github: reporter initialized", "repo", cfg.Repo, "owner", g.owner, "repo_name", g.repo)
	return nil
}

func (g *GitHub) FileReport(ctx context.Context, insight models.Insight) (*models.Report, error) {
	slog.Info("github: filing insight", "title", insight.Title, "severity", insight.Severity, "category", insight.Category)

	body := insight.Description
	if insight.Severity != "" {
		body = fmt.Sprintf("**Severity:** %s\n**Category:** %s\n\n%s", insight.Severity, insight.Category, body)
	}

	req := &gh.IssueRequest{
		Title:  gh.Ptr(insight.Title),
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
		InsightID:    insight.ID,
		ReporterType: "github",
		ExternalID:   fmt.Sprintf("%d", ghIssue.GetNumber()),
		ExternalURL:  ghIssue.GetHTMLURL(),
		Status:       "filed",
	}

	slog.Info("github: issue created", "number", ghIssue.GetNumber(), "url", ghIssue.GetHTMLURL())
	return report, nil
}

func (g *GitHub) UpdateReport(ctx context.Context, externalID string, insight models.Insight) error {
	slog.Debug("github: UpdateReport not implemented", "external_id", externalID)
	return nil
}

func (g *GitHub) FindDuplicate(ctx context.Context, insight models.Insight) (string, error) {
	slog.Info("github: searching for duplicate", "title", insight.Title, "owner", g.owner, "repo", g.repo)

	query := fmt.Sprintf("repo:%s/%s is:open in:title %s", g.owner, g.repo, insight.Title)
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
		if strings.EqualFold(existing.GetTitle(), insight.Title) {
			slog.Info("github: exact duplicate found", "number", existing.GetNumber(), "title", existing.GetTitle())
			return fmt.Sprintf("%d", existing.GetNumber()), nil
		}
	}

	slog.Info("github: no duplicate found", "title", insight.Title)
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
