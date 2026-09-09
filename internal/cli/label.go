package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tcarac/taskboard/internal/models"
)

func labelCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Manage labels (colour-coded tags attached to tickets)",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			labels, err := store.ListLabels()
			if err != nil {
				return err
			}
			if labels == nil {
				labels = []models.Label{}
			}
			return emit(cmd, labels, func() {
				if len(labels) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No labels found.")
					return
				}
				for _, l := range labels {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s (%s)\n", l.Color, l.Name, l.ID)
				}
			})
		},
	}

	var color string
	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a label",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			l, err := store.CreateLabel(models.CreateLabelRequest{Name: args[0], Color: color})
			if err != nil {
				return err
			}
			return emit(cmd, l, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Created label %s %s (%s)\n", l.Color, l.Name, l.ID)
			})
		},
	}
	createCmd.Flags().StringVar(&color, "color", "#6B7280", "hex color")

	deleteCmd := &cobra.Command{
		Use:   "delete [ref]",
		Short: "Delete a label by ID or name and detach it from every ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			ids, err := store.ResolveLabelIDs([]string{args[0]})
			if err != nil {
				return err
			}
			if err := store.DeleteLabel(ids[0]); err != nil {
				return err
			}
			return emit(cmd, deleted(ids[0]), func() {
				fmt.Fprintln(cmd.OutOrStdout(), "Label deleted.")
			})
		},
	}

	cmd.AddCommand(listCmd, createCmd, deleteCmd)
	return cmd
}
