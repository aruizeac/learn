package task

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"stl/heapx"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// - Executor

type ExecutorFunc func(ctx context.Context, jobID string) error

// - Scheduler

type Scheduler struct {
	queue             *heapx.Heap[job]
	mu                sync.Mutex
	cond              *sync.Cond // condition variable for efficient blocking
	config            *SchedulerConfig
	rootProcCtx       context.Context
	rootProcCtxCancel context.CancelFunc

	workChan     chan job
	canceledJobs sync.Map //  using sync.Map eliminates need for separate mutex and provides cleaner semantics
	inFlightJobs sync.WaitGroup
	isStopped    atomic.Bool
	localClock   int64
}

func NewScheduler(opts ...SchedulerOption) *Scheduler {
	schedOptions := defaultSchedulerConfig()
	for _, opt := range opts {
		opt(schedOptions)
	}
	return NewSchedulerWithConfig(schedOptions)
}

// NewSchedulerWithConfig returns a new Scheduler with the specified configuration.
func NewSchedulerWithConfig(config *SchedulerConfig) *Scheduler {
	queue := heapx.NewHeap[job](func(a, b job) bool {
		if a.priority == b.priority {
			return a.clock < b.clock // enforce FIFO ordering
		}
		return a.priority > b.priority
	})
	heap.Init(queue)
	s := &Scheduler{
		queue:  queue,
		config: config,
	}
	// sync.Cond is bound to the mutex. When Wait() is called, it atomically
	// releases the lock and blocks. When signaled, it reacquires the lock before
	// returning. This gives us zero-CPU blocking when the queue is empty.
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *Scheduler) Start() error {
	if s.config == nil {
		s.config = defaultSchedulerConfig()
	}

	s.rootProcCtx, s.rootProcCtxCancel = context.WithCancel(context.Background())
	s.workChan = make(chan job, s.config.MaxWorkers*10)
	for i := 0; i < s.config.MaxWorkers; i++ {
		go func() {
			for j := range s.workChan {
				s.processJob(j)
			}
		}()
	}

	for {
		// Check for shutdown without holding the lock
		if s.isStopped.Load() {
			return nil
		}

		s.mu.Lock()
		// Use condition variable instead of polling.
		// This loop handles spurious wakeups—we re-check the condition after
		// every Wait() return. The isStopped check prevents deadlock on shutdown.
		for s.queue.Len() == 0 && !s.isStopped.Load() {
			s.cond.Wait() // Releases lock, blocks until Signal/Broadcast, reacquires lock
		}

		// Re-check after waking—might have been signaled for shutdown
		if s.isStopped.Load() {
			s.mu.Unlock()
			return nil
		}

		jobBatch := make([]job, 0, s.config.MaxItemsPerPull)
		for i := 0; i < s.config.MaxItemsPerPull && s.queue.Len() > 0; i++ {
			jobBatch = append(jobBatch, heap.Pop(s.queue).(job))
		}
		s.mu.Unlock()

		for _, j := range jobBatch {
			s.inFlightJobs.Add(1)
			s.workChan <- j
		}
	}
}

func (s *Scheduler) Stop(ctx context.Context) error {
	if !s.isStopped.CompareAndSwap(false, true) {
		return errors.New("scheduler is already stopped")
	}

	// Broadcast wakes up the main loop so it can observe isStopped and exit.
	// Without this, the loop would block forever on cond.Wait() if queue is empty.
	s.cond.Broadcast()

	c := make(chan struct{})
	go func() {
		s.config.Logger.InfoContext(ctx, "task.scheduler: stopping scheduler")
		s.rootProcCtxCancel()
		close(s.workChan)
		s.inFlightJobs.Wait()
		close(c)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for scheduler to stop: %w", ctx.Err())
	case <-c:
		s.config.Logger.InfoContext(ctx, "task.scheduler: scheduler stopped")
		return nil
	}
}

