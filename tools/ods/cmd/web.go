package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/envutil"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/paths"
)

type webPackageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

// NewWebCommand creates a command that runs npm scripts from the web directory.
func NewWebCommand() *cobra.Command {
	var worktree string

	cmd := &cobra.Command{
		Use:   "web <script> [args...]",
		Short: "Run web/package.json npm scripts",
		Long:  webHelpDescription(),
		Args:  cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return webScriptNames(), cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().StringVar(&worktree, "worktree", "", "tracked agent-lab worktree to run from instead of the current checkout")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		runWebScript(args, worktree)
	}

	return cmd
}

func runWebScript(args []string, worktree string) {
	repoRoot, manifest, hasManifest := resolveAgentLabTarget(worktree)

	webDir, err := webDirForRoot(repoRoot)
	if err != nil {
		log.Fatalf("Failed to find web directory: %v", err)
	}

	scriptName := args[0]
	scriptArgs := args[1:]
	if len(scriptArgs) > 0 && scriptArgs[0] == "--" {
		scriptArgs = scriptArgs[1:]
	}

	npmArgs := []string{"run", scriptName}
	if len(scriptArgs) > 0 {
		// npm requires "--" to forward flags to the underlying script.
		npmArgs = append(npmArgs, "--")
		npmArgs = append(npmArgs, scriptArgs...)
	}
	log.Debugf("Running in %s: npm %v", webDir, npmArgs)

	webCmd := exec.Command("npm", npmArgs...)
	webCmd.Dir = webDir
	webCmd.Stdout = os.Stdout
	webCmd.Stderr = os.Stderr
	webCmd.Stdin = os.Stdin

	if hasManifest {
		webCmd.Env = envutil.ApplyOverrides(os.Environ(), manifest.RuntimeEnv())
		log.Infof("agent-lab worktree %s detected: web=%s api=%s", manifest.Branch, manifest.URLs.Web, manifest.URLs.API)
		log.Infof("lane=%s base-ref=%s", manifest.ResolvedLane(), manifest.BaseRef)
		log.Infof("dependency mode=%s search-infra=%s", manifest.ResolvedDependencies().Mode, manifest.ResolvedDependencies().SearchInfraMode)
	}

	if err := webCmd.Run(); err != nil {
		// For wrapped commands, preserve the child process's exit code and
		// avoid duplicating already-printed stderr output.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code != -1 {
				os.Exit(code)
			}
		}
		log.Fatalf("Failed to run npm: %v", err)
	}
}

func webScriptNames() []string {
	scripts, err := loadWebScripts()
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(scripts))
	for name := range scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func webHelpDescription() string {
	description := `Run npm scripts from web/package.json.

Examples:
  ods web dev
  ods web lint
  ods web test --watch
  ods web dev --worktree codex/fix/auth-banner-modal`

	scripts := webScriptNames()
	if len(scripts) == 0 {
		return description + "\n\nAvailable scripts: (unable to load)"
	}

	return description + "\n\nAvailable scripts:\n  " + strings.Join(scripts, "\n  ")
}

func loadWebScripts() (map[string]string, error) {
	webDir, err := webDirForRoot("")
	if err != nil {
		return nil, err
	}

	packageJSONPath := filepath.Join(webDir, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", packageJSONPath, err)
	}

	var pkg webPackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", packageJSONPath, err)
	}

	if pkg.Scripts == nil {
		return nil, nil
	}

	return pkg.Scripts, nil
}

func webDirForRoot(root string) (string, error) {
	if root == "" {
		var err error
		root, err = paths.GitRoot()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(root, "web"), nil
}
