package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentlab"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/envutil"
)

// NewBackendCommand creates the parent "backend" command with subcommands for
// running backend services.
// BackendOptions holds options shared across backend subcommands.
type BackendOptions struct {
	NoEE     bool
	Worktree string
}

func NewBackendCommand() *cobra.Command {
	opts := &BackendOptions{}

	cmd := &cobra.Command{
		Use:   "backend",
		Short: "Run backend services (api, model_server)",
		Long: `Run backend services with environment from .vscode/.env.

On first run, copies .vscode/env_template.txt to .vscode/.env if the
.env file does not already exist.

Enterprise Edition features are enabled by default for development,
with license enforcement disabled.

Available subcommands:
  api            Start the FastAPI backend server
  model_server   Start the model server`,
	}

	cmd.PersistentFlags().BoolVar(&opts.NoEE, "no-ee", false, "Disable Enterprise Edition features (enabled by default)")
	cmd.PersistentFlags().StringVar(&opts.Worktree, "worktree", "", "tracked agent-lab worktree to run from instead of the current checkout")

	cmd.AddCommand(newBackendAPICommand(opts))
	cmd.AddCommand(newBackendModelServerCommand(opts))

	return cmd
}

func newBackendAPICommand(opts *BackendOptions) *cobra.Command {
	var port string

	cmd := &cobra.Command{
		Use:   "api",
		Short: "Start the backend API server (uvicorn with hot-reload)",
		Long: `Start the backend API server using uvicorn with hot-reload.

Examples:
  ods backend api
  ods backend api --port 9090
  ods backend api --no-ee
  ods backend api --worktree codex/fix/auth-banner-modal`,
		Run: func(cmd *cobra.Command, args []string) {
			runBackendService("api", "onyx.main:app", port, cmd.Flags().Changed("port"), opts)
		},
	}

	cmd.Flags().StringVar(&port, "port", "8080", "Port to listen on")

	return cmd
}

func newBackendModelServerCommand(opts *BackendOptions) *cobra.Command {
	var port string

	cmd := &cobra.Command{
		Use:   "model_server",
		Short: "Start the model server (uvicorn with hot-reload)",
		Long: `Start the model server using uvicorn with hot-reload.

Examples:
  ods backend model_server
  ods backend model_server --port 9001
  ods backend model_server --worktree codex/fix/auth-banner-modal`,
		Run: func(cmd *cobra.Command, args []string) {
			runBackendService("model_server", "model_server.main:app", port, cmd.Flags().Changed("port"), opts)
		},
	}

	cmd.Flags().StringVar(&port, "port", "9000", "Port to listen on")

	return cmd
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func getProcessOnPort(port int) string {
	out, err := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-t").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return "an unknown process"
	}
	pid := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
	nameOut, err := exec.Command("ps", "-p", pid, "-o", "comm=").Output()
	if err != nil || len(strings.TrimSpace(string(nameOut))) == 0 {
		return fmt.Sprintf("process (PID %s)", pid)
	}
	return fmt.Sprintf("%s (PID %s)", strings.TrimSpace(string(nameOut)), pid)
}

func resolvePort(port string) string {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Invalid port %q: %v", port, err)
	}
	if isPortAvailable(portNum) {
		return port
	}
	proc := getProcessOnPort(portNum)
	candidate := portNum + 1
	for candidate <= 65535 {
		if isPortAvailable(candidate) {
			log.Warnf("⚠ Port %d is in use by %s, using available port %d instead.", portNum, proc, candidate)
			return strconv.Itoa(candidate)
		}
		candidate++
	}
	log.Fatalf("No available ports found starting from %d", portNum)
	return port
}

