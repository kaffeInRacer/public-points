package mocks

import (
	"errors"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// MockRabbitMQ implements RabbitMQInterface for testing
type MockRabbitMQ struct {
	mu          sync.RWMutex
	connected   bool
	queues      map[string]*MockQueue
	exchanges   map[string]*MockExchange
	bindings    map[string][]MockBinding
	messages    map[string][]amqp.Publishing
	consumers   map[string]*MockConsumer
	shouldFail  bool
	lastError   error
	callLog     []string
	channel     *MockChannel
	connection  *MockConnection
}

// MockQueue represents a mock queue
type MockQueue struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
	Messages   []amqp.Publishing
}

// MockExchange represents a mock exchange
type MockExchange struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

// MockBinding represents a queue binding
type MockBinding struct {
	QueueName    string
	RoutingKey   string
	ExchangeName string
	NoWait       bool
	Args         amqp.Table
}

// MockConsumer represents a mock consumer
type MockConsumer struct {
	QueueName string
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
	Channel   chan amqp.Delivery
	Active    bool
}

// MockChannel represents a mock AMQP channel
type MockChannel struct {
	closed bool
}

// MockConnection represents a mock AMQP connection
type MockConnection struct {
	closed bool
}

// NewMockRabbitMQ creates a new mock RabbitMQ client
func NewMockRabbitMQ() *MockRabbitMQ {
	return &MockRabbitMQ{
		connected:  false,
		queues:     make(map[string]*MockQueue),
		exchanges:  make(map[string]*MockExchange),
		bindings:   make(map[string][]MockBinding),
		messages:   make(map[string][]amqp.Publishing),
		consumers:  make(map[string]*MockConsumer),
		callLog:    make([]string, 0),
		channel:    &MockChannel{},
		connection: &MockConnection{},
	}
}

// SetShouldFail sets whether operations should fail
func (m *MockRabbitMQ) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// SetLastError sets the last error
func (m *MockRabbitMQ) SetLastError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = err
}

// GetCallLog returns the call log
func (m *MockRabbitMQ) GetCallLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.callLog...)
}

// ClearCallLog clears the call log
func (m *MockRabbitMQ) ClearCallLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = make([]string, 0)
}

func (m *MockRabbitMQ) logCall(method string) {
	m.callLog = append(m.callLog, method)
}

func (m *MockRabbitMQ) checkError() error {
	if m.shouldFail {
		if m.lastError != nil {
			return m.lastError
		}
		return errors.New("rabbitmq: operation failed")
	}
	return nil
}

// Connect implements RabbitMQInterface
func (m *MockRabbitMQ) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Connect")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	m.connected = true
	m.channel.closed = false
	m.connection.closed = false
	
	return nil
}

// Close implements RabbitMQInterface
func (m *MockRabbitMQ) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Close")
	
	m.connected = false
	m.channel.closed = true
	m.connection.closed = true
	
	// Close all consumer channels
	for _, consumer := range m.consumers {
		if consumer.Channel != nil {
			close(consumer.Channel)
		}
		consumer.Active = false
	}
	
	return nil
}

// DeclareQueue implements RabbitMQInterface
func (m *MockRabbitMQ) DeclareQueue(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("DeclareQueue")
	
	if err := m.checkError(); err != nil {
		return amqp.Queue{}, err
	}
	
	if !m.connected {
		return amqp.Queue{}, errors.New("rabbitmq: not connected")
	}
	
	queue := &MockQueue{
		Name:       name,
		Durable:    durable,
		AutoDelete: autoDelete,
		Exclusive:  exclusive,
		NoWait:     noWait,
		Args:       args,
		Messages:   make([]amqp.Publishing, 0),
	}
	
	m.queues[name] = queue
	m.messages[name] = make([]amqp.Publishing, 0)
	
	return amqp.Queue{
		Name:      name,
		Messages:  0,
		Consumers: 0,
	}, nil
}

// DeclareExchange implements RabbitMQInterface
func (m *MockRabbitMQ) DeclareExchange(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("DeclareExchange")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if !m.connected {
		return errors.New("rabbitmq: not connected")
	}
	
	exchange := &MockExchange{
		Name:       name,
		Kind:       kind,
		Durable:    durable,
		AutoDelete: autoDelete,
		Internal:   internal,
		NoWait:     noWait,
		Args:       args,
	}
	
	m.exchanges[name] = exchange
	
	return nil
}

