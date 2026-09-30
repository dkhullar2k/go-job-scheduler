package scheduler

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"jobscheduler/internal/cronexpr"
	"jobscheduler/internal/model"
)

type Runner interface {
	Execute(context.Context, model.Job) (string, error)
}

type ExecutionLogger interface {
	Execution(jobID, output string, runErr error) error
}

type registration struct {
	job   model.Job
	cron  *cronexpr.Expression
	timer *time.Timer
}

type Service struct {
	mu       sync.Mutex
	entries  map[string]registration
	runner   Runner
	logger   ExecutionLogger
	workers  chan struct{}
	timeout  time.Duration
	stopping bool
	started  bool
	stop     chan struct{}
	loopDone chan struct{}
	runs     sync.WaitGroup
}

func New(maxConcurrency int, timeout time.Duration, runner Runner, logger ExecutionLogger) *Service {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	return &Service{
		entries:  make(map[string]registration),
		runner:   runner,
		logger:   logger,
		workers:  make(chan struct{}, maxConcurrency),
		timeout:  timeout,
		stop:     make(chan struct{}),
		loopDone: make(chan struct{}),
	}
}

func ValidateSchedule(schedule string) error {
	if _, err := time.Parse(time.RFC3339, schedule); err == nil {
		return nil
	}
	if _, err := cronexpr.Parse(schedule); err != nil {
		return fmt.Errorf("schedule must be an RFC 3339 timestamp or five/six-field cron expression: %w", err)
	}
	return nil
}

func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || s.stopping {
		return
	}
	s.started = true
	go s.runCronLoop()
}

func (s *Service) Add(job model.Job) error {
	var parsed *cronexpr.Expression
	if _, err := time.Parse(time.RFC3339, job.Schedule); err != nil {
		expression, parseErr := cronexpr.Parse(job.Schedule)
		if parseErr != nil {
			return fmt.Errorf("invalid schedule %q: %w", job.Schedule, parseErr)
		}
		parsed = &expression
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping {
		return fmt.Errorf("scheduler is stopping")
	}
	s.removeLocked(job.JobID)
	entry := registration{job: job, cron: parsed}
	if parsed == nil {
		if at, _ := time.Parse(time.RFC3339, job.Schedule); time.Until(at) > 0 {
			copyOfJob := job
			entry.timer = time.AfterFunc(time.Until(at), func() { s.launch(copyOfJob) })
		}
	}
	s.entries[job.JobID] = entry
	return nil
}

func (s *Service) Remove(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(jobID)
}

func (s *Service) removeLocked(jobID string) {
	if old, ok := s.entries[jobID]; ok {
		if old.timer != nil {
			old.timer.Stop()
		}
		delete(s.entries, jobID)
	}
}

func (s *Service) runCronLoop() {
	defer close(s.loopDone)
	for {
		now := time.Now().UTC()
		nextSecond := now.Truncate(time.Second).Add(time.Second)
		timer := time.NewTimer(time.Until(nextSecond))
		select {
		case <-s.stop:
			timer.Stop()
			return
		case <-timer.C:
			s.runMatching(time.Now().UTC().Truncate(time.Second))
		}
	}
}

func (s *Service) runMatching(minute time.Time) {
	s.mu.Lock()
	jobs := make([]model.Job, 0)
	for _, entry := range s.entries {
		if entry.cron != nil && entry.cron.Matches(minute) {
			jobs = append(jobs, entry.job)
		}
	}
	s.mu.Unlock()
	for _, job := range jobs {
		s.launch(job)
	}
}

func (s *Service) launch(job model.Job) {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		return
	}
	s.runs.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.runs.Done()
		s.workers <- struct{}{}
		defer func() { <-s.workers }()
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		output, err := s.runner.Execute(ctx, job)
		if logErr := s.logger.Execution(job.JobID, output, err); logErr != nil {
			fmt.Fprintln(os.Stderr, "write execution log:", logErr)
		}
	}()
}

func (s *Service) Stop() {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		return
	}
	s.stopping = true
	for id := range s.entries {
		s.removeLocked(id)
	}
	started := s.started
	close(s.stop)
	s.mu.Unlock()
	if started {
		<-s.loopDone
	}
	s.runs.Wait()
}