func runBackendService(name, module, port string, portExplicit bool, opts *BackendOptions) {
	root, worktreeManifest, hasWorktreeManifest := resolveAgentLabTarget(opts.Worktree)

	if hasWorktreeManifest && !portExplicit {
		switch name {
		case "api":
			port = strconv.Itoa(worktreeManifest.Ports.API)
		case "model_server":
			port = strconv.Itoa(worktreeManifest.Ports.ModelServer)
		}
	}

	port = resolvePort(port)

	envFile := ensureBackendEnvFile(root)
	fileVars, err := envutil.LoadFile(envFile)
	if err != nil {
		log.Fatalf("Failed to load env file %s: %v", envFile, err)
	}

	eeDefaults := eeEnvDefaults(opts.NoEE)
	fileVars = append(fileVars, eeDefaults...)

	backendDir := filepath.Join(root, "backend")

	uvicornArgs := []string{
		"run", "uvicorn", module,
		"--reload",
		"--port", port,
	}
	log.Infof("Starting %s on port %s...", name, port)
	if !opts.NoEE {
		log.Info("Enterprise Edition enabled (use --no-ee to disable)")
	}
	if hasWorktreeManifest {
		log.Infof("agent-lab worktree %s detected: web=%s api=%s", worktreeManifest.Branch, worktreeManifest.URLs.Web, worktreeManifest.URLs.API)
		log.Infof("lane=%s base-ref=%s", worktreeManifest.ResolvedLane(), worktreeManifest.BaseRef)
		log.Infof("dependency mode=%s search-infra=%s", worktreeManifest.ResolvedDependencies().Mode, worktreeManifest.ResolvedDependencies().SearchInfraMode)
	}
	log.Debugf("Running in %s: uv %v", backendDir, uvicornArgs)

	mergedEnv := envutil.Merge(os.Environ(), fileVars)
	if hasWorktreeManifest {
		mergedEnv = envutil.ApplyOverrides(mergedEnv, worktreeManifest.RuntimeEnv())
	}
	log.Debugf("Applied %d env vars from %s (shell takes precedence)", len(fileVars), envFile)

	svcCmd := exec.Command("uv", uvicornArgs...)
	svcCmd.Dir = backendDir
	svcCmd.Stdout = os.Stdout
	svcCmd.Stderr = os.Stderr
	svcCmd.Stdin = os.Stdin
	svcCmd.Env = mergedEnv

	if err := svcCmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code != -1 {
				os.Exit(code)
			}
		}
		log.Fatalf("Failed to run %s: %v", name, err)
	}
}

func currentAgentLabManifest(repoRoot string) (agentlab.Manifest, bool) {
	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		return agentlab.Manifest{}, false
	}
	manifest, found, err := agentlab.FindByRepoRoot(commonGitDir, repoRoot)
	if err != nil {
		return agentlab.Manifest{}, false
	}
	return manifest, found
}

// eeEnvDefaults returns env entries for EE and license enforcement settings.
// These are appended to the file vars so they act as defaults — shell env
// and .env file values still take precedence via mergeEnv.
func eeEnvDefaults(noEE bool) []string {
	if noEE {
		return []string{
			"ENABLE_PAID_ENTERPRISE_EDITION_FEATURES=false",
		}
	}
	return []string{
		"ENABLE_PAID_ENTERPRISE_EDITION_FEATURES=true",
		"LICENSE_ENFORCEMENT_ENABLED=false",
	}
}

// ensureBackendEnvFile copies env_template.txt to .env if .env doesn't exist.
func ensureBackendEnvFile(root string) string {
	vscodeDir := filepath.Join(root, ".vscode")
	envFile := filepath.Join(vscodeDir, ".env")
	templateFile := filepath.Join(vscodeDir, "env_template.txt")

	if _, err := os.Stat(envFile); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("Failed to stat env file %s: %v", envFile, err)
		}
	} else {
		log.Debugf("Using existing env file: %s", envFile)
		return envFile
	}

	templateData, err := os.ReadFile(templateFile)
	if err != nil {
		log.Fatalf("Failed to read env template %s: %v", templateFile, err)
	}

	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		log.Fatalf("Failed to create .vscode directory: %v", err)
	}

	if err := os.WriteFile(envFile, templateData, 0644); err != nil {
		log.Fatalf("Failed to write env file %s: %v", envFile, err)
	}

	log.Infof("Created %s from template (review and fill in <REPLACE THIS> values)", envFile)
	return envFile
}
