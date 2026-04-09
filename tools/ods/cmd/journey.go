package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentlab"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/git"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/journey"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/paths"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/s3"
)

const defaultJourneyHTTPRegion = "us-east-2"

type JourneyRunOptions struct {
	Journey   string
	Label     string
	Worktree  string
	OutputDir string
	Project   string
}

type JourneyCompareOptions struct {
	Journeys       []string
	PlanFile       string
	BeforeRef      string
	AfterRef       string
	AfterWorktree  string
	DependencyMode string
	PR             string
	KeepWorktrees  bool
	Bucket         string
}

type JourneyPublishOptions struct {
	RunDir string
	PR     string
	Bucket string
}

type JourneyCaptureSummary struct {
	Journey      string   `json:"journey"`
	Label        string   `json:"label"`
	Worktree     string   `json:"worktree,omitempty"`
	URL          string   `json:"url"`
	ArtifactDir  string   `json:"artifact_dir"`
	LogPath      string   `json:"log_path"`
	VideoFiles   []string `json:"video_files,omitempty"`
	TraceFiles   []string `json:"trace_files,omitempty"`
	Screenshots  []string `json:"screenshots,omitempty"`
	MetadataJSON []string `json:"metadata_json,omitempty"`
}

type JourneyCompareSummary struct {
	GeneratedAt string                  `json:"generated_at"`
	BeforeRef   string                  `json:"before_ref"`
	AfterRef    string                  `json:"after_ref"`
	RunDir      string                  `json:"run_dir"`
	S3Prefix    string                  `json:"s3_prefix,omitempty"`
	S3HTTPBase  string                  `json:"s3_http_base,omitempty"`
	Captures    []JourneyCaptureSummary `json:"captures"`
}

type managedProcess struct {
	Name    string
	Cmd     *exec.Cmd
	LogPath string
}

// NewJourneyCommand creates the journey command surface.
func NewJourneyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journey",
		Short: "Capture before/after browser journeys as agent artifacts",
	}

	cmd.AddCommand(newJourneyListCommand())
	cmd.AddCommand(newJourneyRunCommand())
	cmd.AddCommand(newJourneyCompareCommand())
	cmd.AddCommand(newJourneyPublishCommand())

	return cmd
}

func newJourneyListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered browser journeys",
		Run: func(cmd *cobra.Command, args []string) {
			runJourneyList()
		},
	}
}

func newJourneyRunCommand() *cobra.Command {
	opts := &JourneyRunOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a single registered journey against the current or target worktree",
		Long: `Run one registered journey against the current checkout or a tracked worktree.

This is the default before/after workflow for product changes:
  1. capture --label before in the target worktree before editing
  2. implement and validate the change in that same worktree
  3. capture --label after in that same worktree

Use journey compare only when you need to recover a missed baseline or compare
two explicit revisions after the fact.`,
		Run: func(cmd *cobra.Command, args []string) {
			runJourneyRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Journey, "journey", "", "registered journey name to run")
	cmd.Flags().StringVar(&opts.Label, "label", "after", "artifact label for this capture (for example before or after)")
	cmd.Flags().StringVar(&opts.Worktree, "worktree", "", "tracked agent-lab worktree to run from instead of the current checkout")
	cmd.Flags().StringVar(&opts.OutputDir, "output-dir", "", "explicit artifact directory for the capture")
	cmd.Flags().StringVar(&opts.Project, "project", "", "override the Playwright project from the journey registry")
	_ = cmd.MarkFlagRequired("journey")

	return cmd
}

