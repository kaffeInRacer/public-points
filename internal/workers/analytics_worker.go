package workers

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"online-shop/internal/infrastructure/queue"
	"online-shop/pkg/config"
)

// AnalyticsWorker handles analytics event processing
type AnalyticsWorker struct {
	config *config.Config
	logger *zap.Logger
}

// NewAnalyticsWorker creates a new analytics worker
func NewAnalyticsWorker(cfg *config.Config, logger *zap.Logger) *AnalyticsWorker {
	return &AnalyticsWorker{
		config: cfg,
		logger: logger,
	}
}

// ProcessMessage processes an analytics message
func (w *AnalyticsWorker) ProcessMessage(message queue.Message) error {
	w.logger.Debug("Processing analytics message", zap.String("message_id", message.ID))

	// Extract event type
	eventType, ok := message.Payload["event_type"].(string)
	if !ok {
		return fmt.Errorf("missing event type")
	}

	// Process based on event type
	switch eventType {
	case "page_view":
		return w.processPageView(message.Payload)
	case "product_view":
		return w.processProductView(message.Payload)
	case "add_to_cart":
		return w.processAddToCart(message.Payload)
	case "purchase":
		return w.processPurchase(message.Payload)
	case "user_registration":
		return w.processUserRegistration(message.Payload)
	case "search":
		return w.processSearch(message.Payload)
	default:
		return w.processGenericEvent(eventType, message.Payload)
	}
}

// processPageView processes page view events
func (w *AnalyticsWorker) processPageView(payload map[string]interface{}) error {
	event := PageViewEvent{
		UserID:    getStringValue(payload, "user_id"),
		SessionID: getStringValue(payload, "session_id"),
		Page:      getStringValue(payload, "page"),
		Referrer:  getStringValue(payload, "referrer"),
		UserAgent: getStringValue(payload, "user_agent"),
		IPAddress: getStringValue(payload, "ip_address"),
		Timestamp: time.Now(),
	}

	w.logger.Debug("Processing page view event",
		zap.String("user_id", event.UserID),
		zap.String("page", event.Page),
	)

	// Store or send to analytics service
	return w.storeEvent("page_view", event)
}

// processProductView processes product view events
func (w *AnalyticsWorker) processProductView(payload map[string]interface{}) error {
	event := ProductViewEvent{
		UserID:     getStringValue(payload, "user_id"),
		SessionID:  getStringValue(payload, "session_id"),
		ProductID:  getStringValue(payload, "product_id"),
		ProductSKU: getStringValue(payload, "product_sku"),
		Category:   getStringValue(payload, "category"),
		Price:      getFloatValue(payload, "price"),
		Timestamp:  time.Now(),
	}

	w.logger.Debug("Processing product view event",
		zap.String("user_id", event.UserID),
		zap.String("product_id", event.ProductID),
	)

	return w.storeEvent("product_view", event)
}

// processAddToCart processes add to cart events
func (w *AnalyticsWorker) processAddToCart(payload map[string]interface{}) error {
	event := AddToCartEvent{
		UserID:     getStringValue(payload, "user_id"),
		SessionID:  getStringValue(payload, "session_id"),
		ProductID:  getStringValue(payload, "product_id"),
		ProductSKU: getStringValue(payload, "product_sku"),
		Quantity:   getIntValue(payload, "quantity"),
		Price:      getFloatValue(payload, "price"),
		Timestamp:  time.Now(),
	}

	w.logger.Debug("Processing add to cart event",
		zap.String("user_id", event.UserID),
		zap.String("product_id", event.ProductID),
		zap.Int("quantity", event.Quantity),
	)

	return w.storeEvent("add_to_cart", event)
}

// processPurchase processes purchase events
func (w *AnalyticsWorker) processPurchase(payload map[string]interface{}) error {
	event := PurchaseEvent{
		UserID:      getStringValue(payload, "user_id"),
		OrderID:     getStringValue(payload, "order_id"),
		OrderNumber: getStringValue(payload, "order_number"),
		TotalAmount: getFloatValue(payload, "total_amount"),
		Currency:    getStringValue(payload, "currency"),
		ItemCount:   getIntValue(payload, "item_count"),
		Timestamp:   time.Now(),
	}

	// Extract items if available
	if items, ok := payload["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				event.Items = append(event.Items, PurchaseItem{
					ProductID:  getStringValue(itemMap, "product_id"),
					ProductSKU: getStringValue(itemMap, "product_sku"),
					Quantity:   getIntValue(itemMap, "quantity"),
					Price:      getFloatValue(itemMap, "price"),
				})
			}
		}
	}

	w.logger.Debug("Processing purchase event",
		zap.String("user_id", event.UserID),
		zap.String("order_id", event.OrderID),
		zap.Float64("total_amount", event.TotalAmount),
	)

	return w.storeEvent("purchase", event)
}

