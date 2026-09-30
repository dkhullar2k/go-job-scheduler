package task

import (
	"context"
	"os/exec"

	"jobscheduler/internal/model"
)

type Handler interface {
	Execute(context.Context, model.Task) (string, error)
}

type CommandHandler struct{}

// Execute captures stdout and stderr together so the logger can store the full
// command output with the execution result.
func (CommandHandler) Execute(ctx context.Context, task model.Task) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", task.Command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
