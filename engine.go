package autocomplete

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type runtimeEngine struct {
	cfg      Config
	provider Provider
	cache    *FileCache
}

func NewEngine(cfg Config, provider Provider) (Engine, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is required")
	}

	cache, err := NewFileCache(cfg.CacheDir, cfg.CacheTTL, cfg.CacheSize)
	if err != nil {
		return nil, err
	}

	return &runtimeEngine{
		cfg:      cfg,
		provider: provider,
		cache:    cache,
	}, nil
}

func (e *runtimeEngine) Suggest(ctx context.Context, req Request) (Result, error) {
	prompt := BuildPrompt(req)
	key := cacheKey(strings.ToLower(defaultString(e.cfg.Provider, "openai")), e.cfg.Model, prompt)

	if cached, ok, err := e.cache.Get(key); err != nil {
		return Result{}, err
	} else if ok {
		cached.Prompt = prompt
		cached.CacheHit = true
		return cached, nil
	}

	completion, err := e.provider.Complete(ctx, PromptRequest{
		Model:        e.cfg.Model,
		Temperature:  e.cfg.Temperature,
		SystemPrompt: systemPrompt,
		Prompt:       prompt,
		Request:      req,
	})
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Suggestions: completion.Suggestions,
		CacheHit:    false,
		Provider:    defaultString(completion.Provider, strings.ToLower(defaultString(e.cfg.Provider, "openai"))),
		Model:       defaultString(completion.Model, e.cfg.Model),
		Prompt:      prompt,
		Usage:       completion.Usage,
		GeneratedAt: time.Now().UTC(),
	}

	if err := e.cache.Set(key, result); err != nil {
		return Result{}, err
	}

	return result, nil
}
