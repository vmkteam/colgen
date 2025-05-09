// Package colgen provides AI-assisted code generation and review capabilities.
// It integrates with Deepseek's API to generate code reviews and README content.
//
//	//colgen@ai:<review|readme|tests>
package colgen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AssistMode represents the type of AI assistance to provide.
type AssistMode string
type AssistantName string

const (
	// ModeReview requests a code review of the provided content.
	ModeReview AssistMode = "review"

	// ModeReadme requests a README generation for the provided content.
	ModeReadme AssistMode = "readme"

	ModeTests AssistMode = "tests"

	AssistantDeepSeek AssistantName = "deepseek"
	AssistantClaude   AssistantName = "claude"
)

var ErrUnsupportedAssistMode = errors.New("unsupported assist mode")

// Assistant provides AI-assisted code generation capabilities.
// It requires a valid Deepseek API key for initialization.
type Assistant struct {
	key string
	c   caller
}

// NewAssistant creates a new Assistant instance with the provided API key.
// The key should be a valid Deepseek API key.
func NewAssistant(n AssistantName, key string) *Assistant {
	var c caller
	switch n {
	case AssistantDeepSeek:
		c = DeepSeekCaller{Key: key}
	case AssistantClaude:
		c = ClaudeCaller{Key: key}
	}

	return &Assistant{
		key: key,
		c:   c,
	}
}

// IsValidMode checks if the provided mode string is a valid assistance mode.
// Valid modes are "review" and "readme".
// Returns ErrUnsupportedAssistMode if the mode is invalid.
func (a *Assistant) IsValidMode(mode AssistMode) error {
	switch mode {
	case ModeReview, ModeReadme, ModeTests:
		return nil
	}

	return fmt.Errorf("%w: %s", ErrUnsupportedAssistMode, mode)
}

// Code represents the input for AI generation, containing both
// a system prompt (context/instructions) and user prompt (content to process).
type Code struct {
	SystemPrompt, Prompt string
}

// Generate produces either a code review or README based on the assistPrompt.
// Returns the generated content or an error if the request fails.
func (a *Assistant) Generate(am AssistMode, content string) (code string, err error) {
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
func (a *Assistant) Review(code string) (string, error) {
	return a.c.call(Code{SystemPrompt: systemPromptReview, Prompt: code})
}

// Readme generates a README for the provided Go code.
// Returns the README as Markdown text or an error if the request fails.
func (a *Assistant) Readme(code string) (string, error) {
	return a.c.call(Code{SystemPrompt: systemPromptReadme, Prompt: code})
}

func (a *Assistant) Tests(code string) (string, error) {
	return a.c.call(Code{SystemPrompt: systemPromptTests, Prompt: code})
}

type UserTestPrompt struct {
	TestPrompt   string
	AppendToFile bool
	TestFilename string
}

func UserPromptForTests(code []byte, filename string) (UserTestPrompt, error) {
	var sb strings.Builder
	sb.WriteString("This is code: \n")
	sb.Write(code)

	r := UserTestPrompt{TestFilename: testFilename(filename)}
	if _, err := os.Stat(r.TestFilename); errors.Is(err, os.ErrNotExist) {
		sb.WriteString("\n Return full test file as go code.")
		r.TestPrompt = sb.String()
		return r, nil
	}

	// append
	tc, err := os.ReadFile(r.TestFilename)
	if err != nil {
		return r, err
	}

	r.AppendToFile = true
	sb.WriteString("\nAdd only new test functions for code with all additional cases included. Return only new test functions as go code.")
	sb.WriteString("\nThis is current test file.\n")
	sb.Write(tc)
	r.TestPrompt = sb.String()

	return r, nil
}

// testFilename returns test filename for filename.
func testFilename(filename string) string {
	if filename == "" {
		return ""
	}

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	name := strings.TrimSuffix(base, ".go")

	return filepath.Join(dir, name+"_test.go")
}

const systemPromptReview = `You are a professional Go developer and testing expert.
You write idiomatic go code.
` + basicLinks + `

---
I will give you one file from go project for review. 
Check code for go idiomatic way. Keep the code clean and readable.
Provide small code examples with recommendations.
Provides sample docs for all missing for function signatures if documentation is missing.  
Return review result in Markdown.
`

const systemPromptReadme = `You are a professional Go developer and Technical Writer.
You write idiomatic go code.
` + basicLinks + `
---
I will give you one file from go project for review. 
Generate readme for this file for github repository.  
Return review result in Markdown.
`

const systemPromptTests = `You are a professional Go developer and testing expert.
You write idiomatic go code.
` + basicLinks + `
---
I will provide you with:
- a Go file for tests (code).
- an existing unit tests for this file (tests).

Your job is to:
- analyze the function and the test
- identify missing test cases (e.g., edge cases, error handling, empty results, multiple providers, nil returns, etc.)
- rewrite the test to cover more cases in a clean and idiomatic way
- keep the style of the original test (e.g., if it uses testify, keep using testify, if it's goconvey, stick to that)
- do not use mocks. If you need mocks for http use http test server.
If the test uses a custom test wrappers, constructors or generators (like tests.Dep()), reuse it as appropriate.
Keep the code clean and readable.

Return code results:
 - as go code without additional markdown comments
 - be ready for append/create test file in go.
 - if you want to add comments - adds it at the end of results in code comment format //.
`

const basicLinks = `
Your essential development resources:
* Go
  * https://go.dev/doc/effective_go
  * https://go.dev/doc/faq
  * https://go.dev/talks/2014/names.slide
  * https://go-proverbs.github.io/
  * https://dave.cheney.net/practical-go
  * https://dave.cheney.net/practical-go/presentations/gophercon-singapore-2019.html
  * https://github.com/diptomondal007/GoLangBooks/blob/master/50%20Shades%20of%20Go%20Traps%20GotchasandCommonMistakesforNewGolangDevs.pdf
  * https://google.github.io/styleguide/go/
  * https://google.github.io/styleguide/go/best-practices
  * https://12factor.net/
`
