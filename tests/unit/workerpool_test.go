package unit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"online-shop/pkg/workerpool"
)

// TestJob implements the Job interface for testing
type TestJob struct {
	id          string
	jobType     string
	priority    int
	retryCount  int
	maxRetries  int
	shouldFail  bool
	executed    bool
	mu          sync.Mutex
	onSuccess   func()
	onFailure   func(error)
	executeFunc func() error
}

func NewTestJob(id, jobType string, priority int) *TestJob {
	return &TestJob{
		id:         id,
		jobType:    jobType,
		priority:   priority,
		maxRetries: 3,
	}
}

func (j *TestJob) Execute() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	
	j.executed = true
	
	if j.executeFunc != nil {
		return j.executeFunc()
	}
	
	if j.shouldFail {
		return errors.New("job failed")
	}
	
	return nil
}

func (j *TestJob) GetID() string {
	return j.id
}

func (j *TestJob) GetType() string {
	return j.jobType
}

func (j *TestJob) GetPriority() int {
	return j.priority
}

func (j *TestJob) GetRetryCount() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.retryCount
}

func (j *TestJob) GetMaxRetries() int {
	return j.maxRetries
}

func (j *TestJob) ShouldRetry() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.retryCount < j.maxRetries
}

func (j *TestJob) OnSuccess() {
	if j.onSuccess != nil {
		j.onSuccess()
	}
}

func (j *TestJob) OnFailure(err error) {
	j.mu.Lock()
	j.retryCount++
	j.mu.Unlock()
	
	if j.onFailure != nil {
		j.onFailure(err)
	}
}

func (j *TestJob) IsExecuted() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.executed
}

func (j *TestJob) SetShouldFail(shouldFail bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.shouldFail = shouldFail
}

func (j *TestJob) SetExecuteFunc(fn func() error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.executeFunc = fn
}

func (j *TestJob) SetOnSuccess(fn func()) {
	j.onSuccess = fn
}

func (j *TestJob) SetOnFailure(fn func(error)) {
	j.onFailure = fn
}

func TestWorkerPool_NewWorkerPool(t *testing.T) {
	tests := []struct {
		name        string
		workers     int
		queueSize   int
		expectError bool
	}{
		{
			name:        "valid configuration",
			workers:     5,
			queueSize:   100,
			expectError: false,
		},
		{
			name:        "zero workers",
			workers:     0,
			queueSize:   100,
			expectError: true,
		},
		{
			name:        "negative workers",
			workers:     -1,
			queueSize:   100,
			expectError: true,
		},
		{
			name:        "zero queue size",
			workers:     5,
			queueSize:   0,
			expectError: true,
		},
		{
			name:        "negative queue size",
			workers:     5,
			queueSize:   -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := workerpool.NewWorkerPool(tt.workers, tt.queueSize)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, pool)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pool)
				assert.Equal(t, tt.workers, pool.GetWorkerCount())
				assert.False(t, pool.IsRunning())
			}
		})
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(3, 10)
	require.NoError(t, err)
	require.NotNil(t, pool)

	// Test start
	err = pool.Start()
	assert.NoError(t, err)
	assert.True(t, pool.IsRunning())

	// Test double start
	err = pool.Start()
	assert.Error(t, err)

	// Test stop
	err = pool.Stop()
	assert.NoError(t, err)
	assert.False(t, pool.IsRunning())

	// Test double stop
	err = pool.Stop()
	assert.Error(t, err)
}

func TestWorkerPool_SubmitJob(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(2, 5)
	require.NoError(t, err)
	require.NotNil(t, pool)

	// Test submit job when pool is not running
	job := NewTestJob("test-1", "test", 1)
	err = pool.SubmitJob(job)
	assert.Error(t, err)

	// Start pool
	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Test submit valid job
	job = NewTestJob("test-2", "test", 1)
	err = pool.SubmitJob(job)
	assert.NoError(t, err)

	// Wait for job to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, job.IsExecuted())

	// Test submit nil job
	err = pool.SubmitJob(nil)
	assert.Error(t, err)
}

