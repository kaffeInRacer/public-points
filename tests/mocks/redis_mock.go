package mocks

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// MockRedis implements RedisInterface for testing
type MockRedis struct {
	mu         sync.RWMutex
	data       map[string]string
	hashes     map[string]map[string]string
	lists      map[string][]string
	sets       map[string]map[string]bool
	sortedSets map[string]map[string]float64
	expiry     map[string]time.Time
	shouldFail bool
	lastError  error
	callLog    []string
	connected  bool
}

// NewMockRedis creates a new mock Redis client
func NewMockRedis() *MockRedis {
	return &MockRedis{
		data:       make(map[string]string),
		hashes:     make(map[string]map[string]string),
		lists:      make(map[string][]string),
		sets:       make(map[string]map[string]bool),
		sortedSets: make(map[string]map[string]float64),
		expiry:     make(map[string]time.Time),
		callLog:    make([]string, 0),
		connected:  true,
	}
}

// SetShouldFail sets whether Redis operations should fail
func (m *MockRedis) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// SetLastError sets the last error
func (m *MockRedis) SetLastError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = err
}

// GetCallLog returns the call log
func (m *MockRedis) GetCallLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.callLog...)
}

// ClearCallLog clears the call log
func (m *MockRedis) ClearCallLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = make([]string, 0)
}

// SetConnected sets the connection status
func (m *MockRedis) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

func (m *MockRedis) logCall(method string) {
	m.callLog = append(m.callLog, method)
}

func (m *MockRedis) checkError() error {
	if !m.connected {
		return errors.New("redis: connection closed")
	}
	if m.shouldFail {
		if m.lastError != nil {
			return m.lastError
		}
		return errors.New("redis: operation failed")
	}
	return nil
}

func (m *MockRedis) isExpired(key string) bool {
	if expiry, exists := m.expiry[key]; exists {
		return time.Now().After(expiry)
	}
	return false
}

func (m *MockRedis) cleanupExpired(key string) {
	if m.isExpired(key) {
		delete(m.data, key)
		delete(m.hashes, key)
		delete(m.lists, key)
		delete(m.sets, key)
		delete(m.sortedSets, key)
		delete(m.expiry, key)
	}
}

// Get implements RedisInterface
func (m *MockRedis) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("Get")
	
	if err := m.checkError(); err != nil {
		return "", err
	}
	
	m.cleanupExpired(key)
	
	if value, exists := m.data[key]; exists {
		return value, nil
	}
	
	return "", errors.New("redis: nil")
}

// Set implements RedisInterface
func (m *MockRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Set")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	m.data[key] = fmt.Sprintf("%v", value)
	
	if expiration > 0 {
		m.expiry[key] = time.Now().Add(expiration)
	} else {
		delete(m.expiry, key)
	}
	
	return nil
}

// Del implements RedisInterface
func (m *MockRedis) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Del")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	for _, key := range keys {
		delete(m.data, key)
		delete(m.hashes, key)
		delete(m.lists, key)
		delete(m.sets, key)
		delete(m.sortedSets, key)
		delete(m.expiry, key)
	}
	
	return nil
}

// Exists implements RedisInterface
func (m *MockRedis) Exists(ctx context.Context, keys ...string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("Exists")
	
	if err := m.checkError(); err != nil {
		return 0, err
	}
	
	count := int64(0)
	for _, key := range keys {
		m.cleanupExpired(key)
		if _, exists := m.data[key]; exists {
			count++
		} else if _, exists := m.hashes[key]; exists {
			count++
		} else if _, exists := m.lists[key]; exists {
			count++
		} else if _, exists := m.sets[key]; exists {
			count++
		} else if _, exists := m.sortedSets[key]; exists {
			count++
		}
	}
	
	return count, nil
}

// Expire implements RedisInterface
func (m *MockRedis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Expire")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	// Check if key exists
	exists := false
	if _, ok := m.data[key]; ok {
		exists = true
	} else if _, ok := m.hashes[key]; ok {
		exists = true
	} else if _, ok := m.lists[key]; ok {
		exists = true
	} else if _, ok := m.sets[key]; ok {
		exists = true
	} else if _, ok := m.sortedSets[key]; ok {
		exists = true
	}
	
	if !exists {
		return errors.New("redis: key does not exist")
	}
	
	m.expiry[key] = time.Now().Add(expiration)
	return nil
}

// TTL implements RedisInterface
func (m *MockRedis) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("TTL")
	
	if err := m.checkError(); err != nil {
		return 0, err
	}
	
	if expiry, exists := m.expiry[key]; exists {
		ttl := time.Until(expiry)
		if ttl <= 0 {
			return -2 * time.Second, nil // Key expired
		}
		return ttl, nil
	}
	
	return -1 * time.Second, nil // Key exists but no expiry
}

// HGet implements RedisInterface
func (m *MockRedis) HGet(ctx context.Context, key, field string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("HGet")
	
	if err := m.checkError(); err != nil {
		return "", err
	}
	
	m.cleanupExpired(key)
	
	if hash, exists := m.hashes[key]; exists {
		if value, fieldExists := hash[field]; fieldExists {
			return value, nil
		}
	}
	
	return "", errors.New("redis: nil")
}

// HSet implements RedisInterface
func (m *MockRedis) HSet(ctx context.Context, key string, values ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("HSet")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if len(values)%2 != 0 {
		return errors.New("redis: wrong number of arguments")
	}
	
	if m.hashes[key] == nil {
		m.hashes[key] = make(map[string]string)
	}
	
	for i := 0; i < len(values); i += 2 {
		field := fmt.Sprintf("%v", values[i])
		value := fmt.Sprintf("%v", values[i+1])
		m.hashes[key][field] = value
	}
	
	return nil
}

