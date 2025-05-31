package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"online-shop/internal/infrastructure/queue"
	"online-shop/pkg/config"
	"online-shop/pkg/workerpool"
)

// EmailJob represents an email processing job
type EmailJob struct {
	workerpool.BaseJob
	Message queue.Message
	Config  *config.Config
	Logger  *logrus.Logger
}

// Execute processes the email job
func (j *EmailJob) Execute(ctx context.Context) error {
	j.Logger.Debug("Executing email job", logrus.Fields{"job_id": j.ID})

	// Parse email data
	var emailData queue.EmailMessage
	if err := mapToStruct(j.Message.Payload, &emailData); err != nil {
		return fmt.Errorf("failed to parse email data: %w", err)
	}

	// Create email worker and process
	emailWorker := NewEmailWorker(j.Config, j.Logger)
	if err := emailWorker.ProcessMessage(j.Message); err != nil {
		return fmt.Errorf("failed to process email: %w", err)
	}

	return nil
}

// GetPriority returns the priority of the email job
func (j *EmailJob) GetPriority() int {
	// Parse email data to get priority
	var emailData queue.EmailMessage
	if err := mapToStruct(j.Message.Payload, &emailData); err == nil {
		return emailData.Priority
	}
	return 0 // Default priority
}

// InvoiceJob represents an invoice processing job
type InvoiceJob struct {
	workerpool.BaseJob
	Message queue.Message
	Config  *config.Config
	Logger  *logrus.Logger
}

// Execute processes the invoice job
func (j *InvoiceJob) Execute(ctx context.Context) error {
	j.Logger.Debug("Executing invoice job", logrus.Fields{"job_id": j.ID})

	// Parse invoice data
	var invoiceData queue.InvoiceMessage
	if err := mapToStruct(j.Message.Payload, &invoiceData); err != nil {
		return fmt.Errorf("failed to parse invoice data: %w", err)
	}

	// Create invoice worker and process
	invoiceWorker := NewInvoiceWorker(j.Config, j.Logger)
	if err := invoiceWorker.ProcessMessage(j.Message); err != nil {
		return fmt.Errorf("failed to process invoice: %w", err)
	}

	return nil
}

// GetPriority returns the priority of the invoice job
func (j *InvoiceJob) GetPriority() int {
	return 5 // High priority for invoices
}

// NotificationJob represents a notification processing job
type NotificationJob struct {
	workerpool.BaseJob
	Message queue.Message
	Config  *config.Config
	Logger  *logrus.Logger
}

// Execute processes the notification job
func (j *NotificationJob) Execute(ctx context.Context) error {
	j.Logger.Debug("Executing notification job", logrus.Fields{"job_id": j.ID})

	// Create notification worker and process
	notificationWorker := NewNotificationWorker(j.Config, j.Logger)
	if err := notificationWorker.ProcessMessage(j.Message); err != nil {
		return fmt.Errorf("failed to process notification: %w", err)
	}

	return nil
}

// GetPriority returns the priority of the notification job
func (j *NotificationJob) GetPriority() int {
	return 3 // Medium priority for notifications
}

// AnalyticsJob represents an analytics processing job
type AnalyticsJob struct {
	workerpool.BaseJob
	Message queue.Message
	Config  *config.Config
	Logger  *logrus.Logger
}

// Execute processes the analytics job
func (j *AnalyticsJob) Execute(ctx context.Context) error {
	j.Logger.Debug("Executing analytics job", logrus.Fields{"job_id": j.ID})

	// Create analytics worker and process
	analyticsWorker := NewAnalyticsWorker(j.Config, j.Logger)
	if err := analyticsWorker.ProcessMessage(j.Message); err != nil {
		return fmt.Errorf("failed to process analytics: %w", err)
	}

	return nil
}

// GetPriority returns the priority of the analytics job
func (j *AnalyticsJob) GetPriority() int {
	return 1 // Low priority for analytics
}