func TestWorkerPool_JobExecution(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(2, 10)
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	t.Run("successful job execution", func(t *testing.T) {
		var successCalled bool
		job := NewTestJob("success-job", "test", 1)
		job.SetOnSuccess(func() {
			successCalled = true
		})

		err = pool.SubmitJob(job)
		assert.NoError(t, err)

		// Wait for job to complete
		time.Sleep(100 * time.Millisecond)
		
		assert.True(t, job.IsExecuted())
		assert.True(t, successCalled)
		assert.Equal(t, 0, job.GetRetryCount())
	})

	t.Run("failed job execution with retry", func(t *testing.T) {
		var failureCalled bool
		var failureError error
		
		job := NewTestJob("failure-job", "test", 1)
		job.SetShouldFail(true)
		job.SetOnFailure(func(err error) {
			failureCalled = true
			failureError = err
		})

		err = pool.SubmitJob(job)
		assert.NoError(t, err)

		// Wait for job to complete with retries
		time.Sleep(500 * time.Millisecond)
		
		assert.True(t, job.IsExecuted())
		assert.True(t, failureCalled)
		assert.NotNil(t, failureError)
		assert.Equal(t, job.GetMaxRetries(), job.GetRetryCount())
	})

	t.Run("job with custom execution function", func(t *testing.T) {
		var customFuncCalled bool
		
		job := NewTestJob("custom-job", "test", 1)
		job.SetExecuteFunc(func() error {
			customFuncCalled = true
			return nil
		})

		err = pool.SubmitJob(job)
		assert.NoError(t, err)

		// Wait for job to complete
		time.Sleep(100 * time.Millisecond)
		
		assert.True(t, job.IsExecuted())
		assert.True(t, customFuncCalled)
	})
}

func TestWorkerPool_ConcurrentJobExecution(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(3, 20)
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	const numJobs = 10
	jobs := make([]*TestJob, numJobs)
	var wg sync.WaitGroup

	// Create jobs with delays to test concurrency
	for i := 0; i < numJobs; i++ {
		job := NewTestJob(
			fmt.Sprintf("concurrent-job-%d", i),
			"test",
			1,
		)
		
		job.SetExecuteFunc(func() error {
			time.Sleep(50 * time.Millisecond) // Simulate work
			return nil
		})
		
		job.SetOnSuccess(func() {
			wg.Done()
		})
		
		jobs[i] = job
		wg.Add(1)
	}

	// Submit all jobs
	start := time.Now()
	for _, job := range jobs {
		err = pool.SubmitJob(job)
		assert.NoError(t, err)
	}

	// Wait for all jobs to complete
	wg.Wait()
	duration := time.Since(start)

	// Verify all jobs were executed
	for i, job := range jobs {
		assert.True(t, job.IsExecuted(), "Job %d was not executed", i)
	}

	// With 3 workers and 10 jobs taking 50ms each, it should take less than
	// 10 * 50ms = 500ms (sequential) but more than 50ms (if all parallel)
	// Expected: around 4 * 50ms = 200ms (10 jobs / 3 workers ≈ 4 batches)
	assert.Less(t, duration, 400*time.Millisecond, "Jobs took too long, concurrency may not be working")
	assert.Greater(t, duration, 100*time.Millisecond, "Jobs completed too quickly")
}

func TestWorkerPool_Metrics(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(2, 10)
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Submit successful jobs
	for i := 0; i < 3; i++ {
		job := NewTestJob(fmt.Sprintf("success-%d", i), "test", 1)
		err = pool.SubmitJob(job)
		assert.NoError(t, err)
	}

	// Submit failing jobs
	for i := 0; i < 2; i++ {
		job := NewTestJob(fmt.Sprintf("failure-%d", i), "test", 1)
		job.SetShouldFail(true)
		err = pool.SubmitJob(job)
		assert.NoError(t, err)
	}

	// Wait for jobs to complete
	time.Sleep(500 * time.Millisecond)

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(5), metrics.TotalJobs)
	assert.Equal(t, int64(3), metrics.CompletedJobs)
	assert.Equal(t, int64(2), metrics.FailedJobs)
	assert.Equal(t, 2, metrics.ActiveWorkers)
	assert.GreaterOrEqual(t, metrics.QueueSize, 0)
	assert.Greater(t, metrics.AverageTime, time.Duration(0))
}

