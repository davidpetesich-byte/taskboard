package db

import (
	"errors"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

// newTestProjectWithPrefix creates an additional project so move tests have a
// distinct target.
func newTestProjectWithPrefix(t *testing.T, s *Store, name, prefix string) *models.Project {
	t.Helper()

	project, err := s.CreateProject(models.CreateProjectRequest{Name: name, Prefix: prefix})
	if err != nil {
		t.Fatalf("creating project %s: %v", prefix, err)
	}
	return project
}

func TestUpdateTicketMovesToAnotherProjectAndRenumbers(t *testing.T) {
	s := newTestStore(t)
	source := newTestProject(t, s)                           // TST
	target := newTestProjectWithPrefix(t, s, "Other", "OTH") // OTH

	// Occupy OTH-1 so the moved ticket cannot keep number 1.
	if _, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: target.ID, Title: "Existing"}); err != nil {
		t.Fatalf("creating existing target ticket: %v", err)
	}
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: source.ID, Title: "Moving"})
	if err != nil {
		t.Fatalf("creating source ticket: %v", err)
	}
	if ticket.Number != 1 {
		t.Fatalf("source ticket number = %d, want 1", ticket.Number)
	}

	moved, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{ProjectID: &target.ID})
	if err != nil {
		t.Fatalf("moving ticket: %v", err)
	}
	if moved == nil {
		t.Fatal("moving ticket returned nil ticket")
	}
	if moved.ProjectID != target.ID {
		t.Errorf("moved.ProjectID = %q, want %q", moved.ProjectID, target.ID)
	}
	if moved.Number != 2 {
		t.Errorf("moved.Number = %d, want 2 (next free in target)", moved.Number)
	}
	if got := moved.DisplayKey(); got != "OTH-2" {
		t.Errorf("moved.DisplayKey() = %q, want %q", got, "OTH-2")
	}

	if _, err := s.ResolveTicketID("TST-1"); !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("ResolveTicketID(\"TST-1\") error = %v, want ErrTicketNotFound after move", err)
	}
	got, err := s.ResolveTicketID("OTH-2")
	if err != nil {
		t.Fatalf("ResolveTicketID(\"OTH-2\"): %v", err)
	}
	if got != ticket.ID {
		t.Errorf("ResolveTicketID(\"OTH-2\") = %q, want %q", got, ticket.ID)
	}
}

func TestUpdateTicketMovePreservesTitleLabelsSubtasksAndComments(t *testing.T) {
	s := newTestStore(t)
	source := newTestProject(t, s)
	target := newTestProjectWithPrefix(t, s, "Other", "OTH")
	label := newTestLabel(t, s, "urgent")

	ticket, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID:   source.ID,
		Title:       "Keeps its contents",
		Description: "body text",
		Labels:      []string{label.ID},
	})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}
	if _, err := s.AddSubtask(ticket.ID, models.CreateSubtaskRequest{Title: "step one"}); err != nil {
		t.Fatalf("adding subtask: %v", err)
	}
	if _, err := s.AddComment(ticket.ID, models.CreateCommentRequest{Author: "dave", Body: "a note"}); err != nil {
		t.Fatalf("adding comment: %v", err)
	}

	moved, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{ProjectID: &target.ID})
	if err != nil {
		t.Fatalf("moving ticket: %v", err)
	}

	if moved.Title != "Keeps its contents" {
		t.Errorf("moved.Title = %q, want %q", moved.Title, "Keeps its contents")
	}
	if moved.Description != "body text" {
		t.Errorf("moved.Description = %q, want %q", moved.Description, "body text")
	}
	if len(moved.Labels) != 1 || moved.Labels[0].ID != label.ID {
		t.Errorf("moved.Labels = %+v, want the single label %q", moved.Labels, label.ID)
	}
	if len(moved.Subtasks) != 1 || moved.Subtasks[0].Title != "step one" {
		t.Errorf("moved.Subtasks = %+v, want one subtask %q", moved.Subtasks, "step one")
	}
	if len(moved.Comments) != 1 || moved.Comments[0].Body != "a note" {
		t.Errorf("moved.Comments = %+v, want one comment %q", moved.Comments, "a note")
	}
}

func TestUpdateTicketMovePlacesTicketAtBottomOfTargetStatusColumn(t *testing.T) {
	s := newTestStore(t)
	source := newTestProject(t, s)
	target := newTestProjectWithPrefix(t, s, "Other", "OTH")

	resident, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: target.ID, Title: "Already here", Status: "todo",
	})
	if err != nil {
		t.Fatalf("creating resident ticket: %v", err)
	}
	ticket, err := s.CreateTicket(models.CreateTicketRequest{
		ProjectID: source.ID, Title: "Moving", Status: "todo",
	})
	if err != nil {
		t.Fatalf("creating source ticket: %v", err)
	}

	moved, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{ProjectID: &target.ID})
	if err != nil {
		t.Fatalf("moving ticket: %v", err)
	}
	if moved.Position <= resident.Position {
		t.Errorf("moved.Position = %v, want greater than resident position %v", moved.Position, resident.Position)
	}
}

func TestUpdateTicketWithoutProjectIDLeavesProjectAndNumberUnchanged(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "Stays put"})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	title := "Renamed"
	updated, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{Title: &title})
	if err != nil {
		t.Fatalf("updating ticket: %v", err)
	}
	if updated.ProjectID != project.ID {
		t.Errorf("updated.ProjectID = %q, want %q", updated.ProjectID, project.ID)
	}
	if updated.Number != ticket.Number {
		t.Errorf("updated.Number = %d, want %d", updated.Number, ticket.Number)
	}
}

func TestUpdateTicketMoveToSameProjectKeepsNumber(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "No-op move"})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	moved, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{ProjectID: &project.ID})
	if err != nil {
		t.Fatalf("moving ticket to same project: %v", err)
	}
	if moved.Number != ticket.Number {
		t.Errorf("moved.Number = %d, want %d (same-project move must not renumber)", moved.Number, ticket.Number)
	}
}

func TestUpdateTicketMoveToMissingProjectFails(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s)
	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "Orphan attempt"})
	if err != nil {
		t.Fatalf("creating ticket: %v", err)
	}

	missing := "does-not-exist"
	if _, err := s.UpdateTicket(ticket.ID, models.UpdateTicketRequest{ProjectID: &missing}); err == nil {
		t.Fatal("moving to a nonexistent project succeeded, want error")
	}

	after, err := s.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("re-reading ticket: %v", err)
	}
	if after.ProjectID != project.ID {
		t.Errorf("after failed move ProjectID = %q, want unchanged %q", after.ProjectID, project.ID)
	}
}

func TestResolveProjectIDAcceptsIDAndPrefix(t *testing.T) {
	s := newTestStore(t)
	project := newTestProject(t, s) // prefix TST

	for _, ref := range []string{project.ID, "TST", "tst"} {
		got, err := s.ResolveProjectID(ref)
		if err != nil {
			t.Errorf("ResolveProjectID(%q): %v", ref, err)
			continue
		}
		if got != project.ID {
			t.Errorf("ResolveProjectID(%q) = %q, want %q", ref, got, project.ID)
		}
	}
}

func TestResolveProjectIDUnknownReturnsErrProjectNotFound(t *testing.T) {
	s := newTestStore(t)
	newTestProject(t, s)

	for _, ref := range []string{"NOPE", "01ZZZZZZZZZZZZZZZZZZZZZZZZ"} {
		if _, err := s.ResolveProjectID(ref); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("ResolveProjectID(%q) error = %v, want ErrProjectNotFound", ref, err)
		}
	}
}
