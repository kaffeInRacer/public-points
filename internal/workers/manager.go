package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"online-shop/internal/infrastructure/queue"
	"online-shop/pkg/config"
	"online-shop/pkg/workerpool"
)

// WorkerManager manages all worker pools
type WorkerManager struct {
	emailPool        *workerpool.WorkerPool
	invoicePool      *workerpool.WorkerPool
	notificationPool *workerpool.WorkerPool
	analyticsPool    *workerpool.WorkerPool
	rabbitmq         *queue.RabbitMQ
	config           *config.Config
	logger           *logrus.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewWorkerManager creates a new worker manager
func NewWorkerManager(cfg *config.Config, rabbitmq *queue.RabbitMQ, logger *logrus.Logger) *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &WorkerManager{
		rabbitmq: rabbitmq,
		config:   cfg,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize worker pools
	manager.initializePools()

	return manager
}

// initializePools creates all worker pools
func (m *WorkerManager) initializePools() {
	// Email worker pool
	m.emailPool = workerpool.NewWorkerPool(workerpool.PoolConfig{
		MaxWorkers: m.config.Workers.EmailWorkers,
		MaxQueue:   m.config.Workers.EmailWorkers * 50,
		Logger:     m.logger,
	})

	// Invoice worker pool
	m.invoicePool = workerpool.NewWorkerPool(workerpool.PoolConfig{
		MaxWorkers: m.config.Workers.InvoiceWorkers,
		MaxQueue:   m.config.Workers.InvoiceWorkers * 30,
		Logger:     m.logger,
	})

	// Notification worker pool
	m.notificationPool = workerpool.NewWorkerPool(workerpool.PoolConfig{
		MaxWorkers: m.config.Workers.NotificationWorkers,
		MaxQueue:   m.config.Workers.NotificationWorkers * 40,
		Logger:     m.logger,
	})

	// Analytics worker pool
	m.analyticsPool = workerpool.NewWorkerPool(workerpool.PoolConfig{
		MaxWorkers: m.config.Workers.AnalyticsWorkers,
		MaxQueue:   m.config.Workers.AnalyticsWorkers * 20,
		Logger:     m.logger,
	})
}

// Start starts all worker pools and consumers
func (m *WorkerManager) Start() error {
	m.logger.Info("Starting worker manager...")

	// Start all worker pools
	m.emailPool.Start(m.ctx)
	m.invoicePool.Start(m.ctx)
	m.notificationPool.Start(m.ctx)
	m.analyticsPool.Start(m.ctx)

	// Start queue consumers
	m.wg.Add(4)
	go m.startEmailConsumer()
	go m.startInvoiceConsumer()
	go m.startNotificationConsumer()
	go m.startAnalyticsConsumer()

	// Start metrics reporter
	go m.startMetricsReporter()

	m.logger.Info("Worker manager started successfully")
	return nil
}

// Stop gracefully stops all workers
func (m *WorkerManager) Stop() {
	m.logger.Info("Stopping worker manager...")

	// Cancel context to stop consumers
	m.cancel()

	// Wait for consumers to finish
	m.wg.Wait()

	// Stop worker pools
	m.emailPool.Stop()
	m.invoicePool.Stop()
	m.notificationPool.Stop()
	m.analyticsPool.Stop()

	m.logger.Info("Worker manager stopped")
}

// startEmailConsumer starts the email queue consumer
func (m *WorkerManager) startEmailConsumer() {
	defer m.wg.Done()

	m.logger.Info("Starting email consumer")

	err := m.rabbitmq.ConsumeMessages(m.ctx, queue.EmailQueue, func(message queue.Message) error {
		job := &EmailJob{
			BaseJob: workerpool.BaseJob{
				ID:   message.ID,
				Type: "email",
			},
			Message: message,
			Config:  m.config,
			Logger:  m.logger,
		}

		return m.emailPool.Submit(job)
	})

	if err != nil && err != context.Canceled {
		m.logger.Error("Email consumer error", logrus.Fields{"error": err})
	}
}

// startInvoiceConsumer starts the invoice queue consumer
func (m *WorkerManager) startInvoiceConsumer() {
	defer m.wg.Done()

	m.logger.Info("Starting invoice consumer")

	err := m.rabbitmq.ConsumeMessages(m.ctx, queue.InvoiceQueue, func(message queue.Message) error {
		job := &InvoiceJob{
			BaseJob: workerpool.BaseJob{
				ID:   message.ID,
				Type: "invoice",
			},
			Message: message,
			Config:  m.config,
			Logger:  m.logger,
		}

		return m.invoicePool.Submit(job)
	})

	if err != nil && err != context.Canceled {
		m.logger.Error("Invoice consumer error", logrus.Fields{"error": err})
	}
}

// startNotificationConsumer starts the notification queue consumer
func (m *WorkerManager) startNotificationConsumer() {
	defer m.wg.Done()

	m.logger.Info("Starting notification consumer")

	err := m.rabbitmq.ConsumeMessages(m.ctx, queue.NotificationQueue, func(message queue.Message) error {
		job := &NotificationJob{
			BaseJob: workerpool.BaseJob{
				ID:   message.ID,
				Type: "notification",
			},
			Message: message,
			Config:  m.config,
			Logger:  m.logger,
		}

		return m.notificationPool.Submit(job)
	})

	if err != nil && err != context.Canceled {
		m.logger.Error("Notification consumer error", logrus.Fields{"error": err})
	}
}

// startAnalyticsConsumer starts the analytics queue consumer
func (m *WorkerManager) startAnalyticsConsumer() {
	defer m.wg.Done()

	m.logger.Info("Starting analytics consumer")

	err := m.rabbitmq.ConsumeMessages(m.ctx, queue.AnalyticsQueue, func(message queue.Message) error {
		job := &AnalyticsJob{
			BaseJob: workerpool.BaseJob{
				ID:   message.ID,
				Type: "analytics",
			},
			Message: message,
			Config:  m.config,
			Logger:  m.logger,
		}

		return m.analyticsPool.Submit(job)
	})

	if err != nil && err != context.Canceled {
		m.logger.Error("Analytics consumer error", logrus.Fields{"error": err})
	}
}

// startMetricsReporter starts the metrics reporting goroutine
func (m *WorkerManager) startMetricsReporter() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.reportMetrics()
		}
	}
}