// processUserRegistration processes user registration events
func (w *AnalyticsWorker) processUserRegistration(payload map[string]interface{}) error {
	event := UserRegistrationEvent{
		UserID:    getStringValue(payload, "user_id"),
		Email:     getStringValue(payload, "email"),
		Source:    getStringValue(payload, "source"),
		Referrer:  getStringValue(payload, "referrer"),
		Timestamp: time.Now(),
	}

	w.logger.Debug("Processing user registration event",
		zap.String("user_id", event.UserID),
		zap.String("email", event.Email),
	)

	return w.storeEvent("user_registration", event)
}

// processSearch processes search events
func (w *AnalyticsWorker) processSearch(payload map[string]interface{}) error {
	event := SearchEvent{
		UserID:      getStringValue(payload, "user_id"),
		SessionID:   getStringValue(payload, "session_id"),
		Query:       getStringValue(payload, "query"),
		ResultCount: getIntValue(payload, "result_count"),
		Timestamp:   time.Now(),
	}

	w.logger.Debug("Processing search event",
		zap.String("user_id", event.UserID),
		zap.String("query", event.Query),
		zap.Int("result_count", event.ResultCount),
	)

	return w.storeEvent("search", event)
}

// processGenericEvent processes generic events
func (w *AnalyticsWorker) processGenericEvent(eventType string, payload map[string]interface{}) error {
	w.logger.Debug("Processing generic analytics event",
		zap.String("event_type", eventType),
		zap.Any("payload", payload),
	)

	return w.storeEvent(eventType, payload)
}

// storeEvent stores an event to the analytics storage
func (w *AnalyticsWorker) storeEvent(eventType string, event interface{}) error {
	// This is a placeholder for storing analytics events
	// In a real implementation, you would:
	// 1. Store in a time-series database (InfluxDB, TimescaleDB)
	// 2. Send to analytics services (Google Analytics, Mixpanel, Amplitude)
	// 3. Store in data warehouse (BigQuery, Redshift, Snowflake)
	// 4. Send to real-time analytics (Apache Kafka, Amazon Kinesis)

	w.logger.Debug("Storing analytics event",
		zap.String("event_type", eventType),
		zap.Any("event", event),
	)

	// Simulate storing event
	// In real implementation:
	// return w.analyticsRepository.Store(eventType, event)
	// or
	// return w.analyticsService.Send(eventType, event)

	return nil
}

// Helper functions to extract values from payload
func getStringValue(payload map[string]interface{}, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}

func getIntValue(payload map[string]interface{}, key string) int {
	if value, ok := payload[key].(float64); ok {
		return int(value)
	}
	if value, ok := payload[key].(int); ok {
		return value
	}
	return 0
}

func getFloatValue(payload map[string]interface{}, key string) float64 {
	if value, ok := payload[key].(float64); ok {
		return value
	}
	if value, ok := payload[key].(int); ok {
		return float64(value)
	}
	return 0.0
}

// Event structures
type PageViewEvent struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Page      string    `json:"page"`
	Referrer  string    `json:"referrer"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

type ProductViewEvent struct {
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	ProductID  string    `json:"product_id"`
	ProductSKU string    `json:"product_sku"`
	Category   string    `json:"category"`
	Price      float64   `json:"price"`
	Timestamp  time.Time `json:"timestamp"`
}

type AddToCartEvent struct {
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	ProductID  string    `json:"product_id"`
	ProductSKU string    `json:"product_sku"`
	Quantity   int       `json:"quantity"`
	Price      float64   `json:"price"`
	Timestamp  time.Time `json:"timestamp"`
}

type PurchaseEvent struct {
	UserID      string         `json:"user_id"`
	OrderID     string         `json:"order_id"`
	OrderNumber string         `json:"order_number"`
	TotalAmount float64        `json:"total_amount"`
	Currency    string         `json:"currency"`
	ItemCount   int            `json:"item_count"`
	Items       []PurchaseItem `json:"items"`
	Timestamp   time.Time      `json:"timestamp"`
}

type PurchaseItem struct {
	ProductID  string  `json:"product_id"`
	ProductSKU string  `json:"product_sku"`
	Quantity   int     `json:"quantity"`
	Price      float64 `json:"price"`
}

type UserRegistrationEvent struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Source    string    `json:"source"`
	Referrer  string    `json:"referrer"`
	Timestamp time.Time `json:"timestamp"`
}

type SearchEvent struct {
	UserID      string    `json:"user_id"`
	SessionID   string    `json:"session_id"`
	Query       string    `json:"query"`
	ResultCount int       `json:"result_count"`
	Timestamp   time.Time `json:"timestamp"`
}