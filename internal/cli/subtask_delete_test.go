package cli

import (
	"strings"
	"testing"
)

func TestSubtaskDeleteUnknownIDFails(t *testing.T) {
	env := newCLIEnv(t)
	out, err := env.run("subtask", "delete", "NOPE")
	if err == nil || !strings.Contains(out, "subtask not found") {
		t.Fatalf("deleting unknown subtask: err=%v out=%q", err, out)
	}
}
