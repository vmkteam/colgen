package assistant

import (
	"context"
	"fmt"

	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/config"
	"github.com/go-deepseek/deepseek/request"
)

const (
	DeepseekName AssistName = "deepseek"
)

// DeepSeek provides AI-assisted code generation capabilities.
// It requires a valid Deepseek API key for initialization.
type DeepSeek struct {
	key string
}

// NewDeepSeek creates a new DeepSeek instance with the provided API key.
// The key should be a valid Deepseek API key.
func NewDeepSeek(key string) *DeepSeek {
	return &DeepSeek{
		key: key,
	}
}

// IsValidMode checks if the provided mode string is a valid assistance mode.
// Valid modes are "review" and "readme".
// Returns ErrUnsupportedAssistMode if the mode is invalid.
func (a *DeepSeek) IsValidMode(mode AssistMode) error {
	switch mode {
	case ModeReview, ModeReadme, ModeTests:
		return nil
	}

	return fmt.Errorf("%w: %s", ErrUnsupportedAssistMode, mode)
}

// Generate produces either a code review or README based on the assistPrompt.
// Returns the generated content or an error if the request fails.
func (a *DeepSeek) Generate(am AssistMode, content string) (code string, err error) {
	switch am {
	case ModeReadme:
		code, err = a.Readme(content)
	case ModeReview:
		code, err = a.Review(content)
	case ModeTests:
		code, err = a.Tests(content)
	default:
		err = fmt.Errorf("%w: %s", ErrUnsupportedAssistMode, am)
	}

	return
}

// Review generates a code review for the provided Go code.
// Returns the review as Markdown text or an error if the request fails.
func (a *DeepSeek) Review(code string) (string, error) {
	return a.call(Code{SystemPrompt: systemPromptReview, Prompt: code})
}

// Readme generates a README for the provided Go code.
// Returns the README as Markdown text or an error if the request fails.
func (a *DeepSeek) Readme(code string) (string, error) {
	return a.call(Code{SystemPrompt: systemPromptReadme, Prompt: code})
}

func (a *DeepSeek) Tests(code string) (string, error) {
	return a.call(Code{SystemPrompt: systemPromptTests, Prompt: code})
}

func (a *DeepSeek) call(c Code) (string, error) {
	const callTimeout = 300
	client, err := deepseek.NewClientWithConfig(config.Config{
		ApiKey:         a.key,
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
