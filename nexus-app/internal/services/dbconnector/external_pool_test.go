package dbconnector

import (
	"context"
	"errors"
	"testing"
)

func TestExternalConnectorDBPoolReusesAndInvalidatesByProfileVersion(t *testing.T) {
	service := New()
	t.Cleanup(func() {
		_ = service.CloseConnectorPools()
	})
	profile := ConnectorProfile{ID: "warehouse", Name: "Warehouse", Kind: "postgres", Host: "db.example.test", Database: "app", Username: "analyst"}
	first, err := service.externalConnectorDB(profile, "sqlite", "file:pool-reuse-a?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("externalConnectorDB first returned error: %v", err)
	}
	second, err := service.externalConnectorDB(profile, "sqlite", "file:pool-reuse-a?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("externalConnectorDB second returned error: %v", err)
	}
	if first != second {
		t.Fatal("expected same profile version to reuse the pool")
	}
	replaced, err := service.externalConnectorDB(profile, "sqlite", "file:pool-reuse-b?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("externalConnectorDB replacement returned error: %v", err)
	}
	if replaced == first {
		t.Fatal("expected changed profile version to invalidate the old pool")
	}
	statuses := service.ConnectorPoolStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected one active pool, got %#v", statuses)
	}
	status := statuses[0]
	if status.ProfileID != "warehouse" || status.Kind != "postgres" || status.Driver != "sqlite" {
		t.Fatalf("unexpected pool status: %#v", status)
	}
	if status.MaxOpenConnections != defaultConnectorPoolMaxOpenConns {
		t.Fatalf("expected max open conns %d, got %#v", defaultConnectorPoolMaxOpenConns, status)
	}
}

func TestExternalConnectorDBPoolRespectsContextCancellationWhenBorrowing(t *testing.T) {
	service := New()
	t.Cleanup(func() {
		_ = service.CloseConnectorPools()
	})
	profile := ConnectorProfile{ID: "warehouse", Name: "Warehouse", Kind: "postgres"}
	db, err := service.externalConnectorDB(profile, "sqlite", "file:pool-cancel?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("externalConnectorDB returned error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := db.Conn(ctx); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled while borrowing pooled connection, got %v", err)
	}
}
