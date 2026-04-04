package claude

import (
	"strings"
	"testing"
)

func TestRedactToken_MasksTokenInLogString(t *testing.T) {
	token := "sk-ant-api03-abc123def456"
	input := "sending request with token sk-ant-api03-abc123def456 to server"
	got := RedactToken(token, input)

	if got == input {
		t.Fatal("expected token to be redacted, but input was unchanged")
	}
	if strings.Contains(got, token) {
		t.Fatalf("redacted output still contains token: %s", got)
	}
}

func TestRedactToken_EmptyTokenReturnsInputUnchanged(t *testing.T) {
	input := "some log line with no token"
	got := RedactToken("", input)

	if got != input {
		t.Fatalf("expected input unchanged, got %q", got)
	}
}

func TestRedactToken_MultipleOccurrencesAllReplaced(t *testing.T) {
	token := "sk-ant-api03-xyz789"
	input := "token=" + token + " retry token=" + token

	got := RedactToken(token, input)

	if strings.Contains(got, token) {
		t.Fatalf("redacted output still contains token: %s", got)
	}
}

func TestRedactToken_ShowsLastFourChars(t *testing.T) {
	token := "sk-ant-api03-abc123def456"
	input := "token: " + token
	got := RedactToken(token, input)

	last4 := token[len(token)-4:]
	if !strings.Contains(got, last4) {
		t.Fatalf("expected redacted output to contain last 4 chars %q, got %q", last4, got)
	}
}

func TestRedactToken_ShortTokenFullyRedacted(t *testing.T) {
	token := "abc"
	input := "short token abc here"
	got := RedactToken(token, input)

	if strings.Contains(got, token) {
		t.Fatalf("redacted output still contains short token: %s", got)
	}
}
