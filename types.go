package autocomplete

import (
	"context"
	"time"
)

type Request struct {
	Shell         string            `json:"shell"`
	InputLine     string            `json:"input_line"`
	CWD           string            `json:"cwd"`
	RecentHistory []string          `json:"recent_history,omitempty"`
	RecentFiles   []string          `json:"recent_files,omitempty"`
	HelpText      string            `json:"help_text,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type Result struct {
	Suggestions []string  `json:"suggestions"`
	CacheHit    bool      `json:"cache_hit"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	Prompt      string    `json:"prompt,omitempty"`
	Usage       Usage     `json:"usage"`
	GeneratedAt time.Time `json:"generated_at"`
}

type PromptRequest struct {
	Model        string
	Temperature  float64
	SystemPrompt string
	Prompt       string
	Request      Request
}

type CompletionResult struct {
	Suggestions []string
	Provider    string
	Model       string
	Usage       Usage
	RawResponse string
}

type Provider interface {
	Complete(ctx context.Context, req PromptRequest) (CompletionResult, error)
}

type Engine interface {
	Suggest(ctx context.Context, req Request) (Result, error)
}

type Config struct {
	Provider           string
	Model              string
	Temperature        float64
	Timeout            time.Duration
	OpenAIAPIKey       string
	OpenAIBaseURL      string
	OllamaHost         string
	CacheDir           string
	CacheTTL           time.Duration
	CacheSize          int
	MaxHistoryCommands int
	MaxRecentFiles     int
}