// BatchJob represents a batch processing job
type BatchJob struct {
	workerpool.BaseJob
	Jobs   []workerpool.Job
	Config *config.Config
	Logger *logrus.Logger
}

// Execute processes multiple jobs in batch
func (j *BatchJob) Execute(ctx context.Context) error {
	j.Logger.Debug("Executing batch job",
		logrus.Fields{
			"job_id":    j.ID,
			"job_count": len(j.Jobs),
		})

	start := time.Now()
	successCount := 0
	errorCount := 0

	for i, job := range j.Jobs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := job.Execute(ctx); err != nil {
				j.Logger.Error("Batch job item failed",
					logrus.Fields{
						"batch_job_id": j.ID,
						"item_index":   i,
						"item_id":      job.GetID(),
						"error":        err.Error(),
					})
				errorCount++
			} else {
				successCount++
			}
		}
	}

	duration := time.Since(start)
	j.Logger.Info("Batch job completed",
		logrus.Fields{
			"job_id":       j.ID,
			"total_jobs":   len(j.Jobs),
			"success_count": successCount,
			"error_count":  errorCount,
			"duration":     duration,
		})

	if errorCount > 0 {
		return fmt.Errorf("batch job completed with %d errors out of %d jobs", errorCount, len(j.Jobs))
	}

	return nil
}

// GetPriority returns the priority of the batch job
func (j *BatchJob) GetPriority() int {
	return 2 // Medium-low priority for batch jobs
}

// RetryableJob wraps a job with retry logic
type RetryableJob struct {
	workerpool.BaseJob
	OriginalJob workerpool.Job
	MaxRetries  int
	RetryDelay  time.Duration
	Config      *config.Config
	Logger      *logrus.Logger
}

// Execute processes the job with retry logic
func (j *RetryableJob) Execute(ctx context.Context) error {
	var lastErr error

	for attempt := 1; attempt <= j.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			j.Logger.Debug("Attempting job execution",
				logrus.Fields{
					"job_id":  j.ID,
					"attempt": attempt,
					"max_retries": j.MaxRetries,
				})

			if err := j.OriginalJob.Execute(ctx); err != nil {
				lastErr = err
				j.Logger.Warn("Job execution failed, will retry",
					logrus.Fields{
						"job_id":  j.ID,
						"attempt": attempt,
						"error":   err.Error(),
					})

				if attempt < j.MaxRetries {
					// Wait before retry with exponential backoff
					delay := j.RetryDelay * time.Duration(attempt)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(delay):
						continue
					}
				}
			} else {
				j.Logger.Debug("Job executed successfully",
					logrus.Fields{
						"job_id":  j.ID,
						"attempt": attempt,
					})
				return nil
			}
		}
	}

	return fmt.Errorf("job failed after %d attempts: %w", j.MaxRetries, lastErr)
}

// GetPriority returns the priority of the original job
func (j *RetryableJob) GetPriority() int {
	return j.OriginalJob.GetPriority()
}

// ScheduledJob represents a job that should be executed at a specific time
type ScheduledJob struct {
	workerpool.BaseJob
	OriginalJob   workerpool.Job
	ScheduledTime time.Time
	Config        *config.Config
	Logger        *logrus.Logger
}

// Execute waits until the scheduled time and then executes the job
func (j *ScheduledJob) Execute(ctx context.Context) error {
	now := time.Now()
	if j.ScheduledTime.After(now) {
		delay := j.ScheduledTime.Sub(now)
		j.Logger.Debug("Waiting for scheduled time",
			logrus.Fields{
				"job_id":         j.ID,
				"scheduled_time": j.ScheduledTime,
				"delay":          delay,
			})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to execute the job
		}
	}

	j.Logger.Debug("Executing scheduled job", logrus.Fields{"job_id": j.ID})
	return j.OriginalJob.Execute(ctx)
}

// GetPriority returns the priority of the original job
func (j *ScheduledJob) GetPriority() int {
	return j.OriginalJob.GetPriority()
}

// Helper function to convert map to struct
func mapToStruct(m map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}