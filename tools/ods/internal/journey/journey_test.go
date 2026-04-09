package journey

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDefinitions(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	registryDir := filepath.Join(root, "web", "tests", "e2e", "journeys")
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "registry.json"), []byte(`{
  "journeys": [
    {
      "name": "auth-landing",
      "description": "test",
      "test_path": "tests/e2e/journeys/auth_landing.spec.ts",
      "project": "journey",
      "requires_model_server": false,
      "skip_global_setup": true
    }
  ]
}`), 0644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	definitions, err := ResolveDefinitions(root, []string{"auth-landing"})
	if err != nil {
		t.Fatalf("resolve definitions: %v", err)
	}
	if len(definitions) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(definitions))
	}
	if definitions[0].Project != "journey" {
		t.Fatalf("expected project journey, got %q", definitions[0].Project)
	}
}

func TestLoadPlanRequiresJourneys(t *testing.T) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "journeys.json")
	if err := os.WriteFile(path, []byte(`{"journeys":["auth-landing"]}`), 0644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	plan, err := LoadPlan(path)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if len(plan.Journeys) != 1 || plan.Journeys[0] != "auth-landing" {
		t.Fatalf("unexpected plan contents: %+v", plan)
	}
}
