package agentcheck

import (
	"reflect"
	"testing"
)

func TestParseAddedLines(t *testing.T) {
	diff := `diff --git a/backend/onyx/server/foo.py b/backend/onyx/server/foo.py
index 1111111..2222222 100644
--- a/backend/onyx/server/foo.py
+++ b/backend/onyx/server/foo.py
@@ -10,1 +11,3 @@
 context = old_value
+from fastapi import HTTPException
-raise OldError()
+raise HTTPException(status_code=400, detail="bad")
@@ -20,0 +23,1 @@
+task.delay (payload)
diff --git a/web/src/sections/Foo.tsx b/web/src/sections/Foo.tsx
index 1111111..2222222 100644
--- a/web/src/sections/Foo.tsx
+++ b/web/src/sections/Foo.tsx
@@ -3,0 +4 @@
+import { Thing } from "@/components/Thing";`

	addedLines, err := ParseAddedLines(diff)
	if err != nil {
		t.Fatalf("ParseAddedLines returned error: %v", err)
	}

	if len(addedLines) != 4 {
		t.Fatalf("expected 4 added lines, got %d", len(addedLines))
	}

	if addedLines[0].Path != "backend/onyx/server/foo.py" || addedLines[0].LineNum != 12 {
		t.Fatalf("unexpected first added line: %+v", addedLines[0])
	}

	if addedLines[2].Path != "backend/onyx/server/foo.py" || addedLines[2].LineNum != 23 {
		t.Fatalf("unexpected third added line: %+v", addedLines[2])
	}

	if addedLines[3].Path != "web/src/sections/Foo.tsx" || addedLines[3].LineNum != 4 {
		t.Fatalf("unexpected final added line: %+v", addedLines[3])
	}
}

func TestParseAddedLinesRejectsMalformedHunkHeader(t *testing.T) {
	diff := `diff --git a/backend/onyx/server/foo.py b/backend/onyx/server/foo.py
--- a/backend/onyx/server/foo.py
+++ b/backend/onyx/server/foo.py
@@ invalid @@
+raise HTTPException(status_code=400, detail="bad")`

	if _, err := ParseAddedLines(diff); err == nil {
		t.Fatal("expected malformed hunk header to return an error")
	}
}

func TestCheckAddedLinesFindsExpectedViolations(t *testing.T) {
	lines := []AddedLine{
		{Path: "backend/onyx/server/foo.py", LineNum: 10, Content: "from fastapi import HTTPException"},
		{Path: "backend/onyx/server/foo.py", LineNum: 11, Content: `raise HTTPException(status_code=400, detail="bad")`},
		{Path: "backend/onyx/server/foo.py", LineNum: 12, Content: "response_model = FooResponse"},
		{Path: "backend/onyx/server/foo.py", LineNum: 13, Content: "my_task.delay (payload)"},
		{Path: "web/src/sections/Foo.tsx", LineNum: 20, Content: `export { Thing } from "@/components/Thing";`},
	}

	violations := CheckAddedLines(lines)

	if len(violations) != 5 {
		t.Fatalf("expected 5 violations, got %d: %+v", len(violations), violations)
	}

	expectedRules := []string{
		"no-new-http-exception",
		"no-new-http-exception",
		"no-new-response-model",
		"no-new-delay",
		"no-new-legacy-component-import",
	}

	for i, expectedRule := range expectedRules {
		if violations[i].RuleID != expectedRule {
			t.Fatalf("expected rule %q at index %d, got %q", expectedRule, i, violations[i].RuleID)
		}
	}
}

func TestCheckAddedLinesIgnoresCommentsStringsAndAllowedScopes(t *testing.T) {
	lines := []AddedLine{
		{Path: "backend/onyx/server/foo.py", LineNum: 1, Content: `message = "HTTPException"`},
		{Path: "backend/onyx/server/foo.py", LineNum: 2, Content: `detail = "response_model="`},
		{Path: "backend/onyx/server/foo.py", LineNum: 3, Content: `note = ".delay("`},
		{Path: "backend/onyx/server/foo.py", LineNum: 4, Content: `# HTTPException`},
		{Path: "backend/onyx/server/foo.py", LineNum: 5, Content: `handler = HTTPExceptionAlias`},
		{Path: "backend/onyx/main.py", LineNum: 6, Content: `raise HTTPException(status_code=400, detail="bad")`},
		{Path: "backend/tests/unit/test_foo.py", LineNum: 7, Content: `from fastapi import HTTPException`},
		{Path: "backend/model_server/foo.py", LineNum: 8, Content: `task.delay(payload)`},
		{Path: "web/src/sections/Foo.tsx", LineNum: 9, Content: `const path = "@/components/Thing";`},
		{Path: "web/src/sections/Foo.tsx", LineNum: 10, Content: `// import { Thing } from "@/components/Thing";`},
		{Path: "web/src/components/Foo.tsx", LineNum: 11, Content: `import { Bar } from "@/components/Bar";`},
	}

	violations := CheckAddedLines(lines)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}

func TestCheckAddedLinesWithRulesSupportsCustomRuleSets(t *testing.T) {
	lines := []AddedLine{
		{Path: "backend/onyx/server/foo.py", LineNum: 12, Content: "response_model = FooResponse"},
		{Path: "web/src/sections/Foo.tsx", LineNum: 20, Content: `import type { Thing } from "@/components/Thing";`},
	}

	rules := []Rule{
		{
			ID:      "python-response-model-only",
			Message: "response_model is not allowed",
			Scope:   backendProductPythonScope(),
			Match: func(line lineView) bool {
				return responseModelPattern.MatchString(line.CodeSansStrings)
			},
		},
	}

	violations := CheckAddedLinesWithRules(lines, rules)
	expected := []Violation{
		{
			RuleID:  "python-response-model-only",
			Path:    "backend/onyx/server/foo.py",
			LineNum: 12,
			Message: "response_model is not allowed",
			Content: "response_model = FooResponse",
		},
	}

	if !reflect.DeepEqual(expected, violations) {
		t.Fatalf("unexpected violations: %+v", violations)
	}
}