func (s *Scheduler) processJob(j job) {
	defer s.inFlightJobs.Done()

	// LoadAndDelete is atomic—if present, we consume the cancellation marker
	// and skip execution. This also cleans up the map entry, preventing memory leaks.
	if _, wasCanceled := s.canceledJobs.LoadAndDelete(j.id); wasCanceled {
		s.config.Logger.InfoContext(context.Background(), "task.scheduler: job was cancelled, skipping",
			slog.String("job_id", j.id))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ExecutionTimeout)
	defer cancel()
	s.config.Logger.InfoContext(ctx, "task.scheduler: processing job",
		slog.String("job_id", j.id),
		slog.Int64("clock", j.clock),
		slog.String("priority", j.priority.String()),
	)
	if err := j.executor(ctx, j.id); err != nil {
		s.config.Logger.ErrorContext(ctx, "task.scheduler: processing failed",
			slog.String("error", err.Error()),
			slog.String("job_id", j.id))
	}
}

func (s *Scheduler) Schedule(exec ExecutorFunc, opts ...ScheduleOption) (string, error) {
	if s.isStopped.Load() {
		return "", errors.New("scheduler is stopped")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	schedOptions := defaultScheduleOptions()
	for _, opt := range opts {
		opt(schedOptions)
	}

	id := strconv.FormatInt(rand.Int64(), 10)
	heap.Push(s.queue, job{
		id:       id,
		executor: exec,
		priority: schedOptions.priority,
		clock:    atomic.AddInt64(&s.localClock, 1),
	})

	// Signal wakes exactly one goroutine waiting on the condition.
	// Since we only have one consumer (the main loop), Signal is sufficient.
	// Use Broadcast if you had multiple consumer goroutines.
	s.cond.Signal()

	return id, nil
}

// JobState represents the state of a job for Cancel() return semantics.
type JobState int

const (
	JobStateQueued   JobState = iota // Job was in heap, now removed
	JobStatePending                  // Job is in workChan or executing, marked for cancellation
	JobStateNotFound                 // Job ID was never scheduled or already completed
)

// CancelResult provides detailed cancellation outcome.
type CancelResult struct {
	State   JobState
	Message string
}

// Cancel cancels the job with the specified ID.
// Returns CancelResult indicating where the job was found:
//   - JobStateQueued: Job was waiting in the priority queue and has been removed
//   - JobStatePending: Job was already dequeued (in channel or executing), marked for skip
//   - JobStateNotFound: Job ID not recognized (never scheduled, already completed, or invalid)
func (s *Scheduler) Cancel(jobID string) (CancelResult, error) {
	if s.isStopped.Load() {
		return CancelResult{}, errors.New("scheduler is stopped")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	jobIdx := s.queue.IndexOf(func(j job) bool { return j.id == jobID })

	if jobIdx != -1 {
		// Job found in heap—remove it directly
		heap.Remove(s.queue, jobIdx)
		return CancelResult{
			State:   JobStateQueued,
			Message: "job removed from queue",
		}, nil
	}

	// Job not in heap. It might be:
	// 1. In workChan buffer (dequeued but not yet picked up by worker)
	// 2. Currently executing in a worker
	// 3. Already completed
	// 4. Never scheduled (invalid ID)
	//
	// We can't distinguish 1/2 from 3/4 without additional tracking.
	// Conservative approach: mark it cancelled. If it's case 3/4, the marker
	// is orphaned but will be cleaned up by periodic GC (see below).

	// Use LoadOrStore to check if we're double-cancelling
	if _, alreadyCancelled := s.canceledJobs.LoadOrStore(jobID, struct{}{}); alreadyCancelled {
		return CancelResult{
			State:   JobStatePending,
			Message: "job already marked for cancellation",
		}, nil
	}

	return CancelResult{
		State:   JobStatePending,
		Message: "job not in queue; marked for cancellation if in-flight",
	}, nil
}

// CancelStrict is the old interface for callers who want the simple error-based API.
// Returns error only if job was definitely not found AND not pending.
// DEPRECATED: Prefer Cancel() for clearer semantics.
func (s *Scheduler) CancelStrict(jobID string) error {
	result, err := s.Cancel(jobID)
	if err != nil {
		return err
	}
	if result.State == JobStateNotFound {
		return errors.New("job not found")
	}
	return nil
}

// GCCancelledJobs removes stale entries from the canceledJobs map.
// Call this periodically if you have high cancellation rates to prevent memory growth
// from orphaned entries (jobs cancelled after completion or invalid IDs).
//
// This addresses the memory leak where Cancel() on invalid/completed job IDs
// would leave entries in the map forever. The sync.Map entries for legitimate
// cancellations are cleaned up in processJob via LoadAndDelete, but orphans remain.
//
// In production, you'd typically run this on a timer:
//
//	go func() {
//	    ticker := time.NewTicker(5 * time.Minute)
//	    for range ticker.C {
//	        scheduler.GCCancelledJobs(1 * time.Minute)
//	    }
//	}()
func (s *Scheduler) GCCancelledJobs(maxAge time.Duration) int {
	// For proper GC, we'd need to store timestamps with each cancellation.
	// This is a simplified version that just clears everything—appropriate if
	// you call it infrequently and can tolerate the race where a just-cancelled
	// job might not get skipped.
	//
	// Production version would store: canceledJobs map[string]time.Time
	// and only delete entries older than maxAge.

	count := 0
	s.canceledJobs.Range(func(key, value any) bool {
		s.canceledJobs.Delete(key)
		count++
		return true
	})
	return count
}

// -- Config/Options

type SchedulerConfig struct {
	Logger           *slog.Logger
	MaxWorkers       int
	MaxItemsPerPull  int
	ExecutionTimeout time.Duration
}

func defaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		MaxWorkers:       10,
		MaxItemsPerPull:  100,
		ExecutionTimeout: time.Second * 30,
		Logger:           slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

// SchedulerOption is a function that configures a Scheduler.
type SchedulerOption func(*SchedulerConfig)

// WithLogger sets the logger for the Scheduler.
func WithLogger(logger *slog.Logger) SchedulerOption {
	return func(c *SchedulerConfig) { c.Logger = logger }
}

// WithMaxWorkers sets the maximum number of workers for the Scheduler.
func WithMaxWorkers(maxWorkers int) SchedulerOption {
	return func(c *SchedulerConfig) { c.MaxWorkers = maxWorkers }
}

// WithMaxItemsPerPull sets the maximum number of items to pull from the queue per pull cycle.
func WithMaxItemsPerPull(maxItemsPerPull int) SchedulerOption {
	return func(c *SchedulerConfig) { c.MaxItemsPerPull = maxItemsPerPull }
}

// WithExecutionTimeout sets the timeout for each job execution.
func WithExecutionTimeout(timeout time.Duration) SchedulerOption {
	return func(c *SchedulerConfig) { c.ExecutionTimeout = timeout }
}

type scheduleOptions struct {
	priority Priority
}

func defaultScheduleOptions() *scheduleOptions {
	return &scheduleOptions{
		priority: LowPriority,
	}
}

// ScheduleOption is a function that configures a Schedule.
type ScheduleOption func(*scheduleOptions)

// WithPriority sets the priority of the job. This will make the Scheduler to execute the job with the specified priority.
func WithPriority(priority Priority) ScheduleOption {
	return func(o *scheduleOptions) { o.priority = priority }
}

// - Job

// Priority represents the priority of a job.
type Priority uint8

const (
	// LowPriority marks a job as low priority. This will make the Scheduler to execute the job after high-priority jobs.
	LowPriority Priority = iota
	// HighPriority marks a job as high priority. This will make the Scheduler to execute the job before low-priority jobs.
	HighPriority
)

var _ fmt.Stringer = Priority(0)

func (p Priority) String() string {
	if p == LowPriority {
		return "LOW"
	}
	return "HIGH"
}

type job struct {
	id       string
	clock    int64 // logical clock for job scheduling
	executor ExecutorFunc
	priority Priority
}
