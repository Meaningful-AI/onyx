package agentlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/alembic"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/envutil"
)

type DependencyResult struct {
	Actions []string
}

type DependencyStatus struct {
	Mode                 DependencyMode `json:"mode"`
	Namespace            string         `json:"namespace,omitempty"`
	PostgresDatabase     string         `json:"postgres_database,omitempty"`
	PostgresReady        bool           `json:"postgres_ready"`
	PostgresTableCount   int            `json:"postgres_table_count,omitempty"`
	RedisPrefix          string         `json:"redis_prefix,omitempty"`
	RedisReady           bool           `json:"redis_ready"`
	RedisKeyCount        int            `json:"redis_key_count,omitempty"`
	FileStoreBucket      string         `json:"file_store_bucket,omitempty"`
	FileStoreReady       bool           `json:"file_store_ready"`
	FileStoreObjectCount int            `json:"file_store_object_count,omitempty"`
	SearchInfraMode      string         `json:"search_infra_mode"`
}

func ProvisionDependencies(commonGitDir string, manifest Manifest) (Manifest, *DependencyResult, error) {
	deps := manifest.ResolvedDependencies()
	result := &DependencyResult{}

	switch deps.Mode {
	case DependencyModeShared:
		result.Actions = append(result.Actions, "using shared Postgres, Redis, and MinIO state")
	case DependencyModeNamespaced:
		if _, err := runPythonScript(manifest, "ensure_database.py"); err != nil {
			return manifest, nil, fmt.Errorf("ensure PostgreSQL database %s: %w", deps.PostgresDatabase, err)
		}
		result.Actions = append(result.Actions, fmt.Sprintf("ensured PostgreSQL database %s", deps.PostgresDatabase))

		envMap, err := runtimeEnvMap(manifest)
		if err != nil {
			return manifest, nil, err
		}
		if err := alembic.UpgradeWithEnv("head", alembic.SchemaDefault, envMap); err != nil {
			return manifest, nil, fmt.Errorf("migrate namespaced database %s: %w", deps.PostgresDatabase, err)
		}
		result.Actions = append(result.Actions, fmt.Sprintf("migrated PostgreSQL database %s", deps.PostgresDatabase))

		if _, err := runPythonScript(manifest, "ensure_bucket.py"); err != nil {
			return manifest, nil, fmt.Errorf("ensure file-store bucket %s: %w", deps.FileStoreBucket, err)
		}
		result.Actions = append(result.Actions, fmt.Sprintf("ensured file-store bucket %s", deps.FileStoreBucket))
		result.Actions = append(result.Actions, fmt.Sprintf("reserved Redis prefix %s", deps.RedisPrefix))
	default:
		return manifest, nil, fmt.Errorf("unsupported dependency mode: %s", deps.Mode)
	}

	result.Actions = append(result.Actions, "search infrastructure remains shared-only")
	manifest.Dependencies = deps
	manifest.Dependencies.LastProvisionedAt = time.Now().UTC().Format(time.RFC3339)
	if err := WriteManifest(commonGitDir, manifest); err != nil {
		return manifest, nil, err
	}
	return manifest, result, nil
}

func InspectDependencies(manifest Manifest) (*DependencyStatus, error) {
	deps := manifest.ResolvedDependencies()
	status := &DependencyStatus{
		Mode:             deps.Mode,
		Namespace:        deps.Namespace,
		PostgresDatabase: deps.PostgresDatabase,
		RedisPrefix:      deps.RedisPrefix,
		FileStoreBucket:  deps.FileStoreBucket,
		SearchInfraMode:  deps.SearchInfraMode,
	}

	if deps.Mode == DependencyModeShared {
		status.PostgresReady = true
		status.RedisReady = true
		status.FileStoreReady = true
		return status, nil
	}

	output, err := runPythonScript(manifest, "dependency_status.py")
	if err != nil {
		return nil, fmt.Errorf("inspect namespaced dependencies: %w", err)
	}
	if err := json.Unmarshal([]byte(output), status); err != nil {
		return nil, fmt.Errorf("parse dependency status: %w", err)
	}
	return status, nil
}

