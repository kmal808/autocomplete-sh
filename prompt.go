package autocomplete

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const systemPrompt = "You are a terminal autocomplete engine. Return 2 to 5 concise, runnable command suggestions that match the user's likely intent. Prefer safe commands, preserve shell syntax, and return plain newline-delimited commands without commentary."

var (
	secretAssignmentPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|token|secret|password)\b[^\n=]*=([^\s]+)`)
	longHexPattern          = regexp.MustCompile(`\b[0-9a-fA-F]{32,}\b`)
	longTokenPattern        = regexp.MustCompile(`\b[A-Za-z0-9_=-]{24,}\b`)
	numberedBulletPattern   = regexp.MustCompile(`^\s*(?:[-*]\s+|\d+[.)]\s+)`)
)

func BuildPrompt(req Request) string {
	var sections []string

	sections = append(sections, fmt.Sprintf("User input: %s", sanitizeText(strings.TrimSpace(req.InputLine))))

	if req.Shell != "" || req.CWD != "" {
		var meta []string
		if req.Shell != "" {
			meta = append(meta, fmt.Sprintf("Shell: %s", sanitizeText(req.Shell)))
		}
		if req.CWD != "" {
			meta = append(meta, fmt.Sprintf("Working directory: %s", sanitizeText(req.CWD)))
		}
		sections = append(sections, formatSection("Terminal Context", meta))
	}

	if len(req.Env) > 0 {
		var keys []string
		for key := range req.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		lines := make([]string, 0, len(keys))
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("%s=%s", key, sanitizeText(req.Env[key])))
		}
		sections = append(sections, formatSection("Environment", lines))
	}

	if len(req.RecentHistory) > 0 {
		lines := make([]string, 0, len(req.RecentHistory))
		for _, line := range req.RecentHistory {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			lines = append(lines, sanitizeText(trimmed))
		}
		if len(lines) > 0 {
			sections = append(sections, formatSection("Recent History", lines))
		}
	}

	if len(req.RecentFiles) > 0 {
		lines := make([]string, 0, len(req.RecentFiles))
		for _, file := range req.RecentFiles {
			trimmed := strings.TrimSpace(file)
			if trimmed == "" {
				continue
			}
			lines = append(lines, sanitizeText(trimmed))
		}
		if len(lines) > 0 {
			sections = append(sections, formatSection("Recent Files", lines))
		}
	}

	if help := strings.TrimSpace(req.HelpText); help != "" {
		sections = append(sections, formatSection("Command Help", []string{sanitizeText(help)}))
	}

	sections = append(sections, "Instructions:\nReturn 2 to 5 command suggestions as plain lines. Do not explain them. Prefer continuing the current command over replacing it when possible.")
	return strings.Join(sections, "\n\n")
}

func sanitizeText(input string) string {
	if input == "" {
		return ""
	}

	sanitized := secretAssignmentPattern.ReplaceAllString(input, `$1=REDACTED`)
	sanitized = longHexPattern.ReplaceAllString(sanitized, "REDACTED")
	sanitized = longTokenPattern.ReplaceAllStringFunc(sanitized, func(token string) string {
		if strings.Contains(token, "/") || strings.Contains(token, ".") || strings.Contains(token, "-") {
			return token
		}
		return "REDACTED"
	})
	return sanitized
}

func parseSuggestions(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	if parsed := parseSuggestionJSON(trimmed); len(parsed) > 0 {
		return parsed
	}

	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
	}

	seen := make(map[string]struct{})
	var suggestions []string
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "`\"'")
		line = numberedBulletPattern.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		suggestions = append(suggestions, line)
		if len(suggestions) == 5 {
			break
		}
	}

	return suggestions
}

func formatSection(title string, lines []string) string {
	var builder strings.Builder
	builder.WriteString(title)
	builder.WriteString(":\n")
	for _, line := range lines {
		builder.WriteString("- ")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}
