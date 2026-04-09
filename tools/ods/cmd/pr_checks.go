package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/git"
)

type PRChecksOptions struct {
	PR string
}

type ghChecksResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				Number  int    `json:"number"`
				Title   string `json:"title"`
				URL     string `json:"url"`
				HeadRef string `json:"headRefName"`
				Commits struct {
					Nodes []struct {
						Commit struct {
							StatusCheckRollup struct {
								Contexts struct {
									Nodes []struct {
										Type         string `json:"__typename"`
										Name         string `json:"name"`
										DisplayTitle string `json:"displayTitle"`
										WorkflowName string `json:"workflowName"`
										Status       string `json:"status"`
										Conclusion   string `json:"conclusion"`
										DetailsURL   string `json:"detailsUrl"`
										Context      string `json:"context"`
										State        string `json:"state"`
										TargetURL    string `json:"targetUrl"`
										Description  string `json:"description"`
									} `json:"nodes"`
								} `json:"contexts"`
							} `json:"statusCheckRollup"`
						} `json:"commit"`
					} `json:"nodes"`
				} `json:"commits"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

// NewPRChecksCommand creates the pr-checks command surface.
func NewPRChecksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr-checks",
		Short: "Inspect GitHub PR checks and surface failing runs for remediation",
	}

	cmd.AddCommand(newPRChecksStatusCommand())
	cmd.AddCommand(newPRChecksDiagnoseCommand())
	return cmd
}

func newPRChecksStatusCommand() *cobra.Command {
	opts := &PRChecksOptions{}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "List all status checks for a pull request",
		Run: func(cmd *cobra.Command, args []string) {
			runPRChecksStatus(opts)
		},
	}
	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number (defaults to the current branch PR)")
	return cmd
}

func newPRChecksDiagnoseCommand() *cobra.Command {
	opts := &PRChecksOptions{}
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "List only failing checks and point to the next remediation command",
		Run: func(cmd *cobra.Command, args []string) {
			runPRChecksDiagnose(opts)
		},
	}
	cmd.Flags().StringVar(&opts.PR, "pr", "", "pull request number (defaults to the current branch PR)")
	return cmd
}

func runPRChecksStatus(opts *PRChecksOptions) {
	prNumber, err := resolvePRNumber(opts.PR)
	if err != nil {
		log.Fatalf("Failed to resolve PR number: %v", err)
	}
	response, err := fetchPRChecks(prNumber)
	if err != nil {
		log.Fatalf("Failed to fetch PR checks: %v", err)
	}

	fmt.Printf("PR #%d %s\n", response.Data.Repository.PullRequest.Number, response.Data.Repository.PullRequest.Title)
	for _, check := range flattenChecks(response) {
		fmt.Printf("[%s] %s (%s) %s\n", check.result(), check.displayName(), check.kind(), check.url())
	}
}

func runPRChecksDiagnose(opts *PRChecksOptions) {
	prNumber, err := resolvePRNumber(opts.PR)
	if err != nil {
		log.Fatalf("Failed to resolve PR number: %v", err)
	}
	response, err := fetchPRChecks(prNumber)
	if err != nil {
		log.Fatalf("Failed to fetch PR checks: %v", err)
	}

	failing := failingChecks(response)
	if len(failing) == 0 {
		fmt.Printf("No failing checks found on PR #%s\n", prNumber)
		return
	}

	fmt.Printf("Failing checks for PR #%s:\n", prNumber)
	for _, check := range failing {
		fmt.Printf("- %s (%s)\n", check.displayName(), check.url())
		if strings.Contains(strings.ToLower(check.displayName()), "playwright") {
			fmt.Printf("  next: ods trace --pr %s\n", prNumber)
		} else {
			fmt.Printf("  next: gh run view <run-id> --log-failed\n")
		}
	}
}

func fetchPRChecks(prNumber string) (*ghChecksResponse, error) {
	repoSlug, err := currentRepoSlug()
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(repoSlug, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected repo slug %q", repoSlug)
	}

	git.CheckGitHubCLI()
	query := `query($owner:String!, $name:String!, $number:Int!) {
  repository(owner:$owner, name:$name) {
    pullRequest(number:$number) {
      number
      title
      url
      headRefName
      commits(last:1) {
        nodes {
          commit {
            statusCheckRollup {
              contexts(first:100) {
                nodes {
                  __typename
                  ... on CheckRun {
                    name
                    status
                    conclusion
                    detailsUrl
                  }
                  ... on StatusContext {
                    context
                    state
                    targetUrl
                    description
                  }
                }
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
		"-F", "owner="+parts[0],
		"-F", "name="+parts[1],
		"-F", "number="+prNumber,
	)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh api graphql failed: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("gh api graphql failed: %w", err)
	}

	var response ghChecksResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("parse PR checks: %w", err)
	}
	return &response, nil
}

type flattenedCheck struct {
	Type         string
	Name         string
	DisplayTitle string
	WorkflowName string
	Status       string
	Conclusion   string
	DetailsURL   string
	Context      string
	State        string
	TargetURL    string
}

func flattenChecks(response *ghChecksResponse) []flattenedCheck {
	result := []flattenedCheck{}
	if response == nil || len(response.Data.Repository.PullRequest.Commits.Nodes) == 0 {
		return result
	}
	for _, node := range response.Data.Repository.PullRequest.Commits.Nodes[0].Commit.StatusCheckRollup.Contexts.Nodes {
		result = append(result, flattenedCheck{
			Type:         node.Type,
			Name:         node.Name,
			DisplayTitle: node.DisplayTitle,
			WorkflowName: node.WorkflowName,
			Status:       node.Status,
			Conclusion:   node.Conclusion,
			DetailsURL:   node.DetailsURL,
			Context:      node.Context,
			State:        node.State,
			TargetURL:    node.TargetURL,
		})
	}
	return result
}

func (c flattenedCheck) displayName() string {
	switch c.Type {
	case "CheckRun":
		if c.DisplayTitle != "" {
			return c.DisplayTitle
		}
		if c.WorkflowName != "" && c.Name != "" {
			return c.WorkflowName + " / " + c.Name
		}
		return c.Name
	default:
		return c.Context
	}
}

func (c flattenedCheck) kind() string {
	if c.Type == "" {
		return "status"
	}
	return c.Type
}

func (c flattenedCheck) result() string {
	if c.Type == "CheckRun" {
		if c.Conclusion != "" {
			return strings.ToLower(c.Conclusion)
		}
		return strings.ToLower(c.Status)
	}
	return strings.ToLower(c.State)
}

func (c flattenedCheck) url() string {
	if c.DetailsURL != "" {
		return c.DetailsURL
	}
	return c.TargetURL
}

func failingChecks(response *ghChecksResponse) []flattenedCheck {
	checks := flattenChecks(response)
	failing := make([]flattenedCheck, 0, len(checks))
	for _, check := range checks {
		result := check.result()
		if result == "failure" || result == "failed" || result == "timed_out" || result == "cancelled" || result == "error" {
			failing = append(failing, check)
		}
	}
	return failing
}
