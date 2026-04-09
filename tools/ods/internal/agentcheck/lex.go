package agentcheck

import "strings"

func stripLineComment(path string, content string) string {
	switch {
	case strings.HasSuffix(path, ".py"):
		return stripCommentMarker(content, "#")
	case isJSLikePath(path):
		return stripCommentMarker(content, "//")
	default:
		return content
	}
}

func isJSLikePath(path string) bool {
	return strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".jsx") ||
		strings.HasSuffix(path, ".ts") ||
		strings.HasSuffix(path, ".tsx")
}

func stripCommentMarker(line string, marker string) string {
	if marker == "" {
		return line
	}

	var builder strings.Builder
	quote := byte(0)
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if quote != 0 {
			builder.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if strings.HasPrefix(line[i:], marker) {
			break
		}

		builder.WriteByte(ch)
		if isQuote(ch) {
			quote = ch
		}
	}

	return builder.String()
}

func stripQuotedStrings(line string) string {
	var builder strings.Builder
	quote := byte(0)
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if isQuote(ch) {
			quote = ch
			builder.WriteByte(' ')
			continue
		}

		builder.WriteByte(ch)
	}

	return builder.String()
}

func isQuote(ch byte) bool {
	return ch == '"' || ch == '\'' || ch == '`'
}
