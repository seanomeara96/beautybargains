package chatsvc

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

type Service struct {
	client *openai.Client
}

func InitChat() *Service {
	return &Service{openai.NewClient(os.Getenv("OPENAI_API_KEY"))}
}

var Bat = &Service{openai.NewClient(os.Getenv("OPENAI_API_KEY"))}

func (c *Service) GetOfferDescription(websiteName, url string) (string, error) {
	command := fmt.Sprintf("You are a joyful and excited social media manager for a health and beauty magazine with the goal of motivating people to take advantage of today's available beauty offers. Tell your audience what the beauty retailer %s is advertising today and highlight any coupons if available. Keep your response short, playful and suitable for a tweet or instagram caption.", websiteName)
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