func newJourneyCompareCommand() *cobra.Command {
	opts := &JourneyCompareOptions{}

	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Capture before and after videos by replaying registered journeys against two revisions",
		Long: `Create or reuse worktrees for the before and after revisions, boot the app in each one,
record the configured journeys, and write a machine-readable summary. If --pr is supplied,
the compare run is also uploaded to S3 and linked from the pull request.

This is the fallback path, not the default workflow. Prefer journey run inside a
single tracked product worktree when you can capture before and after during the
normal edit loop.`,
		Run: func(cmd *cobra.Command, args []string) {
			runJourneyCompare(opts)
		},
	}

	cmd.Flags().StringArrayVar(&opts.Journeys, "journey", nil, "registered journey name to capture (repeatable)")
	cmd.Flags().StringVar(&opts.PlanFile, "plan-file", "", "JSON file containing {\"journeys\":[...]} (defaults to .github/agent-journeys.json when present)")
	cmd.Flags().StringVar(&opts.BeforeRef, "before-ref", "origin/main", "git ref for the before capture")
	cmd.Flags().StringVar(&opts.AfterRef, "after-ref", "HEAD", "git ref for the after capture when --after-worktree is not supplied")
	cmd.Flags().StringVar(&opts.AfterWorktree, "after-worktree", "", "existing tracked worktree to use for the after capture")
	cmd.Flags().StringVar(&opts.DependencyMode, "dependency-mode", string(agentlab.DependencyModeNamespaced), "dependency mode for temporary worktrees: namespaced or shared")
	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number to upload/comment against after capture")
	cmd.Flags().StringVar(&opts.Bucket, "bucket", "", "override the S3 bucket used for uploaded journey artifacts")
	cmd.Flags().BoolVar(&opts.KeepWorktrees, "keep-worktrees", false, "keep temporary journey worktrees after the capture run")

	return cmd
}