// BindQueue implements RabbitMQInterface
func (m *MockRabbitMQ) BindQueue(queueName, routingKey, exchangeName string, noWait bool, args amqp.Table) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("BindQueue")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if !m.connected {
		return errors.New("rabbitmq: not connected")
	}
	
	// Check if queue and exchange exist
	if _, exists := m.queues[queueName]; !exists {
		return errors.New("rabbitmq: queue does not exist")
	}
	
	if _, exists := m.exchanges[exchangeName]; !exists {
		return errors.New("rabbitmq: exchange does not exist")
	}
	
	binding := MockBinding{
		QueueName:    queueName,
		RoutingKey:   routingKey,
		ExchangeName: exchangeName,
		NoWait:       noWait,
		Args:         args,
	}
	
	key := exchangeName + ":" + routingKey
	m.bindings[key] = append(m.bindings[key], binding)
	
	return nil
}

// Publish implements RabbitMQInterface
func (m *MockRabbitMQ) Publish(exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Publish")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if !m.connected {
		return errors.New("rabbitmq: not connected")
	}
	
	// Set timestamp if not provided
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	
	// Find bound queues for this exchange and routing key
	key := exchange + ":" + routingKey
	if bindings, exists := m.bindings[key]; exists {
		for _, binding := range bindings {
			if messages, exists := m.messages[binding.QueueName]; exists {
				m.messages[binding.QueueName] = append(messages, msg)
			} else {
				m.messages[binding.QueueName] = []amqp.Publishing{msg}
			}
			
			// Deliver to active consumers
			if consumer, exists := m.consumers[binding.QueueName]; exists && consumer.Active {
				delivery := amqp.Delivery{
					ConsumerTag:     consumer.Consumer,
					DeliveryTag:     uint64(len(m.messages[binding.QueueName])),
					Redelivered:     false,
					Exchange:        exchange,
					RoutingKey:      routingKey,
					ContentType:     msg.ContentType,
					ContentEncoding: msg.ContentEncoding,
					Headers:         msg.Headers,
					DeliveryMode:    msg.DeliveryMode,
					Priority:        msg.Priority,
					CorrelationId:   msg.CorrelationId,
					ReplyTo:         msg.ReplyTo,
					Expiration:      msg.Expiration,
					MessageId:       msg.MessageId,
					Timestamp:       msg.Timestamp,
					Type:            msg.Type,
					UserId:          msg.UserId,
					AppId:           msg.AppId,
					Body:            msg.Body,
				}
				
				// Non-blocking send to consumer channel
				select {
				case consumer.Channel <- delivery:
				default:
					// Channel is full, message will be queued
				}
			}
		}
	} else {
		// Direct queue publish (when exchange is empty)
		if routingKey != "" {
			if messages, exists := m.messages[routingKey]; exists {
				m.messages[routingKey] = append(messages, msg)
			} else {
				m.messages[routingKey] = []amqp.Publishing{msg}
			}
		}
	}
	
	return nil
}

// Consume implements RabbitMQInterface
func (m *MockRabbitMQ) Consume(queueName, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Consume")
	
	if err := m.checkError(); err != nil {
		return nil, err
	}
	
	if !m.connected {
		return nil, errors.New("rabbitmq: not connected")
	}
	
	// Check if queue exists
	if _, exists := m.queues[queueName]; !exists {
		return nil, errors.New("rabbitmq: queue does not exist")
	}
	
	// Create consumer channel
	deliveryChannel := make(chan amqp.Delivery, 100)
	
	mockConsumer := &MockConsumer{
		QueueName: queueName,
		Consumer:  consumer,
		AutoAck:   autoAck,
		Exclusive: exclusive,
		NoLocal:   noLocal,
		NoWait:    noWait,
		Args:      args,
		Channel:   deliveryChannel,
		Active:    true,
	}
	
	m.consumers[queueName] = mockConsumer
	
	// Deliver existing messages
	if messages, exists := m.messages[queueName]; exists {
		go func() {
			for i, msg := range messages {
				delivery := amqp.Delivery{
					ConsumerTag:     consumer,
					DeliveryTag:     uint64(i + 1),
					Redelivered:     false,
					Exchange:        "",
					RoutingKey:      queueName,
					ContentType:     msg.ContentType,
					ContentEncoding: msg.ContentEncoding,
					Headers:         msg.Headers,
					DeliveryMode:    msg.DeliveryMode,
					Priority:        msg.Priority,
					CorrelationId:   msg.CorrelationId,
					ReplyTo:         msg.ReplyTo,
					Expiration:      msg.Expiration,
					MessageId:       msg.MessageId,
					Timestamp:       msg.Timestamp,
					Type:            msg.Type,
					UserId:          msg.UserId,
					AppId:           msg.AppId,
					Body:            msg.Body,
				}
				
				select {
				case deliveryChannel <- delivery:
				case <-time.After(time.Second):
					// Timeout, stop delivering
					return
				}
			}
		}()
	}
	
	return deliveryChannel, nil
}

// QueuePurge implements RabbitMQInterface
func (m *MockRabbitMQ) QueuePurge(queueName string, noWait bool) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("QueuePurge")
	
	if err := m.checkError(); err != nil {
		return 0, err
	}
	
	if !m.connected {
		return 0, errors.New("rabbitmq: not connected")
	}
	
	messageCount := 0
	if messages, exists := m.messages[queueName]; exists {
		messageCount = len(messages)
		m.messages[queueName] = make([]amqp.Publishing, 0)
	}
	
	return messageCount, nil
}

