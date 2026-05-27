package datasets

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const nativeDatasetSQLEngine = "native-dataset-sql"

var blockedSQLPattern = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|alter|create|truncate|replace|merge|attach|detach|vacuum|pragma)\b`)

func (s *Service) QuerySQL(root string, relPath string, sqlText string) (SQLResult, error) {
	started := time.Now().UTC()
	parsed, err := parseDatasetSQL(sqlText, relPath)
	if err != nil {
		return SQLResult{}, err
	}
	result, err := s.Query(root, relPath, parsed.FilterQuery())
	if err != nil {
		return SQLResult{}, err
	}
	if len(parsed.Columns) > 0 {
		result, err = projectQueryResult(result, parsed.Columns)
		if err != nil {
			return SQLResult{}, err
		}
	}
	completed := time.Now().UTC()
	return SQLResult{
		QueryResult: result,
		SQL:         strings.TrimSpace(sqlText),
		Engine:      nativeDatasetSQLEngine,
		Plan:        datasetSQLPlan(result, parsed),
		StartedAt:   started,
		CompletedAt: completed,
		DurationMs:  completed.Sub(started).Milliseconds(),
	}, nil
}

type parsedDatasetSQL struct {
	Columns []string
	Source  string
	Where   string
	OrderBy string
	Order   string
	Limit   int
}

func parseDatasetSQL(sqlText string, relPath string) (parsedDatasetSQL, error) {
	sqlText = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sqlText), ";"))
	if sqlText == "" {
		return parsedDatasetSQL{}, errors.New("SQL text is required")
	}
	if blockedSQLPattern.MatchString(sqlText) {
		return parsedDatasetSQL{}, errors.New("native dataset SQL is read-only and accepts SELECT only")
	}
	lower := strings.ToLower(sqlText)
	if !strings.HasPrefix(lower, "select ") {
		return parsedDatasetSQL{}, errors.New("native dataset SQL must start with SELECT")
	}
	fromIndex := strings.Index(lower, " from ")
	if fromIndex < 0 {
		return parsedDatasetSQL{}, errors.New("native dataset SQL requires FROM")
	}
	selectPart := strings.TrimSpace(sqlText[len("select "):fromIndex])
	if selectPart == "" {
		return parsedDatasetSQL{}, errors.New("native dataset SQL requires selected columns")
	}
	rest := strings.TrimSpace(sqlText[fromIndex+len(" from "):])
	source, rest := consumeSQLToken(rest)
	if source == "" {
		return parsedDatasetSQL{}, errors.New("native dataset SQL requires a source after FROM")
	}
	if !sourceMatchesSelectedPath(source, relPath) {
		return parsedDatasetSQL{}, fmt.Errorf("native dataset SQL can only query the selected dataset %q", relPath)
	}
	parsed := parsedDatasetSQL{Source: source}
	if strings.TrimSpace(selectPart) != "*" {
		parsed.Columns = splitSQLColumns(selectPart)
		if len(parsed.Columns) == 0 {
			return parsedDatasetSQL{}, errors.New("native dataset SQL did not find selectable columns")
		}
	}
	if err := parseSQLClauses(rest, &parsed); err != nil {
		return parsedDatasetSQL{}, err
	}
	return parsed, nil
}

func parseSQLClauses(rest string, parsed *parsedDatasetSQL) error {
	rest = strings.TrimSpace(rest)
	for rest != "" {
		lower := strings.ToLower(rest)
		switch {
		case strings.HasPrefix(lower, "where "):
			value, remaining := consumeUntilClause(rest[len("where "):], []string{" order by ", " limit "})
			parsed.Where = strings.TrimSpace(value)
			if hasCompoundWhere(parsed.Where) {
				return errors.New("native dataset SQL supports one WHERE predicate; compound AND/OR filters require the future SQL notebook engine")
			}
			rest = strings.TrimSpace(remaining)
		case strings.HasPrefix(lower, "order by "):
			value, remaining := consumeUntilClause(rest[len("order by "):], []string{" limit "})
			fields := strings.Fields(value)
			if len(fields) == 0 {
				return errors.New("ORDER BY requires a column")
			}
			parsed.OrderBy = cleanSQLIdentifier(fields[0])
			if len(fields) > 1 {
				order := strings.ToLower(fields[1])
				if order != "asc" && order != "desc" {
					return errors.New("ORDER BY direction must be ASC or DESC")
				}
				parsed.Order = order
			}
			rest = strings.TrimSpace(remaining)
		case strings.HasPrefix(lower, "limit "):
			fields := strings.Fields(rest[len("limit "):])
			if len(fields) == 0 {
				return errors.New("LIMIT requires a positive integer")
			}
			limit, err := strconv.Atoi(fields[0])
			if err != nil || limit <= 0 {
				return errors.New("LIMIT requires a positive integer")
			}
			parsed.Limit = limit
			if len(fields) > 1 {
				return errors.New("unexpected SQL after LIMIT")
			}
			rest = ""
		default:
			return errors.New("native dataset SQL supports WHERE, ORDER BY, and LIMIT clauses only")
		}
	}
	return nil
}

func (p parsedDatasetSQL) FilterQuery() string {
	parts := []string{}
	if p.Where != "" {
		parts = append(parts, normalizeSQLWhere(p.Where))
	}
	if p.OrderBy != "" {
		order := "asc"
		if p.Order != "" {
			order = p.Order
		}
		parts = append(parts, "order by "+p.OrderBy+" "+order)
	}
	if p.Limit > 0 {
		parts = append(parts, "limit "+strconv.Itoa(p.Limit))
	}
	return strings.Join(parts, " ")
}

func normalizeSQLWhere(where string) string {
	where = strings.TrimSpace(where)
	where = strings.ReplaceAll(where, "<>", "!=")
	where = strings.Join(strings.Fields(where), " ")
	for _, operator := range []string{">=", "<=", "!=", "=", ">", "<"} {
		if left, right, ok := strings.Cut(where, operator); ok {
			return cleanSQLIdentifier(left) + operator + strings.Trim(strings.TrimSpace(right), `"'`)
		}
	}
	return where
}

