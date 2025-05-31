package mocks

import (
	"context"
	"time"

	"github.com/streadway/amqp"
	"gorm.io/gorm"
)

//go:generate mockgen -source=interfaces.go -destination=generated_mocks.go -package=mocks

// DatabaseInterface defines the database operations
type DatabaseInterface interface {
	Create(value interface{}) *gorm.DB
	First(dest interface{}, conds ...interface{}) *gorm.DB
	Find(dest interface{}, conds ...interface{}) *gorm.DB
	Where(query interface{}, args ...interface{}) *gorm.DB
	Update(column string, value interface{}) *gorm.DB
	Updates(values interface{}) *gorm.DB
	Delete(value interface{}, conds ...interface{}) *gorm.DB
	Save(value interface{}) *gorm.DB
	Begin() *gorm.DB
	Commit() *gorm.DB
	Rollback() *gorm.DB
	Raw(sql string, values ...interface{}) *gorm.DB
	Exec(sql string, values ...interface{}) *gorm.DB
	Model(value interface{}) *gorm.DB
	Table(name string) *gorm.DB
	Count(count *int64) *gorm.DB
	Limit(limit int) *gorm.DB
	Offset(offset int) *gorm.DB
	Order(value interface{}) *gorm.DB
	Group(name string) *gorm.DB
	Having(query interface{}, args ...interface{}) *gorm.DB
	Joins(query string, args ...interface{}) *gorm.DB
	Preload(query string, args ...interface{}) *gorm.DB
	Association(column string) *gorm.Association
	Transaction(fc func(tx *gorm.DB) error) error
}

// RedisInterface defines the Redis operations
type RedisInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPop(ctx context.Context, key string) (string, error)
	LLen(ctx context.Context, key string) (int64, error)
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	ZAdd(ctx context.Context, key string, members ...interface{}) error
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	Ping(ctx context.Context) error
	FlushDB(ctx context.Context) error
	Close() error
}

// RabbitMQInterface defines the RabbitMQ operations
type RabbitMQInterface interface {
	Connect() error
	Close() error
	DeclareQueue(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	DeclareExchange(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	BindQueue(queueName, routingKey, exchangeName string, noWait bool, args amqp.Table) error
	Publish(exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(queueName, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	QueuePurge(queueName string, noWait bool) (int, error)
	QueueDelete(queueName string, ifUnused, ifEmpty, noWait bool) (int, error)
	ExchangeDelete(exchangeName string, ifUnused, noWait bool) error
	GetChannel() *amqp.Channel
	GetConnection() *amqp.Connection
	IsConnected() bool
}

// EmailServiceInterface defines the email service operations
type EmailServiceInterface interface {
	SendEmail(to, subject, body string) error
	SendHTMLEmail(to, subject, htmlBody, textBody string) error
	SendEmailWithAttachment(to, subject, body string, attachments []string) error
	SendTemplateEmail(to, subject, templateName string, data interface{}) error
	ValidateEmail(email string) bool
	GetSMTPConfig() SMTPConfig
	SetSMTPConfig(config SMTPConfig)
	TestConnection() error
}

// SMTPConfig represents SMTP configuration
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromEmail   string
	FromName    string
	UseTLS      bool
	UseSSL      bool
	Timeout     int
	MaxRetries  int
	RetryDelay  int
}

// WorkerPoolInterface defines the worker pool operations
type WorkerPoolInterface interface {
	Start() error
	Stop() error
	SubmitJob(job Job) error
	GetMetrics() PoolMetrics
	GetWorkerCount() int
	SetWorkerCount(count int) error
	IsRunning() bool
	GetQueueSize() int
	GetActiveJobs() int
}

// Job represents a job to be processed
type Job interface {
	Execute() error
	GetID() string
	GetType() string
	GetPriority() int
	GetRetryCount() int
	GetMaxRetries() int
	ShouldRetry() bool
	OnSuccess()
	OnFailure(error)
}

// PoolMetrics represents worker pool metrics
type PoolMetrics struct {
	TotalJobs     int64
	CompletedJobs int64
	FailedJobs    int64
	ActiveWorkers int
	QueueSize     int
	AverageTime   time.Duration
}

// LoggerInterface defines the logger operations
type LoggerInterface interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
	WithField(key string, value interface{}) LoggerInterface
	WithFields(fields map[string]interface{}) LoggerInterface
	WithError(err error) LoggerInterface
	SetLevel(level string)
	GetLevel() string
}

// ConfigInterface defines the configuration operations
type ConfigInterface interface {
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat64(key string) float64
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
	GetStringMap(key string) map[string]interface{}
	Set(key string, value interface{})
	IsSet(key string) bool
	AllKeys() []string
	Unmarshal(rawVal interface{}) error
	UnmarshalKey(key string, rawVal interface{}) error
	WatchConfig()
	OnConfigChange(run func())
	ReadInConfig() error
	WriteConfig() error
	SafeWriteConfig() error
}

// CacheInterface defines the cache operations
type CacheInterface interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}, expiration time.Duration) error
	Delete(key string) error
	Exists(key string) bool
	Clear() error
	GetTTL(key string) (time.Duration, error)
	SetTTL(key string, expiration time.Duration) error
	GetKeys(pattern string) ([]string, error)
	GetSize() int64
	GetStats() CacheStats
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64
	Misses      int64
	Keys        int64
	Evictions   int64
	Connections int64
}

