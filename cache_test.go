package autocomplete

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFileCacheSetGet(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewFileCache(dir, time.Hour, 10)
	if err != nil {
		t.Fatal(err)
	}

	key := cacheKey("openai", "gpt-4o-mini", "git status")
	want := Result{
		Suggestions: []string{"git status"},
		GeneratedAt: time.Now().UTC(),
	}
	if err := cache.Set(key, want); err != nil {
		t.Fatal(err)
	}

	got, ok, err := cache.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got.Suggestions) != 1 || got.Suggestions[0] != "git status" {
		t.Fatalf("unexpected cache payload: %#v", got)
	}
}

func TestFileCacheTTLAndEviction(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewFileCache(dir, time.Millisecond, 1)
	if err != nil {
		t.Fatal(err)
	}

	oldKey := cacheKey("openai", "gpt-4o-mini", "git status")
	oldResult := Result{
		Suggestions: []string{"git status"},
		GeneratedAt: time.Now().Add(-time.Second),
	}
	if err := cache.Set(oldKey, oldResult); err != nil {
		t.Fatal(err)
	}

	if _, ok, err := cache.Get(oldKey); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("expected expired cache entry to miss")
	}

	firstKey := cacheKey("openai", "gpt-4o-mini", "git add .")
	secondKey := cacheKey("openai", "gpt-4o-mini", "git commit")
	if err := cache.Set(firstKey, Result{Suggestions: []string{"git add ."}, GeneratedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := cache.Set(secondKey, Result{Suggestions: []string{"git commit"}, GeneratedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	if _, ok, err := cache.Get(firstKey); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatalf("expected older cache entry at %s to be evicted", filepath.Join(dir, firstKey+".json"))
	}
}
