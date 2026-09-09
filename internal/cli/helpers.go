package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tcarac/taskboard/internal/db"
)

// jsonOut is bound to the root --json flag. When set, every command prints
// exactly one JSON document (the API struct) and nothing else on stdout.
var jsonOut bool

var priorities = []string{"urgent", "high", "medium", "low"}

// printJSON writes v as indented JSON to the command's out writer.
func printJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// emit prints v as JSON when --json is set; otherwise it runs the human printer.
func emit(cmd *cobra.Command, v any, human func()) error {
	if jsonOut {
		return printJSON(cmd, v)
	}
	human()
	return nil
}

// deleted is the JSON payload every delete command prints.
func deleted(id string) map[string]any {
	return map[string]any{"deleted": true, "id": id}
}

func validateStatus(status string) error {
	for _, s := range db.BoardStatuses {
		if s == status {
			return nil
		}
	}
	return fmt.Errorf("invalid status %q (valid: %s)", status, strings.Join(db.BoardStatuses, ", "))
}

func validatePriority(priority string) error {
	for _, p := range priorities {
		if p == priority {
			return nil
		}
	}
	return fmt.Errorf("invalid priority %q (valid: %s)", priority, strings.Join(priorities, ", "))
}

// descriptionFromFlags returns the description supplied through --description
// or --description-file (a path, or "-" for stdin). nil means "not supplied".
func descriptionFromFlags(cmd *cobra.Command, inline, file string) (*string, error) {
	if file != "" {
		var data []byte
		var err error
		if file == "-" {
			data, err = io.ReadAll(cmd.InOrStdin())
		} else {
			data, err = os.ReadFile(file)
		}
		if err != nil {
			return nil, fmt.Errorf("reading description file: %w", err)
		}
		s := string(data)
		return &s, nil
	}
	if cmd.Flags().Changed("description") {
		return &inline, nil
	}
	return nil, nil
}
