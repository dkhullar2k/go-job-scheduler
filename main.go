package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jobscheduler/internal/logging"
	"jobscheduler/internal/manager"
	"jobscheduler/internal/parser"
	"jobscheduler/internal/scheduler"
)

type options struct {
	configDir      string
	logFile        string
	maxConcurrency int
	commandTimeout time.Duration
}

func main() {
	if err := execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: jobscheduler <run|validate> [flags]")
	}
	mode := args[0]
	if mode != "run" && mode != "validate" {
		return fmt.Errorf("unknown mode %q (expected run or validate)", mode)
	}
	fs := flag.NewFlagSet(mode, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts := options{}
	fs.StringVar(&opts.configDir, "config-dir", "./jobs", "directory containing JSON job definitions")
	fs.StringVar(&opts.logFile, "log-file", "./logs/executions.jsonl", "append-only JSONL event log")
	fs.IntVar(&opts.maxConcurrency, "max-concurrency", 4, "maximum number of commands running at once")
	fs.DurationVar(&opts.commandTimeout, "command-timeout", 5*time.Minute, "maximum runtime for a command")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if opts.maxConcurrency < 1 {
		return fmt.Errorf("max-concurrency must be at least 1")
	}
	if opts.commandTimeout <= 0 {
		return fmt.Errorf("command-timeout must be positive")
	}

	log, err := logging.Open(opts.logFile)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer log.Close()

	parserService := parser.New()
	schedulerService := scheduler.New(opts.maxConcurrency, opts.commandTimeout, parserService, log)
	managerService := manager.New(opts.configDir, parserService, schedulerService, log)

	if mode == "validate" {
		count, err := managerService.Validate()
		if err != nil {
			return fmt.Errorf("validate job directory: %w", err)
		}
		fmt.Printf("Validated %d job definition(s).\n", count)
		return nil
	}

	schedulerService.Start()
	if err := managerService.Start(); err != nil {
		schedulerService.Stop()
		return fmt.Errorf("start scheduler manager: %w", err)
	}
	defer managerService.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stdout, "Job scheduler ready; watching %s\n", opts.configDir)
	<-ctx.Done()
	fmt.Fprintln(os.Stdout, "Shutting down job scheduler.")
	return nil
}
