package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentcheck"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentdocs"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/paths"
)

type AgentCheckOptions struct {
	Staged   bool
	BaseRef  string
	Worktree string
	RepoRoot string
}

type AgentCheckResult struct {
	Violations    []agentcheck.Violation
	DocViolations []agentdocs.Violation
}

// NewAgentCheckCommand creates the agent-check command.
func NewAgentCheckCommand() *cobra.Command {
	opts := &AgentCheckOptions{}

	cmd := &cobra.Command{
		Use:   "agent-check [paths...]",
		Short: "Run diff-based checks for agent-safe changes",
		Long: `Run diff-based checks for agent-safe changes.

This command inspects added lines in the current git diff and flags a small set
of newly introduced repo-level violations without failing on historical debt.

By default it compares the working tree against HEAD. Use --staged to inspect
the staged diff instead, or --base-ref to compare against a different ref.
Use --worktree to run the same check against a tracked target worktree from the
agent-lab control checkout.

Examples:
  ods agent-check
  ods agent-check --staged
  ods agent-check --base-ref origin/main
  ods agent-check --worktree codex/fix/auth-banner-modal --base-ref origin/main
  ods agent-check web/src backend/onyx/server/features/build`,
		Run: func(cmd *cobra.Command, args []string) {
			runAgentCheck(opts, args)
		},
	}

	cmd.Flags().BoolVar(&opts.Staged, "staged", false, "check staged changes instead of the working tree")
	cmd.Flags().StringVar(&opts.BaseRef, "base-ref", "", "git ref to diff against instead of HEAD")
	cmd.Flags().StringVar(&opts.Worktree, "worktree", "", "tracked agent-lab worktree to check instead of the current checkout")

	return cmd
}

func runAgentCheck(opts *AgentCheckOptions, providedPaths []string) {
	repoRoot, _, _ := resolveAgentLabTarget(opts.Worktree)
	opts.RepoRoot = repoRoot
	result, err := evaluateAgentCheck(opts, providedPaths)
	if err != nil {
		log.Fatalf("Failed to run agent-check: %v", err)
	}

	if len(result.Violations) == 0 && len(result.DocViolations) == 0 {
		log.Info("✅ agent-check found no new violations.")
		return
	}

	sort.Slice(result.Violations, func(i, j int) bool {
		if result.Violations[i].Path != result.Violations[j].Path {
			return result.Violations[i].Path < result.Violations[j].Path
		}
		if result.Violations[i].LineNum != result.Violations[j].LineNum {
			return result.Violations[i].LineNum < result.Violations[j].LineNum
		}
		return result.Violations[i].RuleID < result.Violations[j].RuleID
	})

	for _, violation := range result.Violations {
		log.Errorf("\n❌ %s:%d [%s]", violation.Path, violation.LineNum, violation.RuleID)
		log.Errorf("  %s", violation.Message)
		log.Errorf("  Added line: %s", strings.TrimSpace(violation.Content))
	}

	for _, violation := range result.DocViolations {
		log.Errorf("\n❌ %s [agent-docs]", violation.Path)
		log.Errorf("  %s", violation.Message)
	}

	fmt.Fprintf(
		os.Stderr,
		"\nFound %d agent-check violation(s) and %d agent-docs violation(s).\n",
		len(result.Violations),
		len(result.DocViolations),
	)
	os.Exit(1)
}

func evaluateAgentCheck(opts *AgentCheckOptions, providedPaths []string) (*AgentCheckResult, error) {
	diffOutput, err := getAgentCheckDiff(opts, providedPaths)
	if err != nil {
		return nil, err
	}

	addedLines, err := agentcheck.ParseAddedLines(diffOutput)
	if err != nil {
		return nil, err
	}

	root := opts.RepoRoot
	if root == "" {
		var err error
		root, err = paths.GitRoot()
		if err != nil {
			return nil, fmt.Errorf("determine git root: %w", err)
		}
	}

	result := &AgentCheckResult{
		Violations:    agentcheck.CheckAddedLines(addedLines),
		DocViolations: agentdocs.Validate(root),
	}
	return result, nil
}

func getAgentCheckDiff(opts *AgentCheckOptions, providedPaths []string) (string, error) {
	args := []string{"diff", "--no-color", "--unified=0"}

	if opts.Staged {
		args = append(args, "--cached")
	} else if opts.BaseRef != "" {
		args = append(args, opts.BaseRef)
	} else {
		args = append(args, "HEAD")
	}

	if len(providedPaths) > 0 {
		args = append(args, "--")
		args = append(args, providedPaths...)
	}

	cmd := exec.Command("git", args...)
	if opts.RepoRoot != "" {
		cmd.Dir = opts.RepoRoot
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, string(output))
	}

	return string(output), nil
}
