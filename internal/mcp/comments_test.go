package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func TestCommentToolsLifecycle(t *testing.T) {
	s, store := newTestServer(t)
	p, err := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "CMT"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	tk, err := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	res, err := s.callTool("add_comment", json.RawMessage(`{"ticketId":"`+tk.ID+`","body":"agent finding"}`))
	if err != nil {
		t.Fatalf("add_comment: %v", err)
	}
	c, ok := res.(*models.Comment)
	if !ok {
		t.Fatalf("add_comment returned %#v", res)
	}
	if c.Author != "agent" || c.Body != "agent finding" {
		t.Fatalf("unexpected comment %+v", c)
	}

	got, err := s.callTool("get_ticket", json.RawMessage(`{"id":"`+tk.ID+`"}`))
	if err != nil {
		t.Fatalf("get_ticket: %v", err)
	}
	if len(got.(*models.Ticket).Comments) != 1 {
		t.Fatalf("get_ticket should include the comment")
	}

	if _, err := s.callTool("add_comment", json.RawMessage(`{"ticketId":"`+tk.ID+`"}`)); err == nil || !strings.Contains(err.Error(), "body") {
		t.Fatalf("expected body-required error, got %v", err)
	}

	if _, err := s.callTool("delete_comment", json.RawMessage(`{"id":"`+c.ID+`"}`)); err != nil {
		t.Fatalf("delete_comment: %v", err)
	}
	if _, err := s.callTool("delete_comment", json.RawMessage(`{"id":"`+c.ID+`"}`)); err == nil {
		t.Fatalf("expected error deleting a missing comment")
	}
}
