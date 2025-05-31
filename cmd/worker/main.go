package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"online-shop/internal/infrastructure/queue"
	"online-shop/internal/workers"
	"online-shop/pkg/config"
	"online-shop/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	log, err := logger.NewLogger(cfg.Environment)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer log.Sync()

	log.Info("Starting worker service", zap.String("environment", cfg.Environment))

	// Initialize RabbitMQ
	rabbitmq, err := queue.NewRabbitMQ(cfg, log)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
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
			log.Error("Email worker stopped", zap.Error(err))
		}
	}()

	// Invoice worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting invoice worker")
		if err := rabbitmq.ConsumeMessages(ctx, queue.InvoiceQueue, invoiceWorker.ProcessMessage); err != nil {
			log.Error("Invoice worker stopped", zap.Error(err))
		}
	}()

	// Notification worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting notification worker")
		if err := rabbitmq.ConsumeMessages(ctx, queue.NotificationQueue, notificationWorker.ProcessMessage); err != nil {
			log.Error("Notification worker stopped", zap.Error(err))
		}
	}()

	// Analytics worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting analytics worker")
		if err := rabbitmq.ConsumeMessages(ctx, queue.AnalyticsQueue, analyticsWorker.ProcessMessage); err != nil {
			log.Error("Analytics worker stopped", zap.Error(err))
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
					log.Error("RabbitMQ health check failed", zap.Error(err))
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