package dbconnector

import (
	"errors"
	"strings"
	"unicode"
)

func NormalizeExternalReadOnlySQL(query string) (string, error) {
	return NormalizeExternalReadOnlySQLForKind("", query)
}

func NormalizeExternalReadOnlySQLForKind(kind string, query string) (string, error) {
	return normalizeReadOnlySQL(query,
		"external database connectors only support read-only SELECT queries",
		"external database connector blocks mutating SQL",
		kind,
	)
}

func normalizeReadOnlySQL(query string, unsupportedMessage string, blockedMessage string, kind string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("enter a read-only SELECT query")
	}
	analysis := analyzeSQL(query)
	if err := validateSingleStatement(analysis); err != nil {
		return "", err
	}
	for strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	if query == "" {
		return "", errors.New("enter a read-only SELECT query")
	}
	if len(analysis.tokens) == 0 || (analysis.tokens[0] != "select" && analysis.tokens[0] != "with") {
		return "", errors.New(unsupportedMessage)
	}
	if containsBlockedSQLForKind(analysis.tokens, kind) {
		return "", errors.New(blockedMessage)
	}
	return query, nil
}

type sqlAnalysis struct {
	tokens           []string
	statementCount   int
	invalidStatement bool
}

func analyzeSQL(query string) sqlAnalysis {
	var analysis sqlAnalysis
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
		analysis.tokens = append(analysis.tokens, strings.ToLower(tokenBuilder.String()))
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
				analysis.invalidStatement = true
				continue
			}
			analysis.statementCount++
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
		analysis.statementCount++
	}
	return analysis
}

func validateSingleStatement(analysis sqlAnalysis) error {
	if analysis.invalidStatement {
		return errors.New("query must contain a single SQL statement")
	}
	if analysis.statementCount == 0 {
		return errors.New("enter a read-only SELECT query")
	}
	if analysis.statementCount > 1 {
		return errors.New("query must contain a single SQL statement")
	}
	return nil
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
		return map[string]struct{}{
			"load_file": {},
		}
	case "sqlserver":
		return map[string]struct{}{
			"openquery":                  {},
			"openrowset":                 {},
			"opendatasource":             {},
			"xp_cmdshell":                {},
			"sp_execute_external_script": {},
		}
	case "duckdb":
		return map[string]struct{}{
			"httpfs":  {},
			"install": {},
			"load":    {},
		}
	case "sqlite":
		return map[string]struct{}{
			"load_extension": {},
		}
	default:
		return nil
	}
}
