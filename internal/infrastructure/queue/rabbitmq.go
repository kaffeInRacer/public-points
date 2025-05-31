package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
	"github.com/sirupsen/logrus"

	"online-shop/pkg/config"
)

// RabbitMQ represents a RabbitMQ connection
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  *config.Config
	logger  *logrus.Logger
}

// Message represents a queue message
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Attempts  int                    `json:"attempts"`
	MaxRetries int                   `json:"max_retries"`
}

// EmailMessage represents an email message
type EmailMessage struct {
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	Template string            `json:"template"`
	Data     map[string]interface{} `json:"data"`
	Priority int               `json:"priority"`
}

// InvoiceMessage represents an invoice message
type InvoiceMessage struct {
	OrderID     string  `json:"order_id"`
	UserEmail   string  `json:"user_email"`
	OrderNumber string  `json:"order_number"`
	TotalAmount float64 `json:"total_amount"`
	Items       []InvoiceItem `json:"items"`
}

// InvoiceItem represents an invoice item
type InvoiceItem struct {
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

// Queue names
const (
	EmailQueue   = "email_queue"
	InvoiceQueue = "invoice_queue"
	NotificationQueue = "notification_queue"
	AnalyticsQueue = "analytics_queue"
)

// NewRabbitMQ creates a new RabbitMQ connection
func NewRabbitMQ(cfg *config.Config, logger *logrus.Logger) (*RabbitMQ, error) {
	// Build connection string
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		cfg.RabbitMQ.Username,
		cfg.RabbitMQ.Password,
		cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port,
	)

	// Connect to RabbitMQ
	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	rabbitmq := &RabbitMQ{
		conn:    conn,
		channel: channel,
		config:  cfg,
		logger:  logger,
	}

	// Setup queues
	if err := rabbitmq.setupQueues(); err != nil {
		rabbitmq.Close()
		return nil, fmt.Errorf("failed to setup queues: %w", err)
	}

	logger.Info("Connected to RabbitMQ successfully")
	return rabbitmq, nil
}

// setupQueues declares all required queues
func (r *RabbitMQ) setupQueues() error {
	queues := []string{
		EmailQueue,
		InvoiceQueue,
		NotificationQueue,
		AnalyticsQueue,
	}

	for _, queueName := range queues {
		_, err := r.channel.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			amqp.Table{
				"x-message-ttl": 3600000, // 1 hour TTL
				"x-max-retries": 3,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}

		// Declare dead letter queue
		dlqName := queueName + "_dlq"
		_, err = r.channel.QueueDeclare(
			dlqName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare dead letter queue %s: %w", dlqName, err)
		}
	}

	return nil
}

// PublishEmail publishes an email message to the queue
func (r *RabbitMQ) PublishEmail(ctx context.Context, email EmailMessage) error {
	message := Message{
		ID:        generateMessageID(),
		Type:      "email",
		Payload:   structToMap(email),
		Timestamp: time.Now(),
		Attempts:  0,
		MaxRetries: 3,
	}

	return r.publishMessage(ctx, EmailQueue, message)
}

// PublishInvoice publishes an invoice message to the queue
func (r *RabbitMQ) PublishInvoice(ctx context.Context, invoice InvoiceMessage) error {
	message := Message{
		ID:        generateMessageID(),
		Type:      "invoice",
		Payload:   structToMap(invoice),
		Timestamp: time.Now(),
		Attempts:  0,
		MaxRetries: 3,
	}

	return r.publishMessage(ctx, InvoiceQueue, message)
}

// PublishNotification publishes a notification message to the queue
func (r *RabbitMQ) PublishNotification(ctx context.Context, notification map[string]interface{}) error {
	message := Message{
		ID:        generateMessageID(),
		Type:      "notification",
		Payload:   notification,
		Timestamp: time.Now(),
		Attempts:  0,
		MaxRetries: 3,
	}

	return r.publishMessage(ctx, NotificationQueue, message)
}

// PublishAnalytics publishes an analytics event to the queue
func (r *RabbitMQ) PublishAnalytics(ctx context.Context, event map[string]interface{}) error {
	message := Message{
		ID:        generateMessageID(),
		Type:      "analytics",
		Payload:   event,
		Timestamp: time.Now(),
		Attempts:  0,
		MaxRetries: 1, // Analytics events don't need retries
	}

	return r.publishMessage(ctx, AnalyticsQueue, message)
}

// publishMessage publishes a message to the specified queue
func (r *RabbitMQ) publishMessage(ctx context.Context, queueName string, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Make message persistent
			Timestamp:    time.Now(),
			MessageId:    message.ID,
		},
	)

	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"queue":      queueName,
			"message_id": message.ID,
		}).WithError(err).Error("Failed to publish message")
		return fmt.Errorf("failed to publish message: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"queue":      queueName,
		"message_id": message.ID,
		"type":       message.Type,
	}).Debug("Message published successfully")

	return nil
}

// ConsumeMessages consumes messages from a queue
func (r *RabbitMQ) ConsumeMessages(ctx context.Context, queueName string, handler func(Message) error) error {
	msgs, err := r.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	r.logger.WithField("queue", queueName).Info("Started consuming messages")

	for {
		select {
		case <-ctx.Done():
			r.logger.WithField("queue", queueName).Info("Stopping message consumption")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				r.logger.WithField("queue", queueName).Warn("Message channel closed")
				return fmt.Errorf("message channel closed")
			}

			r.processMessage(msg, handler)
		}
	}
}

// processMessage processes a single message
func (r *RabbitMQ) processMessage(delivery amqp.Delivery, handler func(Message) error) {
	var message Message
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		r.logger.WithError(err).Error("Failed to unmarshal message")
		delivery.Nack(false, false) // Don't requeue malformed messages
		return
	}

	r.logger.WithFields(logrus.Fields{
		"message_id": message.ID,
		"type":       message.Type,
		"attempts":   message.Attempts,
	}).Debug("Processing message")

	// Increment attempt counter
	message.Attempts++

	// Process the message
	if err := handler(message); err != nil {
		r.logger.WithFields(logrus.Fields{
			"message_id": message.ID,
		}).WithError(err).Error("Failed to process message")

		// Check if we should retry
		if message.Attempts < message.MaxRetries {
			r.logger.WithFields(logrus.Fields{
				"message_id":  message.ID,
				"attempt":     message.Attempts,
				"max_retries": message.MaxRetries,
			}).Info("Requeuing message for retry")
			delivery.Nack(false, true) // Requeue for retry
		} else {
			r.logger.WithField("message_id", message.ID).Error("Message exceeded max retries, sending to DLQ")
			delivery.Nack(false, false) // Don't requeue, send to DLQ
		}
		return
	}

	// Acknowledge successful processing
	delivery.Ack(false)
	r.logger.WithField("message_id", message.ID).Debug("Message processed successfully")
}

// Close closes the RabbitMQ connection
func (r *RabbitMQ) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Health check
func (r *RabbitMQ) HealthCheck() error {
	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	}
	return nil
}

// Helper functions
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func structToMap(obj interface{}) map[string]interface{} {
	data, _ := json.Marshal(obj)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}