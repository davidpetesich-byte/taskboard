package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tcarac/taskboard/internal/models"
)

func ticketCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Manage tickets",
	}

	var projectID, status, priority string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tickets",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			tickets, err := store.ListTickets(models.TicketFilter{
				ProjectID: projectID,
				Status:    status,
				Priority:  priority,
			})
			if err != nil {
				return err
			}
			if tickets == nil {
				tickets = []models.Ticket{}
			}
			return emit(cmd, tickets, func() {
				if len(tickets) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No tickets found.")
					return
				}
				for _, t := range tickets {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s - %s (%s, %s)\n", t.DisplayKey(), t.Title, t.Status, t.Priority, t.ID)
				}
			})
		},
	}
	listCmd.Flags().StringVar(&projectID, "project", "", "filter by project ID")
	listCmd.Flags().StringVar(&status, "status", "", "filter by status (backlog|todo|in_progress|in_review|done)")
	listCmd.Flags().StringVar(&priority, "priority", "", "filter by priority (urgent|high|medium|low)")

	var createProject, createPriority, createDue, createTeam string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new ticket",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			title, _ := cmd.Flags().GetString("title")
			req := models.CreateTicketRequest{
				ProjectID: createProject,
				Title:     title,
				Priority:  createPriority,
			}
			if createDue != "" {
				req.DueDate = &createDue
			}
			if createTeam != "" {
				req.TeamID = &createTeam
			}
			t, err := store.CreateTicket(req)
			if err != nil {
				return err
			}
			return emit(cmd, t, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Created ticket %s: %s (%s)\n", t.DisplayKey(), t.Title, t.ID)
			})
		},
	}
	createCmd.Flags().StringVar(&createProject, "project", "", "project ID (required)")
	createCmd.MarkFlagRequired("project")
	createCmd.Flags().String("title", "", "ticket title (required)")
	createCmd.MarkFlagRequired("title")
	createCmd.Flags().StringVar(&createPriority, "priority", "medium", "priority (urgent|high|medium|low)")
	createCmd.Flags().StringVar(&createDue, "due", "", "due date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&createTeam, "team", "", "team ID")

	var moveStatus string
	moveCmd := &cobra.Command{
		Use:   "move [ref]",
		Short: "Move ticket to a different status (ref is an ID or display key like WEB-12)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			if err := validateStatus(moveStatus); err != nil {
				return err
			}
			id, err := store.ResolveTicketID(args[0])
			if err != nil {
				return err
			}
			t, err := store.MoveTicket(id, models.MoveTicketRequest{Status: moveStatus})
			if err != nil {
				return err
			}
			return emit(cmd, t, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Moved %s to %s\n", t.DisplayKey(), t.Status)
			})
		},
	}
	moveCmd.Flags().StringVar(&moveStatus, "status", "", "target status (required)")
	moveCmd.MarkFlagRequired("status")

	deleteCmd := &cobra.Command{
		Use:   "delete [ref]",
		Short: "Delete a ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			id, err := store.ResolveTicketID(args[0])
			if err != nil {
				return err
			}
			if err := store.DeleteTicket(id); err != nil {
				return err
			}
			return emit(cmd, deleted(id), func() {
				fmt.Fprintln(cmd.OutOrStdout(), "Ticket deleted.")
			})
		},
	}

	cmd.AddCommand(listCmd, createCmd, moveCmd, deleteCmd)
	return cmd
}
