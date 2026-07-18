// Package ai optionally drafts a save message from a diff, using the user's
// own free OpenRouter API key. It is entirely opt-in: nothing in gitle
// depends on this package being usable, and every failure mode here should
// be handled by the caller falling back to asking the person directly.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// defaultModel is a free-tier OpenRouter model, picked for being good at
// reading code/diffs. OpenRouter's free lineup rotates — models get
// retired with little notice — so this can be overridden without a code
// change via OPENROUTER_MODEL, and is read fresh on every call rather than
// cached, in case that env var changes between saves.
const defaultModel = "qwen/qwen3-coder:free"

const (
	endpoint     = "https://openrouter.ai/api/v1/chat/completions"
	maxDiffChars = 6000
	maxTokens    = 40
	timeout      = 10 * time.Second
)

// model returns the configured OpenRouter model, honoring OPENROUTER_MODEL
// as an override for when defaultModel gets retired from the free tier.
func model() string {
	if m := os.Getenv("OPENROUTER_MODEL"); m != "" {
		return m
	}
	return defaultModel
}

const systemPrompt = "You write a single short git commit message summarizing a diff. " +
	"Reply with only the message: imperative mood, under 72 characters, no quotes, no trailing period, nothing else."

// Available reports whether an OpenRouter API key is configured.
func Available() bool {
	return os.Getenv("OPENROUTER_API_KEY") != ""
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// SuggestMessage asks the model for a one-line commit message summarizing
// diff. It always sends at most maxDiffChars of the diff and asks for at
// most maxTokens back, so cost and latency are bounded regardless of how
// large the caller's diff is.
func SuggestMessage(diff string) (string, error) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return "", errors.New("OPENROUTER_API_KEY is not set")
	}
	diff = truncate(diff, maxDiffChars)

	reqBody, err := json.Marshal(chatRequest{
		Model: model(),
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: diff},
		},
		MaxTokens: maxTokens,
	})
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", "gitle")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("unreadable response (status %d)", resp.StatusCode)
	}
	if parsed.Error != nil {
		return "", errors.New(parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no suggestion (status %d)", resp.StatusCode)
	}

	msg := cleanSuggestion(parsed.Choices[0].Message.Content)
	if msg == "" {
		return "", errors.New("model returned an empty suggestion")
	}
	return msg, nil
}

// truncate cuts s to at most n bytes, preferring not to split mid-line.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if i := strings.LastIndexByte(cut, '\n'); i > 0 {
		cut = cut[:i]
	}
	return cut + "\n... (diff truncated)"
}

// cleanSuggestion trims a model reply down to a single plain line, in case
// the model wraps its answer in quotes, extra commentary, or a code fence.
func cleanSuggestion(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.Trim(s, " \t`\"'")
	const hardCap = 120
	if len(s) > hardCap {
		s = strings.TrimSpace(s[:hardCap])
	}
	return s
}
