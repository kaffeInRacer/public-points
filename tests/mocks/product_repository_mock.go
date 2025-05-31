package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"online-shop/internal/domain/entities"
	"online-shop/internal/domain/repositories"
)

// MockProductRepository is a mock implementation of ProductRepository
type MockProductRepository struct {
	mock.Mock
}

// NewMockProductRepository creates a new mock product repository
func NewMockProductRepository() *MockProductRepository {
	return &MockProductRepository{}
}

// Create mocks the Create method
func (m *MockProductRepository) Create(ctx context.Context, product *entities.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockProductRepository) GetByID(ctx context.Context, id string) (*entities.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Product), args.Error(1)
}

// GetBySKU mocks the GetBySKU method
func (m *MockProductRepository) GetBySKU(ctx context.Context, sku string) (*entities.Product, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Product), args.Error(1)
}

// Update mocks the Update method
func (m *MockProductRepository) Update(ctx context.Context, product *entities.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockProductRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// List mocks the List method
func (m *MockProductRepository) List(ctx context.Context, limit, offset int) ([]*entities.Product, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Product), args.Error(1)
}

// GetByCategory mocks the GetByCategory method
func (m *MockProductRepository) GetByCategory(ctx context.Context, categoryID string, limit, offset int) ([]*entities.Product, error) {
	args := m.Called(ctx, categoryID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Product), args.Error(1)
}

// UpdateStock mocks the UpdateStock method
func (m *MockProductRepository) UpdateStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}

// GetFeatured mocks the GetFeatured method
func (m *MockProductRepository) GetFeatured(ctx context.Context, limit int) ([]*entities.Product, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Product), args.Error(1)
}

// Search mocks the Search method
func (m *MockProductRepository) Search(ctx context.Context, query string, limit, offset int) ([]*entities.Product, error) {
	args := m.Called(ctx, query, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Product), args.Error(1)
}

// Ensure MockProductRepository implements ProductRepository interface
var _ repositories.ProductRepository = (*MockProductRepository)(nil)