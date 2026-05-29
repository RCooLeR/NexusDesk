package sqlguard

import (
	"errors"
	"strings"
	"unicode"
)

type Options struct {
	UnsupportedMessage string
	BlockedMessage     string
	EmptyMessage       string
	Kind               string
	AllowWith          bool
}

func NormalizeReadOnly(query string, options Options) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New(firstNonEmpty(options.EmptyMessage, "enter a read-only SELECT query"))
	}
	analysis := analyze(query)
	if err := validateSingleStatement(analysis, options.EmptyMessage); err != nil {
		return "", err
	}
	for strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	if query == "" {
		return "", errors.New(firstNonEmpty(options.EmptyMessage, "enter a read-only SELECT query"))
	}
	if len(analysis.tokens) == 0 || !allowedFirstToken(analysis.tokens[0], options.AllowWith) {
		return "", errors.New(firstNonEmpty(options.UnsupportedMessage, "only read-only SELECT queries are supported"))
	}
	if containsBlockedSQLForKind(analysis.tokens, options.Kind) {
		return "", errors.New(firstNonEmpty(options.BlockedMessage, "mutating SQL is blocked"))
	}
	return query, nil
}

func StripComments(query string) string {
	runes := []rune(query)
	var builder strings.Builder
	lineComment := false
	blockCommentDepth := 0
	inSingleQuote := false
	inDoubleQuote := false
	inBacktickQuote := false
	inBracketQuote := false
	dollarQuoteTag := ""

	for index := 0; index < len(runes); index++ {
		current := runes[index]
		if lineComment {
			if current == '\n' {
				lineComment = false
				builder.WriteRune(current)
			}
			continue
		}
		if blockCommentDepth > 0 {
			if current == '/' && index+1 < len(runes) && runes[index+1] == '*' {
				blockCommentDepth++
				index++
				continue
			}
			if current == '*' && index+1 < len(runes) && runes[index+1] == '/' {
				blockCommentDepth--
				index++
				if blockCommentDepth == 0 {
					builder.WriteRune(' ')
				}
			}
			continue
		}
		if dollarQuoteTag != "" {
			builder.WriteRune(current)
			if hasRunePrefix(runes, index, []rune(dollarQuoteTag)) {
				for offset := 1; offset < len([]rune(dollarQuoteTag)); offset++ {
					builder.WriteRune(runes[index+offset])
				}
				index += len([]rune(dollarQuoteTag)) - 1
				dollarQuoteTag = ""
			}
			continue
		}
		if inSingleQuote {
			builder.WriteRune(current)
			if current == '\'' {
				if index+1 < len(runes) && runes[index+1] == '\'' {
					index++
					builder.WriteRune(runes[index])
				} else {
					inSingleQuote = false
				}
			}
			continue
		}
		if inDoubleQuote {
			builder.WriteRune(current)
			if current == '"' {
				if index+1 < len(runes) && runes[index+1] == '"' {
					index++
					builder.WriteRune(runes[index])
				} else {
					inDoubleQuote = false
				}
			}
			continue
		}
		if inBacktickQuote {
			builder.WriteRune(current)
			if current == '`' {
				if index+1 < len(runes) && runes[index+1] == '`' {
					index++
					builder.WriteRune(runes[index])
				} else {
					inBacktickQuote = false
				}
			}
			continue
		}
		if inBracketQuote {
			builder.WriteRune(current)
			if current == ']' {
				if index+1 < len(runes) && runes[index+1] == ']' {
					index++
					builder.WriteRune(runes[index])
				} else {
					inBracketQuote = false
				}
			}
			continue
		}

		switch current {
		case '#':
			lineComment = true
		case '-':
			if index+1 < len(runes) && runes[index+1] == '-' {
				lineComment = true
				index++
				continue
			}
			builder.WriteRune(current)
		case '/':
			if index+1 < len(runes) && runes[index+1] == '*' {
				blockCommentDepth = 1
				index++
				continue
			}
			builder.WriteRune(current)
		case '\'':
			inSingleQuote = true
			builder.WriteRune(current)
		case '"':
			inDoubleQuote = true
			builder.WriteRune(current)
		case '`':
			inBacktickQuote = true
			builder.WriteRune(current)
		case '[':
			inBracketQuote = true
			builder.WriteRune(current)
		case '$':
			if marker, nextIndex, ok := parseDollarQuoteStart(runes, index); ok {
				dollarQuoteTag = marker
				for offset := index; offset < nextIndex; offset++ {
					builder.WriteRune(runes[offset])
				}
				index = nextIndex - 1
				continue
			}
			builder.WriteRune(current)
		default:
			builder.WriteRune(current)
		}
	}
	return strings.TrimSpace(builder.String())
}

