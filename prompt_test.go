package autocomplete

import (
	"strings"
	"testing"
)

func TestBuildPromptRedactsSecrets(t *testing.T) {
	prompt := BuildPrompt(Request{
		Shell:     "bash",
		InputLine: "curl -H 'Authorization: Bearer abcdefghijklmnopqrstuvwxyz123456' https://example.com",
		CWD:       "/tmp/project",
		RecentHistory: []string{
			"export OPENAI_API_KEY=abcdefghijklmnopqrstuvwxyz123456",
		},
		Env: map[string]string{
			"OPENAI_API_KEY": "abcdefghijklmnopqrstuvwxyz123456",
		},
	})

	if strings.Contains(prompt, "abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("prompt leaked a token: %s", prompt)
	}
	if !strings.Contains(prompt, "REDACTED") {
		t.Fatalf("prompt did not redact sensitive content: %s", prompt)
	}
}

func TestParseSuggestionsAcceptsJSONAndPlainText(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		count int
	}{
		{name: "json object", raw: `{"suggestions":["git status","git add ."]}`, count: 2},
		{name: "json array", raw: `["git status","git add ."]`, count: 2},
		{name: "plain text", raw: "1. git status\n2. git add .", count: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSuggestions(tc.raw)
			if len(got) != tc.count {
				t.Fatalf("got %d suggestions, want %d: %#v", len(got), tc.count, got)
			}
		})
	}
}
