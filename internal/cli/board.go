package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// columnTitle turns a status key into its display name: in_review -> In Review.
func columnTitle(status string) string {
	parts := strings.Split(status, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	title := strings.Join(parts, " ")
	if title == "Todo" {
		return "To Do"
	}
	return title
}

func boardCommand() *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "board",
		Short: "Show the Kanban board: every column with its tickets",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore()
			if err != nil {
				return err
			}
			board, err := store.GetBoard(projectID)
			if err != nil {
				return err
			}
			return emit(cmd, board, func() {
				w := cmd.OutOrStdout()
				for i, col := range board.Columns {
					if i > 0 {
						fmt.Fprintln(w)
					}
					fmt.Fprintf(w, "%s (%d)\n", columnTitle(col.Status), len(col.Tickets))
					for _, t := range col.Tickets {
						line := fmt.Sprintf("  %s  %s  [%s]", t.DisplayKey(), t.Title, t.Priority)
						if len(t.Labels) > 0 {
							names := make([]string, 0, len(t.Labels))
							for _, l := range t.Labels {
								names = append(names, l.Name)
							}
							line += "  {" + strings.Join(names, ", ") + "}"
						}
						fmt.Fprintln(w, line)
					}
				}
			})
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "limit to one project ID")
	return cmd
}
