package main

import (
	"2_2-task-scheduler/task"
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"
)

const maxJobs = 10

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	scheduler := task.NewScheduler(
		task.WithLogger(logger),
		task.WithMaxWorkers(1),
	)
	jobIds := make([]string, 0, maxJobs)
	for i := 0; i < maxJobs; i++ {
		isHighPriority := i%2 == 0
		priority := task.LowPriority
		if isHighPriority {
			priority = task.HighPriority
		}
		jobID, err := scheduler.Schedule(sampleExecutor, task.WithPriority(priority))
		if err != nil {
			logger.Error("failed to schedule job", "error", err)
			continue
		}
		jobIds = append(jobIds, jobID)
	}

	go func() {
		if err := scheduler.Start(); err != nil {
			logger.Error("failed to start task scheduler", "error", err)
			return
		}
	}()

	shutdown := make(chan os.Signal, 1)
	defer close(shutdown)
	signal.Notify(shutdown, os.Interrupt)
	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // timeout for shutdown
	defer cancel()
	if err := scheduler.Stop(ctx); err != nil {
		logger.Error("failed to stop task scheduler", "error", err)
	}
}

func sampleExecutor(ctx context.Context, id string) error {
	log.Printf("executing job %s", id)
	return nil
}