// HGetAll implements RedisInterface
func (m *MockRedis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("HGetAll")
	
	if err := m.checkError(); err != nil {
		return nil, err
	}
	
	m.cleanupExpired(key)
	
	if hash, exists := m.hashes[key]; exists {
		result := make(map[string]string)
		for k, v := range hash {
			result[k] = v
		}
		return result, nil
	}
	
	return make(map[string]string), nil
}

// HDel implements RedisInterface
func (m *MockRedis) HDel(ctx context.Context, key string, fields ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("HDel")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if hash, exists := m.hashes[key]; exists {
		for _, field := range fields {
			delete(hash, field)
		}
	}
	
	return nil
}

// LPush implements RedisInterface
func (m *MockRedis) LPush(ctx context.Context, key string, values ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("LPush")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if m.lists[key] == nil {
		m.lists[key] = make([]string, 0)
	}
	
	// Prepend values to the list
	for i := len(values) - 1; i >= 0; i-- {
		value := fmt.Sprintf("%v", values[i])
		m.lists[key] = append([]string{value}, m.lists[key]...)
	}
	
	return nil
}

// RPop implements RedisInterface
func (m *MockRedis) RPop(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("RPop")
	
	if err := m.checkError(); err != nil {
		return "", err
	}
	
	m.cleanupExpired(key)
	
	if list, exists := m.lists[key]; exists && len(list) > 0 {
		value := list[len(list)-1]
		m.lists[key] = list[:len(list)-1]
		return value, nil
	}
	
	return "", errors.New("redis: nil")
}

// LLen implements RedisInterface
func (m *MockRedis) LLen(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("LLen")
	
	if err := m.checkError(); err != nil {
		return 0, err
	}
	
	m.cleanupExpired(key)
	
	if list, exists := m.lists[key]; exists {
		return int64(len(list)), nil
	}
	
	return 0, nil
}

// SAdd implements RedisInterface
func (m *MockRedis) SAdd(ctx context.Context, key string, members ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("SAdd")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if m.sets[key] == nil {
		m.sets[key] = make(map[string]bool)
	}
	
	for _, member := range members {
		m.sets[key][fmt.Sprintf("%v", member)] = true
	}
	
	return nil
}

// SMembers implements RedisInterface
func (m *MockRedis) SMembers(ctx context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("SMembers")
	
	if err := m.checkError(); err != nil {
		return nil, err
	}
	
	m.cleanupExpired(key)
	
	if set, exists := m.sets[key]; exists {
		members := make([]string, 0, len(set))
		for member := range set {
			members = append(members, member)
		}
		return members, nil
	}
	
	return make([]string, 0), nil
}

// ZAdd implements RedisInterface
func (m *MockRedis) ZAdd(ctx context.Context, key string, members ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("ZAdd")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	if len(members)%2 != 0 {
		return errors.New("redis: wrong number of arguments")
	}
	
	if m.sortedSets[key] == nil {
		m.sortedSets[key] = make(map[string]float64)
	}
	
	for i := 0; i < len(members); i += 2 {
		score, err := strconv.ParseFloat(fmt.Sprintf("%v", members[i]), 64)
		if err != nil {
			return err
		}
		member := fmt.Sprintf("%v", members[i+1])
		m.sortedSets[key][member] = score
	}
	
	return nil
}

// ZRange implements RedisInterface
func (m *MockRedis) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("ZRange")
	
	if err := m.checkError(); err != nil {
		return nil, err
	}
	
	m.cleanupExpired(key)
	
	if sortedSet, exists := m.sortedSets[key]; exists {
		// Simple implementation - return all members (should be sorted by score)
		members := make([]string, 0, len(sortedSet))
		for member := range sortedSet {
			members = append(members, member)
		}
		return members, nil
	}
	
	return make([]string, 0), nil
}

// Ping implements RedisInterface
func (m *MockRedis) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("Ping")
	
	if !m.connected {
		return errors.New("redis: connection closed")
	}
	
	return m.checkError()
}

// FlushDB implements RedisInterface
func (m *MockRedis) FlushDB(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("FlushDB")
	
	if err := m.checkError(); err != nil {
		return err
	}
	
	m.data = make(map[string]string)
	m.hashes = make(map[string]map[string]string)
	m.lists = make(map[string][]string)
	m.sets = make(map[string]map[string]bool)
	m.sortedSets = make(map[string]map[string]float64)
	m.expiry = make(map[string]time.Time)
	
	return nil
}

// Close implements RedisInterface
func (m *MockRedis) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Close")
	
	m.connected = false
	return nil
}

// Helper methods for testing

// GetAllData returns all stored data for testing
func (m *MockRedis) GetAllData() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]interface{})
	result["strings"] = m.data
	result["hashes"] = m.hashes
	result["lists"] = m.lists
	result["sets"] = m.sets
	result["sorted_sets"] = m.sortedSets
	result["expiry"] = m.expiry
	
	return result
}

// ClearAllData clears all data
func (m *MockRedis) ClearAllData() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data = make(map[string]string)
	m.hashes = make(map[string]map[string]string)
	m.lists = make(map[string][]string)
	m.sets = make(map[string]map[string]bool)
	m.sortedSets = make(map[string]map[string]float64)
	m.expiry = make(map[string]time.Time)
}

// GetKeyCount returns the total number of keys
func (m *MockRedis) GetKeyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	keys := make(map[string]bool)
	
	for key := range m.data {
		keys[key] = true
	}
	for key := range m.hashes {
		keys[key] = true
	}
	for key := range m.lists {
		keys[key] = true
	}
	for key := range m.sets {
		keys[key] = true
	}
	for key := range m.sortedSets {
		keys[key] = true
	}
	
	return len(keys)
}