func newJourneyPublishCommand() *cobra.Command {
	opts := &JourneyPublishOptions{}

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Upload a previously captured compare run and update the pull request comment",
		Run: func(cmd *cobra.Command, args []string) {
			runJourneyPublish(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RunDir, "run-dir", "", "compare run directory containing summary.json")
	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number to publish against")
	cmd.Flags().StringVar(&opts.Bucket, "bucket", "", "override the S3 bucket used for uploaded journey artifacts")
	_ = cmd.MarkFlagRequired("run-dir")

	return cmd
}

func runJourneyList() {
	repoRoot, err := paths.GitRoot()
	if err != nil {
		log.Fatalf("Failed to determine git root: %v", err)
	}

	registry, err := journey.LoadRegistry(repoRoot)
	if err != nil {
		log.Fatalf("Failed to load journey registry: %v", err)
	}

	for _, definition := range registry.Journeys {
		fmt.Printf("%s\t%s\tproject=%s\tmodel_server=%t\n", definition.Name, definition.Description, definition.Project, definition.RequiresModelServer)
	}
}

func runJourneyRun(opts *JourneyRunOptions) {
	repoRoot, manifest, hasManifest := resolveAgentLabTarget(opts.Worktree)
	harnessRoot, err := resolveJourneyHarnessRoot(repoRoot, manifest, hasManifest)
	if err != nil {
		log.Fatalf("Failed to resolve journey harness root: %v", err)
	}
	capture, err := captureJourney(harnessRoot, repoRoot, manifest, hasManifest, opts.Journey, opts.Label, opts.OutputDir, opts.Project)
	if err != nil {
		log.Fatalf("Journey capture failed: %v", err)
	}

	summaryPath := filepath.Join(capture.ArtifactDir, "summary.json")
	data, err := json.MarshalIndent(capture, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode journey summary: %v", err)
	}
	if err := os.WriteFile(summaryPath, data, 0644); err != nil {
		log.Fatalf("Failed to write journey summary: %v", err)
	}

	log.Infof("Journey %s (%s) captured to %s", capture.Journey, capture.Label, capture.ArtifactDir)
}

func runJourneyCompare(opts *JourneyCompareOptions) {
	repoRoot, err := paths.GitRoot()
	if err != nil {
		log.Fatalf("Failed to determine git root: %v", err)
	}

	definitions, err := resolveJourneyDefinitions(repoRoot, opts.Journeys, opts.PlanFile)
	if err != nil {
		log.Fatalf("Failed to resolve journeys: %v", err)
	}

	currentRoot, currentManifest, hasCurrentManifest := resolveAgentLabTarget("")
	if opts.AfterWorktree == "" && strings.EqualFold(strings.TrimSpace(opts.AfterRef), "HEAD") && !hasCurrentManifest && git.HasUncommittedChanges() {
		log.Fatalf("The current checkout has uncommitted changes, but it is not a tracked agent-lab worktree. Create the product worktree first and rerun with --after-worktree <branch> so the after capture reflects the real patch.")
	}
	_ = currentRoot

	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}

	runDir := filepath.Join(agentlab.StateRoot(commonGitDir), "journeys", time.Now().UTC().Format("20060102-150405"))
	if err := os.MkdirAll(runDir, 0755); err != nil {
		log.Fatalf("Failed to create journey run dir: %v", err)
	}

	beforeTarget, err := createTemporaryJourneyWorktree(opts.BeforeRef, "before", agentlab.DependencyMode(opts.DependencyMode))
	if err != nil {
		log.Fatalf("Failed to create before worktree: %v", err)
	}
	if !opts.KeepWorktrees {
		defer cleanupJourneyTarget(beforeTarget)
	}

	var afterTarget journeyTarget
	if opts.AfterWorktree != "" {
		afterTarget, err = resolveJourneyTarget(opts.AfterWorktree)
		if err != nil {
			log.Fatalf("Failed to resolve after worktree: %v", err)
		}
		if err := runSelfCommand("worktree", "deps", "up", afterTarget.Identifier); err != nil {
			log.Fatalf("Failed to provision dependencies for %s: %v", afterTarget.Identifier, err)
		}
	} else if strings.EqualFold(strings.TrimSpace(opts.AfterRef), "HEAD") {
		if hasCurrentManifest {
			afterTarget = journeyTarget{
				Identifier: currentManifest.Branch,
				Manifest:   currentManifest,
			}
			if err := runSelfCommand("worktree", "deps", "up", afterTarget.Identifier); err != nil {
				log.Fatalf("Failed to provision dependencies for %s: %v", afterTarget.Identifier, err)
			}
			log.Infof("Using current tracked worktree %s for the after capture", afterTarget.Identifier)
		} else {
			afterTarget, err = createTemporaryJourneyWorktree(opts.AfterRef, "after", agentlab.DependencyMode(opts.DependencyMode))
			if err != nil {
				log.Fatalf("Failed to create after worktree: %v", err)
			}
			if !opts.KeepWorktrees {
				defer cleanupJourneyTarget(afterTarget)
			}
		}
	} else {
		afterTarget, err = createTemporaryJourneyWorktree(opts.AfterRef, "after", agentlab.DependencyMode(opts.DependencyMode))
		if err != nil {
			log.Fatalf("Failed to create after worktree: %v", err)
		}
		if !opts.KeepWorktrees {
			defer cleanupJourneyTarget(afterTarget)
		}
	}

	summary := JourneyCompareSummary{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		BeforeRef:   opts.BeforeRef,
		AfterRef:    opts.AfterRef,
		RunDir:      runDir,
		Captures:    []JourneyCaptureSummary{},
	}

	beforeCaptures, err := captureJourneySet(beforeTarget, definitions, "before", runDir)
	if err != nil {
		log.Fatalf("Before capture failed: %v", err)
	}
	summary.Captures = append(summary.Captures, beforeCaptures...)

	afterCaptures, err := captureJourneySet(afterTarget, definitions, "after", runDir)
	if err != nil {
		log.Fatalf("After capture failed: %v", err)
	}
	summary.Captures = append(summary.Captures, afterCaptures...)

	writeJourneyCompareSummary(runDir, summary)
	log.Infof("Journey compare summary written to %s", filepath.Join(runDir, "summary.json"))

	if opts.PR != "" {
		prNumber, err := resolvePRNumber(opts.PR)
		if err != nil {
			log.Fatalf("Failed to resolve PR number: %v", err)
		}
		updated, err := publishJourneyCompare(runDir, prNumber, opts.Bucket)
		if err != nil {
			log.Fatalf("Failed to publish journey compare run: %v", err)
		}
		writeJourneyCompareSummary(runDir, updated)
	}
}

