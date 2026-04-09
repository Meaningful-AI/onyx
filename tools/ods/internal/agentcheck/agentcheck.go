package agentcheck

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var hunkHeaderPattern = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

type AddedLine struct {
	Path    string
	LineNum int
	Content string
}

type Violation struct {
	RuleID  string
	Path    string
	LineNum int
	Message string
	Content string
}

func ParseAddedLines(diff string) ([]AddedLine, error) {
	scanner := bufio.NewScanner(strings.NewReader(diff))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var addedLines []AddedLine
	currentPath := ""
	currentNewLine := 0
	inHunk := false

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "+++ "):
			currentPath = normalizeDiffPath(strings.TrimPrefix(line, "+++ "))
			inHunk = false
		case strings.HasPrefix(line, "@@ "):
			match := hunkHeaderPattern.FindStringSubmatch(line)
			if len(match) != 2 {
				return nil, fmt.Errorf("failed to parse hunk header: %s", line)
			}
			var err error
			currentNewLine, err = parseLineNumber(match[1])
			if err != nil {
				return nil, err
			}
			inHunk = true
		case !inHunk || currentPath == "":
			continue
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			addedLines = append(addedLines, AddedLine{
				Path:    currentPath,
				LineNum: currentNewLine,
				Content: strings.TrimPrefix(line, "+"),
			})
			currentNewLine++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			continue
		default:
			currentNewLine++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan diff: %w", err)
	}

	return addedLines, nil
}

func normalizeDiffPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "b/")
	if path == "/dev/null" {
		return ""
	}
	return filepath.ToSlash(path)
}

func parseLineNumber(value string) (int, error) {
	lineNum := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid line number: %s", value)
		}
		lineNum = lineNum*10 + int(ch-'0')
	}
	return lineNum, nil
}
