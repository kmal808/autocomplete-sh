package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	autocomplete "github.com/kmal808/autocomplete-sh"
)

type fileConfig struct {
	Provider      string  `json:"provider,omitempty"`
	Model         string  `json:"model,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	Timeout       string  `json:"timeout,omitempty"`
	OpenAIBaseURL string  `json:"openai_base_url,omitempty"`
	OllamaHost    string  `json:"ollama_host,omitempty"`
	CacheDir      string  `json:"cache_dir,omitempty"`
	CacheTTL      string  `json:"cache_ttl,omitempty"`
	CacheSize     int     `json:"cache_size,omitempty"`
}

func Default() (autocomplete.Config, error) {
	cacheDir, err := defaultCacheDir()
	if err != nil {
		return autocomplete.Config{}, err
	}

	return autocomplete.Config{
		Provider:           "openai",
		Model:              "gpt-4o-mini",
		Temperature:        0,
		Timeout:            30 * time.Second,
		OpenAIBaseURL:      "https://api.openai.com/v1",
		OllamaHost:         "http://127.0.0.1:11434",
		CacheDir:           cacheDir,
		CacheTTL:           24 * time.Hour,
		CacheSize:          100,
		MaxHistoryCommands: 20,
		MaxRecentFiles:     20,
	}, nil
}

func Path() (string, error) {
	if override := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_CONFIG")); override != "" {
		return override, nil
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, autocomplete.AppName, "config.json"), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(base, autocomplete.AppName, "config.json"), nil
	}
	return filepath.Join(base, autocomplete.AppName, "config.json"), nil
}

func Load() (autocomplete.Config, string, error) {
	cfg, err := Default()
	if err != nil {
		return autocomplete.Config{}, "", err
	}

	path, err := Path()
	if err != nil {
		return autocomplete.Config{}, "", err
	}

	data, err := os.ReadFile(path)
	if err == nil {
		var fileCfg fileConfig
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			return autocomplete.Config{}, path, err
		}
		cfg = mergeFileConfig(cfg, fileCfg)
	} else if !errors.Is(err, os.ErrNotExist) {
		return autocomplete.Config{}, path, err
	}

	cfg = overlayEnv(cfg)
	return cfg, path, nil
}

func Init() (autocomplete.Config, string, error) {
	cfg, path, err := Load()
	if err != nil {
		return autocomplete.Config{}, "", err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := Save(path, cfg); err != nil {
			return autocomplete.Config{}, "", err
		}
	}
	return cfg, path, nil
}

func Save(path string, cfg autocomplete.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	payload := fileConfig{
		Provider:      cfg.Provider,
		Model:         cfg.Model,
		Temperature:   cfg.Temperature,
		Timeout:       cfg.Timeout.String(),
		OpenAIBaseURL: cfg.OpenAIBaseURL,
		OllamaHost:    cfg.OllamaHost,
		CacheDir:      cfg.CacheDir,
		CacheTTL:      cfg.CacheTTL.String(),
		CacheSize:     cfg.CacheSize,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func Get(cfg autocomplete.Config, key string) (string, error) {
	switch normalizeKey(key) {
	case "provider":
		return cfg.Provider, nil
	case "model":
		return cfg.Model, nil
	case "temperature":
		return strconv.FormatFloat(cfg.Temperature, 'f', -1, 64), nil
	case "timeout":
		return cfg.Timeout.String(), nil
	case "openai_base_url":
		return cfg.OpenAIBaseURL, nil
	case "ollama_host":
		return cfg.OllamaHost, nil
	case "cache_dir":
		return cfg.CacheDir, nil
	case "cache_ttl":
		return cfg.CacheTTL.String(), nil
	case "cache_size":
		return strconv.Itoa(cfg.CacheSize), nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func Set(path string, key, value string) (autocomplete.Config, error) {
	cfg, _, err := Load()
	if err != nil {
		return autocomplete.Config{}, err
	}

	switch normalizeKey(key) {
	case "provider":
		cfg.Provider = strings.ToLower(strings.TrimSpace(value))
	case "model":
		cfg.Model = strings.TrimSpace(value)
	case "temperature":
		cfg.Temperature, err = strconv.ParseFloat(strings.TrimSpace(value), 64)
	case "timeout":
		cfg.Timeout, err = time.ParseDuration(strings.TrimSpace(value))
	case "openai_base_url":
		cfg.OpenAIBaseURL = strings.TrimSpace(value)
	case "ollama_host":
		cfg.OllamaHost = strings.TrimSpace(value)
	case "cache_dir":
		cfg.CacheDir = strings.TrimSpace(value)
	case "cache_ttl":
		cfg.CacheTTL, err = time.ParseDuration(strings.TrimSpace(value))
	case "cache_size":
		cfg.CacheSize, err = strconv.Atoi(strings.TrimSpace(value))
	default:
		return autocomplete.Config{}, fmt.Errorf("unknown config key %q", key)
	}
	if err != nil {
		return autocomplete.Config{}, err
	}

	return cfg, SaveAndReturn(path, cfg)
}

func SaveAndReturn(path string, cfg autocomplete.Config) error {
	return Save(path, cfg)
}

func mergeFileConfig(base autocomplete.Config, fileCfg fileConfig) autocomplete.Config {
	if fileCfg.Provider != "" {
		base.Provider = fileCfg.Provider
	}
	if fileCfg.Model != "" {
		base.Model = fileCfg.Model
	}
	if fileCfg.Temperature != 0 {
		base.Temperature = fileCfg.Temperature
	}
	if fileCfg.Timeout != "" {
		if parsed, err := time.ParseDuration(fileCfg.Timeout); err == nil {
			base.Timeout = parsed
		}
	}
	if fileCfg.OpenAIBaseURL != "" {
		base.OpenAIBaseURL = fileCfg.OpenAIBaseURL
	}
	if fileCfg.OllamaHost != "" {
		base.OllamaHost = fileCfg.OllamaHost
	}
	if fileCfg.CacheDir != "" {
		base.CacheDir = fileCfg.CacheDir
	}
	if fileCfg.CacheTTL != "" {
		if parsed, err := time.ParseDuration(fileCfg.CacheTTL); err == nil {
			base.CacheTTL = parsed
		}
	}
	if fileCfg.CacheSize > 0 {
		base.CacheSize = fileCfg.CacheSize
	}
	return base
}

func overlayEnv(cfg autocomplete.Config) autocomplete.Config {
	if provider := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_PROVIDER")); provider != "" {
		cfg.Provider = provider
	}
	if model := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_MODEL")); model != "" {
		cfg.Model = model
	}
	if temperature := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_TEMPERATURE")); temperature != "" {
		if parsed, err := strconv.ParseFloat(temperature, 64); err == nil {
			cfg.Temperature = parsed
		}
	}
	if timeout := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_TIMEOUT")); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = parsed
		}
	}
	if baseURL := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_OPENAI_BASE_URL")); baseURL != "" {
		cfg.OpenAIBaseURL = baseURL
	}
	if host := strings.TrimSpace(os.Getenv("OLLAMA_HOST")); host != "" {
		cfg.OllamaHost = host
	}
	if host := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_OLLAMA_HOST")); host != "" {
		cfg.OllamaHost = host
	}
	if cacheDir := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_CACHE_DIR")); cacheDir != "" {
		cfg.CacheDir = cacheDir
	}
	if cacheTTL := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_CACHE_TTL")); cacheTTL != "" {
		if parsed, err := time.ParseDuration(cacheTTL); err == nil {
			cfg.CacheTTL = parsed
		}
	}
	if cacheSize := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_CACHE_SIZE")); cacheSize != "" {
		if parsed, err := strconv.Atoi(cacheSize); err == nil {
			cfg.CacheSize = parsed
		}
	}
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
		cfg.OpenAIAPIKey = key
	}
	return cfg
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(key, "-", "_")))
}

func defaultCacheDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("AUTOCOMPLETE_CACHE_DIR")); override != "" {
		return override, nil
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); xdg != "" {
		return filepath.Join(xdg, autocomplete.AppName), nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, autocomplete.AppName), nil
}
