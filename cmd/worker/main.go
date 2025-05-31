package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"online-shop/internal/infrastructure/queue"
	"online-shop/internal/workers"
	"online-shop/pkg/config"
	"online-shop/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	logger.Init(&cfg.Logger)
	log := logger.GetLogger()

	log.WithField("environment", cfg.Environment).Info("Starting worker service")

	// Initialize RabbitMQ
	rabbitmq, err := queue.NewRabbitMQ(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to RabbitMQ")
	}
	defer rabbitmq.Close()

	// Initialize workers
	emailWorker := workers.NewEmailWorker(cfg, log)
	invoiceWorker := workers.NewInvoiceWorker(cfg, log)
	notificationWorker := workers.NewNotificationWorker(cfg, log)
	analyticsWorker := workers.NewAnalyticsWorker(cfg, log)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workers
	var wg sync.WaitGroup

	// Email worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting email worker")
		if err := rabbitmq.ConsumeMessages(ctx, queue.EmailQueue, emailWorker.ProcessMessage); err != nil {
			log.WithError(err).Error("Email worker stopped")
		}
	}()

	// Invoice worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting invoice worker")
		if err := rabbitmq.ConsumeMessages(ctx, queue.InvoiceQueue, invoiceWorker.ProcessMessage); err != nil {
			log.WithError(err).Error("Invoice worker stopped")
		}
	}()

	// Notification worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting notification worker")
		if err := rabbitmq.ConsumeMessages(ctx, queue.NotificationQueue, notificationWorker.ProcessMessage); err != nil {
			log.WithError(err).Error("Notification worker stopped")
		}
	}()

	// Analytics worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting analytics worker")
		if err := rabbitmq.ConsumeMessages(ctx, queue.AnalyticsQueue, analyticsWorker.ProcessMessage); err != nil {
			log.WithError(err).Error("Analytics worker stopped")
		}
	}()

	// Health check worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		healthTicker := time.NewTicker(30 * time.Second)
		defer healthTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-healthTicker.C:
				if err := rabbitmq.HealthCheck(); err != nil {
					log.WithError(err).Error("RabbitMQ health check failed")
				}
			}
		}
	}()

	log.Info("All workers started successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Info("Received shutdown signal, stopping workers...")

	// Cancel context to stop all workers
	cancel()

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Warn("Timeout waiting for workers to stop")
	}

	log.Info("Worker service shutdown complete")
}