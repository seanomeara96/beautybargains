package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

type MySchema struct {
	Raw string
}

func (s *MySchema) MarshalJSON() ([]byte, error) {
	return []byte(s.Raw), nil
}

func CreateChatCompletion(params openai.ChatCompletionRequest) (string, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return "", errors.New("OPENAI_API_KEY env var not set")
	}

	params.Model = openai.GPT4o20240806
	params.MaxTokens = 1000

	res, err := openai.NewClient(key).CreateChatCompletion(context.Background(), params)
	if err != nil {
		return "", err
	}

	return res.Choices[0].Message.Content, nil
}

type Score struct {
	Rating int `json:"rating"`
}

func ChatRateBrand(brandName string) (int, error) {
	params := openai.ChatCompletionRequest{
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: fmt.Sprintf(`Give %s a rating out of 100 based on your knowledge of the sentiment of the brand`, brandName)},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:        "BrandRatingResponse",
				Description: "Schema for Brand Rating",
				Schema: &MySchema{`{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
					  "rating": {
						"type": "integer",
						"minimum": 0,
						"maximum": 100,
						"description": "A rating between 0 and 100"
					  }
					},
					"required": ["rating"],
					"additionalProperties": false
				  }`},
			},
		},
	}

	ans, err := CreateChatCompletion(params)
	if err != nil {
		return 0, err
	}
	var score Score
	if err := json.Unmarshal([]byte(ans), &score); err != nil {
		return 0, err
	}
	return score.Rating, nil
}

/* chat service ends */
