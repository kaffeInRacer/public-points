package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"online-shop/internal/application/commands"
	"online-shop/internal/application/handlers"
	"online-shop/internal/domain/entities"
	"online-shop/tests/mocks"
)

func TestUserService_CreateUser(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockUserRepository()
	logger := zap.NewNop()
	handler := handlers.NewCreateUserHandler(mockRepo, logger)

	tests := []struct {
		name          string
		command       commands.CreateUserCommand
		setupMocks    func()
		expectedError string
	}{
		{
			name: "successful user creation",
			command: commands.CreateUserCommand{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+1234567890",
			},
			setupMocks: func() {
				// Mock GetByEmail to return nil (user doesn't exist)
				mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("user not found"))
				// Mock Create to succeed
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "user already exists",
			command: commands.CreateUserCommand{
				Email:     "existing@example.com",
				Password:  "password123",
				FirstName: "Jane",
				LastName:  "Doe",
				Phone:     "+1234567890",
			},
			setupMocks: func() {
				existingUser := &entities.User{
					ID:    "existing-id",
					Email: "existing@example.com",
				}
				// Mock GetByEmail to return existing user
				mockRepo.On("GetByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)
			},
			expectedError: "user already exists",
		},
		{
			name: "invalid email format",
			command: commands.CreateUserCommand{
				Email:     "invalid-email",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			setupMocks:    func() {},
			expectedError: "invalid email format",
		},
		{
			name: "password too short",
			command: commands.CreateUserCommand{
				Email:     "test@example.com",
				Password:  "123",
				FirstName: "John",
				LastName:  "Doe",
			},
			setupMocks:    func() {},
			expectedError: "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil

			// Setup mocks
			tt.setupMocks()

			// Execute
			ctx := context.Background()
			err := handler.Handle(ctx, tt.command)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockUserRepository()
	logger := zap.NewNop()
	handler := handlers.NewGetUserHandler(mockRepo, logger)

	tests := []struct {
		name          string
		userID        string
		setupMocks    func()
		expectedUser  *entities.User
		expectedError string
	}{
		{
			name:   "successful user retrieval",
			userID: "user-123",
			setupMocks: func() {
				user := &entities.User{
					ID:        "user-123",
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					CreatedAt: time.Now(),
				}
				mockRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil)
			},
			expectedUser: &entities.User{
				ID:        "user-123",
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedError: "",
		},
		{
			name:   "user not found",
			userID: "non-existent",
			setupMocks: func() {
				mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("user not found"))
			},
			expectedUser:  nil,
			expectedError: "user not found",
		},
		{
			name:          "empty user ID",
			userID:        "",
			setupMocks:    func() {},
			expectedUser:  nil,
			expectedError: "user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil

			// Setup mocks
			tt.setupMocks()

			// Execute
			ctx := context.Background()
			query := commands.GetUserQuery{ID: tt.userID}
			user, err := handler.Handle(ctx, query)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Email, user.Email)
				assert.Equal(t, tt.expectedUser.FirstName, user.FirstName)
				assert.Equal(t, tt.expectedUser.LastName, user.LastName)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockUserRepository()
	logger := zap.NewNop()
	handler := handlers.NewUpdateUserHandler(mockRepo, logger)

	tests := []struct {
		name          string
		command       commands.UpdateUserCommand
		setupMocks    func()
		expectedError string
	}{
		{
			name: "successful user update",
			command: commands.UpdateUserCommand{
				ID:        "user-123",
				FirstName: "John Updated",
				LastName:  "Doe Updated",
				Phone:     "+9876543210",
			},
			setupMocks: func() {
				existingUser := &entities.User{
					ID:        "user-123",
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Phone:     "+1234567890",
				}
				mockRepo.On("GetByID", mock.Anything, "user-123").Return(existingUser, nil)
				mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.User")).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "user not found",
			command: commands.UpdateUserCommand{
				ID:        "non-existent",
				FirstName: "John",
				LastName:  "Doe",
			},
			setupMocks: func() {
				mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("user not found"))
			},
			expectedError: "user not found",
		},
		{
			name: "empty user ID",
			command: commands.UpdateUserCommand{
				ID:        "",
				FirstName: "John",
				LastName:  "Doe",
			},
			setupMocks:    func() {},
			expectedError: "user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil

			// Setup mocks
			tt.setupMocks()

			// Execute
			ctx := context.Background()
			err := handler.Handle(ctx, tt.command)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockUserRepository()
	logger := zap.NewNop()
	handler := handlers.NewDeleteUserHandler(mockRepo, logger)

	tests := []struct {
		name          string
		userID        string
		setupMocks    func()
		expectedError string
	}{
		{
			name:   "successful user deletion",
			userID: "user-123",
			setupMocks: func() {
				existingUser := &entities.User{
					ID:    "user-123",
					Email: "test@example.com",
				}
				mockRepo.On("GetByID", mock.Anything, "user-123").Return(existingUser, nil)
				mockRepo.On("Delete", mock.Anything, "user-123").Return(nil)
			},
			expectedError: "",
		},
		{
			name:   "user not found",
			userID: "non-existent",
			setupMocks: func() {
				mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("user not found"))
			},
			expectedError: "user not found",
		},
		{
			name:          "empty user ID",
			userID:        "",
			setupMocks:    func() {},
			expectedError: "user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil

			// Setup mocks
			tt.setupMocks()

			// Execute
			ctx := context.Background()
			command := commands.DeleteUserCommand{ID: tt.userID}
			err := handler.Handle(ctx, command)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}