func runJourneyPublish(opts *JourneyPublishOptions) {
	prNumber, err := resolvePRNumber(opts.PR)
	if err != nil {
		log.Fatalf("Failed to resolve PR number: %v", err)
	}
	updated, err := publishJourneyCompare(opts.RunDir, prNumber, opts.Bucket)
	if err != nil {
		log.Fatalf("Failed to publish journey compare run: %v", err)
	}
	writeJourneyCompareSummary(opts.RunDir, updated)
	log.Infof("Published journey compare run from %s", opts.RunDir)
}

func resolveJourneyDefinitions(repoRoot string, requested []string, planFile string) ([]journey.Definition, error) {
	journeyNames := append([]string{}, requested...)
	resolvedPlan := strings.TrimSpace(planFile)
	if resolvedPlan == "" {
		defaultPlan := filepath.Join(repoRoot, journey.DefaultPlanPath)
		if _, err := os.Stat(defaultPlan); err == nil {
			resolvedPlan = defaultPlan
		}
	}
	if resolvedPlan != "" {
		plan, err := journey.LoadPlan(resolvedPlan)
		if err != nil {
			return nil, err
		}
		journeyNames = append(journeyNames, plan.Journeys...)
	}
	if len(journeyNames) == 0 {
		return nil, fmt.Errorf("no journeys requested; pass --journey or provide %s", journey.DefaultPlanPath)
	}

	seen := map[string]bool{}
	deduped := make([]string, 0, len(journeyNames))
	for _, name := range journeyNames {
		if !seen[name] {
			seen[name] = true
			deduped = append(deduped, name)
		}
	}
	return journey.ResolveDefinitions(repoRoot, deduped)
}

func resolveJourneyHarnessRoot(targetRepoRoot string, manifest agentlab.Manifest, hasManifest bool) (string, error) {
	candidates := []string{targetRepoRoot}
	if hasManifest && manifest.CreatedFromPath != "" {
		candidates = append([]string{manifest.CreatedFromPath}, candidates...)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, journey.RegistryPath)); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no journey registry found in target repo %s or control checkout %s", targetRepoRoot, manifest.CreatedFromPath)
}

func captureJourney(harnessRoot, targetRepoRoot string, manifest agentlab.Manifest, hasManifest bool, journeyName, label, outputDir, projectOverride string) (JourneyCaptureSummary, error) {
	definitions, err := journey.ResolveDefinitions(harnessRoot, []string{journeyName})
	if err != nil {
		return JourneyCaptureSummary{}, err
	}
	definition := definitions[0]

	targetDir := strings.TrimSpace(outputDir)
	if targetDir == "" {
		if hasManifest {
			targetDir = filepath.Join(manifest.ArtifactDir, "journeys", journey.Slug(definition.Name), journey.Slug(label))
		} else {
			targetDir = filepath.Join(targetRepoRoot, "web", "output", "journeys", journey.Slug(definition.Name), journey.Slug(label))
		}
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return JourneyCaptureSummary{}, fmt.Errorf("create journey artifact dir: %w", err)
	}

	playwrightOutputDir := filepath.Join(targetDir, "playwright")
	logPath := filepath.Join(targetDir, "journey.log")

	projectName := definition.Project
	if strings.TrimSpace(projectOverride) != "" {
		projectName = projectOverride
	}

	envOverrides := map[string]string{
		"PLAYWRIGHT_JOURNEY_MODE":        "1",
		"PLAYWRIGHT_JOURNEY_CAPTURE_DIR": targetDir,
		"PLAYWRIGHT_OUTPUT_DIR":          playwrightOutputDir,
	}
	if definition.SkipGlobalSetup {
		envOverrides["PLAYWRIGHT_SKIP_GLOBAL_SETUP"] = "1"
	}
	if hasManifest {
		for key, value := range manifest.RuntimeEnv() {
			envOverrides[key] = value
		}
	}

	step, passed := runLoggedCommand(
		"journey-"+definition.Name,
		logPath,
		filepath.Join(harnessRoot, "web"),
		envOverrides,
		"npx",
		"playwright", "test", definition.TestPath, "--project", projectName,
	)
	if !passed {
		return JourneyCaptureSummary{}, fmt.Errorf("%s", strings.Join(step.Details, "\n"))
	}

	artifactSummary, err := summarizeJourneyArtifacts(targetDir)
	if err != nil {
		return JourneyCaptureSummary{}, err
	}
	artifactSummary.Journey = definition.Name
	artifactSummary.Label = label
	artifactSummary.ArtifactDir = targetDir
	artifactSummary.LogPath = logPath
	if hasManifest {
		artifactSummary.Worktree = manifest.Branch
		artifactSummary.URL = manifest.URLs.Web
	} else {
		artifactSummary.URL = envOverrides["BASE_URL"]
	}
	return artifactSummary, nil
}

