package cmd

import (
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/git"
)

type PRMergeOptions struct {
	PR           string
	Auto         bool
	DeleteBranch bool
	Method       string
}

// NewPRMergeCommand creates the pr-merge command.
func NewPRMergeCommand() *cobra.Command {
	opts := &PRMergeOptions{}

	cmd := &cobra.Command{
		Use:   "pr-merge",
		Short: "Merge a GitHub pull request through gh with explicit method flags",
		Run: func(cmd *cobra.Command, args []string) {
			runPRMerge(opts)
		},
	}

	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number (defaults to the current branch PR)")
	cmd.Flags().BoolVar(&opts.Auto, "auto", false, "enable auto-merge instead of merging immediately")
	cmd.Flags().BoolVar(&opts.DeleteBranch, "delete-branch", false, "delete the branch after merge")
	cmd.Flags().StringVar(&opts.Method, "method", "squash", "merge method: squash, merge, or rebase")

	return cmd
}

func runPRMerge(opts *PRMergeOptions) {
	git.CheckGitHubCLI()

	prNumber, err := resolvePRNumber(opts.PR)
	if err != nil {
		log.Fatalf("Failed to resolve PR number: %v", err)
	}

	args := []string{"pr", "merge", prNumber}
	switch opts.Method {
	case "squash":
		args = append(args, "--squash")
	case "merge":
		args = append(args, "--merge")
	case "rebase":
		args = append(args, "--rebase")
	default:
		log.Fatalf("Invalid merge method %q: expected squash, merge, or rebase", opts.Method)
	}
	if opts.Auto {
		args = append(args, "--auto")
	}
	if opts.DeleteBranch {
		args = append(args, "--delete-branch")
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to merge PR #%s: %v", prNumber, err)
	}
}
