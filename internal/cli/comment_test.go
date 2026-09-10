package cli

import (
	"strings"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func TestCommentAddListDeleteRoundTrip(t *testing.T) {
	e := newCLIEnv(t)
	p := e.project()
	var tk models.Ticket
	e.runJSON(&tk, "ticket", "create", "--project", p.ID, "--title", "T")

	var c models.Comment
	e.runJSON(&c, "comment", "add", "SMK-1", "looked at rows 1-10")
	if c.Author != "cli" || c.Body != "looked at rows 1-10" || c.TicketID != tk.ID {
		t.Fatalf("unexpected comment %+v", c)
	}

	e.stdin = "# From stdin\n\n- a\n- b\n"
	var c2 models.Comment
	e.runJSON(&c2, "comment", "add", tk.ID, "--body-file", "-", "--author", "codex")
	e.stdin = ""
	if c2.Author != "codex" || !strings.HasPrefix(c2.Body, "# From stdin") {
		t.Fatalf("unexpected stdin comment %+v", c2)
	}

	var list []models.Comment
	e.runJSON(&list, "comment", "list", "SMK-1")
	if len(list) != 2 || list[0].ID != c.ID || list[1].ID != c2.ID {
		t.Fatalf("expected 2 ordered comments, got %+v", list)
	}

	out, err := e.run("ticket", "get", "SMK-1")
	if err != nil || !strings.Contains(out, "Comments:") || !strings.Contains(out, "codex") {
		t.Fatalf("ticket get should print comments: %v\n%s", err, out)
	}

	var del map[string]any
	e.runJSON(&del, "comment", "delete", c.ID)
	if del["deleted"] != true || del["id"] != c.ID {
		t.Fatalf("expected deleted true, got %v", del)
	}
	e.runJSON(&list, "comment", "list", "SMK-1")
	if len(list) != 1 || list[0].ID != c2.ID {
		t.Fatalf("expected 1 comment after delete, got %+v", list)
	}
}

func TestCommentAddRequiresExactlyOneBodySource(t *testing.T) {
	e := newCLIEnv(t)
	p := e.project()
	e.runJSON(new(models.Ticket), "ticket", "create", "--project", p.ID, "--title", "T")
	if _, err := e.run("comment", "add", "SMK-1"); err == nil {
		t.Fatal("expected error with no body")
	}
	e.stdin = "x"
	if _, err := e.run("comment", "add", "SMK-1", "inline", "--body-file", "-"); err == nil {
		t.Fatal("expected error with both body sources")
	}
	if _, err := e.run("comment", "add", "SMK-1", "   "); err == nil || !strings.Contains(err.Error(), "comment body is empty") {
		t.Fatalf("expected blank inline body to be rejected, got %v", err)
	}
	e.stdin = "\n\t\n"
	if _, err := e.run("comment", "add", "SMK-1", "--body-file", "-"); err == nil || !strings.Contains(err.Error(), "comment body is empty") {
		t.Fatalf("expected blank stdin body to be rejected, got %v", err)
	}
	e.stdin = ""

	// Neither rejection may have written a comment.
	var list []models.Comment
	e.runJSON(&list, "comment", "list", "SMK-1")
	if len(list) != 0 {
		t.Fatalf("rejected add wrote a comment: %+v", list)
	}
}

func TestCommentListEmptyIsJSONArrayAndDeleteUnknownFails(t *testing.T) {
	e := newCLIEnv(t)
	p := e.project()
	e.runJSON(new(models.Ticket), "ticket", "create", "--project", p.ID, "--title", "T")
	out, err := e.run("--json", "comment", "list", "SMK-1")
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("expected [], got %q %v", out, err)
	}
	if out, err := e.run("comment", "list", "SMK-1"); err != nil || !strings.Contains(out, "No comments.") {
		t.Fatalf("expected No comments., got %q %v", out, err)
	}
	if _, err := e.run("comment", "delete", "nope"); err == nil || !strings.Contains(err.Error(), "comment not found") {
		t.Fatalf("expected comment not found, got %v", err)
	}
}
