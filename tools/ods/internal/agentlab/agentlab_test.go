package agentlab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"feat/My Feature": "feat-my-feature",
		"lab/agent_docs":  "lab-agent-docs",
		"  ":              "worktree",
	}

	for input, want := range tests {
		input := input
		want := want
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if got := Slug(input); got != want {
				t.Fatalf("Slug(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestWorktreeIDIsCollisionResistant(t *testing.T) {
	t.Parallel()

	idOne := worktreeID("feat/foo_bar")
	idTwo := worktreeID("feat/foo-bar")
	if idOne == idTwo {
		t.Fatalf("expected distinct worktree ids, got %q", idOne)
	}
	if !strings.HasPrefix(idOne, "feat-foo-bar-") {
		t.Fatalf("unexpected worktree id format: %s", idOne)
	}
}

func TestInferLane(t *testing.T) {
	t.Parallel()

	tests := map[string]WorktreeLane{
		"lab/docs":                 WorktreeLaneLab,
		"codex/lab/docs":           WorktreeLaneLab,
		"fix/auth-banner-modal":    WorktreeLaneProduct,
		"codex/feat/agent-check":   WorktreeLaneProduct,
		"chore/update-readme":      WorktreeLaneProduct,
		"codex/auth-banner-modal":  WorktreeLaneCustom,
		"agent-lab":                WorktreeLaneCustom,
	}

	for branch, want := range tests {
		branch := branch
		want := want
		t.Run(branch, func(t *testing.T) {
			t.Parallel()
			if got := InferLane(branch); got != want {
				t.Fatalf("InferLane(%q) = %q, want %q", branch, got, want)
			}
		})
	}
}

func TestResolveCreateBaseRef(t *testing.T) {
	t.Parallel()

	refExists := func(ref string) bool {
		switch ref {
		case "codex/agent-lab", "origin/main":
			return true
		default:
			return false
		}
	}

	product := ResolveCreateBaseRef("codex/fix/auth-banner-modal", "", refExists)
	if product.Ref != "origin/main" || product.Lane != WorktreeLaneProduct {
		t.Fatalf("unexpected product base selection: %+v", product)
	}

	lab := ResolveCreateBaseRef("codex/lab/bootstrap-docs", "", refExists)
	if lab.Ref != "codex/agent-lab" || lab.Lane != WorktreeLaneLab {
		t.Fatalf("unexpected lab base selection: %+v", lab)
	}

	explicit := ResolveCreateBaseRef("codex/auth-banner-modal", "origin/release", refExists)
	if explicit.Ref != "origin/release" || explicit.Lane != WorktreeLaneCustom {
		t.Fatalf("unexpected explicit base selection: %+v", explicit)
	}

	custom := ResolveCreateBaseRef("codex/auth-banner-modal", "", refExists)
	if custom.Ref != "HEAD" || custom.Lane != WorktreeLaneCustom {
		t.Fatalf("unexpected custom base selection: %+v", custom)
	}
}

func TestBuildManifest(t *testing.T) {
	t.Parallel()

	ports := PortSet{Web: 3301, API: 8381, ModelServer: 9301, MCP: 8391}
	manifest := BuildManifest(
		"/repo/main",
		"/repo/.git",
		"feat/agent-harness",
		WorktreeLaneProduct,
		"origin/main",
		"/worktrees/feat-agent-harness",
		ports,
		DependencyModeNamespaced,
	)

	if manifest.ID != worktreeID("feat/agent-harness") {
		t.Fatalf("unexpected manifest id: %s", manifest.ID)
	}
	if manifest.URLs.Web != "http://127.0.0.1:3301" {
		t.Fatalf("unexpected web url: %s", manifest.URLs.Web)
	}
	if manifest.ComposeProject != "onyx-"+worktreeID("feat/agent-harness") {
		t.Fatalf("unexpected compose project: %s", manifest.ComposeProject)
	}
	if got := manifest.ShellEnv()["INTERNAL_URL"]; got != "http://127.0.0.1:8381" {
		t.Fatalf("unexpected INTERNAL_URL: %s", got)
	}
	if got := manifest.ResolvedDependencies().PostgresDatabase; got != "agentlab_"+strings.ReplaceAll(worktreeID("feat/agent-harness"), "-", "_") {
		t.Fatalf("unexpected postgres database: %s", got)
	}
	if got := manifest.RuntimeEnv()["DEFAULT_REDIS_PREFIX"]; got != "agentlab:"+worktreeID("feat/agent-harness") {
		t.Fatalf("unexpected redis prefix: %s", got)
	}
}

func TestWriteManifestAndLoadAll(t *testing.T) {
	t.Parallel()

	commonGitDir := t.TempDir()
	manifest := BuildManifest(
		"/repo/main",
		commonGitDir,
		"lab/docs",
		WorktreeLaneLab,
		"HEAD",
		"/repo-worktrees/lab-docs",
		PortSet{Web: 3302, API: 8382, ModelServer: 9302, MCP: 8392},
		DependencyModeShared,
	)

	if err := WriteManifest(commonGitDir, manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}

	manifests, err := LoadAll(commonGitDir)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("LoadAll() length = %d, want 1", len(manifests))
	}
	if manifests[0].Branch != manifest.Branch {
		t.Fatalf("unexpected branch: %s", manifests[0].Branch)
	}
}

func TestWriteEnvFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manifest := BuildManifest(
		"/repo/main",
		filepath.Join(root, ".git"),
		"feat/env",
		WorktreeLaneProduct,
		"HEAD",
		root,
		PortSet{Web: 3303, API: 8383, ModelServer: 9303, MCP: 8393},
		DependencyModeNamespaced,
	)

	if err := WriteEnvFiles(manifest); err != nil {
		t.Fatalf("WriteEnvFiles() error = %v", err)
	}

	for _, path := range []string{manifest.EnvFile, manifest.WebEnvFile} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected env file %s to exist: %v", path, err)
		}
	}

	backendEnv, err := os.ReadFile(manifest.EnvFile)
	if err != nil {
		t.Fatalf("read backend env file: %v", err)
	}
	if !containsAll(
		string(backendEnv),
		"POSTGRES_DB=agentlab_"+strings.ReplaceAll(worktreeID("feat/env"), "-", "_"),
		"DEFAULT_REDIS_PREFIX=agentlab:"+worktreeID("feat/env"),
		"S3_FILE_STORE_BUCKET_NAME=onyx-agentlab-"+worktreeID("feat/env"),
	) {
		t.Fatalf("backend env file missing dependency namespace entries: %s", string(backendEnv))
	}
}

