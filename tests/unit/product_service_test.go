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

func TestProductService_CreateProduct(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockProductRepository()
	logger := zap.NewNop()
	handler := handlers.NewCreateProductHandler(mockRepo, logger)

	tests := []struct {
		name          string
		command       commands.CreateProductCommand
		setupMocks    func()
		expectedError string
	}{
		{
			name: "successful product creation",
			command: commands.CreateProductCommand{
				Name:        "Test Product",
				Description: "Test Description",
				SKU:         "TEST-001",
				Price:       99.99,
				CategoryID:  "cat-123",
				Stock:       100,
			},
			setupMocks: func() {
				// Mock GetBySKU to return nil (product doesn't exist)
				mockRepo.On("GetBySKU", mock.Anything, "TEST-001").Return(nil, errors.New("product not found"))
				// Mock Create to succeed
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Product")).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "product with SKU already exists",
			command: commands.CreateProductCommand{
				Name:        "Test Product",
				Description: "Test Description",
				SKU:         "EXISTING-001",
				Price:       99.99,
				CategoryID:  "cat-123",
				Stock:       100,
			},
			setupMocks: func() {
				existingProduct := &entities.Product{
					ID:  "existing-id",
					SKU: "EXISTING-001",
				}
				// Mock GetBySKU to return existing product
				mockRepo.On("GetBySKU", mock.Anything, "EXISTING-001").Return(existingProduct, nil)
			},
			expectedError: "product with SKU already exists",
		},
		{
			name: "invalid price",
			command: commands.CreateProductCommand{
				Name:        "Test Product",
				Description: "Test Description",
				SKU:         "TEST-001",
				Price:       -10.00,
				CategoryID:  "cat-123",
				Stock:       100,
			},
			setupMocks:    func() {},
			expectedError: "price must be positive",
		},
		{
			name: "empty name",
			command: commands.CreateProductCommand{
				Name:        "",
				Description: "Test Description",
				SKU:         "TEST-001",
				Price:       99.99,
				CategoryID:  "cat-123",
				Stock:       100,
			},
			setupMocks:    func() {},
			expectedError: "product name is required",
		},
		{
			name: "empty SKU",
			command: commands.CreateProductCommand{
				Name:        "Test Product",
				Description: "Test Description",
				SKU:         "",
				Price:       99.99,
				CategoryID:  "cat-123",
				Stock:       100,
			},
			setupMocks:    func() {},
			expectedError: "SKU is required",
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

func TestProductService_GetProduct(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockProductRepository()
	logger := zap.NewNop()
	handler := handlers.NewGetProductHandler(mockRepo, logger)

	tests := []struct {
		name            string
		productID       string
		setupMocks      func()
		expectedProduct *entities.Product
		expectedError   string
	}{
		{
			name:      "successful product retrieval",
			productID: "product-123",
			setupMocks: func() {
				product := &entities.Product{
					ID:          "product-123",
					Name:        "Test Product",
					Description: "Test Description",
					SKU:         "TEST-001",
					Price:       99.99,
					Stock:       100,
					CreatedAt:   time.Now(),
				}
				mockRepo.On("GetByID", mock.Anything, "product-123").Return(product, nil)
			},
			expectedProduct: &entities.Product{
				ID:          "product-123",
				Name:        "Test Product",
				Description: "Test Description",
				SKU:         "TEST-001",
				Price:       99.99,
				Stock:       100,
			},
			expectedError: "",
		},
		{
			name:      "product not found",
			productID: "non-existent",
			setupMocks: func() {
				mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("product not found"))
			},
			expectedProduct: nil,
			expectedError:   "product not found",
		},
		{
			name:            "empty product ID",
			productID:       "",
			setupMocks:      func() {},
			expectedProduct: nil,
			expectedError:   "product ID is required",
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
			query := commands.GetProductQuery{ID: tt.productID}
			product, err := handler.Handle(ctx, query)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, product)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, product)
				assert.Equal(t, tt.expectedProduct.ID, product.ID)
				assert.Equal(t, tt.expectedProduct.Name, product.Name)
				assert.Equal(t, tt.expectedProduct.SKU, product.SKU)
				assert.Equal(t, tt.expectedProduct.Price, product.Price)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestProductService_UpdateStock(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockProductRepository()
	logger := zap.NewNop()
	handler := handlers.NewUpdateStockHandler(mockRepo, logger)

	tests := []struct {
		name          string
		command       commands.UpdateStockCommand
		setupMocks    func()
		expectedError string
	}{
		{
			name: "successful stock update",
			command: commands.UpdateStockCommand{
				ProductID: "product-123",
				Quantity:  50,
			},
			setupMocks: func() {
				existingProduct := &entities.Product{
					ID:    "product-123",
					Name:  "Test Product",
					Stock: 100,
				}
				mockRepo.On("GetByID", mock.Anything, "product-123").Return(existingProduct, nil)
				mockRepo.On("UpdateStock", mock.Anything, "product-123", 50).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "product not found",
			command: commands.UpdateStockCommand{
				ProductID: "non-existent",
				Quantity:  50,
			},
			setupMocks: func() {
				mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("product not found"))
			},
			expectedError: "product not found",
		},
		{
			name: "negative stock quantity",
			command: commands.UpdateStockCommand{
				ProductID: "product-123",
				Quantity:  -10,
			},
			setupMocks:    func() {},
			expectedError: "stock quantity cannot be negative",
		},
		{
			name: "empty product ID",
			command: commands.UpdateStockCommand{
				ProductID: "",
				Quantity:  50,
			},
			setupMocks:    func() {},
			expectedError: "product ID is required",
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

func TestProductService_SearchProducts(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockProductRepository()
	logger := zap.NewNop()
	handler := handlers.NewSearchProductsHandler(mockRepo, logger)

	tests := []struct {
		name             string
		query            string
		limit            int
		offset           int
		setupMocks       func()
		expectedProducts []*entities.Product
		expectedError    string
	}{
		{
			name:   "successful product search",
			query:  "test",
			limit:  10,
			offset: 0,
			setupMocks: func() {
				products := []*entities.Product{
					{
						ID:   "product-1",
						Name: "Test Product 1",
						SKU:  "TEST-001",
					},
					{
						ID:   "product-2",
						Name: "Test Product 2",
						SKU:  "TEST-002",
					},
				}
				mockRepo.On("Search", mock.Anything, "test", 10, 0).Return(products, nil)
			},
			expectedProducts: []*entities.Product{
				{
					ID:   "product-1",
					Name: "Test Product 1",
					SKU:  "TEST-001",
				},
				{
					ID:   "product-2",
					Name: "Test Product 2",
					SKU:  "TEST-002",
				},
			},
			expectedError: "",
		},
		{
			name:   "no products found",
			query:  "nonexistent",
			limit:  10,
			offset: 0,
			setupMocks: func() {
				mockRepo.On("Search", mock.Anything, "nonexistent", 10, 0).Return([]*entities.Product{}, nil)
			},
			expectedProducts: []*entities.Product{},
			expectedError:    "",
		},
		{
			name:             "empty search query",
			query:            "",
			limit:            10,
			offset:           0,
			setupMocks:       func() {},
			expectedProducts: nil,
			expectedError:    "search query is required",
		},
		{
			name:             "invalid limit",
			query:            "test",
			limit:            0,
			offset:           0,
			setupMocks:       func() {},
			expectedProducts: nil,
			expectedError:    "limit must be positive",
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
			searchQuery := commands.SearchProductsQuery{
				Query:  tt.query,
				Limit:  tt.limit,
				Offset: tt.offset,
			}
			products, err := handler.Handle(ctx, searchQuery)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, products)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedProducts), len(products))
				for i, expectedProduct := range tt.expectedProducts {
					assert.Equal(t, expectedProduct.ID, products[i].ID)
					assert.Equal(t, expectedProduct.Name, products[i].Name)
					assert.Equal(t, expectedProduct.SKU, products[i].SKU)
				}
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}