func hasCompoundWhere(where string) bool {
	return regexp.MustCompile(`(?i)\s+(and|or)\s+`).MatchString(where)
}

func consumeSQLToken(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	if value[0] == '"' || value[0] == '\'' {
		quote := value[0]
		for index := 1; index < len(value); index++ {
			if value[index] == quote {
				return value[1:index], value[index+1:]
			}
		}
		return strings.Trim(value[1:], `"'`), ""
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", ""
	}
	return cleanSQLIdentifier(fields[0]), strings.TrimSpace(value[len(fields[0]):])
}

func consumeUntilClause(value string, clauses []string) (string, string) {
	lower := strings.ToLower(value)
	best := -1
	for _, clause := range clauses {
		if index := strings.Index(lower, clause); index >= 0 && (best < 0 || index < best) {
			best = index
		}
	}
	if best < 0 {
		return value, ""
	}
	return value[:best], value[best:]
}

func splitSQLColumns(value string) []string {
	parts := strings.Split(value, ",")
	columns := []string{}
	for _, part := range parts {
		column := cleanSQLIdentifier(part)
		if column != "" {
			columns = append(columns, column)
		}
	}
	return columns
}

func cleanSQLIdentifier(value string) string {
	return strings.Trim(strings.TrimSpace(value), "`\"'[]")
}

func sourceMatchesSelectedPath(source string, relPath string) bool {
	source = strings.Trim(strings.ToLower(filepath.ToSlash(source)), "`\"'[]")
	relPath = strings.ToLower(filepath.ToSlash(relPath))
	base := strings.ToLower(filepath.Base(relPath))
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	return source == "dataset" || source == "this" || source == relPath || source == base || source == stem
}

func projectQueryResult(result QueryResult, columns []string) (QueryResult, error) {
	indexes := []int{}
	projectedColumns := []string{}
	missingColumns := []string{}
	for _, column := range columns {
		index := columnIndexByName(result.Columns, column)
		if index < 0 {
			missingColumns = append(missingColumns, column)
			continue
		}
		indexes = append(indexes, index)
		projectedColumns = append(projectedColumns, result.Columns[index])
	}
	if len(missingColumns) > 0 {
		return QueryResult{}, fmt.Errorf("projection column(s) not found: %s", strings.Join(missingColumns, ", "))
	}
	rows := make([][]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		projected := make([]string, len(indexes))
		for i, index := range indexes {
			projected[i] = valueAt(row, index)
		}
		rows = append(rows, projected)
	}
	result.Columns = projectedColumns
	result.Rows = rows
	return result, nil
}

func datasetSQLPlan(result QueryResult, parsed parsedDatasetSQL) []string {
	plan := []string{
		"Validate SELECT-only native dataset SQL.",
		"Read selected dataset through preview-safe bounded loader.",
		fmt.Sprintf("Scan %d loaded rows from %s.", result.TotalRows, result.RelPath),
	}
	if parsed.Where != "" {
		plan = append(plan, "Apply WHERE predicate through bounded row filter.")
	}
	if parsed.OrderBy != "" {
		plan = append(plan, "Order rows by "+parsed.OrderBy+".")
	}
	if parsed.Limit > 0 {
		plan = append(plan, fmt.Sprintf("Limit result to %d row(s).", parsed.Limit))
	}
	if len(parsed.Columns) > 0 {
		plan = append(plan, "Project requested columns.")
	}
	plan = append(plan, fmt.Sprintf("Return %d shown row(s).", len(result.Rows)))
	return plan
}
