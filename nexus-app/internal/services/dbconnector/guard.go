package dbconnector

import (
	"errors"
	"strings"
	"unicode"
)

func tokenizeSQL(query string) []string {
	tokens := []string{}
	var builder strings.Builder
	var quote rune
	lineComment := false
	blockComment := false

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(builder.String()))
		builder.Reset()
	}

	runes := []rune(query)
	for index := 0; index < len(runes); index++ {
		current := runes[index]
		if lineComment {
			if current == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if current == '*' && index+1 < len(runes) && runes[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
		if quote != 0 {
			if current == quote {
				if index+1 < len(runes) && runes[index+1] == quote {
					index++
					continue
				}
				quote = 0
			}
			continue
		}

		switch current {
		case '\'', '"':
			flush()
			quote = current
		case '-':
			if index+1 < len(runes) && runes[index+1] == '-' {
				flush()
				lineComment = true
				index++
				continue
			}
			flush()
		case '/':
			if index+1 < len(runes) && runes[index+1] == '*' {
				flush()
				blockComment = true
				index++
				continue
			}
			flush()
		default:
			if unicode.IsLetter(current) || unicode.IsDigit(current) || current == '_' {
				builder.WriteRune(current)
			} else {
				flush()
			}
		}
	}
	flush()
	return tokens
}

func validateSingleStatement(query string) error {
	segments := splitTopLevelSQLStatements(query)
	if len(segments) == 0 {
		return errors.New("enter a read-only SELECT query")
	}
	nonEmptyCount := 0
	for index, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			if index < len(segments)-1 {
				return errors.New("query must contain a single SQL statement")
			}
			continue
		}
		nonEmptyCount++
		if nonEmptyCount > 1 {
			return errors.New("query must contain a single SQL statement")
		}
	}
	if nonEmptyCount == 0 {
		return errors.New("enter a read-only SELECT query")
	}
	return nil
}

func splitTopLevelSQLStatements(query string) []string {
	segments := []string{}
	var builder strings.Builder
	var quote rune
	lineComment := false
	blockComment := false

	runes := []rune(query)
	for index := 0; index < len(runes); index++ {
		current := runes[index]
		if lineComment {
			if current == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if current == '*' && index+1 < len(runes) && runes[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
		if quote != 0 {
			if current == quote {
				if index+1 < len(runes) && runes[index+1] == quote {
					builder.WriteRune(current)
					index++
					continue
				}
				quote = 0
			}
			builder.WriteRune(current)
			continue
		}

		switch current {
		case '\'':
			quote = '\''
			builder.WriteRune(current)
		case '"':
			quote = '"'
			builder.WriteRune(current)
		case '-':
			if index+1 < len(runes) && runes[index+1] == '-' {
				lineComment = true
				index++
				continue
			}
			builder.WriteRune(current)
		case '/':
			if index+1 < len(runes) && runes[index+1] == '*' {
				blockComment = true
				index++
				continue
			}
			builder.WriteRune(current)
		case ';':
			segments = append(segments, strings.TrimSpace(builder.String()))
			builder.Reset()
		default:
			builder.WriteRune(current)
		}
	}
	segments = append(segments, strings.TrimSpace(builder.String()))
	return segments
}

func containsBlockedSQL(query string) bool {
	for _, token := range tokenizeSQL(query) {
		switch token {
		case "insert", "update", "delete", "drop", "alter", "truncate", "create", "attach", "detach", "replace", "vacuum", "pragma":
			return true
		}
	}
	return false
}