// QueueDelete implements RabbitMQInterface
func (m *MockRabbitMQ) QueueDelete(queueName string, ifUnused, ifEmpty, noWait bool) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("QueueDelete")
	
	if err := m.checkError(); err != nil {
		return 0, err
	}
	
	if !m.connected {
		return 0, errors.New("rabbitmq: not connected")
	}
	
	messageCount := 0
	if messages, exists := m.messages[queueName]; exists {
		messageCount = len(messages)
	}
	
	// Check conditions
	if ifUnused {
		if consumer, exists := m.consumers[queueName]; exists && consumer.Active {
			return 0, errors.New("rabbitmq: queue in use")
		}
	}
	
	if ifEmpty && messageCount > 0 {
		return 0, errors.New("rabbitmq: queue not empty")
	}
	
	// Delete queue
	delete(m.queues, queueName)
	delete(m.messages, queueName)
	
	// Close consumer if exists
	if consumer, exists := m.consumers[queueName]; exists {
		if consumer.Channel != nil {
			close(consumer.Channel)
		}
		consumer.Active = false
		delete(m.consumers, queueName)
	}
	
	// Remove bindings
	for key, bindings := range m.bindings {
		newBindings := make([]MockBinding, 0)
		for _, binding := range bindings {
			if binding.QueueName != queueName {
				newBindings = append(newBindings, binding)
			}
		}
		if len(newBindings) == 0 {
			delete(m.bindings, key)
		} else {
			m.bindings[key] = newBindings
		}
	}
	
	return messageCount, nil
}

// ExchangeDelete implements RabbitMQInterface
func (m *MockRabbitMQ) ExchangeDelete(exchangeName string, ifUnused, noWait bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("ExchangeDelete")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if !m.connected {
		return errors.New("rabbitmq: not connected")
	}
	
	// Check if exchange is in use
	if ifUnused {
		for key := range m.bindings {
			if exchangeName+":" == key[:len(exchangeName)+1] {
				return errors.New("rabbitmq: exchange in use")
			}
		}
	}
	
	// Delete exchange
	delete(m.exchanges, exchangeName)
	
	// Remove bindings
	for key := range m.bindings {
		if exchangeName+":" == key[:len(exchangeName)+1] {
			delete(m.bindings, key)
		}
	}
	
	return nil
}

// GetChannel implements RabbitMQInterface
func (m *MockRabbitMQ) GetChannel() *amqp.Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return nil for mock - this is just for interface compliance
	return nil
}

// GetConnection implements RabbitMQInterface
func (m *MockRabbitMQ) GetConnection() *amqp.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return nil for mock - this is just for interface compliance
	return nil
}

// IsConnected implements RabbitMQInterface
func (m *MockRabbitMQ) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// Helper methods for testing

// GetQueueMessageCount returns the number of messages in a queue
func (m *MockRabbitMQ) GetQueueMessageCount(queueName string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if messages, exists := m.messages[queueName]; exists {
		return len(messages)
	}
	return 0
}

// GetQueueMessages returns all messages in a queue
func (m *MockRabbitMQ) GetQueueMessages(queueName string) []amqp.Publishing {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if messages, exists := m.messages[queueName]; exists {
		result := make([]amqp.Publishing, len(messages))
		copy(result, messages)
		return result
	}
	return make([]amqp.Publishing, 0)
}

// GetQueues returns all declared queues
func (m *MockRabbitMQ) GetQueues() map[string]*MockQueue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]*MockQueue)
	for name, queue := range m.queues {
		result[name] = queue
	}
	return result
}

// GetExchanges returns all declared exchanges
func (m *MockRabbitMQ) GetExchanges() map[string]*MockExchange {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]*MockExchange)
	for name, exchange := range m.exchanges {
		result[name] = exchange
	}
	return result
}

// GetBindings returns all queue bindings
func (m *MockRabbitMQ) GetBindings() map[string][]MockBinding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string][]MockBinding)
	for key, bindings := range m.bindings {
		result[key] = append([]MockBinding{}, bindings...)
	}
	return result
}

// ClearAll clears all queues, exchanges, bindings, and messages
func (m *MockRabbitMQ) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Close all consumer channels
	for _, consumer := range m.consumers {
		if consumer.Channel != nil {
			close(consumer.Channel)
		}
		consumer.Active = false
	}
	
	m.queues = make(map[string]*MockQueue)
	m.exchanges = make(map[string]*MockExchange)
	m.bindings = make(map[string][]MockBinding)
	m.messages = make(map[string][]amqp.Publishing)
	m.consumers = make(map[string]*MockConsumer)
}