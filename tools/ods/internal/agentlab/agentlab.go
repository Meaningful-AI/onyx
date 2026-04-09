package agentlab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	stateDirName        = "onyx-agent-lab"
	worktreesDirName    = "worktrees"
	envFileName         = ".env.agent-lab"
	webEnvFileName      = ".env.web.agent-lab"
	defaultWebPort      = 3300
	defaultAPIPort      = 8380
	defaultModelPort    = 9300
	defaultMCPPort      = 8390
	portSearchWindow    = 400
	dockerProjectPrefix = "onyx"
	searchInfraMode     = "shared"
)

var nonAlphaNumPattern = regexp.MustCompile(`[^a-z0-9]+`)

type DependencyMode string

const (
	DependencyModeShared     DependencyMode = "shared"
	DependencyModeNamespaced DependencyMode = "namespaced"
)

type WorktreeLane string

const (
	WorktreeLaneLab     WorktreeLane = "lab"
	WorktreeLaneProduct WorktreeLane = "product"
	WorktreeLaneCustom  WorktreeLane = "custom"
)

var productBranchPrefixes = []string{
	"build/",
	"chore/",
	"ci/",
	"docs/",
	"feat/",
	"fix/",
	"perf/",
	"refactor/",
	"revert/",
	"style/",
	"test/",
}

type DependencyConfig struct {
	Mode              DependencyMode `json:"mode"`
	Namespace         string         `json:"namespace,omitempty"`
	PostgresDatabase  string         `json:"postgres_database,omitempty"`
	RedisPrefix       string         `json:"redis_prefix,omitempty"`
	FileStoreBucket   string         `json:"file_store_bucket,omitempty"`
	SearchInfraMode   string         `json:"search_infra_mode"`
	LastProvisionedAt string         `json:"last_provisioned_at,omitempty"`
}

type PortSet struct {
	Web         int `json:"web"`
	API         int `json:"api"`
	ModelServer int `json:"model_server"`
	MCP         int `json:"mcp"`
}

type URLSet struct {
	Web string `json:"web"`
	API string `json:"api"`
	MCP string `json:"mcp"`
}

type Manifest struct {
	ID                string           `json:"id"`
	Branch            string           `json:"branch"`
	Lane              WorktreeLane     `json:"lane,omitempty"`
	BaseRef           string           `json:"base_ref"`
	CreatedFromPath   string           `json:"created_from_path"`
	CheckoutPath      string           `json:"checkout_path"`
	StateDir          string           `json:"state_dir"`
	ArtifactDir       string           `json:"artifact_dir"`
	EnvFile           string           `json:"env_file"`
	WebEnvFile        string           `json:"web_env_file"`
	ComposeProject    string           `json:"compose_project"`
	Dependencies      DependencyConfig `json:"dependencies"`
	Ports             PortSet          `json:"ports"`
	URLs              URLSet           `json:"urls"`
	CreatedAt         time.Time        `json:"created_at"`
	LastVerifiedAt    string           `json:"last_verified_at,omitempty"`
	LastVerifySummary string           `json:"last_verify_summary,omitempty"`
}

func Slug(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "/", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = nonAlphaNumPattern.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "worktree"
	}
	return normalized
}

func worktreeID(value string) string {
	slug := Slug(value)
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(sum[:4]))
}

func ComposeProjectName(id string) string {
	slug := Slug(id)
	if len(slug) > 32 {
		slug = slug[:32]
	}
	return fmt.Sprintf("%s-%s", dockerProjectPrefix, slug)
}

func GetCommonGitDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-common-dir failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func StateRoot(commonGitDir string) string {
	return filepath.Join(commonGitDir, stateDirName)
}

func WorktreesRoot(commonGitDir string) string {
	return filepath.Join(StateRoot(commonGitDir), worktreesDirName)
}

func WorktreeStateDir(commonGitDir, id string) string {
	return filepath.Join(WorktreesRoot(commonGitDir), Slug(id))
}

func ManifestPath(commonGitDir, id string) string {
	return filepath.Join(WorktreeStateDir(commonGitDir, id), "manifest.json")
}

func DefaultCheckoutPath(repoRoot, id string) string {
	parent := filepath.Dir(repoRoot)
	worktreesRoot := filepath.Join(parent, filepath.Base(repoRoot)+"-worktrees")
	return filepath.Join(worktreesRoot, worktreeID(id))
}

func NormalizeBranchForLane(branch string) string {
	normalized := strings.TrimSpace(branch)
	normalized = strings.TrimPrefix(normalized, "refs/heads/")
	normalized = strings.TrimPrefix(normalized, "origin/")
	normalized = strings.TrimPrefix(normalized, "codex/")
	return normalized
}

