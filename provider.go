package autocomplete

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type openAIProvider struct {
	client *http.Client
	cfg    Config
}

type ollamaProvider struct {
	client *http.Client
	cfg    Config
}

func NewProvider(cfg Config) (Provider, error) {
	providerName := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerName == "" {
		providerName = "openai"
	}

	client := &http.Client{Timeout: cfg.Timeout}
	switch providerName {
	case "openai":
		return &openAIProvider{client: client, cfg: cfg}, nil
	case "ollama":
		return &ollamaProvider{client: client, cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
}

func (p *openAIProvider) Complete(ctx context.Context, req PromptRequest) (CompletionResult, error) {
	if strings.TrimSpace(p.cfg.OpenAIAPIKey) == "" {
		return CompletionResult{}, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	endpoint := strings.TrimRight(defaultString(p.cfg.OpenAIBaseURL, "https://api.openai.com/v1"), "/") + "/chat/completions"
	payload := map[string]any{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "system", "content": req.SystemPrompt},
			{"role": "user", "content": req.Prompt},
		},
		"temperature": req.Temperature,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return CompletionResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.OpenAIAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return CompletionResult{}, fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return CompletionResult{}, err
	}
	if len(parsed.Choices) == 0 {
		return CompletionResult{}, fmt.Errorf("openai response did not include choices")
	}

	suggestions := parseSuggestions(parsed.Choices[0].Message.Content)
	if len(suggestions) == 0 {
		return CompletionResult{}, fmt.Errorf("openai response did not include suggestions")
	}

	return CompletionResult{
		Suggestions: suggestions,
		Provider:    "openai",
		Model:       defaultString(parsed.Model, req.Model),
		Usage: Usage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
		},
		RawResponse: string(body),
	}, nil
}

func (p *ollamaProvider) Complete(ctx context.Context, req PromptRequest) (CompletionResult, error) {
	endpoint := strings.TrimRight(defaultString(p.cfg.OllamaHost, "http://127.0.0.1:11434"), "/") + "/api/chat"
	payload := map[string]any{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "system", "content": req.SystemPrompt},
			{"role": "user", "content": req.Prompt},
		},
		"stream": false,
		"options": map[string]any{
			"temperature": req.Temperature,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return CompletionResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return CompletionResult{}, fmt.Errorf("ollama request failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Model   string `json:"model"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return CompletionResult{}, err
	}

	suggestions := parseSuggestions(parsed.Message.Content)
	if len(suggestions) == 0 {
		return CompletionResult{}, fmt.Errorf("ollama response did not include suggestions")
	}

	return CompletionResult{
		Suggestions: suggestions,
		Provider:    "ollama",
		Model:       defaultString(parsed.Model, req.Model),
		Usage: Usage{
			PromptTokens:     parsed.PromptEvalCount,
			CompletionTokens: parsed.EvalCount,
		},
		RawResponse: string(body),
	}, nil
}

func parseSuggestionJSON(raw string) []string {
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return normalizeSuggestions(list)
	}

	var object struct {
		Suggestions []string `json:"suggestions"`
		Commands    []string `json:"commands"`
		Completions []string `json:"completions"`
	}
	if err := json.Unmarshal([]byte(raw), &object); err != nil {
		return nil
	}

	switch {
	case len(object.Suggestions) > 0:
		return normalizeSuggestions(object.Suggestions)
	case len(object.Commands) > 0:
		return normalizeSuggestions(object.Commands)
	case len(object.Completions) > 0:
		return normalizeSuggestions(object.Completions)
	default:
		return nil
	}
}

func normalizeSuggestions(input []string) []string {
	seen := make(map[string]struct{})
	suggestions := make([]string, 0, len(input))
	for _, item := range input {
		item = strings.TrimSpace(strings.Trim(item, "`\"'"))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		suggestions = append(suggestions, item)
		if len(suggestions) == 5 {
			break
		}
	}
	return suggestions
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
