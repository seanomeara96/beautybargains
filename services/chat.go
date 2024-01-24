package services

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

type Chat struct {
	client *openai.Client
}

func InitChat() *Chat {
	return &Chat{openai.NewClient(os.Getenv("OPENAI_API_KEY"))}
}

var Bat = &Chat{openai.NewClient(os.Getenv("OPENAI_API_KEY"))}

func (c *Chat) GetOfferDescription(websiteName, url string) (string, error) {
	command := fmt.Sprintf("You are a joyful and excited writer for a health and beauty magazine with the goal of motivating people to take advantage of today's available beauty offers. Tell your audience what the beauty retailer %s is advertising today and highlight any coupons if available. Keep your response short and playful.", websiteName)
	model := openai.GPT4VisionPreview

	res, err := c.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: 1000,
		Messages: []openai.ChatCompletionMessage{
			openai.ChatCompletionMessage{
				Role: "user",
				MultiContent: []openai.ChatMessagePart{
					openai.ChatMessagePart{
						Type: "text",
						Text: command,
					},
					openai.ChatMessagePart{
						Type:     "image_url",
						ImageURL: &openai.ChatMessageImageURL{URL: url},
					},
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	return res.Choices[0].Message.Content, nil

}
