package cli

import (
	"strings"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func TestDueDateIsValidatedOnCreateAndUpdate(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()

	out, err := env.run("ticket", "create", "--project", p.ID, "--title", "x", "--due", "09/30/2026")
	if err == nil || !strings.Contains(out, "invalid due date") {
		t.Fatalf("create with bad --due: err=%v out=%q", err, out)
	}

	var tk models.Ticket
	env.runJSON(&tk, "ticket", "create", "--project", p.ID, "--title", "x", "--due", "2026-09-30")
	if tk.DueDate == nil || tk.DueDate.Format("2006-01-02") != "2026-09-30" {
		t.Fatalf("create with good --due stored %v", tk.DueDate)
	}

	out, err = env.run("ticket", "update", "SMK-1", "--due", "next tuesday")
	if err == nil || !strings.Contains(out, "invalid due date") {
		t.Fatalf("update with bad --due: err=%v out=%q", err, out)
	}
}
