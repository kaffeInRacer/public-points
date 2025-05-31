package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"online-shop/internal/domain/entities"
	"online-shop/internal/domain/repositories"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

// NewMockUserRepository creates a new mock user repository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{}
}

// Create mocks the Create method
func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

// GetByEmail mocks the GetByEmail method
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

// Update mocks the Update method
func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// List mocks the List method
func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*entities.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.User), args.Error(1)
}

// GetAddresses mocks the GetAddresses method
func (m *MockUserRepository) GetAddresses(ctx context.Context, userID string) ([]*entities.UserAddress, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.UserAddress), args.Error(1)
}

// CreateAddress mocks the CreateAddress method
func (m *MockUserRepository) CreateAddress(ctx context.Context, address *entities.UserAddress) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

// UpdateAddress mocks the UpdateAddress method
func (m *MockUserRepository) UpdateAddress(ctx context.Context, address *entities.UserAddress) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

// DeleteAddress mocks the DeleteAddress method
func (m *MockUserRepository) DeleteAddress(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Ensure MockUserRepository implements UserRepository interface
var _ repositories.UserRepository = (*MockUserRepository)(nil)