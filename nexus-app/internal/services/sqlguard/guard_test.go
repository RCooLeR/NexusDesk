package sqlguard

import (
	"strings"
	"testing"
)

func TestNormalizeReadOnlyIgnoresBlockedWordsInLiteralsAndComments(t *testing.T) {
	allowed := []string{
		"select 'delete from users' as note",
		"select \"drop\" as keyword_name",
		"select * from [update]",
		"select $$insert;update$$ as body",
		"/* delete from users */ select id from users",
		"-- drop table users\nselect id from users",
	}
	for _, query := range allowed {
		if _, err := NormalizeReadOnly(query, Options{AllowWith: true}); err != nil {
			t.Fatalf("expected query to pass: %s (%v)", query, err)
		}
	}
}

func TestNormalizeReadOnlyRejectsMutationsAndMultipleStatements(t *testing.T) {
	blocked := []string{
		"delete from users",
		"select * from users; select * from accounts",
		"select * from users;;",
		";select * from users",
		"select id into backup_users from users",
		"pragma table_info(users)",
	}
	for _, query := range blocked {
		if _, err := NormalizeReadOnly(query, Options{AllowWith: true}); err == nil {
			t.Fatalf("expected query to be blocked: %s", query)
		}
	}
}

func TestNormalizeReadOnlyWithOption(t *testing.T) {
	if _, err := NormalizeReadOnly("with q as (select 1) select * from q", Options{AllowWith: false}); err == nil || !strings.Contains(err.Error(), "SELECT") {
		t.Fatalf("expected WITH to be rejected when disabled, got %v", err)
	}
	if _, err := NormalizeReadOnly("with q as (select 1) select * from q", Options{AllowWith: true}); err != nil {
		t.Fatalf("expected WITH to pass when enabled: %v", err)
	}
}

func TestNormalizeReadOnlyBlocksEngineSpecificTokens(t *testing.T) {
	if _, err := NormalizeReadOnly("select load_extension('mod_spatialite')", Options{Kind: "sqlite", AllowWith: true}); err == nil {
		t.Fatal("expected sqlite load_extension to be blocked")
	}
	if _, err := NormalizeReadOnly(`select "load_extension" as keyword_name`, Options{Kind: "sqlite", AllowWith: true}); err != nil {
		t.Fatalf("expected quoted engine token to pass: %v", err)
	}
}

func TestStripCommentsPreservesQuotedContent(t *testing.T) {
	query := "/* delete */ select '-- drop' as note, $$/* keep */$$ as body from dataset -- update\nwhere name = 'a /* b */'"
	stripped := StripComments(query)
	if strings.Contains(stripped, "/* delete */") || strings.Contains(stripped, "-- update") {
		t.Fatalf("expected SQL comments to be removed: %q", stripped)
	}
	for _, preserved := range []string{"'-- drop'", "$$/* keep */$$", "'a /* b */'"} {
		if !strings.Contains(stripped, preserved) {
			t.Fatalf("expected quoted content %q to be preserved in %q", preserved, stripped)
		}
	}
}

func FuzzNormalizeReadOnly(f *testing.F) {
	seeds := []string{
		"select * from dataset",
		"select 'delete;drop' as note",
		"/* update */ select id from dataset",
		"select $$insert; update$$ as body",
		"select * from dataset; select * from dataset",
		";select * from dataset",
		"with rows as (select 1) select * from rows",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, query string) {
		_, _ = NormalizeReadOnly(query, Options{
			UnsupportedMessage: "unsupported",
			BlockedMessage:     "blocked",
			EmptyMessage:       "empty",
			Kind:               "sqlite",
			AllowWith:          true,
		})
		_ = StripComments(query)
	})
}
