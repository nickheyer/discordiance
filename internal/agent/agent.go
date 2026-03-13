package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nickheyer/discordiance/internal/models"
	openai "github.com/sashabaranov/go-openai"
)

// DetectedInsight represents a single insight extracted by the agent from a batch of messages.
type DetectedInsight struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
}

// ExistingInsight is a summary of an already-detected insight for dedup context.
type ExistingInsight struct {
	Title    string
	Status   string
	Category string
}

// AnalysisContext provides product context for more accurate analysis.
type AnalysisContext struct {
	ProductDescription string
	FileContents       map[string]string // filename → content
	RepoContext        string
	ExistingInsights   []ExistingInsight
}

// AnalysisResult is the top-level JSON response from the agent.
type AnalysisResult struct {
	Insights []DetectedInsight `json:"insights"`
}

// Agent wraps an OpenAI-compatible chat completions client for insight detection.
type Agent struct {
	client       *openai.Client
	model        string
	systemPrompt string
}

const defaultSystemPrompt = `You are Discordiance, an AI-powered community insight detection agent. You monitor community platform messages in real time to identify actionable product insights — bugs, feature requests, complaints, questions, and sentiment patterns — on behalf of project maintainers.

# Your Role
You receive batches of timestamped messages from a community platform (e.g. Discord, forums). Your job is to sift through the noise — general chatter, greetings, off-topic discussion, jokes, support that has already been resolved — and surface only genuinely actionable items that a product team should know about.

# Input Format
Messages are provided one per line as:
[YYYY-MM-DD HH:MM] AuthorName: Message content

# Analysis Guidelines

## What IS an actionable insight:
- Bug reports: something is broken, crashes, errors, unexpected behavior
- Feature requests: users asking for new capabilities or improvements
- Complaints: recurring frustration, poor UX, missing documentation, performance problems
- Regressions: something that used to work but no longer does
- Security concerns: vulnerabilities, data exposure, auth problems

## What is NOT an insight:
- General conversation, greetings, jokes, off-topic chat
- One-off user errors or confusion that gets resolved in-thread
- Questions that are answered satisfactorily by other community members
- Vague dissatisfaction with no identifiable actionable item
- Messages that are solely praise or positive feedback

## Consolidation
If multiple messages describe the same underlying problem, consolidate them into a single insight. Reference the pattern (e.g. "Multiple users reported...") rather than creating duplicates.

# Output Schema
Respond with a JSON object containing an "insights" array. Each insight must have:

- "title": A concise, descriptive title written as you would write a bug tracker title (imperative or noun-phrase style, under 80 characters)
- "description": A detailed summary including what the problem is, any reproduction context from the messages, how many users mentioned it, and relevant quotes where helpful. Write this as if filing it directly into a bug tracker.
- "severity": One of:
  - "critical" — Data loss, security vulnerability, complete feature breakage affecting many users
  - "high" — Major functionality broken, significant degradation, blocking workflows
  - "medium" — Notable UX issues, non-blocking bugs, common pain points
  - "low" — Minor inconveniences, cosmetic issues, nice-to-have improvements
- "category": One of:
  - "bug" — Broken or incorrect behavior
  - "feature_request" — New capability or enhancement
  - "complaint" — Usability, performance, or experience grievance
  - "question" — Unanswered question that indicates a documentation or discoverability gap
  - "other" — Does not fit the above

If no actionable insights are found, return: {"insights": []}

Be conservative. It is better to miss a marginal insight than to flood the tracker with noise. Only surface items where there is a clear, actionable signal.`

func New(cfg models.Agent) *Agent {
	config := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}
	if cfg.OrgID != "" {
		config.OrgID = cfg.OrgID
	}

	prompt := cfg.SystemPrompt
	if prompt == "" {
		prompt = defaultSystemPrompt
		slog.Info("agent: using default system prompt")
	} else {
		slog.Info("agent: using custom system prompt", "length", len(prompt))
	}

	slog.Info("agent: created", "model", cfg.Model, "base_url", cfg.BaseURL)

	return &Agent{
		client:       openai.NewClientWithConfig(config),
		model:        cfg.Model,
		systemPrompt: prompt,
	}
}