type analysis struct {
	tokens           []string
	statementCount   int
	invalidStatement bool
}

func analyze(query string) analysis {
	var result analysis
	runes := []rune(query)
	var tokenBuilder strings.Builder
	currentHasContent := false
	lineComment := false
	blockCommentDepth := 0
	inSingleQuote := false
	inDoubleQuote := false
	inBacktickQuote := false
	inBracketQuote := false
	dollarQuoteTag := ""

	flushToken := func() {
		if tokenBuilder.Len() == 0 {
			return
		}
		result.tokens = append(result.tokens, strings.ToLower(tokenBuilder.String()))
		tokenBuilder.Reset()
		currentHasContent = true
	}

	for index := 0; index < len(runes); index++ {
		current := runes[index]
		if lineComment {
			if current == '\n' {
				lineComment = false
			}
			continue
		}
		if blockCommentDepth > 0 {
			if current == '/' && index+1 < len(runes) && runes[index+1] == '*' {
				blockCommentDepth++
				index++
				continue
			}
			if current == '*' && index+1 < len(runes) && runes[index+1] == '/' {
				blockCommentDepth--
				index++
			}
			continue
		}
		if dollarQuoteTag != "" {
			if hasRunePrefix(runes, index, []rune(dollarQuoteTag)) {
				index += len([]rune(dollarQuoteTag)) - 1
				dollarQuoteTag = ""
			}
			continue
		}
		if inSingleQuote {
			if current == '\'' {
				if index+1 < len(runes) && runes[index+1] == '\'' {
					index++
				} else {
					inSingleQuote = false
				}
			}
			continue
		}
		if inDoubleQuote {
			if current == '"' {
				if index+1 < len(runes) && runes[index+1] == '"' {
					index++
				} else {
					inDoubleQuote = false
				}
			}
			continue
		}
		if inBacktickQuote {
			if current == '`' {
				if index+1 < len(runes) && runes[index+1] == '`' {
					index++
				} else {
					inBacktickQuote = false
				}
			}
			continue
		}
		if inBracketQuote {
			if current == ']' {
				if index+1 < len(runes) && runes[index+1] == ']' {
					index++
				} else {
					inBracketQuote = false
				}
			}
			continue
		}

		switch current {
		case '#':
			flushToken()
			lineComment = true
		case '-':
			if index+1 < len(runes) && runes[index+1] == '-' {
				flushToken()
				lineComment = true
				index++
				continue
			}
			flushToken()
			currentHasContent = true
		case '/':
			if index+1 < len(runes) && runes[index+1] == '*' {
				flushToken()
				blockCommentDepth = 1
				index++
				continue
			}
			flushToken()
			currentHasContent = true
		case '\'':
			flushToken()
			inSingleQuote = true
			currentHasContent = true
		case '"':
			flushToken()
			inDoubleQuote = true
			currentHasContent = true
		case '`':
			flushToken()
			inBacktickQuote = true
			currentHasContent = true
		case '[':
			flushToken()
			inBracketQuote = true
			currentHasContent = true
		case '$':
			if marker, nextIndex, ok := parseDollarQuoteStart(runes, index); ok {
				flushToken()
				dollarQuoteTag = marker
				currentHasContent = true
				index = nextIndex - 1
				continue
			}
			tokenBuilder.WriteRune(current)
		case ';':
			flushToken()
			if !currentHasContent {
				result.invalidStatement = true
				continue
			}
			result.statementCount++
			currentHasContent = false
		default:
			if isSQLWordRune(current) {
				tokenBuilder.WriteRune(current)
				continue
			}
			flushToken()
			if !unicode.IsSpace(current) {
				currentHasContent = true
			}
		}
	}
	flushToken()
	if currentHasContent {
		result.statementCount++
	}
	return result
}

