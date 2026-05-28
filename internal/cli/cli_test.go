package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	autocomplete "github.com/kmal808/autocomplete-sh"
	"github.com/kmal808/autocomplete-sh/internal/config"
)

type stubProvider struct {
	suggestions []string
}

func (p stubProvider) Complete(_ context.Context, req autocomplete.PromptRequest) (autocomplete.CompletionResult, error) {
	return autocomplete.CompletionResult{
		Suggestions: p.suggestions,
		Provider:    "openai",
		Model:       req.Model,
	}, nil
}

func TestRunnerCompleteAndPrompt(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	t.Setenv("AUTOCOMPLETE_CONFIG", configPath)

	cfg, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.CacheDir = filepath.Join(tempDir, "cache")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := Runner{
		Stdout: &stdout,
		Stderr: &stderr,
		Env: func(key string) string {
			switch key {
			case "AUTOCOMPLETE_HISTORY":
				return "git checkout main\ngit pull"
			case "AUTOCOMPLETE_ENV":
				return "USER=kvrt\nTERM=xterm-256color"
			default:
				return ""
			}
		},
		NewProvider: func(cfg autocomplete.Config) (autocomplete.Provider, error) {
			return stubProvider{suggestions: []string{"git status", "git stash"}}, nil
		},
	}

	if code := runner.Run(context.Background(), []string{"prompt", "--shell", "bash", "--line", "git sta", "--cwd", tempDir}); code != 0 {
		t.Fatalf("prompt failed: %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "git sta") {
		t.Fatalf("prompt output missing input: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run(context.Background(), []string{"complete", "--shell", "bash", "--line", "git sta", "--cwd", tempDir, "--json"}); code != 0 {
		t.Fatalf("complete failed: %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "git status") {
		t.Fatalf("completion output missing suggestion: %s", stdout.String())
	}

	var parsed autocomplete.Result
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Suggestions) != 2 {
		t.Fatalf("unexpected suggestions: %#v", parsed.Suggestions)
	}
}
