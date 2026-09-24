package db

import (
	"reflect"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func newTitledTicket(t *testing.T, s *Store, project *models.Project, title string) *models.Ticket {
	t.Helper()

	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: title})
	if err != nil {
		t.Fatalf("creating ticket %s: %v", title, err)
	}
	return ticket
}

func moveTo(t *testing.T, s *Store, ticket *models.Ticket, status string) {
	t.Helper()

	if _, err := s.MoveTicket(ticket.ID, models.MoveTicketRequest{Status: status}); err != nil {
		t.Fatalf("moving %s to %s: %v", ticket.Title, status, err)
	}
}

func columnTitles(t *testing.T, s *Store, status string) []string {
	t.Helper()

	tickets, err := s.ListTickets(models.TicketFilter{Status: status})
	if err != nil {
		t.Fatalf("listing %s tickets: %v", status, err)
	}
	titles := []string{}
	for _, ticket := range tickets {
		titles = append(titles, ticket.Title)
	}
	return titles
}

func assertColumn(t *testing.T, s *Store, status string, want []string) {
	t.Helper()

	if got := columnTitles(t, s, status); !reflect.DeepEqual(got, want) {
		t.Errorf("%s column = %v, want %v", status, got, want)
	}
}

func TestMoveTicketIntoDonePlacesItOnTop(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTitledTicket(t, s, project, "A")
	b := newTitledTicket(t, s, project, "B")
	c := newTitledTicket(t, s, project, "C")

	moveTo(t, s, a, "done")
	moveTo(t, s, b, "done")
	moveTo(t, s, c, "done")

	assertColumn(t, s, "done", []string{"C", "B", "A"})
}

func TestMoveTicketAlreadyInDoneKeepsItsPlace(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTitledTicket(t, s, project, "A")
	b := newTitledTicket(t, s, project, "B")
	moveTo(t, s, a, "done")
	moveTo(t, s, b, "done")

	// A drop inside the Done column re-sends the move; it must not reorder.
	moveTo(t, s, a, "done")

	assertColumn(t, s, "done", []string{"B", "A"})
}

func TestUpdateTicketStatusToDonePlacesItOnTop(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTitledTicket(t, s, project, "A")
	b := newTitledTicket(t, s, project, "B")
	moveTo(t, s, a, "done")

	done := "done"
	if _, err := s.UpdateTicket(b.ID, models.UpdateTicketRequest{Status: &done}); err != nil {
		t.Fatalf("updating status: %v", err)
	}

	assertColumn(t, s, "done", []string{"B", "A"})
}

func TestUpdateTicketAlreadyInDoneKeepsItsPlace(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTitledTicket(t, s, project, "A")
	b := newTitledTicket(t, s, project, "B")
	moveTo(t, s, a, "done")
	moveTo(t, s, b, "done")

	// The ticket panel re-sends the current status when saving other edits.
	done, title := "done", "A edited"
	if _, err := s.UpdateTicket(a.ID, models.UpdateTicketRequest{Status: &done, Title: &title}); err != nil {
		t.Fatalf("updating ticket: %v", err)
	}

	assertColumn(t, s, "done", []string{"B", "A edited"})
}

func TestReopenedTicketReturnsToTopOfDone(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTitledTicket(t, s, project, "A")
	b := newTitledTicket(t, s, project, "B")
	moveTo(t, s, a, "done")
	moveTo(t, s, b, "done")

	moveTo(t, s, a, "in_progress")
	assertColumn(t, s, "done", []string{"B"})
	moveTo(t, s, a, "done")

	assertColumn(t, s, "done", []string{"A", "B"})
}

func TestCreateTicketInDonePlacesItOnTop(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTitledTicket(t, s, project, "A")
	moveTo(t, s, a, "done")

	if _, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "B", Status: "done"}); err != nil {
		t.Fatalf("creating done ticket: %v", err)
	}

	assertColumn(t, s, "done", []string{"B", "A"})
}

func TestProjectMoveIntoDonePlacesItOnTop(t *testing.T) {
	s := newTestStore(t)
	source := newTestProject(t, s)
	target := newTestProjectWithPrefix(t, s, "Other", "OTH")
	a := newTitledTicket(t, s, target, "A")
	b := newTitledTicket(t, s, source, "B")
	moveTo(t, s, a, "done")

	done := "done"
	if _, err := s.UpdateTicket(b.ID, models.UpdateTicketRequest{ProjectID: &target.ID, Status: &done}); err != nil {
		t.Fatalf("moving ticket to project and done: %v", err)
	}

	assertColumn(t, s, "done", []string{"B", "A"})
}

func TestMoveTicketIntoOtherColumnsStillAppends(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTitledTicket(t, s, project, "A")
	b := newTitledTicket(t, s, project, "B")

	moveTo(t, s, a, "in_review")
	moveTo(t, s, b, "in_review")

	assertColumn(t, s, "in_review", []string{"A", "B"})
}

func TestDoneBackfillMigrationOrdersByMostRecentlyUpdated(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	oldest := newTitledTicket(t, s, project, "oldest")
	newest := newTitledTicket(t, s, project, "newest")
	middle := newTitledTicket(t, s, project, "middle")
	open := newTitledTicket(t, s, project, "open")

	// Recreate pre-migration data: Done in creation order, with updated_at in
	// the format Go's time.Time is stored as.
	for _, row := range []struct {
		ticket    *models.Ticket
		status    string
		position  float64
		updatedAt string
	}{
		{oldest, "done", 1000, "2026-09-18 14:53:12.97395 -0500 CDT m=+119914.251853126"},
		{newest, "done", 2000, "2026-09-24 14:37:43.711641 -0500 CDT m=+0.003726626"},
		{middle, "done", 3000, "2026-09-24 11:17:28.5 -0500 CDT m=+0.003789168"},
		{open, "todo", 4000, "2026-09-25 09:00:00 -0500 CDT"},
	} {
		if _, err := s.db.Exec("UPDATE tickets SET status=?, position=?, updated_at=? WHERE id=?",
			row.status, row.position, row.updatedAt, row.ticket.ID); err != nil {
			t.Fatalf("seeding %s: %v", row.ticket.Title, err)
		}
	}
	if _, err := s.db.Exec("DELETE FROM schema_migrations WHERE version = '004_order_done_by_recent.sql'"); err != nil {
		t.Fatalf("resetting migration: %v", err)
	}

	if err := runMigrations(s.db); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	assertColumn(t, s, "done", []string{"newest", "middle", "oldest"})
	assertColumn(t, s, "todo", []string{"open"})

	// A ticket finished after the backfill still lands on top.
	moveTo(t, s, open, "done")
	assertColumn(t, s, "done", []string{"open", "newest", "middle", "oldest"})
}
