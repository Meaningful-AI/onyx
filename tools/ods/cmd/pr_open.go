package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/git"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/paths"
)

var conventionalPRTitlePattern = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([^)]+\))?: .+`)

type PROpenOptions struct {
	Title    string
	Base     string
	BodyFile string
	Draft    bool
}

// NewPROpenCommand creates the pr-open command.
func NewPROpenCommand() *cobra.Command {
	opts := &PROpenOptions{}

	cmd := &cobra.Command{
		Use:   "pr-open",
		Short: "Open a GitHub pull request using the repo template and a conventional-commit title",
		Run: func(cmd *cobra.Command, args []string) {
			runPROpen(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Title, "title", "", "PR title (defaults to the latest commit subject)")
	cmd.Flags().StringVar(&opts.Base, "base", "main", "base branch for the PR")
	cmd.Flags().StringVar(&opts.BodyFile, "body-file", "", "explicit PR body file (defaults to .github/pull_request_template.md)")
	cmd.Flags().BoolVar(&opts.Draft, "draft", false, "open the PR as a draft")

	return cmd
}

func runPROpen(opts *PROpenOptions) {
	git.CheckGitHubCLI()

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		subject, err := git.GetCommitMessage("HEAD")
		if err != nil {
			log.Fatalf("Failed to determine PR title from HEAD: %v", err)
		}
		title = subject
	}
	if !conventionalPRTitlePattern.MatchString(title) {
		log.Fatalf("PR title must follow conventional-commit style. Got %q", title)
	}

	bodyFile := strings.TrimSpace(opts.BodyFile)
	if bodyFile == "" {
		repoRoot, err := paths.GitRoot()
		if err != nil {
			log.Fatalf("Failed to determine git root: %v", err)
		}
		bodyFile = filepath.Join(repoRoot, ".github", "pull_request_template.md")
	}
	bodyBytes, err := os.ReadFile(bodyFile)
	if err != nil {
		log.Fatalf("Failed to read PR body file %s: %v", bodyFile, err)
	}

	args := []string{"pr", "create", "--base", opts.Base, "--title", title, "--body", string(bodyBytes)}
	if opts.Draft {
		args = append(args, "--draft")
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to open PR: %v", err)
	}

	fmt.Printf("Opened PR with title %q\n", title)
}
