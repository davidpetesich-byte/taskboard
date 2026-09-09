package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

// cliEnv runs the real root command against a per-test SQLite file.
type cliEnv struct {
	t      *testing.T
	dbPath string
	stdin  string
}

func newCLIEnv(t *testing.T) *cliEnv {
	t.Helper()
	return &cliEnv{t: t, dbPath: filepath.Join(t.TempDir(), "cli.db")}
}

// run executes the CLI with --db pointing at the test database and returns
// everything written to the command's out and err writers.
func (e *cliEnv) run(args ...string) (string, error) {
	e.t.Helper()
	root := NewRootCmd(nil)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(e.stdin))
	root.SetArgs(append([]string{"--db", e.dbPath}, args...))
	err := root.Execute()
	return out.String(), err
}

// runJSON runs with --json and decodes the output into v, failing the test on any error.
func (e *cliEnv) runJSON(v any, args ...string) {
	e.t.Helper()
	out, err := e.run(append([]string{"--json"}, args...)...)
	if err != nil {
		e.t.Fatalf("%v: %v\n%s", args, err, out)
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		e.t.Fatalf("decoding output of %v: %v\n%s", args, err, out)
	}
}

// project creates a project via the CLI and returns it.
func (e *cliEnv) project() models.Project {
	e.t.Helper()
	var p models.Project
	e.runJSON(&p, "project", "create", "Smoke", "--prefix", "SMK")
	return p
}

func TestJSONListIsEmptyArrayNotNull(t *testing.T) {
	env := newCLIEnv(t)
	for _, args := range [][]string{
		{"ticket", "list"},
		{"project", "list"},
		{"team", "list"},
	} {
		out, err := env.run(append([]string{"--json"}, args...)...)
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		if strings.TrimSpace(out) != "[]" {
			t.Errorf("%v --json on empty db = %q, want []", args, strings.TrimSpace(out))
		}
	}
}

func TestJSONProjectCreateRoundTrip(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	if p.ID == "" || p.Prefix != "SMK" || p.Name != "Smoke" {
		t.Fatalf("project create --json returned %+v", p)
	}
	var listed []models.Project
	env.runJSON(&listed, "project", "list")
	if len(listed) != 1 || listed[0].ID != p.ID {
		t.Fatalf("project list --json = %+v, want the one created project", listed)
	}
}

func TestHumanOutputGoesToCommandWriter(t *testing.T) {
	env := newCLIEnv(t)
	env.project()
	out, err := env.run("project", "list")
	if err != nil {
		t.Fatalf("project list: %v", err)
	}
	if !strings.Contains(out, "Smoke [SMK]") {
		t.Fatalf("human project list not captured on cmd writer, got %q", out)
	}
}

func TestRuntimeErrorDoesNotPrintUsage(t *testing.T) {
	env := newCLIEnv(t)
	out, err := env.run("ticket", "delete", "NOPE-1")
	if err == nil {
		t.Fatal("deleting an unknown ticket returned nil error")
	}
	if strings.Contains(out, "Usage:") {
		t.Fatalf("usage text printed on a runtime error:\n%s", out)
	}
}

func TestTicketMoveAndDeleteByDisplayKey(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	var created models.Ticket
	env.runJSON(&created, "ticket", "create", "--project", p.ID, "--title", "First")

	var moved models.Ticket
	env.runJSON(&moved, "ticket", "move", "SMK-1", "--status", "in_review")
	if moved.ID != created.ID || moved.Status != "in_review" {
		t.Fatalf("move by key returned %+v", moved)
	}

	out, err := env.run("ticket", "move", "SMK-1", "--status", "nope")
	if err == nil || !strings.Contains(out, "invalid status") {
		t.Fatalf("move with bad status: err=%v out=%q", err, out)
	}

	var del map[string]any
	env.runJSON(&del, "ticket", "delete", "smk-1")
	if del["deleted"] != true || del["id"] != created.ID {
		t.Fatalf("delete by key returned %v", del)
	}
	var remaining []models.Ticket
	env.runJSON(&remaining, "ticket", "list")
	if len(remaining) != 0 {
		t.Fatalf("ticket still listed after delete: %+v", remaining)
	}
}
