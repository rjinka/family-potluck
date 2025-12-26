package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type SuggestedDish struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DietaryTags []string `json:"dietary_tags"`
}

func SuggestDishes(ctx context.Context, eventName, description, eventType string) ([]SuggestedDish, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")

	prompt := fmt.Sprintf(`Suggest 5-7 potluck dishes for an event named "%s" (Type: %s). 
Description: %s.
Return the suggestions as a JSON array of objects with "name", "description", and "dietary_tags" (array of strings like "Vegan", "Gluten-Free", "Vegetarian", "Dairy-Free", "Nut-Free").
Only return the JSON array, no other text.`, eventName, eventType, description)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	var suggestions []SuggestedDish
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			cleanText := string(text)
			cleanText = strings.TrimSpace(cleanText)
			if strings.HasPrefix(cleanText, "```json") {
				cleanText = strings.TrimPrefix(cleanText, "```json")
				cleanText = strings.TrimSuffix(cleanText, "```")
			} else if strings.HasPrefix(cleanText, "```") {
				cleanText = strings.TrimPrefix(cleanText, "```")
				cleanText = strings.TrimSuffix(cleanText, "```")
			}
			cleanText = strings.TrimSpace(cleanText)

			err := json.Unmarshal([]byte(cleanText), &suggestions)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal suggestions: %v, text: %s", err, cleanText)
			}
			return suggestions, nil
		}
	}

	return nil, fmt.Errorf("no text parts in response")
}
