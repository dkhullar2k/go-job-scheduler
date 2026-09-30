package logging

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExecutionWritesJSONLIncludingEmptyOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	logger, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := logger.Execution("hello", "", nil); err != nil {
		t.Fatal(err)
	}
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var event map[string]any
	if err := json.NewDecoder(bufio.NewReader(file)).Decode(&event); err != nil {
		t.Fatal(err)
	}
	if event["event_type"] != "execution" || event["job_id"] != "hello" || event["status"] != "SUCCESS" || event["output"] != "" {
		t.Fatalf("unexpected event: %#v", event)
	}
}
