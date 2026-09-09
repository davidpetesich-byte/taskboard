package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	taskdb "github.com/tcarac/taskboard/internal/db"
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
	// Each invocation represents a fresh CLI consumer. Clear the destination so
	// omitted `omitempty` fields cannot retain values from a prior response.
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
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

func TestTicketMoveRejectsAmbiguousDisplayKeyWithoutMutation(t *testing.T) {
	env := newCLIEnv(t)
	var upper, lower models.Project
	env.runJSON(&upper, "project", "create", "Upper", "--prefix", "DUP")
	env.runJSON(&lower, "project", "create", "Lower", "--prefix", "dup")
	var upperTicket, lowerTicket models.Ticket
	env.runJSON(&upperTicket, "ticket", "create", "--project", upper.ID, "--title", "Upper ticket")
	env.runJSON(&lowerTicket, "ticket", "create", "--project", lower.ID, "--title", "Lower ticket")

	out, err := env.run("ticket", "move", "DuP-1", "--status", "done")
	if err == nil || !strings.Contains(out, "ambiguous") {
		t.Fatalf("ambiguous move: err=%v out=%q", err, out)
	}

	database, err := taskdb.OpenAt(env.dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer database.Close()
	store := taskdb.NewStore(database)
	for _, id := range []string{upperTicket.ID, lowerTicket.ID} {
		ticket, err := store.GetTicket(id)
		if err != nil {
			t.Fatalf("getting ticket %s: %v", id, err)
		}
		if ticket == nil || ticket.Status != "todo" {
			t.Fatalf("ambiguous move mutated ticket %s: %+v", id, ticket)
		}
	}
}

func TestTicketMoveHandlesDeletionBetweenResolutionAndMove(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	var created models.Ticket
	env.runJSON(&created, "ticket", "create", "--project", p.ID, "--title", "Vanishing ticket")

	database, err := taskdb.OpenAt(env.dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	_, err = database.Exec(`CREATE TRIGGER delete_ticket_after_move
		AFTER UPDATE OF status ON tickets
		BEGIN
			DELETE FROM tickets WHERE id = NEW.id;
		END`)
	if err != nil {
		database.Close()
		t.Fatalf("creating deletion trigger: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("closing test database: %v", err)
	}

	out, err := env.run("ticket", "move", created.ID, "--status", "done")
	if err == nil || !strings.Contains(out, "ticket not found") {
		t.Fatalf("move after concurrent deletion: err=%v out=%q", err, out)
	}
}

func TestTicketGetJSONAndHuman(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	var created models.Ticket
	env.runJSON(&created, "ticket", "create", "--project", p.ID, "--title", "Read me", "--priority", "high")

	var got models.Ticket
	env.runJSON(&got, "ticket", "get", "SMK-1")
	if got.ID != created.ID || got.Title != "Read me" || got.Priority != "high" || got.ProjectPrefix != "SMK" {
		t.Fatalf("ticket get --json = %+v", got)
	}

	out, err := env.run("ticket", "get", created.ID)
	if err != nil {
		t.Fatalf("ticket get: %v", err)
	}
	for _, want := range []string{"SMK-1", "Read me", "todo", "high"} {
		if !strings.Contains(out, want) {
			t.Errorf("human ticket get missing %q:\n%s", want, out)
		}
	}

	if _, err := env.run("ticket", "get", "SMK-2"); err == nil {
		t.Fatal("ticket get on unknown key returned nil error")
	}
}

func TestLabelLifecycleViaCLI(t *testing.T) {
	env := newCLIEnv(t)

	var created models.Label
	env.runJSON(&created, "label", "create", "Blocked", "--color", "#EF4444")
	if created.ID == "" || created.Name != "Blocked" || created.Color != "#EF4444" {
		t.Fatalf("label create --json = %+v", created)
	}

	var plain models.Label
	env.runJSON(&plain, "label", "create", "Plain")
	if plain.Color != "#6B7280" {
		t.Fatalf("label create without --color stored %q, want default #6B7280", plain.Color)
	}

	var listed []models.Label
	env.runJSON(&listed, "label", "list")
	if len(listed) != 2 {
		t.Fatalf("label list --json = %+v, want 2", listed)
	}

	var del map[string]any
	env.runJSON(&del, "label", "delete", "blocked")
	if del["deleted"] != true || del["id"] != created.ID {
		t.Fatalf("label delete by name returned %v", del)
	}
	env.runJSON(&listed, "label", "list")
	if len(listed) != 1 || listed[0].ID != plain.ID {
		t.Fatalf("after delete, label list = %+v", listed)
	}

	if _, err := env.run("label", "delete", "Nope"); err == nil {
		t.Fatal("deleting unknown label returned nil error")
	}
}

func TestTicketCreateWithDescriptionStatusAndLabels(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	var blocked models.Label
	env.runJSON(&blocked, "label", "create", "Blocked")

	env.stdin = "# Heading\n\nbody from stdin\n"
	var created models.Ticket
	env.runJSON(&created, "ticket", "create",
		"--project", p.ID, "--title", "Full", "--status", "in_review",
		"--description-file", "-", "--label", "blocked")
	env.stdin = ""

	if created.Status != "in_review" {
		t.Errorf("status = %q, want in_review", created.Status)
	}
	if created.Description != "# Heading\n\nbody from stdin\n" {
		t.Errorf("description = %q", created.Description)
	}
	if len(created.Labels) != 1 || created.Labels[0].ID != blocked.ID {
		t.Errorf("labels = %+v, want Blocked", created.Labels)
	}

	out, err := env.run("ticket", "create", "--project", p.ID, "--title", "x", "--status", "wat")
	if err == nil || !strings.Contains(out, "invalid status") {
		t.Fatalf("create with bad status: err=%v out=%q", err, out)
	}
	var tickets []models.Ticket
	env.runJSON(&tickets, "ticket", "list")
	if len(tickets) != 1 {
		t.Fatalf("invalid create wrote a ticket: %+v", tickets)
	}
}

func TestTicketUpdateFieldsAndLabelSemantics(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	var a, b models.Label
	env.runJSON(&a, "label", "create", "A")
	env.runJSON(&b, "label", "create", "B")
	var tk models.Ticket
	env.runJSON(&tk, "ticket", "create", "--project", p.ID, "--title", "Orig", "--label", "A")

	// Field update leaves labels alone.
	env.runJSON(&tk, "ticket", "update", "SMK-1", "--title", "Renamed", "--priority", "urgent", "--description", "new body")
	if tk.Title != "Renamed" || tk.Priority != "urgent" || tk.Description != "new body" || len(tk.Labels) != 1 {
		t.Fatalf("after field update: %+v", tk)
	}

	// --add-label is a delta.
	env.runJSON(&tk, "ticket", "update", "SMK-1", "--add-label", "b")
	if ids := labelNames(tk); !reflect.DeepEqual(ids, []string{"A", "B"}) {
		t.Fatalf("after --add-label: %v", ids)
	}

	// --remove-label is a delta.
	env.runJSON(&tk, "ticket", "update", "SMK-1", "--remove-label", "A")
	if ids := labelNames(tk); !reflect.DeepEqual(ids, []string{"B"}) {
		t.Fatalf("after --remove-label: %v", ids)
	}

	// --label replaces the set.
	env.runJSON(&tk, "ticket", "update", "SMK-1", "--label", "A")
	if ids := labelNames(tk); !reflect.DeepEqual(ids, []string{"A"}) {
		t.Fatalf("after --label replace: %v", ids)
	}

	// --clear-labels empties it.
	env.runJSON(&tk, "ticket", "update", "SMK-1", "--clear-labels")
	if len(tk.Labels) != 0 {
		t.Fatalf("after --clear-labels: %+v", tk.Labels)
	}

	// Replace-style and delta-style flags are mutually exclusive.
	for _, flags := range [][]string{
		{"--label", "A", "--add-label", "B"},
		{"--label", "A", "--remove-label", "B"},
		{"--label", "A", "--clear-labels"},
		{"--clear-labels", "--add-label", "A"},
		{"--clear-labels", "--remove-label", "A"},
	} {
		args := append([]string{"ticket", "update", "SMK-1"}, flags...)
		if _, err := env.run(args...); err == nil {
			t.Fatalf("conflicting label flags %v should be rejected", flags)
		}
	}
	// No flags at all is an error, not a silent no-op.
	if _, err := env.run("ticket", "update", "SMK-1"); err == nil {
		t.Fatal("update with no flags should be rejected")
	}
}

func TestTicketUpdateValidatesBeforeWriting(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	var tk models.Ticket
	env.runJSON(&tk, "ticket", "create", "--project", p.ID, "--title", "Original")

	for _, tc := range []struct {
		name  string
		flags []string
		want  string
	}{
		{name: "status", flags: []string{"--title", "Mutated", "--status", "wat"}, want: "invalid status"},
		{name: "priority", flags: []string{"--title", "Mutated", "--priority", "wat"}, want: "invalid priority"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"ticket", "update", tk.ID}, tc.flags...)
			out, err := env.run(args...)
			if err == nil || !strings.Contains(out, tc.want) {
				t.Fatalf("invalid update: err=%v out=%q", err, out)
			}
			var got models.Ticket
			env.runJSON(&got, "ticket", "get", tk.ID)
			if got.Title != "Original" || got.Status != "todo" || got.Priority != "medium" {
				t.Fatalf("invalid update mutated ticket: %+v", got)
			}
		})
	}
}