// Analyze sends a batch of messages to the LLM and returns detected insights.
func (a *Agent) Analyze(ctx context.Context, messages []models.Message, analysisCtx *AnalysisContext) (*AnalysisResult, error) {
	slog.Info("agent: starting analysis", "model", a.model, "message_count", len(messages))

	// Build the composite system prompt with context
	systemPrompt := a.buildSystemPrompt(analysisCtx)

	// Build the user message from the batch
	var sb strings.Builder
	for _, msg := range messages {
		fmt.Fprintf(&sb, "[%s] %s: %s\n", msg.Timestamp.Format("2006-01-02 15:04"), msg.AuthorName, msg.Content)
	}
	userContent := sb.String()
	slog.Debug("agent: request payload", "user_content_length", len(userContent), "system_prompt_length", len(systemPrompt))

	start := time.Now()
	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: a.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	elapsed := time.Since(start)

	if err != nil {
		slog.Error("agent: chat completion failed", "model", a.model, "error", err, "duration", elapsed)
		return nil, fmt.Errorf("agent: chat completion: %w", err)
	}

	slog.Info("agent: LLM response received", "model", a.model, "duration", elapsed,
		"choices", len(resp.Choices),
		"prompt_tokens", resp.Usage.PromptTokens,
		"completion_tokens", resp.Usage.CompletionTokens,
		"total_tokens", resp.Usage.TotalTokens)

	if len(resp.Choices) == 0 {
		slog.Error("agent: no choices in response", "model", a.model)
		return nil, fmt.Errorf("agent: no choices returned")
	}

	content := resp.Choices[0].Message.Content
	slog.Debug("agent: response content", "content", content, "length", len(content))

	result, err := parseAnalysisResult(content)
	if err != nil {
		slog.Error("agent: failed to parse response JSON", "error", err, "content_preview", content[:min(200, len(content))])
		return nil, fmt.Errorf("agent: parsing response: %w", err)
	}

	slog.Info("agent: analysis complete", "insights_found", len(result.Insights), "duration", elapsed)
	for i, insight := range result.Insights {
		slog.Info("agent: detected insight", "index", i, "title", insight.Title, "severity", insight.Severity, "category", insight.Category)
	}

	return result, nil
}

// buildSystemPrompt constructs the full system prompt with optional product context.
func (a *Agent) buildSystemPrompt(analysisCtx *AnalysisContext) string {
	if analysisCtx == nil {
		return a.systemPrompt
	}

	var sb strings.Builder
	sb.WriteString(a.systemPrompt)

	if analysisCtx.ProductDescription != "" {
		sb.WriteString("\n\n# Product Context\n")
		sb.WriteString(analysisCtx.ProductDescription)
	}

	if analysisCtx.RepoContext != "" {
		sb.WriteString("\n\n# Repository\n")
		sb.WriteString(analysisCtx.RepoContext)
	}

	if len(analysisCtx.FileContents) > 0 {
		sb.WriteString("\n\n# Documentation\n")
		for filename, content := range analysisCtx.FileContents {
			sb.WriteString(fmt.Sprintf("## %s\n%s\n\n", filename, truncate(content, 2000)))
		}
	}

	if len(analysisCtx.ExistingInsights) > 0 {
		sb.WriteString("\n\n# Previously Detected (avoid duplicates)\n")
		for _, ei := range analysisCtx.ExistingInsights {
			sb.WriteString(fmt.Sprintf("- [%s] [%s] %s\n", ei.Status, ei.Category, ei.Title))
		}
	}

	return sb.String()
}

// parseAnalysisResult parses the LLM response, supporting both "insights" and legacy "issues" keys.
func parseAnalysisResult(content string) (*AnalysisResult, error) {
	var result AnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, err
	}

	// Fallback: try legacy "issues" key if "insights" is empty
	if len(result.Insights) == 0 {
		var legacy struct {
			Issues []DetectedInsight `json:"issues"`
		}
		if err := json.Unmarshal([]byte(content), &legacy); err == nil && len(legacy.Issues) > 0 {
			result.Insights = legacy.Issues
		}
	}

	return &result, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
