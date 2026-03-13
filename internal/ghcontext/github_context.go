package ghcontext

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	gh "github.com/google/go-github/v68/github"
	"github.com/nickheyer/discordiance/internal/models"
	"gorm.io/gorm"
)

const staleDuration = 24 * time.Hour

// keyFiles are filenames we try to fetch for additional context.
var keyFiles = []string{
	"go.mod", "package.json", "Cargo.toml", "pyproject.toml",
	"requirements.txt", "Gemfile", "pom.xml", "build.gradle",
}

// FetchRepoContext fetches repository context from GitHub and returns it as a string.
// It caches the result in ProductContextCache and refreshes if stale.
func FetchRepoContext(ctx context.Context, db *gorm.DB, product models.Product) (string, error) {
	if product.RepoURL == "" || product.GitHubToken == "" {
		return "", nil
	}

	// Check cache
	var cache models.ProductContextCache
	if err := db.Where("product_id = ?", product.ID).First(&cache).Error; err == nil {
		if time.Since(cache.FetchedAt) < staleDuration {
			slog.Info("ghcontext: using cached context", "product_id", product.ID, "age", time.Since(cache.FetchedAt))
			return cache.Content, nil
		}
		slog.Info("ghcontext: cache stale, refreshing", "product_id", product.ID)
	}

	owner, repo, err := parseRepoURL(product.RepoURL)
	if err != nil {
		return "", err
	}

	content, err := fetchFromGitHub(ctx, product.GitHubToken, owner, repo)
	if err != nil {
		return "", err
	}

	// Upsert cache
	now := time.Now()
	cache = models.ProductContextCache{
		ProductID: product.ID,
		Content:   content,
		FetchedAt: now,
	}
	db.Where("product_id = ?", product.ID).
		Assign(models.ProductContextCache{Content: content, FetchedAt: now}).
		FirstOrCreate(&cache)

	return content, nil
}

// RefreshRepoContext forces a refresh of the cached context.
func RefreshRepoContext(ctx context.Context, db *gorm.DB, product models.Product) (string, error) {
	// Delete existing cache to force refresh
	db.Where("product_id = ?", product.ID).Delete(&models.ProductContextCache{})
	return FetchRepoContext(ctx, db, product)
}

func parseRepoURL(repoURL string) (string, string, error) {
	// Support formats: "owner/repo", "https://github.com/owner/repo", "github.com/owner/repo"
	url := strings.TrimSuffix(repoURL, ".git")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	parts := strings.SplitN(url, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("ghcontext: invalid repo URL format: %s", repoURL)
	}
	return parts[0], parts[1], nil
}

func fetchFromGitHub(ctx context.Context, token, owner, repo string) (string, error) {
	client := gh.NewClient(nil).WithAuthToken(token)

	var sb strings.Builder

	// Repo info
	repoInfo, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("ghcontext: fetching repo: %w", err)
	}

	sb.WriteString(fmt.Sprintf("Repository: %s/%s\n", owner, repo))
	if repoInfo.Description != nil && *repoInfo.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", *repoInfo.Description))
	}
	if repoInfo.Language != nil {
		sb.WriteString(fmt.Sprintf("Primary Language: %s\n", *repoInfo.Language))
	}
	if len(repoInfo.Topics) > 0 {
		sb.WriteString(fmt.Sprintf("Topics: %s\n", strings.Join(repoInfo.Topics, ", ")))
	}

	// README
	readme, _, err := client.Repositories.GetReadme(ctx, owner, repo, nil)
	if err == nil && readme.Content != nil {
		content, err := base64.StdEncoding.DecodeString(*readme.Content)
		if err == nil {
			readmeStr := string(content)
			if len(readmeStr) > 2000 {
				readmeStr = readmeStr[:2000] + "\n... (truncated)"
			}
			sb.WriteString(fmt.Sprintf("\nREADME:\n%s\n", readmeStr))
		}
	}

	// File tree (top-level)
	tree, _, err := client.Git.GetTree(ctx, owner, repo, "HEAD", false)
	if err == nil && tree.Entries != nil {
		sb.WriteString("\nTop-level files:\n")
		for _, entry := range tree.Entries {
			sb.WriteString(fmt.Sprintf("  %s (%s)\n", entry.GetPath(), entry.GetType()))
		}
	}

	// Key dependency files
	for _, filename := range keyFiles {
		fc, _, _, err := client.Repositories.GetContents(ctx, owner, repo, filename, nil)
		if err != nil || fc == nil || fc.Content == nil {
			continue
		}
		content, err := base64.StdEncoding.DecodeString(*fc.Content)
		if err != nil {
			continue
		}
		contentStr := string(content)
		if len(contentStr) > 500 {
			contentStr = contentStr[:500] + "\n... (truncated)"
		}
		sb.WriteString(fmt.Sprintf("\n%s:\n%s\n", filename, contentStr))
	}

	slog.Info("ghcontext: fetched context", "owner", owner, "repo", repo, "length", sb.Len())
	return sb.String(), nil
}