func InferLane(branch string) WorktreeLane {
	normalized := NormalizeBranchForLane(branch)
	if strings.HasPrefix(normalized, "lab/") {
		return WorktreeLaneLab
	}
	for _, prefix := range productBranchPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return WorktreeLaneProduct
		}
	}
	return WorktreeLaneCustom
}

type BaseRefSelection struct {
	Ref    string
	Lane   WorktreeLane
	Reason string
}

func ResolveCreateBaseRef(branch, requested string, refExists func(string) bool) BaseRefSelection {
	lane := InferLane(branch)
	if requested != "" {
		return BaseRefSelection{
			Ref:    requested,
			Lane:   lane,
			Reason: "using explicit --from value",
		}
	}

	switch lane {
	case WorktreeLaneLab:
		for _, candidate := range []string{"codex/agent-lab", "agent-lab", "origin/codex/agent-lab", "origin/agent-lab"} {
			if refExists(candidate) {
				return BaseRefSelection{
					Ref:    candidate,
					Lane:   lane,
					Reason: fmt.Sprintf("inferred lab lane from branch name; using %s as the base ref", candidate),
				}
			}
		}
		return BaseRefSelection{
			Ref:    "HEAD",
			Lane:   lane,
			Reason: "inferred lab lane from branch name, but no agent-lab ref exists locally; falling back to HEAD",
		}
	case WorktreeLaneProduct:
		for _, candidate := range []string{"origin/main", "main"} {
			if refExists(candidate) {
				return BaseRefSelection{
					Ref:    candidate,
					Lane:   lane,
					Reason: fmt.Sprintf("inferred product lane from branch name; using %s as the base ref", candidate),
				}
			}
		}
		return BaseRefSelection{
			Ref:    "HEAD",
			Lane:   lane,
			Reason: "inferred product lane from branch name, but no main ref exists locally; falling back to HEAD",
		}
	default:
		return BaseRefSelection{
			Ref:    "HEAD",
			Lane:   lane,
			Reason: "no lane inferred from branch name; defaulting to HEAD. Prefer codex/lab/... for harness work and codex/fix... or codex/feat... for product work, or pass --from explicitly",
		}
	}
}

func GitRefExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	return cmd.Run() == nil
}

func BuildManifest(repoRoot, commonGitDir, branch string, lane WorktreeLane, baseRef, checkoutPath string, ports PortSet, dependencyMode DependencyMode) Manifest {
	id := worktreeID(branch)
	stateDir := WorktreeStateDir(commonGitDir, id)
	artifactDir := filepath.Join(stateDir, "artifacts")
	envDir := filepath.Join(checkoutPath, ".vscode")

	return Manifest{
		ID:              id,
		Branch:          branch,
		Lane:            lane,
		BaseRef:         baseRef,
		CreatedFromPath: repoRoot,
		CheckoutPath:    checkoutPath,
		StateDir:        stateDir,
		ArtifactDir:     artifactDir,
		EnvFile:         filepath.Join(envDir, envFileName),
		WebEnvFile:      filepath.Join(envDir, webEnvFileName),
		ComposeProject:  ComposeProjectName(id),
		Dependencies:    BuildDependencyConfig(branch, dependencyMode),
		Ports:           ports,
		URLs: URLSet{
			Web: fmt.Sprintf("http://127.0.0.1:%d", ports.Web),
			API: fmt.Sprintf("http://127.0.0.1:%d", ports.API),
			MCP: fmt.Sprintf("http://127.0.0.1:%d", ports.MCP),
		},
		CreatedAt: time.Now().UTC(),
	}
}

func (m Manifest) ResolvedLane() WorktreeLane {
	if m.Lane == "" {
		return InferLane(m.Branch)
	}
	return m.Lane
}

func BuildDependencyConfig(branch string, mode DependencyMode) DependencyConfig {
	if mode == "" {
		mode = DependencyModeShared
	}

	config := DependencyConfig{
		Mode:            mode,
		SearchInfraMode: searchInfraMode,
	}

	if mode != DependencyModeNamespaced {
		return config
	}

	namespace := worktreeID(branch)
	dbSuffix := strings.ReplaceAll(namespace, "-", "_")
	database := fmt.Sprintf("agentlab_%s", dbSuffix)
	if len(database) > 63 {
		database = database[:63]
	}
	bucket := fmt.Sprintf("onyx-agentlab-%s", namespace)
	if len(bucket) > 63 {
		bucket = bucket[:63]
		bucket = strings.Trim(bucket, "-")
	}

	config.Namespace = namespace
	config.PostgresDatabase = database
	config.RedisPrefix = fmt.Sprintf("agentlab:%s", namespace)
	config.FileStoreBucket = bucket
	return config
}

