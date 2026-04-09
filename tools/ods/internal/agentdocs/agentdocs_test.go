package agentdocs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSuccess(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "AGENTS.md"), `[Agent Docs](./docs/agent/README.md)`)
	writeFile(t, filepath.Join(root, "docs/agent/README.md"), `[Architecture](./ARCHITECTURE.md)
[Root](../../AGENTS.md)`)
	writeFile(t, filepath.Join(root, "docs/agent/ARCHITECTURE.md"), `ok`)
	writeFile(t, filepath.Join(root, "docs/agent/BRANCHING.md"), `ok`)
	writeFile(t, filepath.Join(root, "docs/agent/HARNESS.md"), `ok`)
	writeFile(t, filepath.Join(root, "docs/agent/GOLDEN_RULES.md"), `ok`)
	writeFile(t, filepath.Join(root, "docs/agent/LEGACY_ZONES.md"), `ok`)
	writeFile(t, filepath.Join(root, "docs/agent/QUALITY_SCORE.md"), `ok`)

	violations := Validate(root)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}

func TestValidateMissingAndBrokenLinks(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "AGENTS.md"), `[Agent Docs](./docs/agent/README.md)`)
	writeFile(t, filepath.Join(root, "docs/agent/README.md"), `[Missing](./MISSING.md)`)
	writeFile(t, filepath.Join(root, "docs/agent/ARCHITECTURE.md"), `ok`)

	violations := Validate(root)
	if len(violations) < 2 {
		t.Fatalf("expected multiple violations, got %+v", violations)
	}
}

func TestValidateSkipsReposWithoutAgentLabDocs(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "README.md"), `plain repo`)

	violations := Validate(root)
	if len(violations) != 0 {
		t.Fatalf("expected no violations for repo without agent-lab docs, got %+v", violations)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
