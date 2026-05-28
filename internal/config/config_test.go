package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAndSetConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	t.Setenv("AUTOCOMPLETE_CONFIG", configPath)
	t.Setenv("OPENAI_API_KEY", "test-key")

	cfg, path, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	if path != configPath {
		t.Fatalf("got path %q want %q", path, configPath)
	}
	if cfg.OpenAIAPIKey != "test-key" {
		t.Fatalf("expected env overlay to set key")
	}

	updated, err := Set(configPath, "timeout", "5s")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Timeout != 5*time.Second {
		t.Fatalf("unexpected timeout: %s", updated.Timeout)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected config file to be written")
	}
}