// PaymentServiceInterface defines the payment service operations
type PaymentServiceInterface interface {
	CreatePayment(amount float64, currency, description string, metadata map[string]interface{}) (*PaymentResponse, error)
	GetPayment(paymentID string) (*PaymentResponse, error)
	CapturePayment(paymentID string, amount float64) (*PaymentResponse, error)
	RefundPayment(paymentID string, amount float64, reason string) (*PaymentResponse, error)
	CancelPayment(paymentID string, reason string) (*PaymentResponse, error)
	ListPayments(filters PaymentFilters) ([]*PaymentResponse, error)
	ValidateWebhook(payload []byte, signature string) (*WebhookEvent, error)
	GetSupportedMethods() []string
	GetSupportedCurrencies() []string
}

// PaymentResponse represents a payment response
type PaymentResponse struct {
	ID          string                 `json:"id"`
	Amount      float64                `json:"amount"`
	Currency    string                 `json:"currency"`
	Status      string                 `json:"status"`
	Description string                 `json:"description"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
	PaymentURL  string                 `json:"payment_url,omitempty"`
	RedirectURL string                 `json:"redirect_url,omitempty"`
}

// PaymentFilters represents payment filters
type PaymentFilters struct {
	Status    string
	Currency  string
	StartDate time.Time
	EndDate   time.Time
	Limit     int
	Offset    int
}

// WebhookEvent represents a webhook event
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
}

// SearchServiceInterface defines the search service operations
type SearchServiceInterface interface {
	Index(indexName string, docID string, document interface{}) error
	Search(indexName string, query SearchQuery) (*SearchResponse, error)
	Get(indexName string, docID string) (interface{}, error)
	Update(indexName string, docID string, document interface{}) error
	Delete(indexName string, docID string) error
	BulkIndex(indexName string, documents []BulkDocument) error
	CreateIndex(indexName string, mapping interface{}) error
	DeleteIndex(indexName string) error
	IndexExists(indexName string) (bool, error)
	GetMapping(indexName string) (interface{}, error)
	UpdateMapping(indexName string, mapping interface{}) error
	Refresh(indexName string) error
	GetStats(indexName string) (*IndexStats, error)
}

// SearchQuery represents a search query
type SearchQuery struct {
	Query   interface{}            `json:"query"`
	Sort    []interface{}          `json:"sort,omitempty"`
	From    int                    `json:"from,omitempty"`
	Size    int                    `json:"size,omitempty"`
	Source  interface{}            `json:"_source,omitempty"`
	Filters map[string]interface{} `json:"filters,omitempty"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Hits      SearchHits `json:"hits"`
	Took      int        `json:"took"`
	TimedOut  bool       `json:"timed_out"`
	Shards    ShardInfo  `json:"_shards"`
	ScrollID  string     `json:"_scroll_id,omitempty"`
}

// SearchHits represents search hits
type SearchHits struct {
	Total    HitsTotal     `json:"total"`
	MaxScore float64       `json:"max_score"`
	Hits     []SearchHit   `json:"hits"`
}

// HitsTotal represents total hits
type HitsTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

