package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	autocomplete "github.com/kmal808/autocomplete-sh"
	"github.com/kmal808/autocomplete-sh/internal/config"
)

type Runner struct {
	Stdout      io.Writer
	Stderr      io.Writer
	Env         func(string) string
	NewProvider func(autocomplete.Config) (autocomplete.Provider, error)
}

func (r Runner) Run(ctx context.Context, args []string) int {
	if r.Stdout == nil {
		r.Stdout = os.Stdout
	}
	if r.Stderr == nil {
		r.Stderr = os.Stderr
	}
	if r.Env == nil {
		r.Env = os.Getenv
	}
	if r.NewProvider == nil {
		r.NewProvider = autocomplete.NewProvider
	}

	if len(args) == 0 {
		r.printHelp()
		return 0
	}

	switch args[0] {
	case "complete":
		return r.runComplete(ctx, args[1:])
	case "prompt":
		return r.runPrompt(args[1:])
	case "config":
		return r.runConfig(args[1:])
	case "doctor":
		return r.runDoctor()
	case "cache":
		return r.runCache(args[1:])
	case "version":
		fmt.Fprintln(r.Stdout, autocomplete.Version)
		return 0
	case "--help", "-h", "help":
		r.printHelp()
		return 0
	default:
		fmt.Fprintf(r.Stderr, "unknown command %q\n", args[0])
		return 1
	}
}

func (r Runner) runComplete(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("complete", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)

	shell := fs.String("shell", "", "shell name")
	line := fs.String("line", "", "current input line")
	cwd := fs.String("cwd", "", "working directory")
	jsonOutput := fs.Bool("json", false, "print JSON output")
	plainOutput := fs.Bool("plain", false, "print plain suggestions")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}

	provider, err := r.NewProvider(cfg)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	engine, err := autocomplete.NewEngine(cfg, provider)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}

	request := RequestFromEnv(r.Env)
	request.Shell = firstNonEmpty(*shell, request.Shell)
	request.InputLine = firstNonEmpty(*line, request.InputLine)
	request.CWD = firstNonEmpty(*cwd, request.CWD)
	if request.CWD == "" {
		if pwd, err := os.Getwd(); err == nil {
			request.CWD = pwd
		}
	}

	result, err := engine.Suggest(ctx, request)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}

	switch {
	case *jsonOutput:
		encoder := json.NewEncoder(r.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
	default:
		if *plainOutput || !*jsonOutput {
			for _, suggestion := range result.Suggestions {
				fmt.Fprintln(r.Stdout, suggestion)
			}
		}
	}

	return 0
}

func (r Runner) runPrompt(args []string) int {
	fs := flag.NewFlagSet("prompt", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)

	shell := fs.String("shell", "", "shell name")
	line := fs.String("line", "", "current input line")
	cwd := fs.String("cwd", "", "working directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	request := RequestFromEnv(r.Env)
	request.Shell = firstNonEmpty(*shell, request.Shell)
	request.InputLine = firstNonEmpty(*line, request.InputLine)
	request.CWD = firstNonEmpty(*cwd, request.CWD)
	if request.CWD == "" {
		if pwd, err := os.Getwd(); err == nil {
			request.CWD = pwd
		}
	}

	fmt.Fprintln(r.Stdout, autocomplete.BuildPrompt(request))
	return 0
}

func (r Runner) runConfig(args []string) int {
	if len(args) == 0 {
		r.printConfigHelp()
		return 0
	}

	switch args[0] {
	case "init":
		cfg, path, err := config.Init()
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		payload := map[string]any{"path": path, "config": cfg}
		return writeJSON(r.Stdout, r.Stderr, payload)
	case "show":
		cfg, path, err := config.Load()
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		payload := map[string]any{"path": path, "config": cfg}
		return writeJSON(r.Stdout, r.Stderr, payload)
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(r.Stderr, "config get requires a key")
			return 2
		}
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		value, err := config.Get(cfg, args[1])
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		fmt.Fprintln(r.Stdout, value)
		return 0
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(r.Stderr, "config set requires a key and value")
			return 2
		}
		_, path, err := config.Load()
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		cfg, err := config.Set(path, args[1], args[2])
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		payload := map[string]any{"path": path, "config": cfg}
		return writeJSON(r.Stdout, r.Stderr, payload)
	default:
		fmt.Fprintf(r.Stderr, "unknown config subcommand %q\n", args[0])
		return 1
	}
}

