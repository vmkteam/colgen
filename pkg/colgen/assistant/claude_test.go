package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaude_IsValidMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    AssistMode
		wantErr bool
	}{
		{"valid review mode", ModeReview, false},
		{"valid readme mode", ModeReadme, false},
		{"valid tests mode", ModeTests, false},
		{"invalid empty mode", "", true},
		{"invalid random mode", "random", true},
	}

	a := NewDeepSeek("test-key")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := a.IsValidMode(tt.mode)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrUnsupportedAssistMode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaude_Generate(t *testing.T) {
	t.Run("returns error for invalid mode", func(t *testing.T) {
		a := NewDeepSeek("test-key")
		_, err := a.Generate("invalid", "content")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnsupportedAssistMode)
	})

	// Note: Actual API calls would need to be mocked in a real test environment
}
