package mcp

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tcarac/taskboard/internal/db"
	"github.com/tcarac/taskboard/internal/models"
)

func newTestServer(t *testing.T) (*MCPServer, *db.Store) {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	store := db.NewStore(database)
	return NewServer(store), store
}

func findTool(t *testing.T, s *MCPServer, name string) toolDef {
	t.Helper()
	for _, td := range s.toolDefinitions() {
		if td.Name == name {
			return td
		}
	}
	t.Fatalf("tool %q not defined", name)
	return toolDef{}
}

func TestToolDefinitionsCount(t *testing.T) {
	s, _ := newTestServer(t)
	if got := len(s.toolDefinitions()); got != 26 {
		t.Fatalf("expected 26 tools, got %d", got)
	}
}

func TestStatusEnumsMatchBoardStatuses(t *testing.T) {
	s, _ := newTestServer(t)
	for _, name := range []string{"list_tickets", "create_ticket", "update_ticket", "move_ticket"} {
		got := findTool(t, s, name).InputSchema.Properties["status"].Enum
		if !reflect.DeepEqual(got, db.BoardStatuses) {
			t.Errorf("%s status enum: want %v, got %v", name, db.BoardStatuses, got)
		}
	}
}

func TestTicketToolsExposeLabelsParam(t *testing.T) {
	s, _ := newTestServer(t)
	for _, name := range []string{"create_ticket", "update_ticket"} {
		prop, ok := findTool(t, s, name).InputSchema.Properties["labels"]
		if !ok {
			t.Errorf("%s: missing labels param", name)
			continue
		}
		if prop.Type != "array" || prop.Items == nil || prop.Items.Type != "string" {
			t.Errorf("%s: labels must be an array of string, got %+v", name, prop)
		}
	}
}

func TestLabelToolsLifecycle(t *testing.T) {
	s, _ := newTestServer(t)

	created, err := s.callTool("create_label", json.RawMessage(`{"name":"Blocked","color":"#EF4444"}`))
	if err != nil {
		t.Fatalf("create_label: %v", err)
	}
	label, ok := created.(*models.Label)
	if !ok || label.Name != "Blocked" || label.Color != "#EF4444" {
		t.Fatalf("create_label returned %#v", created)
	}

	listed, err := s.callTool("list_labels", nil)
	if err != nil {
		t.Fatalf("list_labels: %v", err)
	}
	if got := len(listed.([]models.Label)); got != 1 {
		t.Fatalf("list_labels: want 1 label, got %d", got)
	}

	if _, err := s.callTool("delete_label", json.RawMessage(`{"id":"`+label.ID+`"}`)); err != nil {
		t.Fatalf("delete_label: %v", err)
	}
	listed, err = s.callTool("list_labels", nil)
	if err != nil {
		t.Fatalf("list_labels after delete: %v", err)
	}
	if got := len(listed.([]models.Label)); got != 0 {
		t.Fatalf("after delete: want 0 labels, got %d", got)
	}
}

func TestCreateLabelDefaultsColor(t *testing.T) {
	s, _ := newTestServer(t)
	created, err := s.callTool("create_label", json.RawMessage(`{"name":"Plain"}`))
	if err != nil {
		t.Fatalf("create_label: %v", err)
	}
	if got := created.(*models.Label).Color; got != "#6B7280" {
		t.Fatalf("want default color #6B7280, got %q", got)
	}
}

func TestCreateTicketToolAttachesLabels(t *testing.T) {
	s, store := newTestServer(t)
	p, err := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "P"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	created, err := s.callTool("create_label", json.RawMessage(`{"name":"Blocked"}`))
	if err != nil {
		t.Fatalf("create_label: %v", err)
	}
	label := created.(*models.Label)

	args := `{"projectId":"` + p.ID + `","title":"t","status":"in_review","labels":["` + label.ID + `"]}`
	res, err := s.callTool("create_ticket", json.RawMessage(args))
	if err != nil {
		t.Fatalf("create_ticket: %v", err)
	}
	tk := res.(*models.Ticket)
	if tk.Status != "in_review" || len(tk.Labels) != 1 || tk.Labels[0].ID != label.ID {
		t.Fatalf("want in_review with one label %s, got status=%s labels=%+v", label.ID, tk.Status, tk.Labels)
	}
}

func TestUpdateTicketToolExposesProjectIDParam(t *testing.T) {
	s, _ := newTestServer(t)
	td := findTool(t, s, "update_ticket")
	if _, ok := td.InputSchema.Properties["projectId"]; !ok {
		t.Fatalf("update_ticket does not advertise a projectId property: %+v", td.InputSchema.Properties)
	}
}

func TestUpdateTicketToolMovesTicketBetweenProjects(t *testing.T) {
	s, store := newTestServer(t)
	source, err := store.CreateProject(models.CreateProjectRequest{Name: "Source", Prefix: "SRC"})
	if err != nil {
		t.Fatalf("CreateProject source: %v", err)
	}
	target, err := store.CreateProject(models.CreateProjectRequest{Name: "Target", Prefix: "TGT"})
	if err != nil {
		t.Fatalf("CreateProject target: %v", err)
	}
	ticket, err := store.CreateTicket(models.CreateTicketRequest{ProjectID: source.ID, Title: "Move me"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	args := `{"id":"` + ticket.ID + `","projectId":"` + target.ID + `"}`
	res, err := s.callTool("update_ticket", json.RawMessage(args))
	if err != nil {
		t.Fatalf("update_ticket: %v", err)
	}
	moved := res.(*models.Ticket)
	if moved.ProjectID != target.ID {
		t.Errorf("moved.ProjectID = %q, want %q", moved.ProjectID, target.ID)
	}
	if moved.ProjectPrefix != "TGT" {
		t.Errorf("moved.ProjectPrefix = %q, want %q", moved.ProjectPrefix, "TGT")
	}
}
