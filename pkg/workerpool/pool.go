package workerpool

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Job represents a unit of work
type Job interface {
	Execute(ctx context.Context) error
	GetID() string
	GetType() string
	GetPriority() int
}

// Worker represents a worker in the pool
type Worker struct {
	ID       int
	JobQueue chan Job
	QuitChan chan bool
	Logger   *logrus.Logger
}

// WorkerPool represents a pool of workers
type WorkerPool struct {
	workers    []*Worker
	jobQueue   chan Job
	quitChan   chan bool
	wg         sync.WaitGroup
	logger     *logrus.Logger
	maxWorkers int
	maxQueue   int
	metrics    *PoolMetrics
	mu         sync.RWMutex
}

// PoolMetrics tracks pool performance
type PoolMetrics struct {
	JobsProcessed   int64
	JobsFailed      int64
	JobsInQueue     int64
	ActiveWorkers   int64
	TotalWorkers    int64
	AverageJobTime  time.Duration
	LastJobTime     time.Time
	mu              sync.RWMutex
}

// PoolConfig contains configuration for the worker pool
type PoolConfig struct {
	MaxWorkers int
	MaxQueue   int
	Logger     *logrus.Logger
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config PoolConfig) *WorkerPool {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}
	if config.MaxQueue <= 0 {
		config.MaxQueue = config.MaxWorkers * 100
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	pool := &WorkerPool{
		workers:    make([]*Worker, 0, config.MaxWorkers),
		jobQueue:   make(chan Job, config.MaxQueue),
		quitChan:   make(chan bool),
		logger:     config.Logger,
		maxWorkers: config.MaxWorkers,
		maxQueue:   config.MaxQueue,
		metrics: &PoolMetrics{
			TotalWorkers: int64(config.MaxWorkers),
		},
	}

	return pool
}

// Start initializes and starts all workers
func (p *WorkerPool) Start(ctx context.Context) {
	p.logger.Info("Starting worker pool",
		logrus.Fields{
			"max_workers": p.maxWorkers,
			"max_queue":   p.maxQueue,
		})

	for i := 0; i < p.maxWorkers; i++ {
		worker := &Worker{
			ID:       i + 1,
			JobQueue: make(chan Job),
			QuitChan: make(chan bool),
			Logger:   p.logger,
		}

		p.workers = append(p.workers, worker)
		p.wg.Add(1)
		go p.startWorker(ctx, worker)
	}

	// Start job dispatcher
	go p.dispatch(ctx)

	p.logger.Info("Worker pool started successfully")
}

// Stop gracefully stops all workers
func (p *WorkerPool) Stop() {
	p.logger.Info("Stopping worker pool...")

	close(p.quitChan)

	// Stop all workers
	for _, worker := range p.workers {
		worker.QuitChan <- true
	}

	// Wait for all workers to finish
	p.wg.Wait()

	p.logger.Info("Worker pool stopped")
}

// Submit adds a job to the queue
func (p *WorkerPool) Submit(job Job) error {
	select {
	case p.jobQueue <- job:
		p.updateMetrics(func(m *PoolMetrics) {
			m.JobsInQueue++
		})
		return nil
	default:
		return fmt.Errorf("job queue is full")
	}
}

// SubmitWithTimeout adds a job to the queue with timeout
func (p *WorkerPool) SubmitWithTimeout(job Job, timeout time.Duration) error {
	select {
	case p.jobQueue <- job:
		p.updateMetrics(func(m *PoolMetrics) {
			m.JobsInQueue++
		})
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout submitting job")
	}
}

