package db

import (
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func TestResolveTicketIDAcceptsIDAndDisplayKey(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s) // prefix TST
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "Ticket"})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	for _, ref := range []string{ticket.ID, "TST-1", "tst-1"} {
		got, err := s.ResolveTicketID(ref)
		if err != nil {
			t.Errorf("ResolveTicketID(%q): %v", ref, err)
			continue
		}
		if got != ticket.ID {
			t.Errorf("ResolveTicketID(%q) = %q, want %q", ref, got, ticket.ID)
		}
	}
}

func TestResolveTicketIDUnknownReturnsErrTicketNotFound(t *testing.T) {
	s := newTestStore(t)
	newTestProject(t, s)
	for _, ref := range []string{"TST-99", "NOPE-1", "not-a-key", "01ZZZZZZZZZZZZZZZZZZZZZZZZ"} {
		_, err := s.ResolveTicketID(ref)
		if !errors.Is(err, ErrTicketNotFound) {
			t.Errorf("ResolveTicketID(%q) error = %v, want ErrTicketNotFound", ref, err)
		}
	}
}

func TestResolveTicketIDRejectsAmbiguousDisplayKey(t *testing.T) {
	s := newTestStore(t)
	upper, err := s.CreateProject(models.CreateProjectRequest{Name: "Upper", Prefix: "DUP"})
	if err != nil {
		t.Fatalf("creating upper-case project: %v", err)
	}
	lower, err := s.CreateProject(models.CreateProjectRequest{Name: "Lower", Prefix: "dup"})
	if err != nil {
		t.Fatalf("creating lower-case project: %v", err)
	}
	for _, project := range []*models.Project{upper, lower} {
		if _, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: project.Name}); err != nil {
			t.Fatalf("creating ticket for %s: %v", project.Name, err)
		}
	}

	id, err := s.ResolveTicketID("DuP-1")
	if !errors.Is(err, ErrTicketReferenceAmbiguous) || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ResolveTicketID(DuP-1) = %q, %v; want an ambiguity error", id, err)
	}
	if id != "" {
		t.Fatalf("ambiguous reference selected ticket %q", id)
	}
}

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

func TestUpdateTicketWithLabelDeltaPreservesLabelAddedDuringUpdate(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	labelA := newTestLabel(t, s, "A")
	labelB := newTestLabel(t, s, "B")
	labelC := newTestLabel(t, s, "C")
	ticket, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: project.ID,
		Title:     "Original",
		Labels:    []string{labelA.ID},
	})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	// Model a label write interleaved after the field update begins. A stale
	// read-modify-replace implementation would delete B when applying its
	// snapshot; an atomic SQL delta preserves it.
	_, err = s.db.Exec(`CREATE TRIGGER attach_label_during_ticket_update
		AFTER UPDATE ON tickets
		BEGIN
			INSERT OR IGNORE INTO ticket_labels (ticket_id, label_id)
			VALUES (NEW.id, '` + labelB.ID + `');
		END`)
	if err != nil {
		t.Fatalf("creating interleaved label trigger: %v", err)
	}

	newTitle := "Renamed"
	updated, err := s.UpdateTicketWithLabelDelta(
		ticket.ID,
		models.UpdateTicketRequest{Title: &newTitle},
		[]string{labelC.ID},
		[]string{labelA.ID},
	)
	if err != nil {
		t.Fatalf("updating ticket with label delta: %v", err)
	}
	if updated.Title != newTitle {
		t.Fatalf("title = %q, want %q", updated.Title, newTitle)
	}
	got := labelIDs(updated.Labels)
	sort.Strings(got)
	want := []string{labelB.ID, labelC.ID}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("labels = %v, want interleaved B plus delta C: %v", got, want)
	}
}

func TestUpdateTicketWithLabelDeltaRollsBackFieldsOnInvalidLabel(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	label := newTestLabel(t, s, "Existing")
	ticket, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: project.ID,
		Title:     "Original",
		Labels:    []string{label.ID},
	})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	newTitle := "Must roll back"
	_, err = s.UpdateTicketWithLabelDelta(
		ticket.ID,
		models.UpdateTicketRequest{Title: &newTitle},
		[]string{"DOES-NOT-EXIST"},
		nil,
	)
	if err == nil {
		t.Fatal("delta update with an unknown label returned nil error")
	}

	got, err := s.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("getting ticket after failed delta: %v", err)
	}
	if got.Title != "Original" {
		t.Fatalf("failed label delta changed title to %q", got.Title)
	}
	if ids := labelIDs(got.Labels); !reflect.DeepEqual(ids, []string{label.ID}) {
		t.Fatalf("failed label delta changed labels to %v", ids)
	}
}

func TestOpenAtSetsBusyTimeout(t *testing.T) {
	s := newTestStore(t)
	var ms int
	if err := s.db.QueryRow("PRAGMA busy_timeout").Scan(&ms); err != nil {
		t.Fatalf("reading busy_timeout: %v", err)
	}
	if ms != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", ms)
	}
}

func TestResolveLabelIDsAcceptsIDAndCaseInsensitiveName(t *testing.T) {
	s := newTestStore(t)
	blocked := newTestLabel(t, s, "Blocked")
	review := newTestLabel(t, s, "Needs Review")

	got, err := s.ResolveLabelIDs([]string{blocked.ID, "needs review", "BLOCKED"})
	if err != nil {
		t.Fatalf("ResolveLabelIDs: %v", err)
	}
	want := []string{blocked.ID, review.ID, blocked.ID}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveLabelIDs = %v, want %v", got, want)
	}

	_, err = s.ResolveLabelIDs([]string{"Blocked", "Nope"})
	if !errors.Is(err, ErrLabelNotFound) {
		t.Fatalf("unknown label error = %v, want ErrLabelNotFound", err)
	}
}

func TestResolveLabelIDsRejectsAmbiguousName(t *testing.T) {
	s := newTestStore(t)
	first := newTestLabel(t, s, "Blocked")
	newTestLabel(t, s, "blocked")

	_, err := s.ResolveLabelIDs([]string{"BLOCKED"})
	if !errors.Is(err, ErrLabelReferenceAmbiguous) {
		t.Fatalf("duplicate-name resolve error = %v, want ErrLabelReferenceAmbiguous", err)
	}

	// An exact ID still resolves even when the name is ambiguous.
	ids, err := s.ResolveLabelIDs([]string{first.ID})
	if err != nil || len(ids) != 1 || ids[0] != first.ID {
		t.Fatalf("resolve by ID with duplicate names = %v, %v", ids, err)
	}
}
