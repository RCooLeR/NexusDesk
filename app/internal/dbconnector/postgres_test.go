package dbconnector

import (
	"strings"
	"testing"

	"NexusAugenticStudio/internal/storage"
)

func TestNormalizeReadOnlyConnectorSQL(t *testing.T) {
	query, err := normalizeReadOnlyConnectorSQL("select * from customers;")
	if err != nil {
		t.Fatalf("normalizeReadOnlyConnectorSQL returned error: %v", err)
	}
	if query != "select * from customers" {
		t.Fatalf("unexpected normalized query: %q", query)
	}

	if _, err := normalizeReadOnlyConnectorSQL("delete from customers"); err == nil {
		t.Fatal("expected mutating query to be rejected")
	}
	if _, err := normalizeReadOnlyConnectorSQL("select 1; select 2"); err == nil {
		t.Fatal("expected multi-statement query to be rejected")
	}
}

func TestPostgresDSNRedactsThroughURL(t *testing.T) {
	dsn := postgresDSN(storage.ConnectorProfile{
		Kind:           "postgres",
		Host:           "db.example.test",
		Port:           5432,
		Database:       "analytics",
		Username:       "analyst",
		Password:       "secret/value",
		SSLMode:        "require",
		TimeoutSeconds: 12,
	}, 12)
	if !strings.Contains(dsn, "postgres://analyst:secret%2Fvalue@db.example.test:5432/analytics") {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
	if !strings.Contains(dsn, "sslmode=require") || !strings.Contains(dsn, "connect_timeout=12") {
		t.Fatalf("expected ssl and timeout query params, got %s", dsn)
	}
}

func TestInferredConnectorRelationships(t *testing.T) {
	relationships := inferredConnectorRelationships([]ConnectorTable{
		{
			Name: "public.customers",
			Columns: []ConnectorColumn{
				{Name: "id", Type: "integer", PrimaryKey: true},
			},
		},
		{
			Name: "orders",
			Columns: []ConnectorColumn{
				{Name: "id", Type: "integer", PrimaryKey: true},
				{Name: "customer_id", Type: "integer"},
			},
		},
	})
	if len(relationships) != 1 {
		t.Fatalf("expected one inferred relationship, got %#v", relationships)
	}
	relationship := relationships[0]
	if relationship.FromTable != "orders" || relationship.FromColumn != "customer_id" || relationship.ToTable != "public.customers" || relationship.ToColumn != "id" {
		t.Fatalf("unexpected relationship: %#v", relationship)
	}
}

func TestRequirePostgresProfile(t *testing.T) {
	err := requirePostgresProfile(storage.ConnectorProfile{ID: "mysql", Kind: "mysql", Host: "db", Database: "app"})
	if err == nil || !strings.Contains(err.Error(), "not PostgreSQL") {
		t.Fatalf("expected kind validation error, got %v", err)
	}
	err = requirePostgresProfile(storage.ConnectorProfile{ID: "pg", Kind: "postgres", Host: "db", Database: "app"})
	if err != nil {
		t.Fatalf("expected valid profile, got %v", err)
	}
}