func (r Runner) runDoctor() int {
	cfg, path, err := config.Load()
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}

	binaryPath, _ := exec.LookPath("autocomplete")
	cacheInfo, cacheErr := os.Stat(cfg.CacheDir)

	report := map[string]any{
		"version":      autocomplete.Version,
		"short_name":   autocomplete.ShortName,
		"repo_url":     autocomplete.RepoURL,
		"config_path":  path,
		"provider":     cfg.Provider,
		"model":        cfg.Model,
		"binary_found": binaryPath != "",
		"binary_path":  binaryPath,
		"openai_key":   cfg.OpenAIAPIKey != "",
		"ollama_host":  cfg.OllamaHost,
		"cache_dir":    cfg.CacheDir,
		"cache_ready":  cacheErr == nil && cacheInfo.IsDir(),
	}

	return writeJSON(r.Stdout, r.Stderr, report)
}

func (r Runner) runCache(args []string) int {
	if len(args) != 1 || args[0] != "clear" {
		fmt.Fprintln(r.Stderr, "usage: autocomplete cache clear")
		return 2
	}

	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	cache, err := autocomplete.NewFileCache(cfg.CacheDir, cfg.CacheTTL, cfg.CacheSize)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if err := cache.Clear(); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	fmt.Fprintln(r.Stdout, "cache cleared")
	return 0
}

func (r Runner) printHelp() {
	fmt.Fprintf(r.Stdout, "autocomplete (%s) - AI-assisted terminal suggestions\n", autocomplete.ShortName)
	fmt.Fprintln(r.Stdout, "")
	fmt.Fprintln(r.Stdout, "Commands:")
	fmt.Fprintln(r.Stdout, "  complete --shell <bash|zsh> --line <input> --cwd <path> [--json|--plain]")
	fmt.Fprintln(r.Stdout, "  prompt --shell <bash|zsh> --line <input> --cwd <path>")
	fmt.Fprintln(r.Stdout, "  config init|show|get|set")
	fmt.Fprintln(r.Stdout, "  doctor")
	fmt.Fprintln(r.Stdout, "  cache clear")
	fmt.Fprintln(r.Stdout, "  version")
	fmt.Fprintln(r.Stdout, "")
	fmt.Fprintf(r.Stdout, "Alias: %s\n", autocomplete.ShortName)
}

func (r Runner) printConfigHelp() {
	fmt.Fprintln(r.Stdout, "autocomplete config init|show|get|set")
}

func RequestFromEnv(getEnv func(string) string) autocomplete.Request {
	request := autocomplete.Request{
		Shell:     strings.TrimSpace(getEnv("AUTOCOMPLETE_SHELL")),
		InputLine: strings.TrimSpace(getEnv("AUTOCOMPLETE_LINE")),
		CWD:       strings.TrimSpace(getEnv("AUTOCOMPLETE_CWD")),
		HelpText:  strings.TrimSpace(getEnv("AUTOCOMPLETE_HELP")),
		Env:       parseEnvList(getEnv("AUTOCOMPLETE_ENV")),
	}

	request.RecentHistory = splitLines(getEnv("AUTOCOMPLETE_HISTORY"))
	request.RecentFiles = splitLines(getEnv("AUTOCOMPLETE_RECENT_FILES"))
	return request
}

func splitLines(raw string) []string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func parseEnvList(raw string) map[string]string {
	result := make(map[string]string)
	for _, line := range splitLines(raw) {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return result
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func CollectHistory(limit int) string {
	if limit <= 0 {
		return ""
	}
	return strings.Join(lastN(splitLines(os.Getenv("AUTOCOMPLETE_HISTORY")), limit), "\n")
}

func CollectRecentFiles(cwd string, limit int) string {
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return ""
	}

	type fileInfo struct {
		name string
		mod  time.Time
	}

	files := make([]fileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: entry.Name(), mod: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].mod.After(files[j].mod)
	})

	if limit > len(files) {
		limit = len(files)
	}

	lines := make([]string, 0, limit)
	for _, file := range files[:limit] {
		lines = append(lines, filepath.Join(cwd, file.name))
	}
	return strings.Join(lines, "\n")
}

func CollectHelp(commandLine string) string {
	fields := strings.Fields(commandLine)
	if len(fields) == 0 {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, fields[0], "--help")
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func lastN(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}
