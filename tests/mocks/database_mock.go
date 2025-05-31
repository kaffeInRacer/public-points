package mocks

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"gorm.io/gorm"
)

// MockDatabase implements DatabaseInterface for testing
type MockDatabase struct {
	mu           sync.RWMutex
	data         map[string][]interface{}
	lastError    error
	shouldFail   bool
	callLog      []string
	transactions map[string]*MockTransaction
	txCounter    int
}

// MockTransaction represents a database transaction
type MockTransaction struct {
	ID        string
	committed bool
	rolledBack bool
	operations []string
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		data:         make(map[string][]interface{}),
		callLog:      make([]string, 0),
		transactions: make(map[string]*MockTransaction),
	}
}

// SetShouldFail sets whether the database should fail operations
func (m *MockDatabase) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// SetLastError sets the last error
func (m *MockDatabase) SetLastError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = err
}

// GetCallLog returns the call log
func (m *MockDatabase) GetCallLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.callLog...)
}

// ClearCallLog clears the call log
func (m *MockDatabase) ClearCallLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = make([]string, 0)
}

// AddData adds test data to the mock database
func (m *MockDatabase) AddData(table string, records ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data[table] == nil {
		m.data[table] = make([]interface{}, 0)
	}
	m.data[table] = append(m.data[table], records...)
}

// GetData returns data from the mock database
func (m *MockDatabase) GetData(table string) []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]interface{}{}, m.data[table]...)
}

// ClearData clears all data
func (m *MockDatabase) ClearData() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]interface{})
}

func (m *MockDatabase) logCall(method string) {
	m.callLog = append(m.callLog, method)
}

func (m *MockDatabase) checkError() *gorm.DB {
	if m.shouldFail {
		return &gorm.DB{Error: m.lastError}
	}
	return &gorm.DB{}
}

// Create implements DatabaseInterface
func (m *MockDatabase) Create(value interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Create")
	
	if db := m.checkError(); db.Error != nil {
		return db
	}
	
	tableName := getTableName(value)
	if m.data[tableName] == nil {
		m.data[tableName] = make([]interface{}, 0)
	}
	m.data[tableName] = append(m.data[tableName], value)
	
	return &gorm.DB{}
}

// First implements DatabaseInterface
func (m *MockDatabase) First(dest interface{}, conds ...interface{}) *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("First")
	
	if db := m.checkError(); db.Error != nil {
		return db
	}
	
	tableName := getTableName(dest)
	records := m.data[tableName]
	
	if len(records) == 0 {
		return &gorm.DB{Error: gorm.ErrRecordNotFound}
	}
	
	// Simple implementation - return first record
	if len(records) > 0 {
		copyValue(records[0], dest)
	}
	
	return &gorm.DB{}
}

// Find implements DatabaseInterface
func (m *MockDatabase) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("Find")
	
	if db := m.checkError(); db.Error != nil {
		return db
	}
	
	tableName := getTableName(dest)
	records := m.data[tableName]
	
	// Simple implementation - return all records
	copySlice(records, dest)
	
	return &gorm.DB{}
}

// Where implements DatabaseInterface
func (m *MockDatabase) Where(query interface{}, args ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Where")
	return &gorm.DB{}
}

// Update implements DatabaseInterface
func (m *MockDatabase) Update(column string, value interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Update")
	return m.checkError()
}

// Updates implements DatabaseInterface
func (m *MockDatabase) Updates(values interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Updates")
	return m.checkError()
}

// Delete implements DatabaseInterface
func (m *MockDatabase) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Delete")
	return m.checkError()
}

// Save implements DatabaseInterface
func (m *MockDatabase) Save(value interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Save")
	return m.checkError()
}

// Begin implements DatabaseInterface
func (m *MockDatabase) Begin() *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Begin")
	
	m.txCounter++
	txID := fmt.Sprintf("tx_%d", m.txCounter)
	m.transactions[txID] = &MockTransaction{
		ID:         txID,
		operations: make([]string, 0),
	}
	
	return &gorm.DB{}
}

// Commit implements DatabaseInterface
func (m *MockDatabase) Commit() *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Commit")
	return m.checkError()
}

// Rollback implements DatabaseInterface
func (m *MockDatabase) Rollback() *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Rollback")
	return m.checkError()
}

// Raw implements DatabaseInterface
func (m *MockDatabase) Raw(sql string, values ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Raw")
	return m.checkError()
}

// Exec implements DatabaseInterface
func (m *MockDatabase) Exec(sql string, values ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Exec")
	return m.checkError()
}

// Model implements DatabaseInterface
func (m *MockDatabase) Model(value interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Model")
	return &gorm.DB{}
}

// Table implements DatabaseInterface
func (m *MockDatabase) Table(name string) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Table")
	return &gorm.DB{}
}

// Count implements DatabaseInterface
func (m *MockDatabase) Count(count *int64) *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.logCall("Count")
	
	if db := m.checkError(); db.Error != nil {
		return db
	}
	
	// Simple implementation - count all records
	total := int64(0)
	for _, records := range m.data {
		total += int64(len(records))
	}
	*count = total
	
	return &gorm.DB{}
}

// Limit implements DatabaseInterface
func (m *MockDatabase) Limit(limit int) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Limit")
	return &gorm.DB{}
}

// Offset implements DatabaseInterface
func (m *MockDatabase) Offset(offset int) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Offset")
	return &gorm.DB{}
}

// Order implements DatabaseInterface
func (m *MockDatabase) Order(value interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Order")
	return &gorm.DB{}
}

// Group implements DatabaseInterface
func (m *MockDatabase) Group(name string) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Group")
	return &gorm.DB{}
}

// Having implements DatabaseInterface
func (m *MockDatabase) Having(query interface{}, args ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Having")
	return &gorm.DB{}
}

// Joins implements DatabaseInterface
func (m *MockDatabase) Joins(query string, args ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Joins")
	return &gorm.DB{}
}

// Preload implements DatabaseInterface
func (m *MockDatabase) Preload(query string, args ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Preload")
	return &gorm.DB{}
}

// Association implements DatabaseInterface
func (m *MockDatabase) Association(column string) *gorm.Association {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Association")
	return &gorm.Association{}
}

// Transaction implements DatabaseInterface
func (m *MockDatabase) Transaction(fc func(tx *gorm.DB) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCall("Transaction")
	
	if m.shouldFail {
		return m.lastError
	}
	
	// Create a mock transaction
	tx := &gorm.DB{}
	return fc(tx)
}

// Helper functions
func getTableName(value interface{}) string {
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		t = t.Elem()
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
	}
	return t.Name()
}

func copyValue(src, dest interface{}) {
	srcVal := reflect.ValueOf(src)
	destVal := reflect.ValueOf(dest)
	
	if destVal.Kind() == reflect.Ptr {
		destVal = destVal.Elem()
	}
	
	if srcVal.Type().AssignableTo(destVal.Type()) {
		destVal.Set(srcVal)
	}
}

func copySlice(src []interface{}, dest interface{}) {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() == reflect.Ptr {
		destVal = destVal.Elem()
	}
	
	if destVal.Kind() != reflect.Slice {
		return
	}
	
	// Create a new slice with the same type
	sliceType := destVal.Type()
	newSlice := reflect.MakeSlice(sliceType, len(src), len(src))
	
	for i, item := range src {
		itemVal := reflect.ValueOf(item)
		if itemVal.Type().AssignableTo(sliceType.Elem()) {
			newSlice.Index(i).Set(itemVal)
		}
	}
	
	destVal.Set(newSlice)
}