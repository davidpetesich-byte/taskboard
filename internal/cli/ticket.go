package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tcarac/taskboard/internal/db"
	"github.com/tcarac/taskboard/internal/models"
)

func ticketCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Manage tickets",
	}

	var projectID, teamID, status, priority, labelRef string
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
				TeamID:    teamID,
				Status:    status,
				Priority:  priority,
			})
			if err != nil {
				return err
			}
			if labelRef != "" {
				ids, err := store.ResolveLabelIDs([]string{labelRef})
				if err != nil {
					return err
				}
				kept := tickets[:0]
				for _, t := range tickets {
					for _, l := range t.Labels {
						if l.ID == ids[0] {
							kept = append(kept, t)
							break
						}
					}
				}
				tickets = kept
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
	listCmd.Flags().StringVar(&teamID, "team", "", "filter by team ID")
	listCmd.Flags().StringVar(&labelRef, "label", "", "filter by label (ID or name); applied after fetching")

	var createProject, createPriority, createDue, createTeam, createStatus, createDesc, createDescFile string
	var createLabels []string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new ticket",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			title, _ := cmd.Flags().GetString("title")
			if err := validatePriority(createPriority); err != nil {
				return err
			}
			if createStatus != "" {
				if err := validateStatus(createStatus); err != nil {
					return err
				}
			}
			desc, err := descriptionFromFlags(cmd, createDesc, createDescFile)
			if err != nil {
				return err
			}
			req := models.CreateTicketRequest{
				ProjectID: createProject,
				Title:     title,
				Priority:  createPriority,
				Status:    createStatus,
			}
			if desc != nil {
				req.Description = *desc
			}
			if createDue != "" {
				if err := validateDue(createDue); err != nil {
					return err
				}
				req.DueDate = &createDue
			}
			if createTeam != "" {
				req.TeamID = &createTeam
			}
			if len(createLabels) > 0 {
				req.Labels, err = store.ResolveLabelIDs(createLabels)
				if err != nil {
					return err
				}
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
	createCmd.Flags().StringVar(&createStatus, "status", "", "initial status (backlog|todo|in_progress|in_review|done; default todo)")
	createCmd.Flags().StringVar(&createDesc, "description", "", "markdown description")
	createCmd.Flags().StringVar(&createDescFile, "description-file", "", "read the description from a file, or - for stdin")
	createCmd.MarkFlagsMutuallyExclusive("description", "description-file")
	createCmd.Flags().StringArrayVar(&createLabels, "label", nil, "label to attach (ID or name); repeatable")

	getCmd := &cobra.Command{
		Use:   "get [ref]",
		Short: "Show a ticket in full (ref is an ID or display key like WEB-12)",
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
			t, err := store.GetTicket(id)
			if err != nil {
				return err
			}
			if t == nil {
				return db.ErrTicketNotFound
			}
			return emit(cmd, t, func() {
				w := cmd.OutOrStdout()
				fmt.Fprintf(w, "%s  %s  [%s, %s]\n", t.DisplayKey(), t.Title, t.Status, t.Priority)
				fmt.Fprintf(w, "ID: %s\n", t.ID)
				team, due := "-", "-"
				if t.TeamID != nil {
					team = *t.TeamID
				}
				if t.DueDate != nil {
					due = t.DueDate.Format("2006-01-02")
				}
				fmt.Fprintf(w, "Project: %s   Team: %s   Due: %s\n", t.ProjectID, team, due)
				names := make([]string, 0, len(t.Labels))
				for _, l := range t.Labels {
					names = append(names, l.Name)
				}
				if len(names) > 0 {
					fmt.Fprintf(w, "Labels: %s\n", strings.Join(names, ", "))
				}
				if len(t.BlockedBy) > 0 {
					fmt.Fprintf(w, "Blocked by: %s\n", strings.Join(t.BlockedBy, ", "))
				}
				if len(t.Subtasks) > 0 {
					fmt.Fprintln(w, "Subtasks:")
					for _, st := range t.Subtasks {
						mark := " "
						if st.Completed {
							mark = "x"
						}
						fmt.Fprintf(w, "  [%s] %s (%s)\n", mark, st.Title, st.ID)
					}
				}
				if t.Description != "" {
					fmt.Fprintf(w, "\n%s\n", t.Description)
				}
			})
		},
	}

	var upTitle, upDesc, upDescFile, upStatus, upPriority, upDue, upTeam string
	var upLabels, upAddLabels, upRemoveLabels []string
	var upClearLabels bool
	updateCmd := &cobra.Command{
		Use:   "update [ref]",
		Short: "Update ticket fields and labels (only flags that are set are changed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			changed := false
			for _, name := range []string{"title", "description", "description-file", "status", "priority", "due", "team", "label", "add-label", "remove-label", "clear-labels"} {
				if cmd.Flags().Changed(name) {
					changed = true
					break
				}
			}
			if !changed {
				return fmt.Errorf("nothing to update: pass at least one flag (see --help)")
			}
			store, err := openStore()
			if err != nil {
				return err
			}
			id, err := store.ResolveTicketID(args[0])
			if err != nil {
				return err
			}

			var req models.UpdateTicketRequest
			if cmd.Flags().Changed("title") {
				req.Title = &upTitle
			}
			if cmd.Flags().Changed("status") {
				if err := validateStatus(upStatus); err != nil {
					return err
				}
				req.Status = &upStatus
			}
			if cmd.Flags().Changed("priority") {
				if err := validatePriority(upPriority); err != nil {
					return err
				}
				req.Priority = &upPriority
			}
			if cmd.Flags().Changed("due") {
				if err := validateDue(upDue); err != nil {
					return err
				}
				req.DueDate = &upDue
			}
			if cmd.Flags().Changed("team") {
				req.TeamID = &upTeam
			}
			req.Description, err = descriptionFromFlags(cmd, upDesc, upDescFile)
			if err != nil {
				return err
			}

			switch {
			case upClearLabels:
				req.Labels = []string{}
			case len(upLabels) > 0:
				req.Labels, err = store.ResolveLabelIDs(upLabels)
				if err != nil {
					return err
				}
			case len(upAddLabels) > 0 || len(upRemoveLabels) > 0:
				current, err := store.GetTicket(id)
				if err != nil {
					return err
				}
				if current == nil {
					return db.ErrTicketNotFound
				}
				set := map[string]bool{}
				order := []string{}
				for _, l := range current.Labels {
					set[l.ID] = true
					order = append(order, l.ID)
				}
				add, err := store.ResolveLabelIDs(upAddLabels)
				if err != nil {
					return err
				}
				for _, lid := range add {
					if !set[lid] {
						set[lid] = true
						order = append(order, lid)
					}
				}
				remove, err := store.ResolveLabelIDs(upRemoveLabels)
				if err != nil {
					return err
				}
				for _, lid := range remove {
					delete(set, lid)
				}
				req.Labels = []string{}
				for _, lid := range order {
					if set[lid] {
						req.Labels = append(req.Labels, lid)
					}
				}
			}

			t, err := store.UpdateTicket(id, req)
			if err != nil {
				return err
			}
			if t == nil {
				return db.ErrTicketNotFound
			}
			return emit(cmd, t, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated %s: %s [%s, %s]\n", t.DisplayKey(), t.Title, t.Status, t.Priority)
			})
		},
	}
	updateCmd.Flags().StringVar(&upTitle, "title", "", "new title")
	updateCmd.Flags().StringVar(&upDesc, "description", "", "new markdown description")
	updateCmd.Flags().StringVar(&upDescFile, "description-file", "", "read the new description from a file, or - for stdin")
	updateCmd.Flags().StringVar(&upStatus, "status", "", "new status (backlog|todo|in_progress|in_review|done)")
	updateCmd.Flags().StringVar(&upPriority, "priority", "", "new priority (urgent|high|medium|low)")
	updateCmd.Flags().StringVar(&upDue, "due", "", "new due date (YYYY-MM-DD)")
	updateCmd.Flags().StringVar(&upTeam, "team", "", "new team ID")
	updateCmd.Flags().StringArrayVar(&upLabels, "label", nil, "replace the label set with these (ID or name); repeatable")
	updateCmd.Flags().StringArrayVar(&upAddLabels, "add-label", nil, "add a label (ID or name); repeatable")
	updateCmd.Flags().StringArrayVar(&upRemoveLabels, "remove-label", nil, "remove a label (ID or name); repeatable")
	updateCmd.Flags().BoolVar(&upClearLabels, "clear-labels", false, "remove every label")
	updateCmd.MarkFlagsMutuallyExclusive("description", "description-file")
	updateCmd.MarkFlagsMutuallyExclusive("label", "add-label")
	updateCmd.MarkFlagsMutuallyExclusive("label", "remove-label")
	updateCmd.MarkFlagsMutuallyExclusive("label", "clear-labels")
	updateCmd.MarkFlagsMutuallyExclusive("clear-labels", "add-label")
	updateCmd.MarkFlagsMutuallyExclusive("clear-labels", "remove-label")

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
			if t == nil {
				return fmt.Errorf("ticket not found")
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

	cmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, moveCmd, deleteCmd)
	return cmd
}
