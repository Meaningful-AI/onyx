package prreview

import "testing"

func TestClassifySource(t *testing.T) {
	t.Helper()

	cases := map[string]Source{
		"openai-codex-reviewer[bot]": SourceCodex,
		"greptile-ai[bot]":           SourceGreptile,
		"cubic-review[bot]":          SourceCubic,
		"renovate[bot]":              SourceBot,
		"human-user":                 SourceHuman,
	}

	for login, expected := range cases {
		if actual := ClassifySource(login); actual != expected {
			t.Fatalf("classify %q: expected %s, got %s", login, expected, actual)
		}
	}
}

func TestTriageMarksDuplicates(t *testing.T) {
	t.Helper()

	result := Triage(PullRequest{
		Number: 42,
		Threads: []Thread{
			{
				ID:   "thread-1",
				Path: "web/src/foo.tsx",
				Line: 10,
				Comments: []Comment{
					{ID: 1, AuthorLogin: "greptile-ai[bot]", Body: "Handle null values here."},
				},
			},
			{
				ID:   "thread-2",
				Path: "web/src/foo.tsx",
				Line: 10,
				Comments: []Comment{
					{ID: 2, AuthorLogin: "openai-codex-reviewer[bot]", Body: "Handle null values here"},
				},
			},
		},
	})

	if len(result.Summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(result.Summaries))
	}

	var duplicateFound bool
	for _, summary := range result.Summaries {
		if summary.Thread.ID == "thread-2" && summary.Category == "duplicate" {
			duplicateFound = true
		}
	}
	if !duplicateFound {
		t.Fatalf("expected duplicate thread to be detected: %+v", result.Summaries)
	}
}