type journeyTarget struct {
	Identifier string
	Manifest   agentlab.Manifest
	Temporary  bool
}

func resolveJourneyTarget(identifier string) (journeyTarget, error) {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		return journeyTarget{}, err
	}
	manifest, found, err := agentlab.FindByIdentifier(commonGitDir, identifier)
	if err != nil {
		return journeyTarget{}, err
	}
	if !found {
		return journeyTarget{}, fmt.Errorf("no worktree found for %q", identifier)
	}
	return journeyTarget{Identifier: manifest.Branch, Manifest: manifest}, nil
}

func createTemporaryJourneyWorktree(ref, label string, mode agentlab.DependencyMode) (journeyTarget, error) {
	branch := fmt.Sprintf("codex/journey-%s-%s-%d", journey.Slug(label), journey.Slug(strings.ReplaceAll(ref, "/", "-")), time.Now().UTC().UnixNano())
	if err := runSelfCommand("worktree", "create", branch, "--from", ref, "--dependency-mode", string(mode)); err != nil {
		return journeyTarget{}, err
	}
	if err := runSelfCommand("worktree", "deps", "up", branch); err != nil {
		return journeyTarget{}, err
	}
	target, err := resolveJourneyTarget(branch)
	if err != nil {
		return journeyTarget{}, err
	}
	target.Temporary = true
	return target, nil
}

func cleanupJourneyTarget(target journeyTarget) {
	if !target.Temporary {
		return
	}
	if err := runSelfCommand("worktree", "remove", target.Identifier, "--force", "--drop-deps"); err != nil {
		log.Warnf("Failed to remove temporary worktree %s: %v", target.Identifier, err)
	}
	if err := exec.Command("git", "branch", "-D", target.Identifier).Run(); err != nil {
		log.Warnf("Failed to delete temporary branch %s: %v", target.Identifier, err)
	}
}

func captureJourneySet(target journeyTarget, definitions []journey.Definition, label, runDir string) ([]JourneyCaptureSummary, error) {
	harnessRoot, err := resolveJourneyHarnessRoot(target.Manifest.CheckoutPath, target.Manifest, true)
	if err != nil {
		return nil, err
	}
	requiresModelServer := false
	for _, definition := range definitions {
		if definition.RequiresModelServer {
			requiresModelServer = true
			break
		}
	}

	processes, err := startJourneyServices(target, runDir, requiresModelServer)
	if err != nil {
		return nil, err
	}
	defer stopManagedProcesses(processes)

	captures := make([]JourneyCaptureSummary, 0, len(definitions))
	for _, definition := range definitions {
		outputDir := filepath.Join(runDir, journey.Slug(definition.Name), journey.Slug(label))
		capture, err := captureJourney(harnessRoot, target.Manifest.CheckoutPath, target.Manifest, true, definition.Name, label, outputDir, "")
		if err != nil {
			return nil, err
		}
		captures = append(captures, capture)
	}
	return captures, nil
}

