package db

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/tcarac/taskboard/internal/models"
)

func newStoreWithTicket(t *testing.T) (*Store, *models.Ticket) {
	t.Helper()

	database, err := OpenAt(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	s := NewStore(database)

	project, err := s.CreateProject(models.CreateProjectRequest{Name: "Comments", Prefix: "CMT"})
	if err != nil {
		t.Fatalf("creating test project: %v", err)
	}

	ticket, err := s.CreateTicket(models.CreateTicketRequest{ProjectID: project.ID, Title: "Ticket"})
	if err != nil {
		t.Fatalf("creating test ticket: %v", err)
	}

	return s, ticket
}

func TestAddCommentReturnsPersistedComment(t *testing.T) {
	store, ticket := newStoreWithTicket(t)
	c, err := store.AddComment(ticket.ID, models.CreateCommentRequest{Author: "David", Body: "# Found it\n\nrow 12 is wrong"})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if c.ID == "" || c.TicketID != ticket.ID || c.Author != "David" || c.Body != "# Found it\n\nrow 12 is wrong" || c.CreatedAt.IsZero() {
		t.Fatalf("unexpected comment: %+v", c)
	}
	got, err := store.GetComment(c.ID)
	if err != nil || got == nil || got.Body != c.Body {
		t.Fatalf("GetComment: %+v %v", got, err)
	}
}

func TestAddCommentDefaultsAuthorAndRejectsUnknownTicket(t *testing.T) {
	store, ticket := newStoreWithTicket(t)
	c, err := store.AddComment(ticket.ID, models.CreateCommentRequest{Body: "no author"})
	if err != nil || c.Author != "unknown" {
		t.Fatalf("expected author fallback 'unknown', got %+v %v", c, err)
	}
	if _, err := store.AddComment("nope", models.CreateCommentRequest{Author: "x", Body: "y"}); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestGetTicketHydratesCommentsInOrder(t *testing.T) {
	store, ticket := newStoreWithTicket(t)
	for _, body := range []string{"first", "second", "third"} {
		if _, err := store.AddComment(ticket.ID, models.CreateCommentRequest{Author: "a", Body: body}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.GetTicket(ticket.ID)
	if err != nil || got == nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if len(got.Comments) != 3 || got.Comments[0].Body != "first" || got.Comments[2].Body != "third" {
		t.Fatalf("expected 3 ordered comments, got %+v", got.Comments)
	}
	list, err := store.ListTickets(models.TicketFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, lt := range list {
		if len(lt.Comments) != 0 {
			t.Fatalf("ListTickets must not hydrate comments, got %+v", lt.Comments)
		}
	}
}

func TestDeleteCommentAndCascade(t *testing.T) {
	store, ticket := newStoreWithTicket(t)
	c, _ := store.AddComment(ticket.ID, models.CreateCommentRequest{Author: "a", Body: "b"})
	if err := store.DeleteComment(c.ID); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	if err := store.DeleteComment(c.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows on second delete, got %v", err)
	}
	keep, _ := store.AddComment(ticket.ID, models.CreateCommentRequest{Author: "a", Body: "kept until cascade"})
	if err := store.DeleteTicket(ticket.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := store.GetComment(keep.ID); err != nil || got != nil {
		t.Fatalf("expected cascade delete, got %+v %v", got, err)
	}
}