func validateSingleStatement(analysis analysis, emptyMessage string) error {
	if analysis.invalidStatement {
		return errors.New("query must contain a single SQL statement")
	}
	if analysis.statementCount == 0 {
		return errors.New(firstNonEmpty(emptyMessage, "enter a read-only SELECT query"))
	}
	if analysis.statementCount > 1 {
		return errors.New("query must contain a single SQL statement")
	}
	return nil
}

func allowedFirstToken(token string, allowWith bool) bool {
	return token == "select" || (allowWith && token == "with")
}

func hasRunePrefix(runes []rune, start int, prefix []rune) bool {
	if start+len(prefix) > len(runes) {
		return false
	}
	for index := range prefix {
		if runes[start+index] != prefix[index] {
			return false
		}
	}
	return true
}

func parseDollarQuoteStart(runes []rune, start int) (string, int, bool) {
	if start >= len(runes) || runes[start] != '$' {
		return "", 0, false
	}
	end := start + 1
	for end < len(runes) && (unicode.IsLetter(runes[end]) || unicode.IsDigit(runes[end]) || runes[end] == '_') {
		end++
	}
	if end < len(runes) && runes[end] == '$' {
		return string(runes[start : end+1]), end + 1, true
	}
	return "", 0, false
}

func isSQLWordRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_' || value == '$'
}

func containsBlockedSQL(tokens []string) bool {
	for _, token := range tokens {
		switch token {
		case "insert", "update", "delete", "drop", "alter", "truncate", "create", "attach", "detach", "replace",
			"vacuum", "pragma", "grant", "revoke", "comment", "reindex", "analyze", "cluster", "refresh",
			"call", "execute", "do", "use", "set", "reset", "lock", "unlock", "begin", "commit", "rollback",
			"savepoint", "release", "merge", "upsert", "into", "outfile", "dumpfile", "load", "install":
			return true
		}
	}
	return false
}

func containsBlockedSQLForKind(tokens []string, kind string) bool {
	if containsBlockedSQL(tokens) {
		return true
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return false
	}
	extra := extraBlockedSQLTokensForKind(kind)
	if len(extra) == 0 {
		return false
	}
	for _, token := range tokens {
		if _, blocked := extra[token]; blocked {
			return true
		}
	}
	return false
}

func extraBlockedSQLTokensForKind(kind string) map[string]struct{} {
	switch kind {
	case "postgres":
		return map[string]struct{}{
			"copy":                 {},
			"dblink":               {},
			"pg_ls_dir":            {},
			"pg_read_binary_file":  {},
			"pg_read_file":         {},
			"pg_rotate_logfile":    {},
			"pg_stat_file":         {},
			"pg_write_file":        {},
			"postgres_fdw_handler": {},
		}
	case "mysql", "mariadb":
		return map[string]struct{}{"load_file": {}}
	case "sqlserver":
		return map[string]struct{}{
			"openquery":                  {},
			"openrowset":                 {},
			"opendatasource":             {},
			"xp_cmdshell":                {},
			"sp_execute_external_script": {},
		}
	case "duckdb":
		return map[string]struct{}{"httpfs": {}, "install": {}, "load": {}}
	case "sqlite":
		return map[string]struct{}{"load_extension": {}}
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
