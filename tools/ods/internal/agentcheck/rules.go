package agentcheck

import (
	"regexp"
	"strings"
)

var (
	httpExceptionPattern = regexp.MustCompile(`\bHTTPException\b`)
	responseModelPattern = regexp.MustCompile(`\bresponse_model\s*=`)
	delayCallPattern     = regexp.MustCompile(`\.\s*delay\s*\(`)
	componentPathPattern = regexp.MustCompile(`["'](?:@/components/|\.\.?/components/|\.\.?/.*/components/)`)
	importExportPattern  = regexp.MustCompile(`^\s*(?:import|export)\b`)
)

type Scope func(path string) bool

type Matcher func(line lineView) bool

type Rule struct {
	ID      string
	Message string
	Scope   Scope
	Match   Matcher
}

type lineView struct {
	AddedLine
	Path            string
	Code            string
	CodeSansStrings string
	TrimmedCode     string
}

func CheckAddedLines(lines []AddedLine) []Violation {
	return CheckAddedLinesWithRules(lines, DefaultRules())
}

func CheckAddedLinesWithRules(lines []AddedLine, rules []Rule) []Violation {
	var violations []Violation

	for _, addedLine := range lines {
		line := buildLineView(addedLine)
		if line.Path == "" {
			continue
		}

		for _, rule := range rules {
			if rule.Scope != nil && !rule.Scope(line.Path) {
				continue
			}
			if rule.Match == nil || !rule.Match(line) {
				continue
			}

			violations = append(violations, Violation{
				RuleID:  rule.ID,
				Path:    line.Path,
				LineNum: line.LineNum,
				Message: rule.Message,
				Content: line.Content,
			})
		}
	}

	return violations
}

func DefaultRules() []Rule {
	return append([]Rule(nil), defaultRules...)
}

var defaultRules = []Rule{
	{
		ID:      "no-new-http-exception",
		Message: "Do not introduce new HTTPException usage in backend product code. Raise OnyxError instead.",
		Scope:   backendProductPythonScope(exactPath("backend/onyx/main.py")),
		Match: func(line lineView) bool {
			return hasPythonCode(line) && httpExceptionPattern.MatchString(line.CodeSansStrings)
		},
	},
	{
		ID:      "no-new-response-model",
		Message: "Do not introduce response_model on new FastAPI APIs. Type the function directly instead.",
		Scope:   backendProductPythonScope(),
		Match: func(line lineView) bool {
			return hasPythonCode(line) && responseModelPattern.MatchString(line.CodeSansStrings)
		},
	},
	{
		ID:      "no-new-delay",
		Message: "Do not introduce Celery .delay() calls. Use an enqueue path that sets expires= explicitly.",
		Scope:   backendProductPythonScope(),
		Match: func(line lineView) bool {
			return hasPythonCode(line) && delayCallPattern.MatchString(line.CodeSansStrings)
		},
	},
	{
		ID:      "no-new-legacy-component-import",
		Message: "Do not introduce new imports from web/src/components. Prefer Opal or refresh-components.",
		Scope:   nonLegacyWebSourceScope(),
		Match: func(line lineView) bool {
			return isLegacyComponentImport(line)
		},
	},
}

func buildLineView(line AddedLine) lineView {
	path := normalizeDiffPath(line.Path)
	code := stripLineComment(path, line.Content)
	return lineView{
		AddedLine:       line,
		Path:            path,
		Code:            code,
		CodeSansStrings: stripQuotedStrings(code),
		TrimmedCode:     strings.TrimSpace(code),
	}
}

func backendProductPythonScope(excluded ...Scope) Scope {
	return func(path string) bool {
		if !strings.HasPrefix(path, "backend/") || !strings.HasSuffix(path, ".py") {
			return false
		}
		if strings.HasPrefix(path, "backend/tests/") {
			return false
		}
		if strings.HasPrefix(path, "backend/model_server/") {
			return false
		}
		if strings.Contains(path, "/__pycache__/") {
			return false
		}
		for _, exclude := range excluded {
			if exclude != nil && exclude(path) {
				return false
			}
		}
		return true
	}
}

func nonLegacyWebSourceScope() Scope {
	return func(path string) bool {
		if !strings.HasPrefix(path, "web/src/") {
			return false
		}
		return !strings.HasPrefix(path, "web/src/components/")
	}
}

func exactPath(target string) Scope {
	return func(path string) bool {
		return path == target
	}
}

func hasPythonCode(line lineView) bool {
	return strings.TrimSpace(line.CodeSansStrings) != ""
}

func isLegacyComponentImport(line lineView) bool {
	if line.TrimmedCode == "" {
		return false
	}
	if !importExportPattern.MatchString(line.TrimmedCode) {
		return false
	}
	return componentPathPattern.MatchString(line.Code)
}
