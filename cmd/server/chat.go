package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

/* chat service begins */
func generateOfferDescription(websiteName string, banner BannerData) (string, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return "", errors.New("OPENAI_API_KEY env var not set")
	}

	c := openai.NewClient(key)

	model := openai.GPT4o

	content := []openai.ChatMessagePart{
		{
			Type: "text",
			Text: fmt.Sprintf(`You are a joyful and excited social media manager for a health and beauty magazine with the goal of motivating people to take advantage of today's available beauty offers. 
			Tell your audience what the beauty retailer %s is advertising today and highlight any coupons if available. Keep your response short, playful and suitable for a tweet or instagram caption. 
			Do not acknowledge that you are AI.`, websiteName),
		},
	}

	if banner.SupportingText != "" {
		content = append(content, openai.ChatMessagePart{
			Type: "text",
			Text: fmt.Sprintf("For some additional context regarding this promotion please see the quoted text '%s'", banner.SupportingText),
		})
	}

	content = append(content, openai.ChatMessagePart{
		Type:     "image_url",
		ImageURL: &openai.ChatMessageImageURL{URL: banner.Src},
	})

	res, err := c.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: 1000,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:         "user",
				MultiContent: content,
			},
		},
	})

	if err != nil {
		return "", err
	}

	return res.Choices[0].Message.Content, nil

}

/* chat service ends */
