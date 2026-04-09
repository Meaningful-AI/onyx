package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentlab"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/paths"
)

type WorktreeCreateOptions struct {
	From           string
	Path           string
	Bootstrap      bool
	DependencyMode string
}

type WorktreeRemoveOptions struct {
	Force    bool
	DropDeps bool
}

type WorktreeBootstrapOptions struct {
	EnvMode    string
	PythonMode string
	WebMode    string
}

// NewWorktreeCommand creates the parent worktree command.
func NewWorktreeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage agent-lab git worktrees and harness metadata",
		Long: `Manage agent-lab git worktrees and the local harness state that makes
them bootable with isolated ports, URLs, and artifact directories.`,
	}

	cmd.AddCommand(newWorktreeCreateCommand())
	cmd.AddCommand(newWorktreeBootstrapCommand())
	cmd.AddCommand(newWorktreeDepsCommand())
	cmd.AddCommand(newWorktreeStatusCommand())
	cmd.AddCommand(newWorktreeShowCommand())
	cmd.AddCommand(newWorktreeRemoveCommand())

	return cmd
}

func newWorktreeCreateCommand() *cobra.Command {
	opts := &WorktreeCreateOptions{}

	cmd := &cobra.Command{
		Use:   "create <branch>",
		Short: "Create a new agent-lab worktree with isolated runtime metadata",
		Long: `Create a tracked agent-lab worktree and bootstrap its local runtime state.

Branch lanes control the default base ref when --from is not supplied:
  codex/lab/<name>   -> codex/agent-lab
  codex/fix/<name>   -> origin/main
  codex/feat/<name>  -> origin/main

Use conventional branch lanes for product work so the base stays explicit.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runWorktreeCreate(args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.From, "from", "", "git ref to branch from (defaults are inferred from the branch lane)")
	cmd.Flags().StringVar(&opts.Path, "path", "", "custom checkout path for the new worktree")
	cmd.Flags().BoolVar(&opts.Bootstrap, "bootstrap", true, "bootstrap env, Python, and frontend dependencies for the worktree")
	cmd.Flags().StringVar(&opts.DependencyMode, "dependency-mode", string(agentlab.DependencyModeNamespaced), "dependency mode: namespaced or shared")

	return cmd
}

func newWorktreeBootstrapCommand() *cobra.Command {
	opts := &WorktreeBootstrapOptions{}

	cmd := &cobra.Command{
		Use:   "bootstrap [worktree]",
		Short: "Bootstrap env files and dependencies for an agent-lab worktree",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			identifier := ""
			if len(args) == 1 {
				identifier = args[0]
			}
			runWorktreeBootstrap(identifier, opts)
		},
	}

	cmd.Flags().StringVar(&opts.EnvMode, "env-mode", string(agentlab.BootstrapModeAuto), "env bootstrap mode: auto, skip, link, copy")
	cmd.Flags().StringVar(&opts.PythonMode, "python-mode", string(agentlab.BootstrapModeAuto), "Python bootstrap mode: auto, skip, link, copy")
	cmd.Flags().StringVar(&opts.WebMode, "web-mode", string(agentlab.BootstrapModeAuto), "frontend bootstrap mode: auto, skip, clone, copy, npm")

	return cmd
}

func newWorktreeDepsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Manage namespaced external dependencies for an agent-lab worktree",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "up [worktree]",
		Short: "Provision external dependency state for a worktree",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			identifier := ""
			if len(args) == 1 {
				identifier = args[0]
			}
			runWorktreeDepsUp(identifier)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "status [worktree]",
		Short: "Inspect external dependency state for a worktree",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			identifier := ""
			if len(args) == 1 {
				identifier = args[0]
			}
			runWorktreeDepsStatus(identifier)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "reset [worktree]",
		Short: "Reset namespaced external dependency state for a worktree",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			identifier := ""
			if len(args) == 1 {
				identifier = args[0]
			}
			runWorktreeDepsReset(identifier)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "down [worktree]",
		Short: "Tear down namespaced external dependency state for a worktree",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			identifier := ""
			if len(args) == 1 {
				identifier = args[0]
			}
			runWorktreeDepsDown(identifier)
		},
	})

	return cmd
}

func newWorktreeStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List tracked agent-lab worktrees",
		Run: func(cmd *cobra.Command, args []string) {
			runWorktreeStatus()
		},
	}
}

func newWorktreeShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [worktree]",
		Short: "Show detailed metadata for an agent-lab worktree",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			identifier := ""
			if len(args) == 1 {
				identifier = args[0]
			}
			runWorktreeShow(identifier)
		},
	}
}

func newWorktreeRemoveCommand() *cobra.Command {
	opts := &WorktreeRemoveOptions{}

	cmd := &cobra.Command{
		Use:   "remove <worktree>",
		Short: "Remove an agent-lab worktree and its local state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runWorktreeRemove(args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "force removal even if git reports uncommitted changes")
	cmd.Flags().BoolVar(&opts.DropDeps, "drop-deps", false, "tear down namespaced dependencies before removing the worktree")

	return cmd
}

func runWorktreeCreate(branch string, opts *WorktreeCreateOptions) {
	repoRoot, err := paths.GitRoot()
	if err != nil {
		log.Fatalf("Failed to determine git root: %v", err)
	}

	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}

	if manifest, found, err := agentlab.FindByIdentifier(commonGitDir, branch); err != nil {
		log.Fatalf("Failed to inspect existing worktrees: %v", err)
	} else if found {
		log.Fatalf("Worktree already exists for %s at %s", manifest.Branch, manifest.CheckoutPath)
	}

	manifests, err := agentlab.LoadAll(commonGitDir)
	if err != nil {
		log.Fatalf("Failed to load worktree metadata: %v", err)
	}

	ports, err := agentlab.AllocatePorts(manifests)
	if err != nil {
		log.Fatalf("Failed to allocate worktree ports: %v", err)
	}

	dependencyMode := agentlab.DependencyMode(opts.DependencyMode)
	switch dependencyMode {
	case agentlab.DependencyModeShared, agentlab.DependencyModeNamespaced:
	default:
		log.Fatalf("Invalid dependency mode %q: must be shared or namespaced", opts.DependencyMode)
	}

	checkoutPath := opts.Path
	if checkoutPath == "" {
		checkoutPath = agentlab.DefaultCheckoutPath(repoRoot, branch)
	}
	checkoutPath, err = filepath.Abs(checkoutPath)
	if err != nil {
		log.Fatalf("Failed to resolve checkout path: %v", err)
	}

	if _, err := os.Stat(checkoutPath); err == nil {
		log.Fatalf("Checkout path already exists: %s", checkoutPath)
	}

	baseSelection := agentlab.ResolveCreateBaseRef(branch, opts.From, agentlab.GitRefExists)
	manifest := agentlab.BuildManifest(
		repoRoot,
		commonGitDir,
		branch,
		baseSelection.Lane,
		baseSelection.Ref,
		checkoutPath,
		ports,
		dependencyMode,
	)
	args := []string{"-c", "core.hooksPath=/dev/null", "worktree", "add", "-b", branch, checkoutPath, baseSelection.Ref}
	log.Infof("Creating worktree %s at %s", branch, checkoutPath)
	gitCmd := exec.Command("git", args...)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	gitCmd.Stdin = os.Stdin
	if err := gitCmd.Run(); err != nil {
		log.Fatalf("git worktree add failed: %v", err)
	}

	if resolvedPath, err := filepath.EvalSymlinks(checkoutPath); err == nil {
		manifest.CheckoutPath = resolvedPath
	}

	if err := agentlab.WriteEnvFiles(manifest); err != nil {
		log.Fatalf("Failed to write worktree env files: %v", err)
	}
	if err := agentlab.WriteManifest(commonGitDir, manifest); err != nil {
		log.Fatalf("Failed to write worktree manifest: %v", err)
	}

	if opts.Bootstrap {
		bootstrapResult, err := agentlab.Bootstrap(manifest, agentlab.BootstrapOptions{
			EnvMode:    agentlab.BootstrapModeAuto,
			PythonMode: agentlab.BootstrapModeAuto,
			WebMode:    agentlab.BootstrapModeAuto,
		})
		if err != nil {
			log.Fatalf("Failed to bootstrap worktree: %v", err)
		}
		for _, action := range bootstrapResult.Actions {
			fmt.Printf("  bootstrap: %s\n", action)
		}
	}

	manifest, dependencyResult, err := agentlab.ProvisionDependencies(commonGitDir, manifest)
	if err != nil {
		log.Fatalf("Failed to provision worktree dependencies: %v", err)
	}
	for _, action := range dependencyResult.Actions {
		fmt.Printf("  deps: %s\n", action)
	}

	fmt.Printf("Created agent-lab worktree %s\n", manifest.Branch)
	fmt.Printf("  checkout: %s\n", manifest.CheckoutPath)
	fmt.Printf("  lane: %s\n", manifest.ResolvedLane())
	fmt.Printf("  base ref: %s\n", manifest.BaseRef)
	fmt.Printf("  base selection: %s\n", baseSelection.Reason)
	fmt.Printf("  dependency mode: %s\n", manifest.ResolvedDependencies().Mode)
	if manifest.ResolvedDependencies().Namespace != "" {
		fmt.Printf("  dependency namespace: %s\n", manifest.ResolvedDependencies().Namespace)
	}
	if manifest.ResolvedDependencies().Mode == agentlab.DependencyModeNamespaced {
		fmt.Printf("  postgres database: %s\n", manifest.ResolvedDependencies().PostgresDatabase)
		fmt.Printf("  redis prefix: %s\n", manifest.ResolvedDependencies().RedisPrefix)
		fmt.Printf("  file-store bucket: %s\n", manifest.ResolvedDependencies().FileStoreBucket)
	}
	fmt.Printf("  web url:  %s\n", manifest.URLs.Web)
	fmt.Printf("  api url:  %s\n", manifest.URLs.API)
	fmt.Printf("  mcp url:  %s\n", manifest.URLs.MCP)
	fmt.Printf("  artifacts: %s\n", manifest.ArtifactDir)
	for _, warning := range manifest.DependencyWarnings() {
		fmt.Printf("  note: %s\n", warning)
	}
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", manifest.CheckoutPath)
	fmt.Printf("  # Make edits in the worktree itself.\n")
	if manifest.ResolvedLane() == agentlab.WorktreeLaneProduct {
		fmt.Printf("  # Run harness commands from the control checkout with --worktree %s.\n", manifest.Branch)
		fmt.Printf("  ods verify --worktree %s\n", manifest.Branch)
		fmt.Printf("  ods backend api --worktree %s\n", manifest.Branch)
		fmt.Printf("  ods web dev --worktree %s\n", manifest.Branch)
	} else {
		fmt.Printf("  ods backend api\n")
		fmt.Printf("  ods backend model_server\n")
		fmt.Printf("  ods web dev\n")
		fmt.Printf("  ods verify\n")
	}
}

func runWorktreeBootstrap(identifier string, opts *WorktreeBootstrapOptions) {
	manifest := mustResolveWorktree(identifier)
	bootstrapResult, err := agentlab.Bootstrap(manifest, agentlab.BootstrapOptions{
		EnvMode:    agentlab.BootstrapMode(opts.EnvMode),
		PythonMode: agentlab.BootstrapMode(opts.PythonMode),
		WebMode:    agentlab.BootstrapMode(opts.WebMode),
	})
	if err != nil {
		log.Fatalf("Failed to bootstrap worktree %s: %v", manifest.Branch, err)
	}

	fmt.Printf("Bootstrapped %s\n", manifest.Branch)
	for _, action := range bootstrapResult.Actions {
		fmt.Printf("  %s\n", action)
	}
}

func runWorktreeDepsUp(identifier string) {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}
	manifest := mustResolveWorktree(identifier)
	manifest, result, err := agentlab.ProvisionDependencies(commonGitDir, manifest)
	if err != nil {
		log.Fatalf("Failed to provision dependencies for %s: %v", manifest.Branch, err)
	}

	fmt.Printf("Provisioned dependencies for %s\n", manifest.Branch)
	for _, action := range result.Actions {
		fmt.Printf("  %s\n", action)
	}
	for _, warning := range manifest.DependencyWarnings() {
		fmt.Printf("  note: %s\n", warning)
	}
}

func runWorktreeDepsStatus(identifier string) {
	manifest := mustResolveWorktree(identifier)
	status, err := agentlab.InspectDependencies(manifest)
	if err != nil {
		log.Fatalf("Failed to inspect dependencies for %s: %v", manifest.Branch, err)
	}

	fmt.Printf("branch: %s\n", manifest.Branch)
	fmt.Printf("mode: %s\n", status.Mode)
	if status.Namespace != "" {
		fmt.Printf("namespace: %s\n", status.Namespace)
	}
	if status.PostgresDatabase != "" {
		fmt.Printf("postgres database: %s (ready=%t tables=%d)\n", status.PostgresDatabase, status.PostgresReady, status.PostgresTableCount)
	}
	if status.RedisPrefix != "" {
		fmt.Printf("redis prefix: %s (ready=%t keys=%d)\n", status.RedisPrefix, status.RedisReady, status.RedisKeyCount)
	}
	if status.FileStoreBucket != "" {
		fmt.Printf("file-store bucket: %s (ready=%t objects=%d)\n", status.FileStoreBucket, status.FileStoreReady, status.FileStoreObjectCount)
	}
	fmt.Printf("search infra: %s\n", status.SearchInfraMode)
	for _, warning := range manifest.DependencyWarnings() {
		fmt.Printf("note: %s\n", warning)
	}
}

func runWorktreeDepsReset(identifier string) {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}
	manifest := mustResolveWorktree(identifier)
	manifest, result, err := agentlab.ResetDependencies(commonGitDir, manifest)
	if err != nil {
		log.Fatalf("Failed to reset dependencies for %s: %v", manifest.Branch, err)
	}

	fmt.Printf("Reset dependencies for %s\n", manifest.Branch)
	for _, action := range result.Actions {
		fmt.Printf("  %s\n", action)
	}
}

func runWorktreeDepsDown(identifier string) {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}
	manifest := mustResolveWorktree(identifier)
	manifest, result, err := agentlab.TeardownDependencies(commonGitDir, manifest)
	if err != nil {
		log.Fatalf("Failed to tear down dependencies for %s: %v", manifest.Branch, err)
	}

	fmt.Printf("Tore down dependencies for %s\n", manifest.Branch)
	for _, action := range result.Actions {
		fmt.Printf("  %s\n", action)
	}
}

func runWorktreeStatus() {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}

	repoRoot, _ := paths.GitRoot()
	current, _, _ := agentlab.FindByRepoRoot(commonGitDir, repoRoot)

	manifests, err := agentlab.LoadAll(commonGitDir)
	if err != nil {
		log.Fatalf("Failed to load worktree manifests: %v", err)
	}

	if len(manifests) == 0 {
		log.Info("No agent-lab worktrees tracked yet.")
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "CURRENT\tBRANCH\tLANE\tMODE\tWEB\tAPI\tPATH"); err != nil {
		log.Fatalf("Failed to write worktree header: %v", err)
	}
	for _, manifest := range manifests {
		marker := ""
		if manifest.ID == current.ID && manifest.ID != "" {
			marker = "*"
		}
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			marker,
			manifest.Branch,
			manifest.ResolvedLane(),
			manifest.ResolvedDependencies().Mode,
			manifest.URLs.Web,
			manifest.URLs.API,
			manifest.CheckoutPath,
		); err != nil {
			log.Fatalf("Failed to write worktree row for %s: %v", manifest.Branch, err)
		}
	}
	_ = tw.Flush()
}

func runWorktreeShow(identifier string) {
	manifest := mustResolveWorktree(identifier)

	fmt.Printf("branch: %s\n", manifest.Branch)
	fmt.Printf("id: %s\n", manifest.ID)
	fmt.Printf("lane: %s\n", manifest.ResolvedLane())
	fmt.Printf("checkout: %s\n", manifest.CheckoutPath)
	fmt.Printf("base-ref: %s\n", manifest.BaseRef)
	fmt.Printf("state-dir: %s\n", manifest.StateDir)
	fmt.Printf("artifacts: %s\n", manifest.ArtifactDir)
	fmt.Printf("backend env: %s\n", manifest.EnvFile)
	fmt.Printf("web env: %s\n", manifest.WebEnvFile)
	fmt.Printf("compose project: %s\n", manifest.ComposeProject)
	fmt.Printf("dependency mode: %s\n", manifest.ResolvedDependencies().Mode)
	if manifest.ResolvedDependencies().Namespace != "" {
		fmt.Printf("dependency namespace: %s\n", manifest.ResolvedDependencies().Namespace)
	}
	if manifest.ResolvedDependencies().PostgresDatabase != "" {
		fmt.Printf("postgres database: %s\n", manifest.ResolvedDependencies().PostgresDatabase)
		fmt.Printf("redis prefix: %s\n", manifest.ResolvedDependencies().RedisPrefix)
		fmt.Printf("file-store bucket: %s\n", manifest.ResolvedDependencies().FileStoreBucket)
	}
	fmt.Printf("search infra: %s\n", manifest.ResolvedDependencies().SearchInfraMode)
	fmt.Printf("web url: %s\n", manifest.URLs.Web)
	fmt.Printf("api url: %s\n", manifest.URLs.API)
	fmt.Printf("mcp url: %s\n", manifest.URLs.MCP)
	fmt.Printf("ports: web=%d api=%d model_server=%d mcp=%d\n", manifest.Ports.Web, manifest.Ports.API, manifest.Ports.ModelServer, manifest.Ports.MCP)
	if manifest.LastVerifiedAt != "" {
		fmt.Printf("last verified: %s\n", manifest.LastVerifiedAt)
	}
	if manifest.LastVerifySummary != "" {
		fmt.Printf("last summary: %s\n", manifest.LastVerifySummary)
	}
	for _, warning := range manifest.DependencyWarnings() {
		fmt.Printf("note: %s\n", warning)
	}
}

func mustResolveWorktree(identifier string) agentlab.Manifest {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}

	if identifier == "" {
		repoRoot, err := paths.GitRoot()
		if err != nil {
			log.Fatalf("Failed to determine git root: %v", err)
		}
		manifest, found, err := agentlab.FindByRepoRoot(commonGitDir, repoRoot)
		if err != nil {
			log.Fatalf("Failed to resolve current worktree manifest: %v", err)
		}
		if !found {
			log.Fatalf("No agent-lab worktree found for %q", identifier)
		}
		return manifest
	}

	manifest, found, err := agentlab.FindByIdentifier(commonGitDir, identifier)
	if err != nil {
		log.Fatalf("Failed to resolve worktree manifest: %v", err)
	}
	if !found {
		log.Fatalf("No agent-lab worktree found for %q", identifier)
	}
	return manifest
}

func runWorktreeRemove(identifier string, opts *WorktreeRemoveOptions) {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}

	manifest, found, err := agentlab.FindByIdentifier(commonGitDir, identifier)
	if err != nil {
		log.Fatalf("Failed to resolve worktree: %v", err)
	}
	if !found {
		log.Fatalf("No agent-lab worktree found for %q", identifier)
	}

	if opts.DropDeps {
		var teardownResult *agentlab.DependencyResult
		manifest, teardownResult, err = agentlab.TeardownDependencies(commonGitDir, manifest)
		if err != nil {
			log.Fatalf("Failed to tear down worktree dependencies: %v", err)
		}
		for _, action := range teardownResult.Actions {
			fmt.Printf("  deps: %s\n", action)
		}
	}

	args := []string{"worktree", "remove"}
	if opts.Force {
		args = append(args, "--force")
	}
	args = append(args, manifest.CheckoutPath)

	log.Infof("Removing worktree %s", manifest.Branch)
	gitCmd := exec.Command("git", args...)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	gitCmd.Stdin = os.Stdin
	if err := gitCmd.Run(); err != nil {
		if opts.Force && isOrphanedWorktree(manifest.CheckoutPath) {
			log.Warnf("git detached %s but left an orphaned checkout behind; removing %s", manifest.Branch, manifest.CheckoutPath)
			if removeErr := os.RemoveAll(manifest.CheckoutPath); removeErr != nil {
				log.Fatalf("git worktree remove failed: %v (fallback cleanup failed: %v)", err, removeErr)
			}
		} else {
			log.Fatalf("git worktree remove failed: %v", err)
		}
	}

	if err := agentlab.RemoveState(commonGitDir, manifest.ID); err != nil {
		log.Fatalf("Failed to remove worktree state: %v", err)
	}

	fmt.Printf("Removed agent-lab worktree %s\n", manifest.Branch)
	if manifest.ResolvedDependencies().Mode == agentlab.DependencyModeNamespaced && !opts.DropDeps {
		fmt.Printf("  note: namespaced Postgres/Redis/MinIO state was left in place. Use `ods worktree deps down %s` before removal if you want cleanup.\n", manifest.Branch)
	}
}

func isOrphanedWorktree(checkoutPath string) bool {
	output, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err == nil && strings.Contains(string(output), "worktree "+checkoutPath+"\n") {
		return false
	}
	if _, statErr := os.Stat(checkoutPath); os.IsNotExist(statErr) {
		return true
	}
	if statusErr := exec.Command("git", "-C", checkoutPath, "status", "--short").Run(); statusErr != nil {
		return true
	}
	return false
}
