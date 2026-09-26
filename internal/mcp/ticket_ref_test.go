package mcp

import (
	"encoding/json"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func TestGetTicketToolAcceptsKeyAndBoardLink(t *testing.T) {
	s, store := newTestServer(t)
	p, err := store.CreateProject(models.CreateProjectRequest{Name: "TCA", Prefix: "TCA"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	created, err := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "Dashboard"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	for _, ref := range []string{
		created.ID,
		"TCA-1",
		"tca-1",
		"http://localhost:3010/?ticket=TCA-1",
		" http://localhost:3010/?ticket=TCA-1 ",
	} {
		res, err := s.callTool("get_ticket", json.RawMessage(`{"id":"`+ref+`"}`))
		if err != nil {
			t.Errorf("get_ticket %q: %v", ref, err)
			continue
		}
		if got := res.(*models.Ticket); got.ID != created.ID {
			t.Errorf("get_ticket %q returned %s, want %s", ref, got.ID, created.ID)
		}
	}

	if _, err := s.callTool("get_ticket", json.RawMessage(`{"id":"TCA-99"}`)); err == nil {
		t.Error("get_ticket TCA-99: want not-found error, got nil")
	}
}
