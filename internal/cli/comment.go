package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/tcarac/taskboard/internal/db"
	"github.com/tcarac/taskboard/internal/models"
)

func commentCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage comments (discussion notes on a ticket)",
	}

	var bodyFile, author string
	addCmd := &cobra.Command{
		Use:   "add [ticket-ref] [body]",
		Short: "Add a Markdown comment to a ticket (ref is an ID or display key like WEB-12)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := commentBody(cmd, args, bodyFile)
			if err != nil {
				return err
			}
			store, err := openStore()
			if err != nil {
				return err
			}
			ticketID, err := store.ResolveTicketID(args[0])
			if err != nil {
				return err
			}
			c, err := store.AddComment(ticketID, models.CreateCommentRequest{Author: author, Body: body})
			if err != nil {
				return err
			}
			return emit(cmd, c, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Added comment %s by %s\n", c.ID, c.Author)
			})
		},
	}
	addCmd.Flags().StringVar(&bodyFile, "body-file", "", "read the body from a file, or - for stdin")
	addCmd.Flags().StringVar(&author, "author", "cli", "author name")

	listCmd := &cobra.Command{
		Use:   "list [ticket-ref]",
		Short: "List a ticket's comments oldest first",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			ticketID, err := store.ResolveTicketID(args[0])
			if err != nil {
				return err
			}
			t, err := store.GetTicket(ticketID)
			if err != nil {
				return err
			}
			if t == nil {
				return db.ErrTicketNotFound
			}
			comments := t.Comments
			if comments == nil {
				comments = []models.Comment{}
			}
			return emit(cmd, comments, func() {
				w := cmd.OutOrStdout()
				if len(comments) == 0 {
					fmt.Fprintln(w, "No comments.")
					return
				}
				for _, c := range comments {
					fmt.Fprintf(w, "--- %s · %s (%s)\n%s\n", c.Author, c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), c.ID, c.Body)
				}
			})
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			if err := store.DeleteComment(args[0]); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("comment not found")
				}
				return err
			}
			return emit(cmd, deleted(args[0]), func() {
				fmt.Fprintln(cmd.OutOrStdout(), "Comment deleted.")
			})
		},
	}

	cmd.AddCommand(addCmd, listCmd, deleteCmd)
	return cmd
}

// commentBody returns the body from the positional argument or --body-file,
// requiring exactly one of the two. It runs before the database is opened.
func commentBody(cmd *cobra.Command, args []string, bodyFile string) (string, error) {
	hasInline := len(args) == 2
	if hasInline == (bodyFile != "") {
		return "", fmt.Errorf("provide the body as an argument or with --body-file, not both")
	}
	if hasInline {
		return args[1], nil
	}
	var data []byte
	var err error
	if bodyFile == "-" {
		data, err = io.ReadAll(cmd.InOrStdin())
	} else {
		data, err = os.ReadFile(bodyFile)
	}
	if err != nil {
		return "", fmt.Errorf("reading body file: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("comment body is empty")
	}
	return string(data), nil
}
