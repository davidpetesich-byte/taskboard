package cli

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tcarac/taskboard/internal/models"
)

func subtaskCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subtask",
		Short: "Manage subtasks (checklist items on a ticket)",
	}

	addCmd := &cobra.Command{
		Use:   "add [ticket-ref] [title]",
		Short: "Add a subtask to a ticket (ref is an ID or display key like WEB-12)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			ticketID, err := store.ResolveTicketID(args[0])
			if err != nil {
				return err
			}
			st, err := store.AddSubtask(ticketID, models.CreateSubtaskRequest{Title: args[1]})
			if err != nil {
				return err
			}
			return emit(cmd, st, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Added subtask %s (%s)\n", st.Title, st.ID)
			})
		},
	}

	toggleCmd := &cobra.Command{
		Use:   "toggle [id]",
		Short: "Flip a subtask between done and not done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			st, err := store.ToggleSubtask(args[0])
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("subtask not found")
			}
			if err != nil {
				return err
			}
			return emit(cmd, st, func() {
				state := "not done"
				if st.Completed {
					state = "done"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Subtask %s is now %s\n", st.Title, state)
			})
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a subtask",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			if err := store.DeleteSubtask(args[0]); err != nil {
				return err
			}
			return emit(cmd, deleted(args[0]), func() {
				fmt.Fprintln(cmd.OutOrStdout(), "Subtask deleted.")
			})
		},
	}

	cmd.AddCommand(addCmd, toggleCmd, deleteCmd)
	return cmd
}
