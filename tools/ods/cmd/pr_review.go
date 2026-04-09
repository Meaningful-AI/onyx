package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/agentlab"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/git"
	"github.com/onyx-dot-app/onyx/tools/ods/internal/prreview"
)

type PRReviewFetchOptions struct {
	PR     string
	Output string
}

type PRReviewTriageOptions struct {
	PR     string
	Output string
}

type PRReviewRespondOptions struct {
	PR        string
	CommentID int
	Body      string
	ThreadID  string
}

type ghReviewResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				Number        int    `json:"number"`
				Title         string `json:"title"`
				URL           string `json:"url"`
				ReviewThreads struct {
					Nodes []struct {
						ID         string `json:"id"`
						IsResolved bool   `json:"isResolved"`
						IsOutdated bool   `json:"isOutdated"`
						Path       string `json:"path"`
						Line       int    `json:"line"`
						StartLine  int    `json:"startLine"`
						Comments   struct {
							Nodes []struct {
								DatabaseID int `json:"databaseId"`
								Body       string
								URL        string `json:"url"`
								CreatedAt  string `json:"createdAt"`
								Author     struct {
									Login string `json:"login"`
								} `json:"author"`
							} `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

// NewPRReviewCommand creates the pr-review command surface.
func NewPRReviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr-review",
		Short: "Fetch, triage, and respond to GitHub pull request review threads",
	}

	cmd.AddCommand(newPRReviewFetchCommand())
	cmd.AddCommand(newPRReviewTriageCommand())
	cmd.AddCommand(newPRReviewRespondCommand())
	cmd.AddCommand(newPRReviewResolveCommand())

	return cmd
}

func newPRReviewFetchCommand() *cobra.Command {
	opts := &PRReviewFetchOptions{}

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch pull request review threads and write them to local harness state",
		Run: func(cmd *cobra.Command, args []string) {
			runPRReviewFetch(opts)
		},
	}

	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number (defaults to the current branch PR)")
	cmd.Flags().StringVar(&opts.Output, "output", "", "explicit output path for the fetched review JSON")
	return cmd
}

func newPRReviewTriageCommand() *cobra.Command {
	opts := &PRReviewTriageOptions{}

	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Classify unresolved review threads into actionable, duplicate, outdated, or resolved",
		Run: func(cmd *cobra.Command, args []string) {
			runPRReviewTriage(opts)
		},
	}

	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number (defaults to the current branch PR)")
	cmd.Flags().StringVar(&opts.Output, "output", "", "explicit output path for the triage JSON")
	return cmd
}

func newPRReviewRespondCommand() *cobra.Command {
	opts := &PRReviewRespondOptions{}

	cmd := &cobra.Command{
		Use:   "respond",
		Short: "Reply to an inline pull request review comment and optionally resolve the thread",
		Run: func(cmd *cobra.Command, args []string) {
			runPRReviewRespond(opts)
		},
	}

	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number (defaults to the current branch PR)")
	cmd.Flags().IntVar(&opts.CommentID, "comment-id", 0, "top-level pull request review comment ID to reply to")
	cmd.Flags().StringVar(&opts.ThreadID, "thread-id", "", "GraphQL review thread ID to resolve after replying")
	cmd.Flags().StringVar(&opts.Body, "body", "", "reply body to post")
	_ = cmd.MarkFlagRequired("comment-id")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

func newPRReviewResolveCommand() *cobra.Command {
	opts := &PRReviewRespondOptions{}

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve a review thread without posting a reply",
		Run: func(cmd *cobra.Command, args []string) {
			runPRReviewResolve(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ThreadID, "thread-id", "", "GraphQL review thread ID to resolve")
	_ = cmd.MarkFlagRequired("thread-id")

	return cmd
}

func runPRReviewFetch(opts *PRReviewFetchOptions) {
	prNumber, err := resolvePRNumber(opts.PR)
	if err != nil {
		log.Fatalf("Failed to resolve PR number: %v", err)
	}

	review, err := fetchPRReview(prNumber)
	if err != nil {
		log.Fatalf("Failed to fetch PR review threads: %v", err)
	}

	outputPath, err := reviewOutputPath(prNumber, opts.Output, "threads.json")
	if err != nil {
		log.Fatalf("Failed to determine output path: %v", err)
	}
	writeJSON(outputPath, review)
	log.Infof("Fetched %d review threads into %s", len(review.Threads), outputPath)
}

func runPRReviewTriage(opts *PRReviewTriageOptions) {
	prNumber, err := resolvePRNumber(opts.PR)
	if err != nil {
		log.Fatalf("Failed to resolve PR number: %v", err)
	}

	review, err := fetchPRReview(prNumber)
	if err != nil {
		log.Fatalf("Failed to fetch PR review threads: %v", err)
	}
	triage := prreview.Triage(review)

	outputPath, err := reviewOutputPath(prNumber, opts.Output, "triage.json")
	if err != nil {
		log.Fatalf("Failed to determine output path: %v", err)
	}
	writeJSON(outputPath, triage)

	for _, summary := range triage.Summaries {
		lineRef := ""
		if summary.Thread.Path != "" {
			lineRef = summary.Thread.Path
			if summary.Thread.Line > 0 {
				lineRef = fmt.Sprintf("%s:%d", lineRef, summary.Thread.Line)
			}
		}
		fmt.Printf("[%s] %s %s %s\n", summary.Category, summary.Source, summary.Thread.ID, lineRef)
		for _, reason := range summary.Reasons {
			fmt.Printf("  - %s\n", reason)
		}
	}
	log.Infof("Wrote PR review triage to %s", outputPath)
}

func runPRReviewRespond(opts *PRReviewRespondOptions) {
	prNumber, err := resolvePRNumber(opts.PR)
	if err != nil {
		log.Fatalf("Failed to resolve PR number: %v", err)
	}
	repoSlug, err := currentRepoSlug()
	if err != nil {
		log.Fatalf("Failed to resolve repo slug: %v", err)
	}

	if err := replyToReviewComment(repoSlug, prNumber, opts.CommentID, opts.Body); err != nil {
		log.Fatalf("Failed to reply to review comment: %v", err)
	}
	if strings.TrimSpace(opts.ThreadID) != "" {
		if err := resolveReviewThread(opts.ThreadID); err != nil {
			log.Fatalf("Failed to resolve review thread: %v", err)
		}
	}
	log.Infof("Posted reply to review comment %d on PR #%s", opts.CommentID, prNumber)
}

func runPRReviewResolve(opts *PRReviewRespondOptions) {
	if err := resolveReviewThread(opts.ThreadID); err != nil {
		log.Fatalf("Failed to resolve review thread: %v", err)
	}
	log.Infof("Resolved review thread %s", opts.ThreadID)
}

func fetchPRReview(prNumber string) (prreview.PullRequest, error) {
	repoSlug, err := currentRepoSlug()
	if err != nil {
		return prreview.PullRequest{}, err
	}
	parts := strings.SplitN(repoSlug, "/", 2)
	if len(parts) != 2 {
		return prreview.PullRequest{}, fmt.Errorf("unexpected repo slug %q", repoSlug)
	}

	response, err := ghGraphQL(parts[0], parts[1], prNumber)
	if err != nil {
		return prreview.PullRequest{}, err
	}

	pr := prreview.PullRequest{
		Number:  response.Data.Repository.PullRequest.Number,
		Title:   response.Data.Repository.PullRequest.Title,
		URL:     response.Data.Repository.PullRequest.URL,
		Threads: []prreview.Thread{},
	}

	for _, thread := range response.Data.Repository.PullRequest.ReviewThreads.Nodes {
		item := prreview.Thread{
			ID:         thread.ID,
			IsResolved: thread.IsResolved,
			IsOutdated: thread.IsOutdated,
			Path:       thread.Path,
			Line:       thread.Line,
			StartLine:  thread.StartLine,
			Comments:   []prreview.Comment{},
		}
		for _, comment := range thread.Comments.Nodes {
			item.Comments = append(item.Comments, prreview.Comment{
				ID:          comment.DatabaseID,
				Body:        comment.Body,
				AuthorLogin: comment.Author.Login,
				URL:         comment.URL,
				CreatedAt:   comment.CreatedAt,
			})
		}
		pr.Threads = append(pr.Threads, item)
	}

	return pr, nil
}

func ghGraphQL(owner, name, prNumber string) (*ghReviewResponse, error) {
	git.CheckGitHubCLI()
	query := `query($owner:String!, $name:String!, $number:Int!) {
  repository(owner:$owner, name:$name) {
    pullRequest(number:$number) {
      number
      title
      url
      reviewThreads(first:100) {
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          comments(first:100) {
            nodes {
              databaseId
              body
              url
              createdAt
              author {
                login
              }
            }
          }
        }
      }
    }
  }
}`

	cmd := exec.Command(
		"gh", "api", "graphql",
		"-f", "query="+query,
		"-F", "owner="+owner,
		"-F", "name="+name,
		"-F", "number="+prNumber,
	)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh api graphql failed: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("gh api graphql failed: %w", err)
	}

	var response ghReviewResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("parse graphql response: %w", err)
	}
	return &response, nil
}

func replyToReviewComment(repoSlug, prNumber string, commentID int, body string) error {
	_, err := ghString(
		"api",
		"--method", "POST",
		fmt.Sprintf("repos/%s/pulls/%s/comments/%d/replies", repoSlug, prNumber, commentID),
		"-f", "body="+body,
	)
	return err
}

func resolveReviewThread(threadID string) error {
	git.CheckGitHubCLI()
	mutation := `mutation($threadId:ID!) {
  resolveReviewThread(input:{threadId:$threadId}) {
    thread {
      id
      isResolved
    }
  }
}`

	cmd := exec.Command(
		"gh", "api", "graphql",
		"-f", "query="+mutation,
		"-F", "threadId="+threadID,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("resolve review thread: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func reviewOutputPath(prNumber, explicit, fileName string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}

	commonGitDir, err := agentlab.GetCommonGitDir()
	if err != nil {
		return "", err
	}
	stateDir := filepath.Join(agentlab.StateRoot(commonGitDir), "reviews", "pr-"+prNumber)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return "", fmt.Errorf("create review state dir: %w", err)
	}
	return filepath.Join(stateDir, fileName), nil
}

func writeJSON(path string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode JSON for %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("Failed to write %s: %v", path, err)
	}
}
