package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/tcarac/taskboard/internal/db"
	"github.com/tcarac/taskboard/internal/models"
)

// newTestAPI serves the REST API over httptest with an empty embedded
// frontend so handler behavior can be exercised end to end.
func newTestAPI(t *testing.T) (*httptest.Server, *db.Store) {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	store := db.NewStore(database)
	ts := httptest.NewServer(New(store, fstest.MapFS{}))
	t.Cleanup(ts.Close)
	return ts, store
}

func do(t *testing.T, method, url, body string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := res.Body.Read(buf)
		sb.Write(buf[:n])
		if readErr != nil {
			break
		}
	}
	return res, sb.String()
}

func TestCommentEndpointsLifecycle(t *testing.T) {
	ts, store := newTestAPI(t)
	p, err := store.CreateProject(models.CreateProjectRequest{Name: "P", Prefix: "API"})
	if err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTicket(models.CreateTicketRequest{ProjectID: p.ID, Title: "T"})
	if err != nil {
		t.Fatal(err)
	}

	res, body := do(t, http.MethodPost, ts.URL+"/api/tickets/"+tk.ID+"/comments", `{"author":"David","body":"**seen** it"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: want 201, got %d %s", res.StatusCode, body)
	}
	var created models.Comment
	if err := json.Unmarshal([]byte(body), &created); err != nil || created.Author != "David" || created.Body != "**seen** it" {
		t.Fatalf("create: unexpected body %s (%v)", body, err)
	}

	res, body = do(t, http.MethodGet, ts.URL+"/api/tickets/"+tk.ID, "")
	var got models.Ticket
	if res.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &got) != nil || len(got.Comments) != 1 || got.Comments[0].ID != created.ID {
		t.Fatalf("get ticket: want one comment, got %d %s", res.StatusCode, body)
	}

	res, body = do(t, http.MethodGet, ts.URL+"/api/tickets", "")
	if res.StatusCode != http.StatusOK || strings.Contains(body, `"comments"`) {
		t.Fatalf("list must not carry comments: %d %s", res.StatusCode, body)
	}

	res, body = do(t, http.MethodPost, ts.URL+"/api/tickets/"+tk.ID+"/comments", `{"author":"x","body":"   "}`)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, "body is required") {
		t.Fatalf("blank body: want 400, got %d %s", res.StatusCode, body)
	}

	res, body = do(t, http.MethodPost, ts.URL+"/api/tickets/nope/comments", `{"body":"hi"}`)
	if res.StatusCode != http.StatusNotFound || !strings.Contains(body, "ticket not found") {
		t.Fatalf("unknown ticket: want 404, got %d %s", res.StatusCode, body)
	}

	res, _ = do(t, http.MethodDelete, ts.URL+"/api/comments/"+created.ID, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d", res.StatusCode)
	}
	res, body = do(t, http.MethodDelete, ts.URL+"/api/comments/"+created.ID, "")
	if res.StatusCode != http.StatusNotFound || !strings.Contains(body, "comment not found") {
		t.Fatalf("second delete: want 404, got %d %s", res.StatusCode, body)
	}
}

func TestUpdateTicketEndpointMovesTicketToAnotherProject(t *testing.T) {
	ts, store := newTestAPI(t)
	source, err := store.CreateProject(models.CreateProjectRequest{Name: "Source", Prefix: "SRC"})
	if err != nil {
		t.Fatal(err)
	}
	target, err := store.CreateProject(models.CreateProjectRequest{Name: "Target", Prefix: "TGT"})
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := store.CreateTicket(models.CreateTicketRequest{ProjectID: source.ID, Title: "Move me"})
	if err != nil {
		t.Fatal(err)
	}

	res, body := do(t, http.MethodPut, ts.URL+"/api/tickets/"+ticket.ID,
		`{"projectId":"`+target.ID+`"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s; want 200", res.StatusCode, body)
	}

	var got models.Ticket
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.ProjectID != target.ID {
		t.Errorf("response projectId = %q, want %q", got.ProjectID, target.ID)
	}

	stored, err := store.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("re-reading ticket: %v", err)
	}
	if stored.ProjectID != target.ID {
		t.Errorf("stored projectId = %q, want %q", stored.ProjectID, target.ID)
	}
}