func TestTicketDescriptionFromFile(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	path := filepath.Join(t.TempDir(), "description.md")
	if err := os.WriteFile(path, []byte("body from file\n"), 0o600); err != nil {
		t.Fatalf("write description fixture: %v", err)
	}

	var tk models.Ticket
	env.runJSON(&tk, "ticket", "create", "--project", p.ID, "--title", "File", "--description-file", path)
	if tk.Description != "body from file\n" {
		t.Fatalf("create description = %q", tk.Description)
	}

	if err := os.WriteFile(path, []byte("updated from file\n"), 0o600); err != nil {
		t.Fatalf("update description fixture: %v", err)
	}
	env.runJSON(&tk, "ticket", "update", tk.ID, "--description-file", path)
	if tk.Description != "updated from file\n" {
		t.Fatalf("update description = %q", tk.Description)
	}
}

func TestTicketListFiltersByLabel(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	env.runJSON(new(models.Label), "label", "create", "Blocked")
	env.runJSON(new(models.Ticket), "ticket", "create", "--project", p.ID, "--title", "tagged", "--label", "Blocked")
	env.runJSON(new(models.Ticket), "ticket", "create", "--project", p.ID, "--title", "plain")

	var tickets []models.Ticket
	env.runJSON(&tickets, "ticket", "list", "--label", "blocked")
	if len(tickets) != 1 || tickets[0].Title != "tagged" {
		t.Fatalf("ticket list --label = %+v", tickets)
	}
}

func TestTicketListFiltersByTeam(t *testing.T) {
	env := newCLIEnv(t)
	p := env.project()
	var team models.Team
	env.runJSON(&team, "team", "create", "CLI")
	env.runJSON(new(models.Ticket), "ticket", "create", "--project", p.ID, "--title", "assigned", "--team", team.ID)
	env.runJSON(new(models.Ticket), "ticket", "create", "--project", p.ID, "--title", "unassigned")

	var tickets []models.Ticket
	env.runJSON(&tickets, "ticket", "list", "--team", team.ID)
	if len(tickets) != 1 || tickets[0].Title != "assigned" {
		t.Fatalf("ticket list --team = %+v", tickets)
	}
}

// labelNames returns a ticket's label names sorted, for stable comparison.
func labelNames(t models.Ticket) []string {
	names := make([]string, 0, len(t.Labels))
	for _, l := range t.Labels {
		names = append(names, l.Name)
	}
	sort.Strings(names)
	return names
}
