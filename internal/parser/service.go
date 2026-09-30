package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"jobscheduler/internal/model"
	"jobscheduler/internal/task"
)

type Service struct {
	handlers map[string]task.Handler
}

func New() *Service {
	return &Service{handlers: map[string]task.Handler{"execute_command": task.CommandHandler{}}}
}

func (s *Service) ParseFile(path string) (model.Job, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Job{}, err
	}
	var job model.Job
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&job); err != nil {
		return model.Job{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return model.Job{}, fmt.Errorf("parse %s: multiple JSON values", path)
		}
		return model.Job{}, fmt.Errorf("parse %s: trailing data: %w", path, err)
	}
	if strings.TrimSpace(job.JobID) == "" || strings.TrimSpace(job.Schedule) == "" {
		return model.Job{}, errors.New("job_id and schedule are required")
	}
	if job.Task.Type == "" {
		return model.Job{}, errors.New("task.type is required")
	}
	if _, ok := s.handlers[job.Task.Type]; !ok {
		return model.Job{}, fmt.Errorf("unsupported task type %q", job.Task.Type)
	}
	if job.Task.Type == "execute_command" && strings.TrimSpace(job.Task.Command) == "" {
		return model.Job{}, errors.New("task.command is required for execute_command")
	}
	return job, nil
}

func (s *Service) Execute(ctx context.Context, job model.Job) (string, error) {
	handler, ok := s.handlers[job.Task.Type]
	if !ok {
		return "", fmt.Errorf("unsupported task type %q", job.Task.Type)
	}
	return handler.Execute(ctx, job.Task)
}
