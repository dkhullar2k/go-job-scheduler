package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"jobscheduler/internal/logging"
	"jobscheduler/internal/parser"
	"jobscheduler/internal/scheduler"
)

type Service struct {
	dir       string
	parser    *parser.Service
	scheduler *scheduler.Service
	logger    *logging.Logger
	watcher   *fsnotify.Watcher
	watchDone chan struct{}
	mu        sync.Mutex
	byPath    map[string]string
	byID      map[string]string
}

func New(dir string, p *parser.Service, s *scheduler.Service, l *logging.Logger) *Service {
	return &Service{dir: dir, parser: p, scheduler: s, logger: l, byPath: make(map[string]string), byID: make(map[string]string)}
}

func (m *Service) Start() error {
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(m.dir); err != nil {
		watcher.Close()
		return err
	}
	m.watcher = watcher
	if err := m.scan(); err != nil {
		watcher.Close()
		return err
	}
	m.watchDone = make(chan struct{})
	go m.watch()
	return nil
}

func (m *Service) scan() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			paths = append(paths, filepath.Join(m.dir, entry.Name()))
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		m.handle(path)
	}
	return nil
}

func (m *Service) watch() {
	defer close(m.watchDone)
	pending := make(map[string]time.Time)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	defer m.watcher.Close()
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename|fsnotify.Remove) != 0 && strings.EqualFold(filepath.Ext(event.Name), ".json") {
				pending[event.Name] = time.Now()
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintln(os.Stderr, "job directory watcher:", err)
		case now := <-ticker.C:
			for path, changed := range pending {
				if now.Sub(changed) >= 200*time.Millisecond {
					m.handle(path)
					delete(pending, path)
				}
			}
		}
	}
}

func (m *Service) handle(path string) {
	path, _ = filepath.Abs(path)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		m.removePath(path)
		return
	}
	job, err := m.parser.ParseFile(path)
	if err != nil {
		m.logParse(path, "", err)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if other, exists := m.byID[job.JobID]; exists && other != path {
		m.logParse(path, job.JobID, fmt.Errorf("duplicate job_id also defined by %s", other))
		return
	}
	if err := m.scheduler.Add(job); err != nil {
		m.logParse(path, job.JobID, err)
		return
	}
	if oldID := m.byPath[path]; oldID != "" && oldID != job.JobID {
		m.scheduler.Remove(oldID)
		delete(m.byID, oldID)
	}
	m.byPath[path], m.byID[job.JobID] = job.JobID, path
	m.logParse(path, job.JobID, nil)
}

func (m *Service) removePath(path string) {
	path, _ = filepath.Abs(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if id := m.byPath[path]; id != "" {
		m.scheduler.Remove(id)
		delete(m.byPath, path)
		delete(m.byID, id)
	}
}

func (m *Service) logParse(path, jobID string, err error) {
	if logErr := m.logger.Parse(path, jobID, err); logErr != nil {
		fmt.Fprintln(os.Stderr, "write parse log:", logErr)
	}
}

func (m *Service) Close() {
	if m.watcher != nil {
		_ = m.watcher.Close()
	}
	if m.watchDone != nil {
		<-m.watchDone
	}
	m.scheduler.Stop()
}

func (m *Service) Validate() (int, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return 0, err
	}
	count := 0
	seen := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(m.dir, entry.Name())
		job, parseErr := m.parser.ParseFile(path)
		if parseErr == nil {
			parseErr = scheduler.ValidateSchedule(job.Schedule)
		}
		if parseErr == nil {
			if previous, ok := seen[job.JobID]; ok {
				parseErr = fmt.Errorf("duplicate job_id also defined by %s", previous)
			}
		}
		if parseErr == nil {
			seen[job.JobID] = path
			count++
		}
		m.logParse(path, job.JobID, parseErr)
		if parseErr != nil {
			fmt.Fprintln(os.Stderr, path+":", parseErr)
		}
	}
	return count, nil
}
