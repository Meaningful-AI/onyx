package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/onyx-dot-app/onyx/tools/ods/internal/git"
)

func ghString(args ...string) (string, error) {
	git.CheckGitHubCLI()

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("gh %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func resolvePRNumber(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	return ghString("pr", "view", "--json", "number", "--jq", ".number")
}

func currentRepoSlug() (string, error) {
	return ghString("repo", "view", "--json", "owner,name", "--jq", `.owner.login + "/" + .name`)
}

func upsertIssueComment(repoSlug, prNumber, marker, body string) error {
	commentID, err := ghString(
		"api",
		fmt.Sprintf("repos/%s/issues/%s/comments", repoSlug, prNumber),
		"--jq",
		fmt.Sprintf(".[] | select(.body | startswith(%q)) | .id", marker),
	)
	if err != nil {
		return err
	}
	if commentID != "" {
		_, err := ghString(
			"api",
			"--method", "PATCH",
			fmt.Sprintf("repos/%s/issues/comments/%s", repoSlug, commentID),
			"-f", fmt.Sprintf("body=%s", body),
		)
		return err
	}

	_, err = ghString(
		"api",
		"--method", "POST",
		fmt.Sprintf("repos/%s/issues/%s/comments", repoSlug, prNumber),
		"-f", fmt.Sprintf("body=%s", body),
	)
	return err
}
