package workers

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"online-shop/internal/infrastructure/queue"
	"online-shop/internal/utils"
	"online-shop/pkg/config"
)

// AnalyticsWorker handles analytics event processing
type AnalyticsWorker struct {
	config *config.Config
	logger *logrus.Logger
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	EventID     string                 `json:"event_id"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	EventType   string                 `json:"event_type"`
	EventName   string                 `json:"event_name"`
	Properties  map[string]interface{} `json:"properties"`
	Timestamp   time.Time              `json:"timestamp"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Referrer    string                 `json:"referrer,omitempty"`
	PageURL     string                 `json:"page_url,omitempty"`
	DeviceType  string                 `json:"device_type,omitempty"`
	Platform    string                 `json:"platform,omitempty"`
	Country     string                 `json:"country,omitempty"`
	City        string                 `json:"city,omitempty"`
}

// NewAnalyticsWorker creates a new analytics worker
func NewAnalyticsWorker(cfg *config.Config, logger *logrus.Logger) *AnalyticsWorker {
	return &AnalyticsWorker{
		config: cfg,
		logger: logger,
	}
}

// ProcessMessage processes an analytics message
func (w *AnalyticsWorker) ProcessMessage(message queue.Message) error {
	startTime := time.Now()
	w.logger.Debug("Processing analytics message", logrus.Fields{"message_id": message.ID})

	// Parse analytics event
	var event AnalyticsEvent
	if err := utils.MapToStruct(message.Payload, &event); err != nil {
		return fmt.Errorf("failed to parse analytics event: %w", err)
	}

	// Validate event
	if err := w.validateEvent(event); err != nil {
		return fmt.Errorf("invalid analytics event: %w", err)
	}

	// Process event
	if err := w.processEvent(event); err != nil {
		return fmt.Errorf("failed to process analytics event: %w", err)
	}

	processingTime := time.Since(startTime)
	w.logger.Info("Analytics event processed successfully",
		logrus.Fields{
			"message_id":      message.ID,
			"event_id":        event.EventID,
			"event_type":      event.EventType,
			"event_name":      event.EventName,
			"user_id":         event.UserID,
			"processing_time": processingTime,
		})

	return nil
}

// validateEvent validates the analytics event
func (w *AnalyticsWorker) validateEvent(event AnalyticsEvent) error {
	if event.EventID == "" {
		return fmt.Errorf("event_id is required")
	}

	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}

	if event.EventName == "" {
		return fmt.Errorf("event_name is required")
	}

	if event.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Validate timestamp is not too old or in the future
	now := time.Now()
	if event.Timestamp.Before(now.Add(-24*time.Hour)) {
		return fmt.Errorf("event timestamp is too old")
	}

	if event.Timestamp.After(now.Add(1*time.Hour)) {
		return fmt.Errorf("event timestamp is in the future")
	}

	return nil
}

// processEvent processes the analytics event
func (w *AnalyticsWorker) processEvent(event AnalyticsEvent) error {
	w.logger.Debug("Processing analytics event",
		logrus.Fields{
			"event_id":   event.EventID,
			"event_type": event.EventType,
			"event_name": event.EventName,
			"user_id":    event.UserID,
		})

	// Store event data
	if err := w.storeEvent(event); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Update real-time metrics
	if err := w.updateRealTimeMetrics(event); err != nil {
		w.logger.Warn("Failed to update real-time metrics",
			logrus.Fields{
				"event_id": event.EventID,
				"error":    err.Error(),
			})
		// Don't fail the entire processing for real-time metrics
	}

	return nil
}

// storeEvent stores the analytics event
func (w *AnalyticsWorker) storeEvent(event AnalyticsEvent) error {
	w.logger.Debug("Storing analytics event",
		logrus.Fields{
			"event_id":   event.EventID,
			"event_type": event.EventType,
			"event_name": event.EventName,
		})

	// In a real implementation, you would:
	// 1. Store in time-series database (InfluxDB, TimescaleDB)
	// 2. Store in data warehouse (BigQuery, Redshift, Snowflake)
	// 3. Send to analytics platforms (Google Analytics, Mixpanel, etc.)

	// Simulate data storage
	time.Sleep(50 * time.Millisecond)

	w.logger.Debug("Analytics event stored successfully",
		logrus.Fields{
			"event_id": event.EventID,
		})

	return nil
}

// updateRealTimeMetrics updates real-time metrics
func (w *AnalyticsWorker) updateRealTimeMetrics(event AnalyticsEvent) error {
	w.logger.Debug("Updating real-time metrics",
		logrus.Fields{
			"event_id":   event.EventID,
			"event_type": event.EventType,
		})

	// In a real implementation, you would:
	// 1. Update Redis counters
	// 2. Send to real-time analytics dashboard
	// 3. Update WebSocket connections for live data
	// 4. Trigger alerts if thresholds are met

	// Simulate real-time metrics update
	time.Sleep(10 * time.Millisecond)

	return nil
}