// SearchHit represents a search hit
type SearchHit struct {
	Index  string                 `json:"_index"`
	Type   string                 `json:"_type"`
	ID     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

// ShardInfo represents shard information
type ShardInfo struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

// BulkDocument represents a bulk document
type BulkDocument struct {
	ID       string      `json:"id"`
	Document interface{} `json:"document"`
}

// IndexStats represents index statistics
type IndexStats struct {
	DocsCount    int64 `json:"docs_count"`
	DocsDeleted  int64 `json:"docs_deleted"`
	StoreSize    int64 `json:"store_size"`
	IndexingTime int64 `json:"indexing_time"`
	SearchTime   int64 `json:"search_time"`
}

// NotificationServiceInterface defines the notification service operations
type NotificationServiceInterface interface {
	SendNotification(notification Notification) error
	SendBulkNotifications(notifications []Notification) error
	GetNotificationStatus(notificationID string) (*NotificationStatus, error)
	GetNotificationHistory(userID string, filters NotificationFilters) ([]*NotificationStatus, error)
	RegisterDevice(userID, deviceToken, platform string) error
	UnregisterDevice(userID, deviceToken string) error
	CreateTemplate(template NotificationTemplate) error
	UpdateTemplate(templateID string, template NotificationTemplate) error
	DeleteTemplate(templateID string) error
	GetTemplate(templateID string) (*NotificationTemplate, error)
	ListTemplates() ([]*NotificationTemplate, error)
}

// Notification represents a notification
type Notification struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Type        string                 `json:"type"`
	Channel     string                 `json:"channel"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	TemplateID  string                 `json:"template_id,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	Priority    int                    `json:"priority"`
	TTL         time.Duration          `json:"ttl,omitempty"`
}

// NotificationStatus represents notification status
type NotificationStatus struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	ReadAt      *time.Time             `json:"read_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationFilters represents notification filters
type NotificationFilters struct {
	Type      string
	Channel   string
	Status    string
	StartDate time.Time
	EndDate   time.Time
	Limit     int
	Offset    int
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Channel     string                 `json:"channel"`
	Subject     string                 `json:"subject,omitempty"`
	Body        string                 `json:"body"`
	Variables   []string               `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AnalyticsServiceInterface defines the analytics service operations
type AnalyticsServiceInterface interface {
	TrackEvent(event AnalyticsEvent) error
	TrackEvents(events []AnalyticsEvent) error
	GetMetrics(query MetricsQuery) (*MetricsResponse, error)
	GetReport(reportType string, filters ReportFilters) (*Report, error)
	CreateDashboard(dashboard Dashboard) error
	UpdateDashboard(dashboardID string, dashboard Dashboard) error
	GetDashboard(dashboardID string) (*Dashboard, error)
	ListDashboards() ([]*Dashboard, error)
	DeleteDashboard(dashboardID string) error
	GetRealTimeMetrics(metricNames []string) (map[string]interface{}, error)
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	ID         string                 `json:"id"`
	EventType  string                 `json:"event_type"`
	EventName  string                 `json:"event_name"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source,omitempty"`
	Version    string                 `json:"version,omitempty"`
}

// MetricsQuery represents a metrics query
type MetricsQuery struct {
	Metrics   []string               `json:"metrics"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	GroupBy   []string               `json:"group_by,omitempty"`
	StartDate time.Time              `json:"start_date"`
	EndDate   time.Time              `json:"end_date"`
	Interval  string                 `json:"interval,omitempty"`
}

// MetricsResponse represents a metrics response
type MetricsResponse struct {
	Metrics   map[string]interface{} `json:"metrics"`
	Data      []MetricDataPoint      `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricDataPoint represents a metric data point
type MetricDataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"`
	Labels    map[string]string      `json:"labels,omitempty"`
}

// ReportFilters represents report filters
type ReportFilters struct {
	StartDate time.Time              `json:"start_date"`
	EndDate   time.Time              `json:"end_date"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	GroupBy   []string               `json:"group_by,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Offset    int                    `json:"offset,omitempty"`
}

// Report represents a report
type Report struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Summary   map[string]interface{} `json:"summary,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Dashboard represents a dashboard
type Dashboard struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Widgets     []DashboardWidget      `json:"widgets"`
	Layout      map[string]interface{} `json:"layout,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// DashboardWidget represents a dashboard widget
type DashboardWidget struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Config   map[string]interface{} `json:"config"`
	Position map[string]int         `json:"position"`
}