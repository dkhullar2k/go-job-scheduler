package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileValidCommandJob(t *testing.T) {
	path := writeJob(t, `{"job_id":"hello","schedule":"*/10 * * * * *","task":{"type":"execute_command","command":"echo hello"}}`)

	job, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if job.JobID != "hello" || job.Schedule != "*/10 * * * * *" || job.Task.Command != "echo hello" {
		t.Fatalf("unexpected parsed job: %+v", job)
	}
}

func TestParseFileRejectsUnknownFieldAndTrailingValue(t *testing.T) {
	tests := map[string]string{
		"unknown field":  `{"job_id":"hello","schedule":"* * * * *","task":{"type":"execute_command","command":"echo hello"},"extra":true}`,
		"trailing value": `{"job_id":"hello","schedule":"* * * * *","task":{"type":"execute_command","command":"echo hello"}} {}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := New().ParseFile(writeJob(t, content)); err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}

func writeJob(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "job.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