func (m Manifest) ResolvedDependencies() DependencyConfig {
	if m.Dependencies.Mode == "" {
		return BuildDependencyConfig(m.Branch, DependencyModeShared)
	}
	resolved := m.Dependencies
	if resolved.SearchInfraMode == "" {
		resolved.SearchInfraMode = searchInfraMode
	}
	return resolved
}

func (m Manifest) RuntimeEnv() map[string]string {
	env := map[string]string{
		"AGENT_LAB_ARTIFACT_DIR":      m.ArtifactDir,
		"AGENT_LAB_DEPENDENCY_MODE":   string(m.ResolvedDependencies().Mode),
		"AGENT_LAB_SEARCH_INFRA_MODE": m.ResolvedDependencies().SearchInfraMode,
		"AGENT_LAB_WORKTREE_ID":       m.ID,
		"AGENT_LAB_WORKTREE_URL":      m.URLs.Web,
		"BASE_URL":                    m.URLs.Web,
		"INTERNAL_URL":                m.URLs.API,
		"MCP_INTERNAL_URL":            m.URLs.MCP,
		"PORT":                        fmt.Sprintf("%d", m.Ports.Web),
		"WEB_DOMAIN":                  m.URLs.Web,
	}

	deps := m.ResolvedDependencies()
	if deps.Namespace != "" {
		env["AGENT_LAB_NAMESPACE"] = deps.Namespace
	}
	if deps.Mode == DependencyModeNamespaced {
		env["POSTGRES_DB"] = deps.PostgresDatabase
		env["DEFAULT_REDIS_PREFIX"] = deps.RedisPrefix
		env["S3_FILE_STORE_BUCKET_NAME"] = deps.FileStoreBucket
	}

	return env
}

func (m Manifest) ShellEnv() map[string]string {
	return m.RuntimeEnv()
}

func (m Manifest) DependencyWarnings() []string {
	deps := m.ResolvedDependencies()
	if deps.SearchInfraMode == searchInfraMode {
		return []string{
			"Search infrastructure remains shared across worktrees. OpenSearch/Vespa state is not namespaced or torn down by agent-lab.",
		}
	}
	return nil
}

