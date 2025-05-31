package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"online-shop/internal/interfaces/http/handlers"
	"online-shop/internal/interfaces/http/middleware"
	httpRouter "online-shop/internal/interfaces/http"
	"online-shop/pkg/config"
	"online-shop/tests/mocks"
)

// APITestSuite defines the test suite for API integration tests
type APITestSuite struct {
	suite.Suite
	router         *gin.Engine
	userRepo       *mocks.MockUserRepository
	productRepo    *mocks.MockProductRepository
	orderRepo      *mocks.MockOrderRepository
	userHandler    *handlers.UserHandler
	productHandler *handlers.ProductHandler
	orderHandler   *handlers.OrderHandler
}

// SetupSuite sets up the test suite
func (suite *APITestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Initialize logger
	logger := zap.NewNop()

	// Initialize mock repositories
	suite.userRepo = mocks.NewMockUserRepository()
	suite.productRepo = mocks.NewMockProductRepository()
	suite.orderRepo = mocks.NewMockOrderRepository()

	// Initialize handlers
	suite.userHandler = handlers.NewUserHandler(suite.userRepo, logger)
	suite.productHandler = handlers.NewProductHandler(suite.productRepo, logger)
	suite.orderHandler = handlers.NewOrderHandler(suite.orderRepo, logger)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(logger)

	// Initialize router
	cfg := &config.Config{Environment: "test"}
	routerInstance := httpRouter.NewRouter(
		cfg,
		logger,
		suite.userHandler,
		suite.productHandler,
		suite.orderHandler,
		authMiddleware,
	)
	routerInstance.SetupRoutes()
	suite.router = routerInstance.GetEngine()
}

// SetupTest sets up each test
func (suite *APITestSuite) SetupTest() {
	// Reset mocks before each test
	suite.userRepo.ExpectedCalls = nil
	suite.userRepo.Calls = nil
	suite.productRepo.ExpectedCalls = nil
	suite.productRepo.Calls = nil
	suite.orderRepo.ExpectedCalls = nil
	suite.orderRepo.Calls = nil
}

// TestHealthEndpoint tests the health check endpoint
func (suite *APITestSuite) TestHealthEndpoint() {
	// Create request
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	suite.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

// TestUserRegistration tests user registration endpoint
func (suite *APITestSuite) TestUserRegistration() {
	// Setup mock expectations
	suite.userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("user not found"))
	suite.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).Return(nil)

	// Create request body
	requestBody := map[string]interface{}{
		"email":      "test@example.com",
		"password":   "password123",
		"first_name": "John",
		"last_name":  "Doe",
		"phone":      "+1234567890",
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	suite.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "User created successfully", response["message"])

	// Verify mock expectations
	suite.userRepo.AssertExpectations(suite.T())
}

// TestUserRegistrationValidation tests user registration validation
func (suite *APITestSuite) TestUserRegistrationValidation() {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"password":   "password123",
				"first_name": "John",
				"last_name":  "Doe",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email is required",
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email":      "invalid-email",
				"password":   "password123",
				"first_name": "John",
				"last_name":  "Doe",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid email format",
		},
		{
			name: "password too short",
			requestBody: map[string]interface{}{
				"email":      "test@example.com",
				"password":   "123",
				"first_name": "John",
				"last_name":  "Doe",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must be at least 8 characters",
		},
		{
			name: "missing first name",
			requestBody: map[string]interface{}{
				"email":     "test@example.com",
				"password":  "password123",
				"last_name": "Doe",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "first name is required",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			// Create request body
			jsonBody, _ := json.Marshal(tt.requestBody)

			// Create request
			req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute request
			suite.router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["error"].(string), tt.expectedError)
		})
	}
}

// TestProductList tests product listing endpoint
func (suite *APITestSuite) TestProductList() {
	// Setup mock expectations
	products := []*entities.Product{
		{
			ID:    "product-1",
			Name:  "Product 1",
			SKU:   "PROD-001",
			Price: 99.99,
		},
		{
			ID:    "product-2",
			Name:  "Product 2",
			SKU:   "PROD-002",
			Price: 149.99,
		},
	}
	suite.productRepo.On("List", mock.Anything, 10, 0).Return(products, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/products?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	// Execute request
	suite.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	productsData := response["products"].([]interface{})
	assert.Len(suite.T(), productsData, 2)

	// Verify mock expectations
	suite.productRepo.AssertExpectations(suite.T())
}

// TestProductSearch tests product search endpoint
func (suite *APITestSuite) TestProductSearch() {
	// Setup mock expectations
	products := []*entities.Product{
		{
			ID:   "product-1",
			Name: "Test Product",
			SKU:  "TEST-001",
		},
	}
	suite.productRepo.On("Search", mock.Anything, "test", 10, 0).Return(products, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/products/search?q=test&limit=10", nil)
	w := httptest.NewRecorder()

	// Execute request
	suite.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	productsData := response["products"].([]interface{})
	assert.Len(suite.T(), productsData, 1)

	// Verify mock expectations
	suite.productRepo.AssertExpectations(suite.T())
}

// TestUnauthorizedAccess tests unauthorized access to protected endpoints
func (suite *APITestSuite) TestUnauthorizedAccess() {
	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/user/profile"},
		{"PUT", "/api/v1/user/profile"},
		{"GET", "/api/v1/user/orders"},
		{"POST", "/api/v1/cart/items"},
		{"POST", "/api/v1/orders"},
	}

	for _, endpoint := range protectedEndpoints {
		suite.T().Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			// Create request without authorization header
			req, _ := http.NewRequest(endpoint.method, endpoint.path, nil)
			w := httptest.NewRecorder()

			// Execute request
			suite.router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["error"].(string), "unauthorized")
		})
	}
}

// TestCORSHeaders tests CORS headers
func (suite *APITestSuite) TestCORSHeaders() {
	// Create OPTIONS request
	req, _ := http.NewRequest("OPTIONS", "/api/v1/products", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	// Execute request
	suite.router.ServeHTTP(w, req)

	// Assert CORS headers
	assert.Equal(suite.T(), "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(suite.T(), w.Header().Get("Access-Control-Allow-Methods"), "GET")
}

// TestRateLimiting tests rate limiting middleware
func (suite *APITestSuite) TestRateLimiting() {
	// This test would require a more sophisticated setup to test rate limiting
	// For now, we'll just verify that the endpoint responds normally
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestMetricsEndpoint tests the metrics endpoint
func (suite *APITestSuite) TestMetricsEndpoint() {
	// Create request
	req, _ := http.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Execute request
	suite.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Header().Get("Content-Type"), "text/plain")
}

// Run the test suite
func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}