package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vmkteam/colgen/pkg/colgen"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAIPrompts(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMode    colgen.AssistMode
		wantName    colgen.AssistantName
		wantErr     bool
		expectedErr string
	}{
		{
			name:     "simple mode without assistant",
			input:    "readme",
			wantMode: "readme",
			wantName: colgen.AssistantDeepSeek,
			wantErr:  false,
		},
		{
			name:     "mode with deepseek assistant",
			input:    "readme(deepseek)",
			wantMode: "readme",
			wantName: colgen.AssistantDeepSeek,
			wantErr:  false,
		},
		{
			name:     "mode with claude assistant",
			input:    "review(claude)",
			wantMode: "review",
			wantName: colgen.AssistantClaude,
			wantErr:  false,
		},
		{
			name:     "tests mode with empty parentheses",
			input:    "tests()",
			wantMode: "tests",
			wantName: colgen.AssistantDeepSeek,
			wantErr:  false,
		},
		{
			name:        "missing closing parenthesis",
			input:       "readme(invalid",
			wantMode:    "",
			wantName:    "",
			wantErr:     true,
			expectedErr: "invalid AI prompt, \")\" is not found or has invalid position",
		},
		{
			name:        "invalid parentheses order",
			input:       "readme)(invalid",
			wantMode:    "",
			wantName:    "",
			wantErr:     true,
			expectedErr: "invalid AI prompt, \")\" is not found or has invalid position",
		},
		{
			name:     "empty input",
			input:    "",
			wantMode: "",
			wantName: colgen.AssistantDeepSeek,
			wantErr:  false,
		},
		{
			name:     "whitespace in parentheses",
			input:    "readme( )",
			wantMode: "readme",
			wantName: colgen.AssistantDeepSeek,
			wantErr:  false,
		},
		{
			name:     "whitespace before parentheses",
			input:    "readme (claude)",
			wantMode: "readme",
			wantName: colgen.AssistantClaude,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotName, err := extractAIPrompts(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Equal(t, tt.expectedErr, err.Error())
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, gotMode)
			assert.Equal(t, tt.wantName, gotName)
		})
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple file",
			path:     "file.go",
			expected: "file",
		},
		{
			name:     "file with path",
			path:     "/path/to/file.go",
			expected: "file",
		},
		{
			name:     "file without extension",
			path:     "file",
			expected: "file",
		},
		{
			name:     "file with multiple dots",
			path:     "file.test.go",
			expected: "file.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := baseName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAppVersion(t *testing.T) {
	version := appVersion()
	assert.NotEmpty(t, version)
}

func TestConfigFillByAssistName(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		assistName colgen.AssistantName
		key        string
		wantErr    bool
	}{
		{
			name:       "nil config",
			config:     nil,
			assistName: colgen.AssistantDeepSeek,
			key:        "test-key",
			wantErr:    true,
		},
		{
			name:       "deepseek assistant",
			config:     &Config{},
			assistName: colgen.AssistantDeepSeek,
			key:        "deepseek-key",
			wantErr:    false,
		},
		{
			name:       "claude assistant",
			config:     &Config{},
			assistName: colgen.AssistantClaude,
			key:        "claude-key",
			wantErr:    false,
		},
		{
			name:       "unknown assistant",
			config:     &Config{},
			assistName: "unknown",
			key:        "unknown-key",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.fillByName(tt.assistName, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.key, tt.config.keyByName(tt.assistName))
		})
	}
}

func TestConfigKeyByName(t *testing.T) {
	cfg := &Config{
		DeepSeekKey: "deepseek-key",
		ClaudeKey:   "claude-key",
	}

	tests := []struct {
		name       string
		assistName colgen.AssistantName
		expected   string
	}{
		{
			name:       "deepseek assistant",
			assistName: colgen.AssistantDeepSeek,
			expected:   "deepseek-key",
		},
		{
			name:       "claude assistant",
			assistName: colgen.AssistantClaude,
			expected:   "claude-key",
		},
		{
			name:       "unknown assistant",
			assistName: "unknown",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.keyByName(tt.assistName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigPath(t *testing.T) {
	path, err := configPath()
	require.NoError(t, err)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	expected := filepath.Join(homeDir, configFile)
	assert.Equal(t, expected, path)
}

func TestReadConfig(t *testing.T) {
	// This is a basic test that just ensures the function doesn't crash
	// A more comprehensive test would require mocking the filesystem
	_, err := readConfig()
	require.NoError(t, err)
}

func TestColgenLines(t *testing.T) {
	// Create a temporary file for testing
	content := `package testpkg

//colgen:User,Category
//colgen:User:IDs,UniqueIDs
//colgen@ai:tests(claude)
//colgen@replace:something

func main() {
	// code
}
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "colgen_test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Test readFile function
	cl, err := readFile(tmpFile.Name())
	require.NoError(t, err)

	// Verify the parsed content
	assert.Equal(t, "testpkg", cl.pkgName)
	assert.Equal(t, []string{"User,Category", "User:IDs,UniqueIDs"}, cl.lines)
	assert.Equal(t, []string{"tests(claude)"}, cl.assistant)
	assert.Equal(t, []string{"//colgen@replace:something"}, cl.injection)
}
