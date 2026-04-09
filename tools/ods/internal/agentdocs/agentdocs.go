package agentdocs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

var requiredFiles = []string{
	"AGENTS.md",
	"docs/agent/README.md",
	"docs/agent/ARCHITECTURE.md",
	"docs/agent/BRANCHING.md",
	"docs/agent/HARNESS.md",
	"docs/agent/GOLDEN_RULES.md",
	"docs/agent/LEGACY_ZONES.md",
	"docs/agent/QUALITY_SCORE.md",
}

type Violation struct {
	Path    string
	Message string
}

func Validate(root string) []Violation {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash("docs/agent/README.md"))); err != nil {
		return nil
	}

	var violations []Violation

	for _, relPath := range requiredFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relPath))); err != nil {
			violations = append(violations, Violation{
				Path:    relPath,
				Message: "required agent-lab knowledge-base file is missing",
			})
		}
	}

	for _, relPath := range requiredFiles {
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		violations = append(violations, validateMarkdownLinks(root, relPath, string(content))...)
	}

	return violations
}

func validateMarkdownLinks(root string, relPath string, content string) []Violation {
	var violations []Violation
	matches := markdownLinkPattern.FindAllStringSubmatch(content, -1)
	docDir := filepath.Dir(filepath.Join(root, filepath.FromSlash(relPath)))

	for _, match := range matches {
		if len(match) != 2 {
			continue
		}

		target := strings.TrimSpace(match[1])
		if target == "" {
			continue
		}
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}
		if strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
			continue
		}

		target = stripAnchor(target)

		var absTarget string
		if filepath.IsAbs(target) {
			absTarget = target
		} else {
			absTarget = filepath.Join(docDir, target)
		}

		if _, err := os.Stat(absTarget); err != nil {
			violations = append(violations, Violation{
				Path: relPath,
				Message: fmt.Sprintf(
					"broken markdown link target: %s",
					target,
				),
			})
		}
	}

	return violations
}

func stripAnchor(target string) string {
	if idx := strings.Index(target, "#"); idx >= 0 {
		return target[:idx]
	}
	return target
}
