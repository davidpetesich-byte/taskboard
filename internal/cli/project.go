package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tcarac/taskboard/internal/models"
)

func projectCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			projects, err := store.ListProjects("")
			if err != nil {
				return err
			}
			if projects == nil {
				projects = []models.Project{}
			}
			return emit(cmd, projects, func() {
				if len(projects) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No projects found.")
					return
				}
				for _, p := range projects {
					icon := p.Icon
					if icon == "" {
						icon = " "
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s [%s] (%s) - %s\n", icon, p.Name, p.Prefix, p.Status, p.ID)
				}
			})
		},
	}

	var prefix, icon, color string
	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			p, err := store.CreateProject(models.CreateProjectRequest{
				Name:   args[0],
				Prefix: prefix,
				Icon:   icon,
				Color:  color,
			})
			if err != nil {
				return err
			}
			return emit(cmd, p, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Created project %s [%s] (%s)\n", p.Name, p.Prefix, p.ID)
			})
		},
	}
	createCmd.Flags().StringVar(&prefix, "prefix", "", "project prefix (required)")
	createCmd.MarkFlagRequired("prefix")
	createCmd.Flags().StringVar(&icon, "icon", "", "emoji icon")
	createCmd.Flags().StringVar(&color, "color", "#3B82F6", "hex color")

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			if err := store.DeleteProject(args[0]); err != nil {
				return err
			}
			return emit(cmd, deleted(args[0]), func() {
				fmt.Fprintln(cmd.OutOrStdout(), "Project deleted.")
			})
		},
	}

	cmd.AddCommand(listCmd, createCmd, deleteCmd)
	return cmd
}