func ResetDependencies(commonGitDir string, manifest Manifest) (Manifest, *DependencyResult, error) {
	deps := manifest.ResolvedDependencies()
	result := &DependencyResult{}
	if deps.Mode == DependencyModeShared {
		result.Actions = append(result.Actions, "shared dependency mode selected; reset is a no-op")
		return manifest, result, nil
	}

	if _, err := runPythonScript(manifest, "reset_dependencies.py"); err != nil {
		return manifest, nil, fmt.Errorf("reset namespaced dependencies: %w", err)
	}
	result.Actions = append(result.Actions, fmt.Sprintf("dropped and recreated PostgreSQL database %s", deps.PostgresDatabase))
	result.Actions = append(result.Actions, fmt.Sprintf("cleared Redis prefix %s", deps.RedisPrefix))
	result.Actions = append(result.Actions, fmt.Sprintf("emptied file-store bucket %s", deps.FileStoreBucket))

	envMap, err := runtimeEnvMap(manifest)
	if err != nil {
		return manifest, nil, err
	}
	if err := alembic.UpgradeWithEnv("head", alembic.SchemaDefault, envMap); err != nil {
		return manifest, nil, fmt.Errorf("re-migrate namespaced database %s: %w", deps.PostgresDatabase, err)
	}
	result.Actions = append(result.Actions, fmt.Sprintf("re-migrated PostgreSQL database %s", deps.PostgresDatabase))
	result.Actions = append(result.Actions, "search infrastructure remains shared-only and was not reset")

	manifest.Dependencies = deps
	manifest.Dependencies.LastProvisionedAt = time.Now().UTC().Format(time.RFC3339)
	if err := WriteManifest(commonGitDir, manifest); err != nil {
		return manifest, nil, err
	}
	return manifest, result, nil
}

func TeardownDependencies(commonGitDir string, manifest Manifest) (Manifest, *DependencyResult, error) {
	deps := manifest.ResolvedDependencies()
	result := &DependencyResult{}
	if deps.Mode == DependencyModeShared {
		result.Actions = append(result.Actions, "shared dependency mode selected; teardown is a no-op")
		return manifest, result, nil
	}

	if _, err := runPythonScript(manifest, "teardown_dependencies.py"); err != nil {
		return manifest, nil, fmt.Errorf("tear down namespaced dependencies: %w", err)
	}
	result.Actions = append(result.Actions, fmt.Sprintf("dropped PostgreSQL database %s", deps.PostgresDatabase))
	result.Actions = append(result.Actions, fmt.Sprintf("cleared Redis prefix %s", deps.RedisPrefix))
	result.Actions = append(result.Actions, fmt.Sprintf("deleted file-store bucket %s", deps.FileStoreBucket))
	result.Actions = append(result.Actions, "search infrastructure remains shared-only and was not torn down")

	manifest.Dependencies = deps
	manifest.Dependencies.LastProvisionedAt = ""
	if err := WriteManifest(commonGitDir, manifest); err != nil {
		return manifest, nil, err
	}
	return manifest, result, nil
}

func runtimeEnvMap(manifest Manifest) (map[string]string, error) {
	envMap := make(map[string]string)
	repoRoot := runtimeRepoRoot(manifest)

	backendEnvPath := filepath.Join(repoRoot, ".vscode", ".env")
	if _, err := os.Stat(backendEnvPath); err == nil {
		fileVars, err := envutil.LoadFile(backendEnvPath)
		if err != nil {
			return nil, err
		}
		for _, entry := range fileVars {
			if idx := strings.Index(entry, "="); idx > 0 {
				envMap[entry[:idx]] = entry[idx+1:]
			}
		}
	}

	for key, value := range manifest.RuntimeEnv() {
		envMap[key] = value
	}
	return envMap, nil
}

func runPythonScript(manifest Manifest, scriptName string) (string, error) {
	pythonBinary, err := findPythonBinary(manifest)
	if err != nil {
		return "", err
	}
	code, err := loadPythonScript(scriptName)
	if err != nil {
		return "", err
	}

	envMap, err := runtimeEnvMap(manifest)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(pythonBinary, "-c", code)
	cmd.Dir = filepath.Join(runtimeRepoRoot(manifest), "backend")
	cmd.Env = envutil.ApplyOverrides(os.Environ(), envMap)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func findPythonBinary(manifest Manifest) (string, error) {
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join(manifest.CheckoutPath, ".venv", "Scripts", "python.exe"),
			filepath.Join(manifest.CreatedFromPath, ".venv", "Scripts", "python.exe"),
		}
	} else {
		candidates = []string{
			filepath.Join(manifest.CheckoutPath, ".venv", "bin", "python"),
			filepath.Join(manifest.CreatedFromPath, ".venv", "bin", "python"),
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find a Python interpreter in %s/.venv or %s/.venv", manifest.CheckoutPath, manifest.CreatedFromPath)
}

func runtimeRepoRoot(manifest Manifest) string {
	if manifest.CheckoutPath != "" {
		if _, err := os.Stat(filepath.Join(manifest.CheckoutPath, "backend")); err == nil {
			return manifest.CheckoutPath
		}
	}
	return manifest.CreatedFromPath
}