// GetMetrics returns current pool metrics
func (p *WorkerPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// GetQueueSize returns current queue size
func (p *WorkerPool) GetQueueSize() int {
	return len(p.jobQueue)
}

// GetActiveWorkers returns number of active workers
func (p *WorkerPool) GetActiveWorkers() int64 {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return p.metrics.ActiveWorkers
}

// startWorker starts a single worker
func (p *WorkerPool) startWorker(ctx context.Context, worker *Worker) {
	defer p.wg.Done()

	p.logger.Debug("Starting worker", logrus.Fields{"worker_id": worker.ID})

	for {
		select {
		case <-ctx.Done():
			p.logger.Debug("Worker stopping due to context cancellation",
				logrus.Fields{"worker_id": worker.ID})
			return
		case <-worker.QuitChan:
			p.logger.Debug("Worker stopping due to quit signal",
				logrus.Fields{"worker_id": worker.ID})
			return
		case job := <-worker.JobQueue:
			p.processJob(ctx, worker, job)
		}
	}
}

// processJob processes a single job
func (p *WorkerPool) processJob(ctx context.Context, worker *Worker, job Job) {
	startTime := time.Now()

	p.updateMetrics(func(m *PoolMetrics) {
		m.ActiveWorkers++
		m.JobsInQueue--
	})

	defer func() {
		duration := time.Since(startTime)
		p.updateMetrics(func(m *PoolMetrics) {
			m.ActiveWorkers--
			m.JobsProcessed++
			m.LastJobTime = time.Now()
			// Update average job time (simple moving average)
			if m.AverageJobTime == 0 {
				m.AverageJobTime = duration
			} else {
				m.AverageJobTime = (m.AverageJobTime + duration) / 2
			}
		})
	}()

	worker.Logger.Debug("Processing job",
		logrus.Fields{
			"worker_id": worker.ID,
			"job_id":    job.GetID(),
			"job_type":  job.GetType(),
		})

	if err := job.Execute(ctx); err != nil {
		p.updateMetrics(func(m *PoolMetrics) {
			m.JobsFailed++
		})

		worker.Logger.Error("Job execution failed",
			logrus.Fields{
				"worker_id": worker.ID,
				"job_id":    job.GetID(),
				"job_type":  job.GetType(),
				"error":     err.Error(),
				"duration":  time.Since(startTime),
			})
		return
	}

	worker.Logger.Debug("Job completed successfully",
		logrus.Fields{
			"worker_id": worker.ID,
			"job_id":    job.GetID(),
			"job_type":  job.GetType(),
			"duration":  time.Since(startTime),
		})
}

// dispatch distributes jobs to workers
func (p *WorkerPool) dispatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Job dispatcher stopping due to context cancellation")
			return
		case <-p.quitChan:
			p.logger.Info("Job dispatcher stopping due to quit signal")
			return
		case job := <-p.jobQueue:
			// Find available worker
			go func(j Job) {
				select {
				case <-ctx.Done():
					return
				default:
					// Round-robin job assignment
					workerIndex := int(p.metrics.JobsProcessed) % len(p.workers)
					worker := p.workers[workerIndex]
					
					select {
					case worker.JobQueue <- j:
						// Job assigned successfully
					case <-time.After(5 * time.Second):
						// Worker is busy, try next worker
						for i := 0; i < len(p.workers); i++ {
							nextIndex := (workerIndex + i + 1) % len(p.workers)
							nextWorker := p.workers[nextIndex]
							select {
							case nextWorker.JobQueue <- j:
								return
							default:
								continue
							}
						}
						// All workers are busy, log warning
						p.logger.Warn("All workers are busy, job may be delayed",
							logrus.Fields{
								"job_id":   j.GetID(),
								"job_type": j.GetType(),
							})
						// Try to assign to first available worker (blocking)
						p.workers[0].JobQueue <- j
					}
				}
			}(job)
		}
	}
}

// updateMetrics safely updates pool metrics
func (p *WorkerPool) updateMetrics(updateFunc func(*PoolMetrics)) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	updateFunc(p.metrics)
}

// PriorityJob represents a job with priority
type PriorityJob struct {
	BaseJob
	Priority int
}

// BaseJob provides basic job implementation
type BaseJob struct {
	ID   string
	Type string
}

func (b BaseJob) GetID() string {
	return b.ID
}

func (b BaseJob) GetType() string {
	return b.Type
}

func (b BaseJob) GetPriority() int {
	return 0 // Default priority
}

func (p PriorityJob) GetPriority() int {
	return p.Priority
}