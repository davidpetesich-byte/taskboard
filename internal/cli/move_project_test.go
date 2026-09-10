package cli

import (
	"strings"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

// newMoveEnv creates two projects and one ticket in the source project,
// returning the env plus both project IDs.
func newMoveEnv(t *testing.T) (*cliEnv, string, string) {
	t.Helper()
	e := newCLIEnv(t)

	var source, target models.Project
	e.runJSON(&source, "project", "create", "Source", "--prefix", "SRC")
	e.runJSON(&target, "project", "create", "Target", "--prefix", "TGT")

	var ticket models.Ticket
	e.runJSON(&ticket, "ticket", "create", "--project", source.ID, "--title", "Move me")
	return e, source.ID, target.ID
}

func TestTicketUpdateProjectMovesTicketByPrefix(t *testing.T) {
	e, _, _ := newMoveEnv(t)

	var moved models.Ticket
	e.runJSON(&moved, "ticket", "update", "SRC-1", "--project", "TGT")

	if moved.ProjectPrefix != "TGT" {
		t.Errorf("moved.ProjectPrefix = %q, want %q", moved.ProjectPrefix, "TGT")
	}
	if got := moved.DisplayKey(); got != "TGT-1" {
		t.Errorf("moved.DisplayKey() = %q, want %q", got, "TGT-1")
	}

	if _, err := e.run("ticket", "get", "SRC-1"); err == nil {
		t.Error("getting the old key SRC-1 succeeded, want not-found after the move")
	}
}

func TestTicketUpdateProjectAcceptsProjectID(t *testing.T) {
	e, _, targetID := newMoveEnv(t)

	var moved models.Ticket
	e.runJSON(&moved, "ticket", "update", "SRC-1", "--project", targetID)

	if moved.ProjectID != targetID {
		t.Errorf("moved.ProjectID = %q, want %q", moved.ProjectID, targetID)
	}
}

func TestTicketUpdateProjectRejectsUnknownProject(t *testing.T) {
	e, sourceID, _ := newMoveEnv(t)

	out, err := e.run("ticket", "update", "SRC-1", "--project", "NOPE")
	if err == nil {
		t.Fatalf("update with unknown project succeeded, want error; output = %s", out)
	}
	if !strings.Contains(err.Error(), "NOPE") {
		t.Errorf("error = %v, want it to name the unknown project %q", err, "NOPE")
	}

	var still models.Ticket
	e.runJSON(&still, "ticket", "get", "SRC-1")
	if still.ProjectID != sourceID {
		t.Errorf("ProjectID = %q, want unchanged %q", still.ProjectID, sourceID)
	}
}

func TestTicketUpdateWithOnlyProjectFlagIsNotNothingToUpdate(t *testing.T) {
	e, _, _ := newMoveEnv(t)

	// --project alone must satisfy the "at least one flag" guard.
	if _, err := e.run("ticket", "update", "SRC-1", "--project", "TGT"); err != nil {
		t.Fatalf("update with only --project failed: %v", err)
	}
}