func TestWorkerPool_QueueManagement(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(1, 3) // Small queue to test overflow
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Submit jobs that will block the worker
	blockingJob := NewTestJob("blocking", "test", 1)
	blockingJob.SetExecuteFunc(func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	err = pool.SubmitJob(blockingJob)
	assert.NoError(t, err)

	// Submit jobs to fill the queue
	for i := 0; i < 3; i++ {
		job := NewTestJob(fmt.Sprintf("queued-%d", i), "test", 1)
		err = pool.SubmitJob(job)
		assert.NoError(t, err)
	}

	// Queue should be full now, next job should fail
	overflowJob := NewTestJob("overflow", "test", 1)
	err = pool.SubmitJob(overflowJob)
	assert.Error(t, err)

	// Check queue size
	assert.Equal(t, 3, pool.GetQueueSize())

	// Wait for jobs to complete
	time.Sleep(300 * time.Millisecond)

	// Queue should be empty now
	assert.Equal(t, 0, pool.GetQueueSize())
}

func TestWorkerPool_WorkerScaling(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(2, 10)
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Test initial worker count
	assert.Equal(t, 2, pool.GetWorkerCount())

	// Test scaling up
	err = pool.SetWorkerCount(5)
	assert.NoError(t, err)
	assert.Equal(t, 5, pool.GetWorkerCount())

	// Test scaling down
	err = pool.SetWorkerCount(3)
	assert.NoError(t, err)
	assert.Equal(t, 3, pool.GetWorkerCount())

	// Test invalid worker count
	err = pool.SetWorkerCount(0)
	assert.Error(t, err)
	assert.Equal(t, 3, pool.GetWorkerCount()) // Should remain unchanged

	err = pool.SetWorkerCount(-1)
	assert.Error(t, err)
	assert.Equal(t, 3, pool.GetWorkerCount()) // Should remain unchanged
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(2, 10)
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)

	var completedJobs int32
	var mu sync.Mutex

	// Submit long-running jobs
	for i := 0; i < 5; i++ {
		job := NewTestJob(fmt.Sprintf("long-job-%d", i), "test", 1)
		job.SetExecuteFunc(func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		job.SetOnSuccess(func() {
			mu.Lock()
			completedJobs++
			mu.Unlock()
		})

		err = pool.SubmitJob(job)
		assert.NoError(t, err)
	}

	// Give jobs time to start
	time.Sleep(50 * time.Millisecond)

	// Stop pool (should wait for running jobs to complete)
	start := time.Now()
	err = pool.Stop()
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.False(t, pool.IsRunning())

	// Should have taken some time to complete running jobs
	assert.Greater(t, duration, 50*time.Millisecond)

	// All jobs should have completed
	mu.Lock()
	finalCompletedJobs := completedJobs
	mu.Unlock()
	
	assert.Equal(t, int32(5), finalCompletedJobs)
}

func TestWorkerPool_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	pool, err := workerpool.NewWorkerPool(10, 100)
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	const numJobs = 1000
	var completedJobs int64
	var failedJobs int64
	var wg sync.WaitGroup

	start := time.Now()

	// Submit many jobs
	for i := 0; i < numJobs; i++ {
		job := NewTestJob(fmt.Sprintf("stress-job-%d", i), "test", 1)
		
		// Make some jobs fail randomly
		if i%10 == 0 {
			job.SetShouldFail(true)
		}

		job.SetOnSuccess(func() {
			atomic.AddInt64(&completedJobs, 1)
			wg.Done()
		})

		job.SetOnFailure(func(err error) {
			if !job.ShouldRetry() {
				atomic.AddInt64(&failedJobs, 1)
				wg.Done()
			}
		})

		wg.Add(1)
		err = pool.SubmitJob(job)
		assert.NoError(t, err)
	}

	// Wait for all jobs to complete
	wg.Wait()
	duration := time.Since(start)

	t.Logf("Processed %d jobs in %v", numJobs, duration)
	t.Logf("Completed: %d, Failed: %d", completedJobs, failedJobs)

	// Verify metrics
	metrics := pool.GetMetrics()
	assert.Equal(t, int64(numJobs), metrics.TotalJobs)
	assert.Equal(t, completedJobs, metrics.CompletedJobs)
	assert.Equal(t, failedJobs, metrics.FailedJobs)
	assert.Equal(t, 10, metrics.ActiveWorkers)

	// Should process jobs reasonably quickly
	assert.Less(t, duration, 10*time.Second)
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	pool, err := workerpool.NewWorkerPool(2, 10)
	require.NoError(t, err)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var cancelledJobs int32

	// Submit jobs that would take longer than the context timeout
	for i := 0; i < 5; i++ {
		job := NewTestJob(fmt.Sprintf("context-job-%d", i), "test", 1)
		job.SetExecuteFunc(func() error {
			select {
			case <-ctx.Done():
				atomic.AddInt32(&cancelledJobs, 1)
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return nil
			}
		})

		err = pool.SubmitJob(job)
		assert.NoError(t, err)
	}

	// Wait for context to timeout
	<-ctx.Done()

	// Wait a bit more for jobs to process the cancellation
	time.Sleep(150 * time.Millisecond)

	// Some jobs should have been cancelled
	assert.Greater(t, atomic.LoadInt32(&cancelledJobs), int32(0))
}