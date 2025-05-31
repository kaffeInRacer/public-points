package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"online-shop/internal/domain/entities"
	"online-shop/internal/domain/repositories"
)

// MockOrderRepository is a mock implementation of OrderRepository
type MockOrderRepository struct {
	mock.Mock
}

// NewMockOrderRepository creates a new mock order repository
func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{}
}

// Create mocks the Create method
func (m *MockOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockOrderRepository) GetByID(ctx context.Context, id string) (*entities.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Order), args.Error(1)
}

// GetByOrderNumber mocks the GetByOrderNumber method
func (m *MockOrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*entities.Order, error) {
	args := m.Called(ctx, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Order), args.Error(1)
}

// Update mocks the Update method
func (m *MockOrderRepository) Update(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockOrderRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetByUserID mocks the GetByUserID method
func (m *MockOrderRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*entities.Order, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Order), args.Error(1)
}

// List mocks the List method
func (m *MockOrderRepository) List(ctx context.Context, limit, offset int) ([]*entities.Order, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Order), args.Error(1)
}

// UpdateStatus mocks the UpdateStatus method
func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id string, status entities.OrderStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// GetItems mocks the GetItems method
func (m *MockOrderRepository) GetItems(ctx context.Context, orderID string) ([]*entities.OrderItem, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.OrderItem), args.Error(1)
}

// CreateItem mocks the CreateItem method
func (m *MockOrderRepository) CreateItem(ctx context.Context, item *entities.OrderItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

// Ensure MockOrderRepository implements OrderRepository interface
var _ repositories.OrderRepository = (*MockOrderRepository)(nil)