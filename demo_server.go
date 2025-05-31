package main

import (
	"fmt"
	"net/http"
	"online-shop/pkg/config"
	"online-shop/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize logger
	logger.Init()
	log := logger.GetLogger()
	log.Info("🚀 Starting Online Shop Demo Server...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Failed to load config: ", err)
		// Use default config
		cfg = &config.Config{
			Server: config.ServerConfig{
				Host: "0.0.0.0",
				Port: "12000",
			},
		}
	}

	// Create Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "online-shop-api",
			"version": "1.0.0",
		})
	})

	// API info endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "🛍️ Welcome to Online Shop API",
			"version": "1.0.0",
			"architecture": gin.H{
				"patterns": []string{"DDD", "CQRS", "Clean Architecture"},
				"technologies": []string{
					"Go 1.19+",
					"Gin Framework",
					"PostgreSQL + GORM",
					"Redis Caching",
					"Elasticsearch",
					"gRPC Services",
					"JWT Authentication",
					"Midtrans Payment",
				},
			},
			"features": []string{
				"User Management & Authentication",
				"Product Catalog & Search",
				"Shopping Cart & Orders",
				"Payment Processing",
				"Real-time Notifications",
				"Admin Dashboard",
			},
			"endpoints": gin.H{
				"health":   "/health",
				"users":    "/api/v1/users/*",
				"products": "/api/v1/products/*",
				"orders":   "/api/v1/orders/*",
				"payments": "/api/v1/payments/*",
			},
			"documentation": "https://github.com/your-repo/online-shop",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Users endpoints
		users := v1.Group("/users")
		{
			users.POST("/register", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "👤 User Registration Endpoint",
					"description": "Register a new user account",
					"method": "POST",
					"body": gin.H{
						"name":     "string (required)",
						"email":    "string (required, unique)",
						"password": "string (required, min 8 chars)",
						"role":     "string (optional, default: customer)",
					},
				})
			})
			users.POST("/login", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "🔐 User Login Endpoint",
					"description": "Authenticate user and get JWT token",
					"method": "POST",
					"body": gin.H{
						"email":    "string (required)",
						"password": "string (required)",
					},
				})
			})
			users.GET("/profile", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "👤 User Profile Endpoint",
					"description": "Get authenticated user profile",
					"method": "GET",
					"headers": gin.H{
						"Authorization": "Bearer <jwt_token>",
					},
				})
			})
		}

		// Products endpoints
		products := v1.Group("/products")
		{
			products.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "📦 Products List Endpoint",
					"description": "Get paginated list of products",
					"method": "GET",
					"query_params": gin.H{
						"page":     "int (optional, default: 1)",
						"limit":    "int (optional, default: 10)",
						"category": "string (optional)",
						"sort":     "string (optional: name, price, created_at)",
					},
				})
			})
			products.GET("/:id", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "📦 Product Details Endpoint",
					"description": "Get detailed product information",
					"method": "GET",
					"params": gin.H{
						"id": "string (required, product UUID)",
					},
				})
			})
			products.POST("/search", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "🔍 Product Search Endpoint",
					"description": "Search products using Elasticsearch",
					"method": "POST",
					"body": gin.H{
						"query":    "string (required, search term)",
						"filters":  "object (optional, category, price range, etc.)",
						"page":     "int (optional, default: 1)",
						"limit":    "int (optional, default: 10)",
					},
				})
			})
		}

		// Orders endpoints
		orders := v1.Group("/orders")
		{
			orders.POST("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "🛒 Create Order Endpoint",
					"description": "Create a new order from cart items",
					"method": "POST",
					"headers": gin.H{
						"Authorization": "Bearer <jwt_token>",
					},
					"body": gin.H{
						"items": "array (required, cart items)",
						"shipping_address": "object (required)",
						"payment_method": "string (required)",
					},
				})
			})
			orders.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "📋 User Orders Endpoint",
					"description": "Get user's order history",
					"method": "GET",
					"headers": gin.H{
						"Authorization": "Bearer <jwt_token>",
					},
					"query_params": gin.H{
						"status": "string (optional, filter by status)",
						"page":   "int (optional, default: 1)",
						"limit":  "int (optional, default: 10)",
					},
				})
			})
			orders.GET("/:id", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "📋 Order Details Endpoint",
					"description": "Get detailed order information",
					"method": "GET",
					"headers": gin.H{
						"Authorization": "Bearer <jwt_token>",
					},
					"params": gin.H{
						"id": "string (required, order UUID)",
					},
				})
			})
		}

		// Payments endpoints
		payments := v1.Group("/payments")
		{
			payments.POST("/webhook", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "💳 Midtrans Payment Webhook",
					"description": "Handle payment notifications from Midtrans",
					"method": "POST",
					"note": "This endpoint is called by Midtrans payment gateway",
				})
			})
		}

		// Admin endpoints (protected)
		admin := v1.Group("/admin")
		{
			admin.GET("/dashboard", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "📊 Admin Dashboard Endpoint",
					"description": "Get admin dashboard statistics",
					"method": "GET",
					"headers": gin.H{
						"Authorization": "Bearer <admin_jwt_token>",
					},
				})
			})
			admin.POST("/products", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "➕ Create Product Endpoint",
					"description": "Create a new product (admin only)",
					"method": "POST",
					"headers": gin.H{
						"Authorization": "Bearer <admin_jwt_token>",
					},
				})
			})
		}
	}

	// Start server
	address := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Info("🌐 Server starting on ", address)
	log.Info("📖 API Documentation: http://", address)
	log.Info("❤️  Health Check: http://", address, "/health")
	
	if err := router.Run(address); err != nil {
		log.Fatal("❌ Failed to start server: ", err)
	}
}