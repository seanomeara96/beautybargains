package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

type OfferDescriptionResponse struct {
	// text description of the offer
	Description string `json:"description"`
	// any coupon codes found in the resource
	CouponCodes []CouponCode `json:"coupon_codes"`
	// typical health and beauty categories
	Categories []string `json:"categories"`
	// any brands mentioned
	Brands []string `json:"brands"`
}

const responseSchema = `{
  "type": "object",
  "properties": {
    "description": {
      "type": "string",
      "description": "Text description of the offer"
    },
    "coupon_codes": {
      "type": "array",
      "description": "Any coupon codes found in the resource",
      "items": {
        "type": "object",
        "properties": {
          "code": {
            "type": "string",
            "description": "The coupon code"
          },
          "description": {
            "type": "string",
            "description": "Description of the coupon code"
          },
          "valid_until": {
            "type": ["string", "null"],
            "format": "date-time",
            "description": "Expiration date of the coupon, if any"
          }
        },
        "required": ["code", "description"]
      }
    },
    "categories": {
      "type": "array",
      "description": "Typical health and beauty categories",
      "items": {
        "type": "string"
      }
    },
    "brands": {
      "type": "array",
      "description": "Any brands mentioned",
      "items": {
        "type": "string"
      }
    }
  },
  "required": ["description", "coupon_codes", "categories", "brands"]
}`

type MySchema struct {
	Raw string
}

func (s *MySchema) MarshalJSON() ([]byte, error) {
	return []byte(s.Raw), nil
}

var mySchema = MySchema{responseSchema}

/* chat service begins */
func analyzeOffer(websiteName string, banner BannerData) (*OfferDescriptionResponse, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY env var not set")
	}

	c := openai.NewClient(key)

	model := openai.GPT4o20240806

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
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:        "OfferDescriptionResponse",
				Description: "Schema for Offer Description response including coupon codes and related information.",
				Schema:      &mySchema,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var data = new(OfferDescriptionResponse)
	if err := json.Unmarshal([]byte(res.Choices[0].Message.Content), data); err != nil {
		return nil, err
	}

	return data, nil
}

/* chat service ends */
