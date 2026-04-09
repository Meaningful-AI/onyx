package journey

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	RegistryPath    = "web/tests/e2e/journeys/registry.json"
	DefaultPlanPath = ".github/agent-journeys.json"
)

type Definition struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	TestPath            string `json:"test_path"`
	Project             string `json:"project"`
	RequiresModelServer bool   `json:"requires_model_server"`
	SkipGlobalSetup     bool   `json:"skip_global_setup"`
}

type Registry struct {
	Journeys []Definition `json:"journeys"`
}

type Plan struct {
	Journeys []string `json:"journeys"`
}

func LoadRegistry(repoRoot string) (Registry, error) {
	var registry Registry

	data, err := os.ReadFile(filepath.Join(repoRoot, RegistryPath))
	if err != nil {
		return registry, fmt.Errorf("read journey registry: %w", err)
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return registry, fmt.Errorf("parse journey registry: %w", err)
	}
	if len(registry.Journeys) == 0 {
		return registry, fmt.Errorf("journey registry is empty")
	}

	for _, journey := range registry.Journeys {
		if strings.TrimSpace(journey.Name) == "" {
			return registry, fmt.Errorf("journey registry contains an entry with an empty name")
		}
		if strings.TrimSpace(journey.TestPath) == "" {
			return registry, fmt.Errorf("journey %q is missing test_path", journey.Name)
		}
		if strings.TrimSpace(journey.Project) == "" {
			return registry, fmt.Errorf("journey %q is missing project", journey.Name)
		}
	}

	return registry, nil
}

func LoadPlan(planPath string) (Plan, error) {
	var plan Plan

	data, err := os.ReadFile(planPath)
	if err != nil {
		return plan, fmt.Errorf("read journey plan: %w", err)
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return plan, fmt.Errorf("parse journey plan: %w", err)
	}
	if len(plan.Journeys) == 0 {
		return plan, fmt.Errorf("journey plan contains no journeys")
	}
	return plan, nil
}

func ResolveDefinitions(repoRoot string, names []string) ([]Definition, error) {
	registry, err := LoadRegistry(repoRoot)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]Definition, len(registry.Journeys))
	for _, definition := range registry.Journeys {
		byName[definition.Name] = definition
	}

	definitions := make([]Definition, 0, len(names))
	for _, name := range names {
		definition, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("unknown journey %q", name)
		}
		definitions = append(definitions, definition)
	}

	return definitions, nil
}

func Slug(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.ReplaceAll(normalized, "/", "-")
	var builder strings.Builder
	lastDash := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "journey"
	}
	return slug
}
