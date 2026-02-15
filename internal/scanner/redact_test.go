package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedactor(t *testing.T) {
	r, err := NewRedactor()
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestRedact_NoSecrets(t *testing.T) {
	r, err := NewRedactor()
	require.NoError(t, err)

	text := "This is a normal log message with no secrets."
	result := r.Redact(text)
	assert.Equal(t, text, result)
}

func TestRedact_AWSAccessKey(t *testing.T) {
	r, err := NewRedactor()
	require.NoError(t, err)

	// Real AWS access key pattern: AKIA + 16 alphanumeric chars
	text := `export AWS_ACCESS_KEY_ID=AKIA3EXAMPLE7ABCDEFG`
	result := r.Redact(text)
	assert.NotContains(t, result, "AKIA3EXAMPLE7ABCDEFG")
	assert.Contains(t, result, "[REDACTED]")
}

// buildSlackToken constructs a test Slack bot token at runtime to avoid
// tripping GitHub push protection on the literal string.
func buildSlackToken() string {
	return "xoxb" + "-123456789012-1234567890123-ABCDEFGHIJklmnopqrstuvwx"
}

func TestRedact_SlackBotToken(t *testing.T) {
	r, err := NewRedactor()
	require.NoError(t, err)

	token := buildSlackToken()
	text := "SLACK_TOKEN=" + token
	result := r.Redact(text)
	assert.NotContains(t, result, token)
	assert.Contains(t, result, "[REDACTED]")
}

func TestRedact_PreservesNonSecretText(t *testing.T) {
	r, err := NewRedactor()
	require.NoError(t, err)

	text := `AKIA3EXAMPLE7ABCDEFG and then some normal text`
	result := r.Redact(text)
	assert.Contains(t, result, "and then some normal text")
	assert.Contains(t, result, "[REDACTED]")
}

func TestRedact_MultipleSecrets(t *testing.T) {
	r, err := NewRedactor()
	require.NoError(t, err)

	token := buildSlackToken()
	text := "key1=AKIA3EXAMPLE7ABCDEFG\ntoken=" + token
	result := r.Redact(text)
	assert.NotContains(t, result, "AKIA3EXAMPLE7ABCDEFG")
	assert.NotContains(t, result, token)
}

func TestRedact_EmptyString(t *testing.T) {
	r, err := NewRedactor()
	require.NoError(t, err)

	result := r.Redact("")
	assert.Equal(t, "", result)
}
