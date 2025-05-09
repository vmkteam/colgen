package colgen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/config"
	"github.com/go-deepseek/deepseek/request"
)

type caller interface {
	call(c Code) (string, error)
}

type DeepSeekCaller struct {
	Key string
}

func (d DeepSeekCaller) call(c Code) (string, error) {
	const callTimeout = 300
	client, err := deepseek.NewClientWithConfig(config.Config{
		ApiKey:         d.Key,
		TimeoutSeconds: callTimeout,
	})
	if err != nil {
		return "", err
	}

	temperature := float32(0)
	chatReq := &request.ChatCompletionsRequest{
		Messages: []*request.Message{
			{
				Role:    "system",
				Content: c.SystemPrompt,
			},
			{
				Role:    "user",
				Content: c.Prompt,
			},
		},
		Model:       deepseek.DEEPSEEK_CHAT_MODEL,
		Temperature: &temperature,
	}

	chatResp, err := client.CallChatCompletionsChat(context.Background(), chatReq)
	if err != nil {
		return "", err
	}
	return chatResp.Choices[0].Message.Content, nil
}

type ClaudeCaller struct {
	Key string
}

func (d ClaudeCaller) call(c Code) (string, error) {
	const callTimeout = 300 * time.Second
	client := anthropic.NewClient(option.WithAPIKey(d.Key), option.WithRequestTimeout(callTimeout), option.WithEnvironmentProduction())
	message, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		System: []anthropic.TextBlockParam{
			{Text: c.SystemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				[]anthropic.ContentBlockParamUnion{
					{OfRequestTextBlock: &anthropic.TextBlockParam{Text: c.Prompt}},
				}...,
			),
		},
		Model:       anthropic.ModelClaude3_7SonnetLatest,
		Temperature: anthropic.Float(0),
		MaxTokens:   10000,
	})

	if err != nil {
		return "", fmt.Errorf("claude message, err=%w", err)
	} else if message == nil {
		return "", errors.New("claude message is nil")
	}

	return message.Content[0].Text, nil
}
