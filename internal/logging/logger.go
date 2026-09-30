package logging

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

type Event struct {
	EventType string  `json:"event_type"`
	Time      string  `json:"time"`
	Status    string  `json:"status"`
	JobID     string  `json:"job_id,omitempty"`
	File      string  `json:"file,omitempty"`
	Output    *string `json:"output,omitempty"`
	Error     string  `json:"error,omitempty"`
}

func Open(path string) (*Logger, error) {
	if path == "" {
		return nil, errors.New("log file path cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f, enc: json.NewEncoder(f)}, nil
}

func (l *Logger) Parse(file, jobID string, parseErr error) error {
	e := Event{EventType: "parse", Time: time.Now().UTC().Format(time.RFC3339Nano), File: file, JobID: jobID, Status: "SUCCESS"}
	if parseErr != nil {
		e.Status, e.Error = "FAILURE", parseErr.Error()
	}
	return l.write(e)
}

func (l *Logger) Execution(jobID, output string, runErr error) error {
	// Keep output present even for successful commands that print nothing.
	e := Event{EventType: "execution", Time: time.Now().UTC().Format(time.RFC3339Nano), JobID: jobID, Output: &output, Status: "SUCCESS"}
	if runErr != nil {
		e.Status, e.Error = "FAILURE", runErr.Error()
	}
	return l.write(e)
}

func (l *Logger) write(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.enc.Encode(e)
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
