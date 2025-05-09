package assistant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	ClaudeName AssistName = "claude"
)

// Claude provides AI-assisted code generation capabilities.
// It requires a valid Claude API key for initialization.
type Claude struct {
	key string
}

// NewClaude creates a new Claude instance with the provided API key.
// The key should be a valid Claude API key.
func NewClaude(key string) *Claude {
	return &Claude{
		key: key,
	}
}

// IsValidMode checks if the provided mode string is a valid assistance mode.
// Valid modes are "review" and "readme".
// Returns ErrUnsupportedAssistMode if the mode is invalid.
func (a *Claude) IsValidMode(mode AssistMode) error {
	switch mode {
	case ModeReview, ModeReadme, ModeTests:
		return nil
	}

	return fmt.Errorf("%w: %s", ErrUnsupportedAssistMode, mode)
}

// Generate produces either a code review or README based on the assistPrompt.
// Returns the generated content or an error if the request fails.
func (a *Claude) Generate(am AssistMode, content string) (code string, err error) {
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
func (a *Claude) Review(code string) (string, error) {
	return a.call(Code{SystemPrompt: systemPromptReview, Prompt: code})
}

// Readme generates a README for the provided Go code.
// Returns the README as Markdown text or an error if the request fails.
func (a *Claude) Readme(code string) (string, error) {
	return a.call(Code{SystemPrompt: systemPromptReadme, Prompt: code})
}

func (a *Claude) Tests(code string) (string, error) {
	return a.call(Code{SystemPrompt: systemPromptTests, Prompt: code})
}

func (a *Claude) call(c Code) (string, error) {
	const callTimeout = 300 * time.Second
	client := anthropic.NewClient(option.WithAPIKey(a.key), option.WithRequestTimeout(callTimeout), option.WithEnvironmentProduction())
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
	})

	if err != nil {
		return "", fmt.Errorf("claude message, err=%w", err)
	} else if message == nil {
		return "", errors.New("claude message is nil")
	}

	return message.JSON.Content.Raw(), nil
}
