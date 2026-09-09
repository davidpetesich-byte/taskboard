package db

import (
	"path/filepath"
	"reflect"
	"sort"
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

func newTestLabel(t *testing.T, s *Store, name string) *models.Label {
	t.Helper()

	label, err := s.CreateLabel(models.CreateLabelRequest{Name: name, Color: "#EF4444"})
	if err != nil {
		t.Fatalf("creating label: %v", err)
	}
	return label
}

func labelIDs(labels []models.Label) []string {
	ids := make([]string, len(labels))
	for i, label := range labels {
		ids[i] = label.ID
	}
	return ids
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

func TestUpdateTicketWithUnknownLabelReturnsError(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "Ticket"})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	_, err = s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Labels: []string{"DOES-NOT-EXIST"}})
	if err == nil {
		t.Fatal("updating ticket with an unknown label returned nil error")
	}
}

func TestCreateTicketWithUnknownLabelReturnsError(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)

	_, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: project.ID,
		Title:     "Ticket",
		Labels:    []string{"DOES-NOT-EXIST"},
	})
	if err == nil {
		t.Fatal("creating ticket with an unknown label returned nil error")
	}
}

func TestLabelRoundTripReplacesNotAppends(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	labelA := newTestLabel(t, s, "A")
	labelB := newTestLabel(t, s, "B")
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "Ticket"})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	for _, tc := range []struct {
		name   string
		labels []string
		want   []string
	}{
		{name: "A", labels: []string{labelA.ID}, want: []string{labelA.ID}},
		{name: "A and B", labels: []string{labelA.ID, labelB.ID}, want: []string{labelA.ID, labelB.ID}},
		{name: "B", labels: []string{labelB.ID}, want: []string{labelB.ID}},
		{name: "empty", labels: []string{}, want: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updated, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Labels: tc.labels})
			if err != nil {
				t.Fatalf("updating ticket labels: %v", err)
			}

			got := labelIDs(updated.Labels)
			sort.Strings(got)
			sort.Strings(tc.want)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("label IDs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCreateTicketWithUnknownLabelLeavesNoOrphanTicket(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)

	_, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: project.ID,
		Title:     "Ticket",
		Labels:    []string{"DOES-NOT-EXIST"},
	})
	if err == nil {
		t.Fatal("creating ticket with an unknown label returned nil error")
	}

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM tickets WHERE project_id = ?", project.ID).Scan(&count); err != nil {
		t.Fatalf("counting tickets: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no ticket rows after failed create, got %d", count)
	}
}

func TestUpdateTicketWithUnknownLabelKeepsExistingLabels(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	labelA := newTestLabel(t, s, "A")
	ticket, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: project.ID,
		Title:     "Ticket",
		Labels:    []string{labelA.ID},
	})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	newTitle := "Renamed"
	_, err = s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{
		Title:  &newTitle,
		Labels: []string{labelA.ID, "DOES-NOT-EXIST"},
	})
	if err == nil {
		t.Fatal("updating ticket with an unknown label returned nil error")
	}

	got, err := s.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("getting ticket: %v", err)
	}
	if got.Title != "Ticket" {
		t.Errorf("title = %q, want unchanged %q", got.Title, "Ticket")
	}
	if ids := labelIDs(got.Labels); !reflect.DeepEqual(ids, []string{labelA.ID}) {
		t.Errorf("labels = %v, want unchanged %v", ids, []string{labelA.ID})
	}
}