func startJourneyServices(target journeyTarget, runDir string, includeModelServer bool) ([]managedProcess, error) {
	logDir := filepath.Join(runDir, "services", journey.Slug(target.Manifest.Branch))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create service log dir: %w", err)
	}

	processes := make([]managedProcess, 0, 3)

	apiProcess, err := startManagedProcess(
		"api",
		filepath.Join(logDir, "api.log"),
		"backend", "api", "--worktree", target.Identifier,
	)
	if err != nil {
		return nil, err
	}
	processes = append(processes, apiProcess)

	if includeModelServer {
		modelProcess, err := startManagedProcess(
			"model_server",
			filepath.Join(logDir, "model_server.log"),
			"backend", "model_server", "--worktree", target.Identifier,
		)
		if err != nil {
			stopManagedProcesses(processes)
			return nil, err
		}
		processes = append(processes, modelProcess)
	}

	webProcess, err := startManagedProcess(
		"web",
		filepath.Join(logDir, "web.log"),
		"web", "dev", "--worktree", target.Identifier, "--", "--webpack",
	)
	if err != nil {
		stopManagedProcesses(processes)
		return nil, err
	}
	processes = append(processes, webProcess)

	if err := waitForJourneyURL(target.Manifest.URLs.API+"/health", 2*time.Minute, processes...); err != nil {
		stopManagedProcesses(processes)
		return nil, err
	}
	if err := waitForJourneyURL(target.Manifest.URLs.Web+"/api/health", 3*time.Minute, processes...); err != nil {
		stopManagedProcesses(processes)
		return nil, err
	}
	return processes, nil
}

func startManagedProcess(name, logPath string, args ...string) (managedProcess, error) {
	executable, err := os.Executable()
	if err != nil {
		return managedProcess{}, fmt.Errorf("determine ods executable: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return managedProcess{}, fmt.Errorf("create log dir: %w", err)
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return managedProcess{}, fmt.Errorf("create log file: %w", err)
	}

	cmd := exec.Command(executable, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return managedProcess{}, fmt.Errorf("start %s: %w", name, err)
	}
	_ = logFile.Close()

	return managedProcess{Name: name, Cmd: cmd, LogPath: logPath}, nil
}

func stopManagedProcesses(processes []managedProcess) {
	for i := len(processes) - 1; i >= 0; i-- {
		process := processes[i]
		if process.Cmd == nil || process.Cmd.Process == nil {
			continue
		}
		_ = process.Cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func(cmd *exec.Cmd) {
			_, _ = cmd.Process.Wait()
			close(done)
		}(process.Cmd)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = process.Cmd.Process.Kill()
		}
	}
}

func waitForJourneyURL(url string, timeout time.Duration, processes ...managedProcess) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := ensureManagedProcessesRunning(processes); err != nil {
			return fmt.Errorf("while waiting for %s: %w", url, err)
		}
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(3 * time.Second)
	}
	if err := ensureManagedProcessesRunning(processes); err != nil {
		return fmt.Errorf("while waiting for %s: %w", url, err)
	}
	return fmt.Errorf("timed out waiting for %s", url)
}

func ensureManagedProcessesRunning(processes []managedProcess) error {
	for _, process := range processes {
		if process.Cmd == nil || process.Cmd.Process == nil {
			continue
		}
		if err := syscall.Kill(process.Cmd.Process.Pid, 0); err != nil {
			if err == syscall.ESRCH {
				return fmt.Errorf("%s exited early\n%s", process.Name, readJourneyLogTail(process.LogPath, 40))
			}
			if err != syscall.EPERM {
				return fmt.Errorf("check %s process health: %w", process.Name, err)
			}
		}
	}
	return nil
}

func readJourneyLogTail(path string, lineCount int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("failed to read %s: %v", path, err)
	}
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return fmt.Sprintf("%s is empty", path)
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > lineCount {
		lines = lines[len(lines)-lineCount:]
	}
	return fmt.Sprintf("recent log tail from %s:\n%s", path, strings.Join(lines, "\n"))
}

func summarizeJourneyArtifacts(root string) (JourneyCaptureSummary, error) {
	summary := JourneyCaptureSummary{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		switch {
		case strings.HasSuffix(path, ".webm"):
			summary.VideoFiles = append(summary.VideoFiles, relative)
		case strings.HasSuffix(path, "trace.zip"):
			summary.TraceFiles = append(summary.TraceFiles, relative)
		case strings.HasSuffix(path, ".png"):
			summary.Screenshots = append(summary.Screenshots, relative)
		case strings.HasSuffix(path, ".json") && filepath.Base(path) != "summary.json":
			summary.MetadataJSON = append(summary.MetadataJSON, relative)
		}
		return nil
	})
	if err != nil {
		return summary, fmt.Errorf("walk journey artifacts: %w", err)
	}
	sort.Strings(summary.VideoFiles)
	sort.Strings(summary.TraceFiles)
	sort.Strings(summary.Screenshots)
	sort.Strings(summary.MetadataJSON)
	return summary, nil
}

