package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// ClassificationResult is the output of a single insight classification.
type ClassificationResult struct {
	InsightID string
	State     v1.InsightState
	Reason    string
}

// Client talks to any OpenAI-compatible chat completions endpoint.
type Client struct {
	http *http.Client
}

// NewClient creates a new agent client.
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

// Classify sends insight content with product context to the LLM and returns
// the classified state and reasoning for each insight.
func (c *Client) Classify(ctx context.Context, agent *models.Agent, product *models.Product, insights []models.Insight) ([]ClassificationResult, error) {
	systemPrompt := BuildClassificationPrompt(product)

	var userContent strings.Builder
	for i, ins := range insights {
		if i > 0 {
			userContent.WriteString("\n---\n")
		}
		fmt.Fprintf(&userContent, "[Insight ID: %s]\n", ins.ID)
		fmt.Fprintf(&userContent, "Platform: %s | Author: %s | Medium: %s\n",
			ins.PlatformID, ins.Author, v1.InsightMedium_name[ins.Medium])
		fmt.Fprintf(&userContent, "Content:\n%s\n", ins.Content)
	}

	reqBody := chatCompletionRequest{
		Model: agent.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent.String()},
		},
		Temperature: agent.Temperature,
	}
	if agent.MaxTokens > 0 {
		reqBody.MaxTokens = int(agent.MaxTokens)
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(agent.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+agent.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agent returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in agent response")
	}

	return parseClassificationResponse(chatResp.Choices[0].Message.Content, insights)
}

func parseClassificationResponse(content string, insights []models.Insight) ([]ClassificationResult, error) {
	// Try JSON array parse first.
	var classifications []classificationEntry
	if err := json.Unmarshal([]byte(content), &classifications); err == nil && len(classifications) > 0 {
		return mapClassifications(classifications, insights), nil
	}

	// Try extracting JSON from markdown code block.
	if idx := strings.Index(content, "```"); idx >= 0 {
		start := strings.Index(content[idx:], "\n")
		if start >= 0 {
			inner := content[idx+start+1:]
			if end := strings.Index(inner, "```"); end >= 0 {
				inner = inner[:end]
				if err := json.Unmarshal([]byte(strings.TrimSpace(inner)), &classifications); err == nil && len(classifications) > 0 {
					return mapClassifications(classifications, insights), nil
				}
			}
		}
	}

	// Single insight fallback: parse as single object.
	if len(insights) == 1 {
		var single classificationEntry
		if err := json.Unmarshal([]byte(content), &single); err == nil {
			single.ID = insights[0].ID
			return mapClassifications([]classificationEntry{single}, insights), nil
		}

		// Last resort: try to extract state from text.
		state := extractStateFromText(content)
		return []ClassificationResult{{
			InsightID: insights[0].ID,
			State:     state,
			Reason:    content,
		}}, nil
	}

	return nil, fmt.Errorf("could not parse classification response")
}

func mapClassifications(entries []classificationEntry, insights []models.Insight) []ClassificationResult {
	results := make([]ClassificationResult, 0, len(entries))
	insightMap := make(map[string]bool, len(insights))
	for _, ins := range insights {
		insightMap[ins.ID] = true
	}

	for _, e := range entries {
		id := e.ID
		if !insightMap[id] && len(insights) == 1 {
			id = insights[0].ID
		}
		if !insightMap[id] {
			continue
		}
		results = append(results, ClassificationResult{
			InsightID: id,
			State:     parseState(e.State),
			Reason:    e.Reason,
		})
	}
	return results
}

func parseState(s string) v1.InsightState {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "cold":
		return v1.InsightState_INSIGHT_STATE_COLD
	case "hot":
		return v1.InsightState_INSIGHT_STATE_HOT
	case "burn":
		return v1.InsightState_INSIGHT_STATE_BURN
	default:
		return v1.InsightState_INSIGHT_STATE_COLD
	}
}

func extractStateFromText(text string) v1.InsightState {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "burn") {
		return v1.InsightState_INSIGHT_STATE_BURN
	}
	if strings.Contains(lower, "hot") {
		return v1.InsightState_INSIGHT_STATE_HOT
	}
	return v1.InsightState_INSIGHT_STATE_COLD
}

// OpenAI-compatible types.
type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type classificationEntry struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}
