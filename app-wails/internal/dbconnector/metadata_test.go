package dbconnector

import "testing"

func TestQuoteConnectorIdentifiers(t *testing.T) {
	cases := map[string]struct {
		actual string
		want   string
	}{
		"double quote": {
			actual: quoteDoubleIdent(`user"data`),
			want:   `"user""data"`,
		},
		"backtick": {
			actual: quoteBacktickIdent("user`data"),
			want:   "`user``data`",
		},
		"sql server": {
			actual: quoteSQLServerIdent("user]data"),
			want:   "[user]]data]",
		},
		"postgres public table": {
			actual: quotePostgresQualifiedName("public", `user"data`),
			want:   `"public"."user""data"`,
		},
		"postgres schema table": {
			actual: quotePostgresQualifiedName("sales", "orders"),
			want:   `"sales"."orders"`,
		},
		"mysql qualified": {
			actual: quoteMySQLQualifiedName("analytics.orders"),
			want:   "`analytics`.`orders`",
		},
		"sql server qualified": {
			actual: quoteSQLServerQualifiedName("dbo.orders"),
			want:   "[dbo].[orders]",
		},
	}
	for name, testCase := range cases {
		if testCase.actual != testCase.want {
			t.Fatalf("%s = %q, want %q", name, testCase.actual, testCase.want)
		}
	}
}

func TestSplitQualifiedConnectorName(t *testing.T) {
	parts := splitQualifiedConnectorName("  analytics . public . orders  ")
	if len(parts) != 3 || parts[0] != "analytics" || parts[1] != "public" || parts[2] != "orders" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
}
