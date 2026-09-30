package task

import (
	"context"
	"testing"

	"jobscheduler/internal/model"
)

func TestCommandHandlerCapturesCombinedOutput(t *testing.T) {
	output, err := (CommandHandler{}).Execute(context.Background(), model.Task{
		Type:    "execute_command",
		Command: "printf stdout; printf stderr >&2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output != "stdoutstderr" {
		t.Fatalf("output = %q, want combined stdout and stderr", output)
	}
}
