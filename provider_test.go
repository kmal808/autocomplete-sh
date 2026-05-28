package autocomplete

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestOpenAIProviderParsesResponse(t *testing.T) {
	provider := &openAIProvider{
		client: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"model":"gpt-4o-mini",
						"choices":[{"message":{"content":"{\"suggestions\":[\"git status\",\"git add .\"]}"}}],
						"usage":{"prompt_tokens":11,"completion_tokens":7}
					}`)),
				}, nil
			}),
		},
		cfg: Config{
			Provider:      "openai",
			OpenAIAPIKey:  "test-key",
			OpenAIBaseURL: "https://example.invalid/v1",
		},
	}

	result, err := provider.Complete(context.Background(), PromptRequest{
		Model:        "gpt-4o-mini",
		Temperature:  0,
		SystemPrompt: systemPrompt,
		Prompt:       "User input: git sta",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Suggestions) != 2 {
		t.Fatalf("unexpected suggestions: %#v", result.Suggestions)
	}
}

func TestOllamaProviderParsesResponse(t *testing.T) {
	provider := &ollamaProvider{
		client: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"model":"llama3.1",
						"message":{"content":"git status\ngit add ."},
						"prompt_eval_count":12,
						"eval_count":6
					}`)),
				}, nil
			}),
		},
		cfg: Config{
			Provider:   "ollama",
			OllamaHost: "http://127.0.0.1:11434",
		},
	}

	result, err := provider.Complete(context.Background(), PromptRequest{
		Model:        "llama3.1",
		Temperature:  0,
		SystemPrompt: systemPrompt,
		Prompt:       "User input: git sta",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Suggestions) != 2 {
		t.Fatalf("unexpected suggestions: %#v", result.Suggestions)
	}
}
