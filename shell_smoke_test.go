package autocomplete_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellAdaptersSourceAndRequestSuggestions(t *testing.T) {
	tempDir := t.TempDir()
	stubBinary := filepath.Join(tempDir, "autocomplete")
	stubScript := "#!/bin/sh\nif [ \"$1\" = \"complete\" ]; then\n  printf 'git status\\ngit stash\\n'\n  exit 0\nfi\nprintf 'unsupported\\n' >&2\nexit 1\n"
	if err := os.WriteFile(stubBinary, []byte(stubScript), 0o755); err != nil {
		t.Fatal(err)
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		shell   string
		command string
	}{
		{
			name:    "bash",
			shell:   "bash",
			command: fmt.Sprintf("source %q && _autocomplete_request_suggestions 'git sta' %q", filepath.Join(repoRoot, "autocomplete.sh"), repoRoot),
		},
		{
			name:    "zsh",
			shell:   "zsh",
			command: fmt.Sprintf("source %q && _autocomplete_request_suggestions 'git sta' %q", filepath.Join(repoRoot, "autocomplete.zsh"), repoRoot),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := exec.LookPath(tc.shell); err != nil {
				t.Skipf("%s is not installed", tc.shell)
			}
			cmd := exec.Command(tc.shell, "-lc", tc.command)
			cmd.Env = append(os.Environ(),
				"PATH="+filepath.Dir(stubBinary)+string(os.PathListSeparator)+os.Getenv("PATH"),
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s adapter failed: %v\n%s", tc.name, err, output)
			}
			if !strings.Contains(string(output), "git status") {
				t.Fatalf("%s adapter output missing suggestion: %s", tc.name, output)
			}
		})
	}
}
