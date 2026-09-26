package db

import (
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func newTicketIn(t *testing.T, s *Store, project *models.Project, title, status string) *models.Ticket {
	t.Helper()

	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: title, Status: status})
	if err != nil {
		t.Fatalf("creating ticket %s: %v", title, err)
	}
	return ticket
}

func TestCreateTicketAppendsBelowOtherProjectsTickets(t *testing.T) {
	s := newTestStore(t)
	tca := newTestProjectWithPrefix(t, s, "TCA", "TCA")
	ait := newTestProjectWithPrefix(t, s, "AIT", "AIT")
	newTicketIn(t, s, tca, "TCA-1", "in_progress")
	newTicketIn(t, s, tca, "TCA-2", "in_progress")

	// AIT-1 has a lower number than every TCA ticket but is the newest card.
	newTicketIn(t, s, ait, "AIT-1", "in_progress")

	assertColumn(t, s, "in_progress", []string{"TCA-1", "TCA-2", "AIT-1"})
}

func TestCreateTicketAppendsBelowTicketMovedIn(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTicketIn(t, s, project, "A", "backlog")
	newTicketIn(t, s, project, "B", "backlog")
	newTicketIn(t, s, project, "C", "todo")
	moveTo(t, s, a, "todo")

	newTicketIn(t, s, project, "D", "todo")

	assertColumn(t, s, "todo", []string{"C", "A", "D"})
}

func TestUpdateTicketStatusAppendsToBottom(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	c := newTicketIn(t, s, project, "C", "backlog")
	newTicketIn(t, s, project, "A", "todo")
	newTicketIn(t, s, project, "B", "todo")

	todo := "todo"
	if _, err := s.UpdateTicket(c.ID, models.UpdateTicketRequest{Status: &todo}); err != nil {
		t.Fatalf("updating status: %v", err)
	}

	assertColumn(t, s, "todo", []string{"A", "B", "C"})
}

func TestUpdateTicketSameStatusKeepsPlace(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTicketIn(t, s, project, "A", "todo")
	newTicketIn(t, s, project, "B", "todo")

	// The ticket panel re-sends the current status when saving other edits.
	todo, title := "todo", "A edited"
	if _, err := s.UpdateTicket(a.ID, models.UpdateTicketRequest{Status: &todo, Title: &title}); err != nil {
		t.Fatalf("updating ticket: %v", err)
	}

	assertColumn(t, s, "todo", []string{"A edited", "B"})
}

func TestMoveTicketWithinColumnKeepsPlace(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTicketIn(t, s, project, "A", "todo")
	newTicketIn(t, s, project, "B", "todo")

	moveTo(t, s, a, "todo")

	assertColumn(t, s, "todo", []string{"A", "B"})
}

func TestMoveTicketToExplicitPosition(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	a := newTicketIn(t, s, project, "A", "todo")
	b := newTicketIn(t, s, project, "B", "todo")
	c := newTicketIn(t, s, project, "C", "in_progress")

	between := (a.Position + b.Position) / 2
	if _, err := s.MoveTicket(c.ID, models.MoveTicketRequest{Status: "todo", Position: &between}); err != nil {
		t.Fatalf("moving ticket: %v", err)
	}

	assertColumn(t, s, "todo", []string{"A", "C", "B"})
}

func TestProjectMoveAppendsBelowWholeColumn(t *testing.T) {
	s := newTestStore(t)
	source := newTestProject(t, s)
	target := newTestProjectWithPrefix(t, s, "Other", "OTH")
	newTicketIn(t, s, target, "T1", "todo")
	s1 := newTicketIn(t, s, source, "S1", "todo")
	newTicketIn(t, s, source, "S2", "todo")

	if _, err := s.UpdateTicket(s1.ID, models.UpdateTicketRequest{ProjectID: &target.ID}); err != nil {
		t.Fatalf("moving ticket to project: %v", err)
	}

	assertColumn(t, s, "todo", []string{"T1", "S2", "S1"})
}
