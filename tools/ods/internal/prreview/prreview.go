package prreview

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Source string

const (
	SourceHuman    Source = "human"
	SourceCodex    Source = "codex"
	SourceGreptile Source = "greptile"
	SourceCubic    Source = "cubic"
	SourceBot      Source = "bot"
)

type Comment struct {
	ID          int    `json:"id"`
	Body        string `json:"body"`
	AuthorLogin string `json:"author_login"`
	URL         string `json:"url,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type Thread struct {
	ID         string    `json:"id"`
	IsResolved bool      `json:"is_resolved"`
	IsOutdated bool      `json:"is_outdated"`
	Path       string    `json:"path,omitempty"`
	Line       int       `json:"line,omitempty"`
	StartLine  int       `json:"start_line,omitempty"`
	Comments   []Comment `json:"comments"`
}

type PullRequest struct {
	Number  int      `json:"number"`
	Title   string   `json:"title"`
	URL     string   `json:"url,omitempty"`
	Threads []Thread `json:"threads"`
}

type ThreadSummary struct {
	Thread      Thread   `json:"thread"`
	Source      Source   `json:"source"`
	Category    string   `json:"category"`
	DuplicateOf string   `json:"duplicate_of,omitempty"`
	Reasons     []string `json:"reasons,omitempty"`
}

type TriageResult struct {
	PullRequest PullRequest     `json:"pull_request"`
	Summaries   []ThreadSummary `json:"summaries"`
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func ClassifySource(login string) Source {
	lower := strings.ToLower(strings.TrimSpace(login))
	switch {
	case strings.Contains(lower, "codex"):
		return SourceCodex
	case strings.Contains(lower, "greptile"):
		return SourceGreptile
	case strings.Contains(lower, "cubic"):
		return SourceCubic
	case strings.HasSuffix(lower, "[bot]") || strings.Contains(lower, "bot"):
		return SourceBot
	default:
		return SourceHuman
	}
}

func Triage(pr PullRequest) TriageResult {
	summaries := make([]ThreadSummary, 0, len(pr.Threads))
	seen := map[string]string{}

	for _, thread := range pr.Threads {
		source := SourceHuman
		if len(thread.Comments) > 0 {
			source = ClassifySource(thread.Comments[0].AuthorLogin)
		}

		summary := ThreadSummary{
			Thread:   thread,
			Source:   source,
			Category: "actionable",
		}

		if thread.IsResolved {
			summary.Category = "resolved"
			summary.Reasons = append(summary.Reasons, "thread already resolved")
		} else if thread.IsOutdated {
			summary.Category = "outdated"
			summary.Reasons = append(summary.Reasons, "thread marked outdated by GitHub")
		}

		key := duplicateKey(thread)
		if existing, ok := seen[key]; ok && summary.Category == "actionable" {
			summary.Category = "duplicate"
			summary.DuplicateOf = existing
			summary.Reasons = append(summary.Reasons, fmt.Sprintf("duplicates %s", existing))
		} else if summary.Category == "actionable" {
			seen[key] = thread.ID
		}

		if source == SourceHuman && summary.Category == "actionable" {
			summary.Reasons = append(summary.Reasons, "human review requires explicit response or fix")
		}
		if source != SourceHuman && summary.Category == "actionable" {
			summary.Reasons = append(summary.Reasons, fmt.Sprintf("%s-generated review comment", source))
		}

		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Category != summaries[j].Category {
			return summaries[i].Category < summaries[j].Category
		}
		if summaries[i].Source != summaries[j].Source {
			return summaries[i].Source < summaries[j].Source
		}
		return summaries[i].Thread.ID < summaries[j].Thread.ID
	})

	return TriageResult{
		PullRequest: pr,
		Summaries:   summaries,
	}
}

func duplicateKey(thread Thread) string {
	parts := []string{thread.Path, fmt.Sprintf("%d", thread.Line)}
	if len(thread.Comments) > 0 {
		parts = append(parts, normalizeBody(thread.Comments[0].Body))
	}
	return strings.Join(parts, "::")
}

func normalizeBody(body string) string {
	normalized := strings.ToLower(strings.TrimSpace(body))
	normalized = nonAlphaNum.ReplaceAllString(normalized, " ")
	return strings.Join(strings.Fields(normalized), " ")
}
