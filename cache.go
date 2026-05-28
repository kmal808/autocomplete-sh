package autocomplete

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type FileCache struct {
	dir     string
	ttl     time.Duration
	maxSize int
	enabled bool
}

func NewFileCache(dir string, ttl time.Duration, maxSize int) (*FileCache, error) {
	if dir == "" || maxSize <= 0 {
		return &FileCache{enabled: false}, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	return &FileCache{
		dir:     dir,
		ttl:     ttl,
		maxSize: maxSize,
		enabled: true,
	}, nil
}

func (c *FileCache) Get(key string) (Result, bool, error) {
	if c == nil || !c.enabled {
		return Result{}, false, nil
	}

	path := filepath.Join(c.dir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, false, nil
		}
		return Result{}, false, err
	}

	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return Result{}, false, err
	}

	if c.ttl > 0 && !result.GeneratedAt.IsZero() && time.Since(result.GeneratedAt) > c.ttl {
		_ = os.Remove(path)
		return Result{}, false, nil
	}

	now := time.Now()
	_ = os.Chtimes(path, now, now)
	result.CacheHit = true
	return result, true, nil
}

func (c *FileCache) Set(key string, result Result) error {
	if c == nil || !c.enabled {
		return nil
	}

	if result.GeneratedAt.IsZero() {
		result.GeneratedAt = time.Now().UTC()
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(c.dir, key+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	return c.evict()
}

func (c *FileCache) Clear() error {
	if c == nil || !c.enabled {
		return nil
	}

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(c.dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}

func cacheKey(provider, model, prompt string) string {
	sum := sha256.Sum256([]byte(provider + "\x00" + model + "\x00" + prompt))
	return hex.EncodeToString(sum[:])
}

func (c *FileCache) evict() error {
	if c == nil || !c.enabled || c.maxSize <= 0 {
		return nil
	}

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}

	type fileInfo struct {
		name    string
		modTime time.Time
	}

	files := make([]fileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, fileInfo{name: entry.Name(), modTime: info.ModTime()})
	}

	if len(files) <= c.maxSize {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	for _, file := range files[:len(files)-c.maxSize] {
		if err := os.Remove(filepath.Join(c.dir, file.name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}
