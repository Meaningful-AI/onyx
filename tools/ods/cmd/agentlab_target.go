package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentlab"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/paths"
)

func resolveAgentLabTarget(identifier string) (string, agentlab.Manifest, bool) {
	if identifier == "" {
		repoRoot, err := paths.GitRoot()
		if err != nil {
			log.Fatalf("Failed to determine git root: %v", err)
		}
		manifest, found := currentAgentLabManifest(repoRoot)
		return repoRoot, manifest, found
	}

	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		log.Fatalf("Failed to determine git common dir: %v", err)
	}
	manifest, found, err := agentlab.FindByIdentifier(commonGitDir, identifier)
	if err != nil {
		log.Fatalf("Failed to resolve worktree %q: %v", identifier, err)
	}
	if !found {
		log.Fatalf("No agent-lab worktree found for %q", identifier)
	}
	return manifest.CheckoutPath, manifest, true
}
