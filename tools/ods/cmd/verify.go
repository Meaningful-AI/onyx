package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentlab"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/envutil"
)

type VerifyOptions struct {
	BaseRef           string
	SkipAgentCheck    bool
	Worktree          string
	PytestPaths       []string
	PlaywrightPaths   []string
	PlaywrightGrep    string
	PlaywrightProject string
}

type VerifySummary struct {
	GeneratedAt string              `json:"generated_at"`
	RepoRoot    string              `json:"repo_root"`
	Worktree    *agentlab.Manifest  `json:"worktree,omitempty"`
	Steps       []VerifyStepSummary `json:"steps"`
}

type VerifyStepSummary struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Command     []string `json:"command,omitempty"`
	DurationMS  int64    `json:"duration_ms"`
	LogPath     string   `json:"log_path,omitempty"`
	ArtifactDir string   `json:"artifact_dir,omitempty"`
	Details     []string `json:"details,omitempty"`
}

// NewVerifyCommand creates the verify command.
func NewVerifyCommand() *cobra.Command {
	opts := &VerifyOptions{}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Run the agent-lab verification ladder and write a machine-readable summary",
		Long: `Run the agent-lab verification ladder for the current checkout.

This command composes the diff-based agent-check with optional pytest and
Playwright execution, then writes a JSON summary into the worktree artifact
directory so agents can inspect the result without re-parsing console output.

Use --worktree to run the same flow against a tracked target worktree from the
agent-lab control checkout.`,
		Run: func(cmd *cobra.Command, args []string) {
			runVerify(opts)
		},
	}

	cmd.Flags().StringVar(&opts.BaseRef, "base-ref", "", "git ref to compare against for agent-check (defaults to the worktree base ref or HEAD)")
	cmd.Flags().BoolVar(&opts.SkipAgentCheck, "skip-agent-check", false, "skip the diff-based agent-check step")
	cmd.Flags().StringVar(&opts.Worktree, "worktree", "", "tracked agent-lab worktree to verify from instead of the current checkout")
	cmd.Flags().StringArrayVar(&opts.PytestPaths, "pytest", nil, "pytest path or node id to run (repeatable)")
	cmd.Flags().StringArrayVar(&opts.PlaywrightPaths, "playwright", nil, "Playwright test path to run (repeatable)")
	cmd.Flags().StringVar(&opts.PlaywrightGrep, "playwright-grep", "", "grep passed through to Playwright")
	cmd.Flags().StringVar(&opts.PlaywrightProject, "playwright-project", "", "Playwright project to run")

	return cmd
}

func runVerify(opts *VerifyOptions) {
	repoRoot, manifest, hasManifest := resolveAgentLabTarget(opts.Worktree)

	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}

	runAt := time.Now().UTC()
	artifactRoot := filepath.Join(repoRoot, "web", "output")
	if hasManifest {
		artifactRoot = filepath.Join(manifest.ArtifactDir, "verify", runAt.Format("20060102-150405"))
	}
	if err := os.MkdirAll(artifactRoot, 0755); err != nil {
		log.Fatalf("Failed to create verify artifact dir: %v", err)
	}

	summary := VerifySummary{
		GeneratedAt: runAt.Format(time.RFC3339),
		RepoRoot:    repoRoot,
		Steps:       make([]VerifyStepSummary, 0, 3),
	}
	if hasManifest {
		manifestCopy := manifest
		summary.Worktree = &manifestCopy
	}

	if !opts.SkipAgentCheck {
		baseRef := opts.BaseRef
		if baseRef == "" && hasManifest {
			baseRef = manifest.BaseRef
		}
		if baseRef == "" {
			baseRef = "HEAD"
		}

		step, passed := runAgentCheckVerifyStep(repoRoot, opts.Worktree, baseRef)
		summary.Steps = append(summary.Steps, step)
		if !passed {
			writeVerifySummary(summary, artifactRoot, commonGitDir, manifest, hasManifest, runAt)
			os.Exit(1)
		}
	}

	if len(opts.PytestPaths) > 0 {
		step, passed := runPytestVerifyStep(repoRoot, artifactRoot, manifest, hasManifest, opts.PytestPaths)
		summary.Steps = append(summary.Steps, step)
		if !passed {
			writeVerifySummary(summary, artifactRoot, commonGitDir, manifest, hasManifest, runAt)
			os.Exit(1)
		}
	}

	if len(opts.PlaywrightPaths) > 0 || opts.PlaywrightGrep != "" {
		step, passed := runPlaywrightVerifyStep(repoRoot, artifactRoot, manifest, hasManifest, opts)
		summary.Steps = append(summary.Steps, step)
		if !passed {
			writeVerifySummary(summary, artifactRoot, commonGitDir, manifest, hasManifest, runAt)
			os.Exit(1)
		}
	}

	writeVerifySummary(summary, artifactRoot, commonGitDir, manifest, hasManifest, runAt)
	log.Infof("Verification summary written to %s", filepath.Join(artifactRoot, "summary.json"))
}

func runAgentCheckVerifyStep(repoRoot, worktree, baseRef string) (VerifyStepSummary, bool) {
	startedAt := time.Now()
	opts := &AgentCheckOptions{BaseRef: baseRef, Worktree: worktree, RepoRoot: repoRoot}
	result, err := evaluateAgentCheck(opts, nil)

	step := VerifyStepSummary{
		Name:       "agent-check",
		Command:    []string{"ods", "agent-check", "--base-ref", baseRef},
		DurationMS: time.Since(startedAt).Milliseconds(),
	}
	if worktree != "" {
		step.Command = append(step.Command, "--worktree", worktree)
	}
	if err != nil {
		step.Status = "failed"
		step.Details = []string{err.Error()}
		return step, false
	}

	if len(result.Violations) == 0 && len(result.DocViolations) == 0 {
		step.Status = "passed"
		return step, true
	}

	step.Status = "failed"
	for _, violation := range result.Violations {
		step.Details = append(step.Details, fmt.Sprintf("%s:%d [%s] %s", violation.Path, violation.LineNum, violation.RuleID, violation.Message))
	}
	for _, violation := range result.DocViolations {
		step.Details = append(step.Details, fmt.Sprintf("%s [agent-docs] %s", violation.Path, violation.Message))
	}
	return step, false
}