func runSelfCommand(args ...string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func writeJourneyCompareSummary(runDir string, summary JourneyCompareSummary) {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode journey compare summary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), data, 0644); err != nil {
		log.Fatalf("Failed to write journey compare summary: %v", err)
	}
}

func publishJourneyCompare(runDir, prNumber, bucketOverride string) (JourneyCompareSummary, error) {
	var summary JourneyCompareSummary
	data, err := os.ReadFile(filepath.Join(runDir, "summary.json"))
	if err != nil {
		return summary, fmt.Errorf("read compare summary: %w", err)
	}
	if err := json.Unmarshal(data, &summary); err != nil {
		return summary, fmt.Errorf("parse compare summary: %w", err)
	}

	bucket := bucketOverride
	if bucket == "" {
		bucket = getS3Bucket()
	}

	timestamp := filepath.Base(runDir)
	s3Prefix := fmt.Sprintf("s3://%s/journeys/pr-%s/%s/", bucket, prNumber, timestamp)
	if err := s3.SyncUp(runDir, s3Prefix, true); err != nil {
		return summary, err
	}

	httpBase := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/journeys/pr-%s/%s/", bucket, defaultJourneyHTTPRegion, prNumber, timestamp)
	summary.S3Prefix = s3Prefix
	summary.S3HTTPBase = httpBase

	repoSlug, err := currentRepoSlug()
	if err != nil {
		return summary, err
	}
	body := buildJourneyPRComment(summary)
	if err := upsertIssueComment(repoSlug, prNumber, "<!-- agent-journey-report -->", body); err != nil {
		return summary, err
	}
	return summary, nil
}

func buildJourneyPRComment(summary JourneyCompareSummary) string {
	type capturePair struct {
		before *JourneyCaptureSummary
		after  *JourneyCaptureSummary
	}
	byJourney := map[string]*capturePair{}
	for i := range summary.Captures {
		capture := &summary.Captures[i]
		pair := byJourney[capture.Journey]
		if pair == nil {
			pair = &capturePair{}
			byJourney[capture.Journey] = pair
		}
		switch capture.Label {
		case "before":
			pair.before = capture
		case "after":
			pair.after = capture
		}
	}

	names := make([]string, 0, len(byJourney))
	for name := range byJourney {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := []string{
		"<!-- agent-journey-report -->",
		"### Agent Journey Report",
		"",
		fmt.Sprintf("Before ref: `%s`", summary.BeforeRef),
		fmt.Sprintf("After ref: `%s`", summary.AfterRef),
		"",
		"| Journey | Before | After |",
		"|---------|--------|-------|",
	}

	for _, name := range names {
		pair := byJourney[name]
		before := journeyLink(summary.RunDir, summary.S3HTTPBase, pair.before)
		after := journeyLink(summary.RunDir, summary.S3HTTPBase, pair.after)
		lines = append(lines, fmt.Sprintf("| `%s` | %s | %s |", name, before, after))
	}

	return strings.Join(lines, "\n")
}

func journeyLink(runDir, httpBase string, capture *JourneyCaptureSummary) string {
	if capture == nil {
		return "_not captured_"
	}
	artifactDir, err := filepath.Rel(runDir, capture.ArtifactDir)
	if err != nil {
		artifactDir = capture.ArtifactDir
	}
	if len(capture.VideoFiles) > 0 {
		return fmt.Sprintf("[video](%s%s)", httpBase, pathJoin(artifactDir, capture.VideoFiles[0]))
	}
	if len(capture.Screenshots) > 0 {
		return fmt.Sprintf("[screenshot](%s%s)", httpBase, pathJoin(artifactDir, capture.Screenshots[0]))
	}
	return "_no artifact_"
}

func pathJoin(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		clean = append(clean, strings.Trim(part, "/"))
	}
	return strings.Join(clean, "/")
}