func (m Manifest) EnvFileContents(kind string) string {
	values := m.RuntimeEnv()
	deps := m.ResolvedDependencies()
	var lines []string
	lines = append(lines, "# Generated by `ods worktree create` for agent-lab.")
	lines = append(lines, "# This file only contains worktree-local overrides.")
	lines = append(lines, fmt.Sprintf("AGENT_LAB_WORKTREE_ID=%s", m.ID))
	lines = append(lines, fmt.Sprintf("AGENT_LAB_ARTIFACT_DIR=%s", m.ArtifactDir))
	lines = append(lines, fmt.Sprintf("AGENT_LAB_DEPENDENCY_MODE=%s", deps.Mode))
	lines = append(lines, fmt.Sprintf("AGENT_LAB_SEARCH_INFRA_MODE=%s", deps.SearchInfraMode))
	if deps.Namespace != "" {
		lines = append(lines, fmt.Sprintf("AGENT_LAB_NAMESPACE=%s", deps.Namespace))
	}
	switch kind {
	case "web":
		lines = append(lines, fmt.Sprintf("PORT=%d", m.Ports.Web))
		lines = append(lines, fmt.Sprintf("BASE_URL=%s", values["BASE_URL"]))
		lines = append(lines, fmt.Sprintf("WEB_DOMAIN=%s", values["WEB_DOMAIN"]))
		lines = append(lines, fmt.Sprintf("INTERNAL_URL=%s", values["INTERNAL_URL"]))
		lines = append(lines, fmt.Sprintf("MCP_INTERNAL_URL=%s", values["MCP_INTERNAL_URL"]))
	default:
		lines = append(lines, fmt.Sprintf("WEB_DOMAIN=%s", values["WEB_DOMAIN"]))
		lines = append(lines, fmt.Sprintf("INTERNAL_URL=%s", values["INTERNAL_URL"]))
		lines = append(lines, fmt.Sprintf("MCP_INTERNAL_URL=%s", values["MCP_INTERNAL_URL"]))
		if deps.Mode == DependencyModeNamespaced {
			lines = append(lines, fmt.Sprintf("POSTGRES_DB=%s", deps.PostgresDatabase))
			lines = append(lines, fmt.Sprintf("DEFAULT_REDIS_PREFIX=%s", deps.RedisPrefix))
			lines = append(lines, fmt.Sprintf("S3_FILE_STORE_BUCKET_NAME=%s", deps.FileStoreBucket))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func WriteManifest(commonGitDir string, manifest Manifest) error {
	stateDir := WorktreeStateDir(commonGitDir, manifest.ID)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("create worktree state dir: %w", err)
	}
	if err := os.MkdirAll(manifest.ArtifactDir, 0755); err != nil {
		return fmt.Errorf("create artifact dir: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(ManifestPath(commonGitDir, manifest.ID), data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func WriteEnvFiles(manifest Manifest) error {
	if err := os.MkdirAll(filepath.Dir(manifest.EnvFile), 0755); err != nil {
		return fmt.Errorf("create env dir: %w", err)
	}
	if err := os.WriteFile(manifest.EnvFile, []byte(manifest.EnvFileContents("backend")), 0644); err != nil {
		return fmt.Errorf("write backend env file: %w", err)
	}
	if err := os.WriteFile(manifest.WebEnvFile, []byte(manifest.EnvFileContents("web")), 0644); err != nil {
		return fmt.Errorf("write web env file: %w", err)
	}
	return nil
}

func LoadAll(commonGitDir string) ([]Manifest, error) {
	worktreesRoot := WorktreesRoot(commonGitDir)
	entries, err := os.ReadDir(worktreesRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read worktrees dir: %w", err)
	}

	manifests := make([]Manifest, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := LoadManifest(filepath.Join(worktreesRoot, entry.Name(), "manifest.json"))
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].Branch < manifests[j].Branch
	})

	return manifests, nil
}

func LoadManifest(path string) (Manifest, error) {
	var manifest Manifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, fmt.Errorf("read manifest %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	return manifest, nil
}

func FindByRepoRoot(commonGitDir, repoRoot string) (Manifest, bool, error) {
	manifests, err := LoadAll(commonGitDir)
	if err != nil {
		return Manifest{}, false, err
	}

	repoRoot = normalizePath(repoRoot)
	for _, manifest := range manifests {
		if normalizePath(manifest.CheckoutPath) == repoRoot {
			return manifest, true, nil
		}
	}

	return Manifest{}, false, nil
}

func FindByIdentifier(commonGitDir, identifier string) (Manifest, bool, error) {
	manifests, err := LoadAll(commonGitDir)
	if err != nil {
		return Manifest{}, false, err
	}

	slug := Slug(identifier)
	cleanIdentifier := normalizePath(identifier)
	var slugMatches []Manifest

	for _, manifest := range manifests {
		switch {
		case manifest.ID == slug:
			return manifest, true, nil
		case manifest.Branch == identifier:
			return manifest, true, nil
		case normalizePath(manifest.CheckoutPath) == cleanIdentifier:
			return manifest, true, nil
		case slug != "" && Slug(manifest.Branch) == slug:
			slugMatches = append(slugMatches, manifest)
		}
	}

	if len(slugMatches) == 1 {
		return slugMatches[0], true, nil
	}
	if len(slugMatches) > 1 {
		return Manifest{}, false, fmt.Errorf("identifier %q matches multiple worktrees; use the branch, full id, or checkout path", identifier)
	}

	return Manifest{}, false, nil
}

func RemoveState(commonGitDir, id string) error {
	if err := os.RemoveAll(WorktreeStateDir(commonGitDir, id)); err != nil {
		return fmt.Errorf("remove worktree state: %w", err)
	}
	return nil
}

func UpdateVerification(commonGitDir string, manifest Manifest, summaryPath string, verifiedAt time.Time) error {
	manifest.LastVerifySummary = summaryPath
	manifest.LastVerifiedAt = verifiedAt.UTC().Format(time.RFC3339)
	return WriteManifest(commonGitDir, manifest)
}

func AllocatePorts(existing []Manifest) (PortSet, error) {
	reserved := make(map[int]bool)
	for _, manifest := range existing {
		reserved[manifest.Ports.Web] = true
		reserved[manifest.Ports.API] = true
		reserved[manifest.Ports.ModelServer] = true
		reserved[manifest.Ports.MCP] = true
	}

	for offset := 0; offset < portSearchWindow; offset++ {
		ports := PortSet{
			Web:         defaultWebPort + offset,
			API:         defaultAPIPort + offset,
			ModelServer: defaultModelPort + offset,
			MCP:         defaultMCPPort + offset,
		}

		if reserved[ports.Web] || reserved[ports.API] || reserved[ports.ModelServer] || reserved[ports.MCP] {
			continue
		}

		if portsAvailable(ports) {
			return ports, nil
		}
	}

	return PortSet{}, fmt.Errorf("failed to allocate an available worktree port set after %d attempts", portSearchWindow)
}

func portsAvailable(ports PortSet) bool {
	candidates := []int{ports.Web, ports.API, ports.ModelServer, ports.MCP}
	for _, port := range candidates {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return false
		}
		_ = ln.Close()
	}
	return true
}

func normalizePath(path string) string {
	clean := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(clean)
	if err == nil {
		return filepath.Clean(resolved)
	}
	return clean
}
