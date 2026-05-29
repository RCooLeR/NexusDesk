package dbconnector

import "nexusdesk/internal/services/sqlguard"

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
	return sqlguard.NormalizeReadOnly(query, sqlguard.Options{
		UnsupportedMessage: unsupportedMessage,
		BlockedMessage:     blockedMessage,
		EmptyMessage:       "enter a read-only SELECT query",
		Kind:               kind,
		AllowWith:          true,
	})
}