// reportMetrics logs current worker pool metrics
func (m *WorkerManager) reportMetrics() {
	emailMetrics := m.emailPool.GetMetrics()
	invoiceMetrics := m.invoicePool.GetMetrics()
	notificationMetrics := m.notificationPool.GetMetrics()
	analyticsMetrics := m.analyticsPool.GetMetrics()

	m.logger.Info("Worker pool metrics",
		logrus.Fields{
			"email_pool": logrus.Fields{
				"jobs_processed":  emailMetrics.JobsProcessed,
				"jobs_failed":     emailMetrics.JobsFailed,
				"jobs_in_queue":   emailMetrics.JobsInQueue,
				"active_workers":  emailMetrics.ActiveWorkers,
				"avg_job_time":    emailMetrics.AverageJobTime,
			},
			"invoice_pool": logrus.Fields{
				"jobs_processed":  invoiceMetrics.JobsProcessed,
				"jobs_failed":     invoiceMetrics.JobsFailed,
				"jobs_in_queue":   invoiceMetrics.JobsInQueue,
				"active_workers":  invoiceMetrics.ActiveWorkers,
				"avg_job_time":    invoiceMetrics.AverageJobTime,
			},
			"notification_pool": logrus.Fields{
				"jobs_processed":  notificationMetrics.JobsProcessed,
				"jobs_failed":     notificationMetrics.JobsFailed,
				"jobs_in_queue":   notificationMetrics.JobsInQueue,
				"active_workers":  notificationMetrics.ActiveWorkers,
				"avg_job_time":    notificationMetrics.AverageJobTime,
			},
			"analytics_pool": logrus.Fields{
				"jobs_processed":  analyticsMetrics.JobsProcessed,
				"jobs_failed":     analyticsMetrics.JobsFailed,
				"jobs_in_queue":   analyticsMetrics.JobsInQueue,
				"active_workers":  analyticsMetrics.ActiveWorkers,
				"avg_job_time":    analyticsMetrics.AverageJobTime,
			},
		})
}

// GetPoolMetrics returns metrics for all pools
func (m *WorkerManager) GetPoolMetrics() map[string]workerpool.PoolMetrics {
	return map[string]workerpool.PoolMetrics{
		"email":        m.emailPool.GetMetrics(),
		"invoice":      m.invoicePool.GetMetrics(),
		"notification": m.notificationPool.GetMetrics(),
		"analytics":    m.analyticsPool.GetMetrics(),
	}
}

// HealthCheck performs health check on all worker pools
func (m *WorkerManager) HealthCheck() error {
	// Check if context is cancelled
	if m.ctx.Err() != nil {
		return fmt.Errorf("worker manager context cancelled")
	}

	// Check RabbitMQ connection
	if err := m.rabbitmq.HealthCheck(); err != nil {
		return fmt.Errorf("rabbitmq health check failed: %w", err)
	}

	// Check worker pool queue sizes
	if m.emailPool.GetQueueSize() > m.config.Workers.EmailWorkers*40 {
		return fmt.Errorf("email queue is too full")
	}

	if m.invoicePool.GetQueueSize() > m.config.Workers.InvoiceWorkers*25 {
		return fmt.Errorf("invoice queue is too full")
	}

	if m.notificationPool.GetQueueSize() > m.config.Workers.NotificationWorkers*35 {
		return fmt.Errorf("notification queue is too full")
	}

	if m.analyticsPool.GetQueueSize() > m.config.Workers.AnalyticsWorkers*15 {
		return fmt.Errorf("analytics queue is too full")
	}

	return nil
}