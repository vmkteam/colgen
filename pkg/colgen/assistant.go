// Package colgen
// AI Assistant
//
//	//go:generate go run ../cmd/colgen/colgen.go -ai=<deepseek-key>
//	//colgen@ai:<review|readme>
package colgen

import (
	"context"
	"errors"

	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
)

const (
	AssistantModeReview = "review"
	AssistantModeReadme = "readme"
)

var ErrInvalidAssistMode = errors.New("invalid assist mode")

type Assistant struct {
	key string
}

func NewAssistant(key string) *Assistant {
	return &Assistant{
		key: key,
	}
}

func (aa Assistant) IsValidMode(mode string) error {
	if mode == AssistantModeReview || mode == AssistantModeReadme {
		return nil
	}

	return ErrInvalidAssistMode
}

type Code struct {
	SystemPrompt, Prompt string
}

func (aa Assistant) Generate(assistPrompt, content string) (code string, err error) {
	switch assistPrompt {
	case AssistantModeReadme:
		code, err = aa.Readme(content)
	case AssistantModeReview:
		code, err = aa.Review(content)
	}

	return
}

func (aa Assistant) Review(code string) (string, error) {
	return aa.call(Code{SystemPrompt: systemPromptReview, Prompt: code})
}

func (aa Assistant) Readme(code string) (string, error) {
	return aa.call(Code{SystemPrompt: systemPromptReadme, Prompt: code})
}

func (aa Assistant) call(c Code) (string, error) {
	client, err := deepseek.NewClient(aa.key)
	if err != nil {
		return "", err
	}

	var t float32 = 0
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
		Temperature: &t,
	}

	chatResp, err := client.CallChatCompletionsChat(context.Background(), chatReq)
	if err != nil {
		return "", err
	}
	return chatResp.Choices[0].Message.Content, nil
}

const systemPromptReview = `You are a professional Go developer and testing expert.
You write idiomatic go code.
Your essential development resources:
* Go
  * https://go.dev/doc/effective_go
  * https://go.dev/doc/faq
  * https://go.dev/talks/2014/names.slide
  * https://go-proverbs.github.io/
  * https://dave.cheney.net/practical-go/presentations/gophercon-singapore-2019.html
  * https://github.com/diptomondal007/GoLangBooks/blob/master/50%20Shades%20of%20Go%20Traps%20GotchasandCommonMistakesforNewGolangDevs.pdf
  * https://google.github.io/styleguide/go/
  * https://google.github.io/styleguide/go/best-practices
  * https://12factor.net/

---
I will give you one file from go project for review. 
Check code for go idiomatic way. Keep the code clean and readable.
Provide small code examples with recommendations.
Provides sample docs for all missing for function signatures if documentation is missing.  
Return review result in Markdown.
`

const systemPromptReadme = `You are a professional Go developer and Technical Writer.
You write idiomatic go code.
Your essential development resources:
* Go
  * https://go.dev/doc/effective_go
  * https://go.dev/doc/faq
  * https://go.dev/talks/2014/names.slide
  * https://go-proverbs.github.io/
  * https://dave.cheney.net/practical-go/presentations/gophercon-singapore-2019.html
  * https://github.com/diptomondal007/GoLangBooks/blob/master/50%20Shades%20of%20Go%20Traps%20GotchasandCommonMistakesforNewGolangDevs.pdf
  * https://google.github.io/styleguide/go/
  * https://google.github.io/styleguide/go/best-practices
  * https://12factor.net/

---
I will give you one file from go project for review. 
Generate readme for this file for github repository.  
Return review result in Markdown.
`