func TestFindByIdentifierRejectsAmbiguousSlug(t *testing.T) {
	t.Parallel()

	commonGitDir := t.TempDir()
	manifests := []Manifest{
		BuildManifest(
			"/repo/main",
			commonGitDir,
			"feat/foo_bar",
			WorktreeLaneProduct,
			"HEAD",
			"/repo-worktrees/"+worktreeID("feat/foo_bar"),
			PortSet{Web: 3302, API: 8382, ModelServer: 9302, MCP: 8392},
			DependencyModeNamespaced,
		),
		BuildManifest(
			"/repo/main",
			commonGitDir,
			"feat/foo-bar",
			WorktreeLaneProduct,
			"HEAD",
			"/repo-worktrees/"+worktreeID("feat/foo-bar"),
			PortSet{Web: 3303, API: 8383, ModelServer: 9303, MCP: 8393},
			DependencyModeNamespaced,
		),
	}

	for _, manifest := range manifests {
		if err := WriteManifest(commonGitDir, manifest); err != nil {
			t.Fatalf("WriteManifest() error = %v", err)
		}
	}

	if _, found, err := FindByIdentifier(commonGitDir, "feat-foo-bar"); err == nil || found {
		t.Fatalf("expected ambiguous slug lookup to fail, found=%t err=%v", found, err)
	}
}

func TestBootstrapLinksAndClonesFromSource(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	checkoutRoot := t.TempDir()
	commonGitDir := filepath.Join(sourceRoot, ".git")

	writeTestFile(t, filepath.Join(sourceRoot, ".vscode", ".env"), "OPENAI_API_KEY=test\n")
	writeTestFile(t, filepath.Join(sourceRoot, ".vscode", ".env.web"), "AUTH_TYPE=basic\n")
	writeTestFile(t, filepath.Join(sourceRoot, ".venv", "bin", "python"), "#!/bin/sh\n")
	writeTestFile(t, filepath.Join(sourceRoot, "web", "node_modules", ".bin", "next"), "#!/bin/sh\n")

	manifest := BuildManifest(
		sourceRoot,
		commonGitDir,
		"feat/bootstrap",
		WorktreeLaneProduct,
		"HEAD",
		checkoutRoot,
		PortSet{Web: 3305, API: 8385, ModelServer: 9305, MCP: 8395},
		DependencyModeNamespaced,
	)

	result, err := Bootstrap(manifest, BootstrapOptions{
		EnvMode:    BootstrapModeLink,
		PythonMode: BootstrapModeLink,
		WebMode:    BootstrapModeClone,
	})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if len(result.Actions) == 0 {
		t.Fatal("expected bootstrap actions to be recorded")
	}

	if target, err := os.Readlink(filepath.Join(checkoutRoot, ".vscode", ".env")); err != nil || target == "" {
		t.Fatalf("expected .vscode/.env symlink, err=%v target=%q", err, target)
	}
	if target, err := os.Readlink(filepath.Join(checkoutRoot, ".venv")); err != nil || target == "" {
		t.Fatalf("expected .venv symlink, err=%v target=%q", err, target)
	}
	if _, err := os.Stat(filepath.Join(checkoutRoot, "web", "node_modules", ".bin", "next")); err != nil {
		t.Fatalf("expected cloned node_modules marker: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(checkoutRoot, "web", "node_modules")); err != nil {
		t.Fatalf("expected node_modules to exist: %v", err)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
