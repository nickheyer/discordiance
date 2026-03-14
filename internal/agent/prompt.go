package agent

import (
	"fmt"
	"strings"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// BuildClassificationPrompt constructs the system prompt for insight classification.
func BuildClassificationPrompt(product *models.Product) string {
	var b strings.Builder

	b.WriteString("You are an insight classifier for the product: ")
	b.WriteString(product.Name)
	b.WriteString("\n\n")

	if product.Description != "" {
		b.WriteString("Product description: ")
		b.WriteString(product.Description)
		b.WriteString("\n\n")
	}

	// Append product context.
	for _, ctx := range product.Contexts {
		label := ctx.Label
		if label == "" {
			label = v1.ProductContextType_name[ctx.Type]
		}
		fmt.Fprintf(&b, "Context (%s): %s\n", label, ctx.Value)
	}

	b.WriteString(`
Your task is to classify each insight into exactly one of three states:

- "cold": Uninteresting, irrelevant, unrelated, general conversation, or too subjective to classify. This is the default when unsure.
- "hot": Clearly articulated opinion or sentiment about the product (positive or negative), quantifiable and measurable.
- "burn": Bugs, issues, red alerts, reported incidents, or problems directly related to the product.

For each insight, respond with a JSON array. Each element must have:
- "id": the insight ID provided
- "state": one of "cold", "hot", or "burn"
- "reason": a brief explanation of why you chose this classification

Respond ONLY with the JSON array, no other text.
`)

	return b.String()
}
