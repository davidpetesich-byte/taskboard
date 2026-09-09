package db

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	database, err := OpenAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	return NewStore(database)
}

func TestOpenAtRunsMigrations(t *testing.T) {
	s := newTestStore(t)

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 migrations applied, got %d", count)
	}
}