func runPytestVerifyStep(repoRoot, artifactRoot string, manifest agentlab.Manifest, hasManifest bool, pytestPaths []string) (VerifyStepSummary, bool) {
	pythonExecutable := pythonForRepo(repoRoot)
	args := append([]string{"-m", "dotenv", "-f", ".vscode/.env", "run", "--", "pytest"}, pytestPaths...)
	extraEnv := map[string]string{}
	if hasManifest {
		for key, value := range manifest.RuntimeEnv() {
			extraEnv[key] = value
		}
	}

	step, passed := runLoggedCommand(
		"pytest",
		filepath.Join(artifactRoot, "pytest.log"),
		filepath.Join(repoRoot, "backend"),
		extraEnv,
		pythonExecutable,
		args...,
	)
	if hasManifest {
		step.Details = append(step.Details, fmt.Sprintf("dependency mode: %s", manifest.ResolvedDependencies().Mode))
		step.Details = append(step.Details, fmt.Sprintf("search infra: %s", manifest.ResolvedDependencies().SearchInfraMode))
	}
	return step, passed
}

func runPlaywrightVerifyStep(repoRoot, artifactRoot string, manifest agentlab.Manifest, hasManifest bool, opts *VerifyOptions) (VerifyStepSummary, bool) {
	args := []string{"playwright", "test"}
	args = append(args, opts.PlaywrightPaths...)
	if opts.PlaywrightGrep != "" {
		args = append(args, "--grep", opts.PlaywrightGrep)
	}
	if opts.PlaywrightProject != "" {
		args = append(args, "--project", opts.PlaywrightProject)
	}

	extraEnv := map[string]string{}
	if hasManifest {
		for key, value := range manifest.RuntimeEnv() {
			extraEnv[key] = value
		}
	}

	step, passed := runLoggedCommand(
		"playwright",
		filepath.Join(artifactRoot, "playwright.log"),
		filepath.Join(repoRoot, "web"),
		extraEnv,
		"npx",
		args...,
	)
	step.ArtifactDir = filepath.Join(repoRoot, "web", "output")
	if hasManifest {
		step.Details = append(step.Details, fmt.Sprintf("base url: %s", manifest.URLs.Web))
		step.Details = append(step.Details, fmt.Sprintf("dependency mode: %s", manifest.ResolvedDependencies().Mode))
		step.Details = append(step.Details, fmt.Sprintf("search infra: %s", manifest.ResolvedDependencies().SearchInfraMode))
		step.Details = append(step.Details, fmt.Sprintf("reuse Chrome DevTools MCP against %s for interactive browser validation", manifest.URLs.Web))
		step.Details = append(step.Details, manifest.DependencyWarnings()...)
	}
	return step, passed
}

func runLoggedCommand(name, logPath, workdir string, extraEnv map[string]string, executable string, args ...string) (VerifyStepSummary, bool) {
	startedAt := time.Now()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return VerifyStepSummary{
			Name:       name,
			Status:     "failed",
			DurationMS: time.Since(startedAt).Milliseconds(),
			Details:    []string{fmt.Sprintf("create log dir: %v", err)},
		}, false
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return VerifyStepSummary{
			Name:       name,
			Status:     "failed",
			DurationMS: time.Since(startedAt).Milliseconds(),
			Details:    []string{fmt.Sprintf("create log file: %v", err)},
		}, false
	}
	defer func() { _ = logFile.Close() }()

	cmd := exec.Command(executable, args...)
	cmd.Dir = workdir
	cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
	cmd.Stderr = io.MultiWriter(os.Stderr, logFile)
	if len(extraEnv) > 0 {
		cmd.Env = envutil.ApplyOverrides(os.Environ(), extraEnv)
	}

	step := VerifyStepSummary{
		Name:       name,
		Command:    append([]string{executable}, args...),
		LogPath:    logPath,
		DurationMS: 0,
	}

	err = cmd.Run()
	step.DurationMS = time.Since(startedAt).Milliseconds()
	if err != nil {
		step.Status = "failed"
		step.Details = []string{err.Error()}
		return step, false
	}

	step.Status = "passed"
	return step, true
}

func writeVerifySummary(summary VerifySummary, artifactRoot, commonGitDir string, manifest agentlab.Manifest, hasManifest bool, runAt time.Time) {
	summaryPath := filepath.Join(artifactRoot, "summary.json")
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode verify summary: %v", err)
	}
	if err := os.WriteFile(summaryPath, data, 0644); err != nil {
		log.Fatalf("Failed to write verify summary: %v", err)
	}

	if hasManifest {
		if err := agentlab.UpdateVerification(commonGitDir, manifest, summaryPath, runAt); err != nil {
			log.Warnf("Failed to update worktree verification metadata: %v", err)
		}
	}
}

func pythonForRepo(repoRoot string) string {
	candidate := filepath.Join(repoRoot, ".venv", "bin", "python")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	if manifest, found := currentAgentLabManifest(repoRoot); found {
		sharedCandidate := filepath.Join(manifest.CreatedFromPath, ".venv", "bin", "python")
		if _, err := os.Stat(sharedCandidate); err == nil {
			return sharedCandidate
		}
	}

	return "python"
}
