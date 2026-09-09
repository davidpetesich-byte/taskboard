package db

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
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

func newTestProject(t *testing.T, s *Store) *models.Project {
	t.Helper()

	project, err := s.CreateProject(models.CreateProjectRequest{Name: "Test", Prefix: "TST"})
	if err != nil {
		t.Fatalf("creating test project: %v", err)
	}
	return project
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

func TestGetBoardReturnsFiveColumnsInOrder(t *testing.T) {
	s := newTestStore(t)

	board, err := s.GetBoard("")
	if err != nil {
		t.Fatalf("getting board: %v", err)
	}

	got := make([]string, len(board.Columns))
	for i, column := range board.Columns {
		got[i] = column.Status
	}
	want := []string{"backlog", "todo", "in_progress", "in_review", "done"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("board statuses = %v, want %v", got, want)
	}
}

func TestTicketInReviewLandsInReviewColumn(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	ticket, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: project.ID,
		Title:     "Review me",
		Status:    "in_review",
	})
	if err != nil {
		t.Fatalf("creating in-review ticket: %v", err)
	}

	board, err := s.GetBoard(project.ID)
	if err != nil {
		t.Fatalf("getting project board: %v", err)
	}
	for _, column := range board.Columns {
		if column.Status != "in_review" {
			continue
		}
		if len(column.Tickets) != 1 || column.Tickets[0].ID != ticket.ID {
			t.Errorf("in_review tickets = %v, want only %s", column.Tickets, ticket.ID)
		}
		return
	}
	t.Errorf("in_review column not found")
}
