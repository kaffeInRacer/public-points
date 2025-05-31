package http

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/sirupsen/logrus"

	"online-shop/internal/interfaces/http/handlers"
	"online-shop/internal/interfaces/http/middleware"
	"online-shop/pkg/config"
)

// Router represents the HTTP router
type Router struct {
	engine      *gin.Engine
	config      *config.Config
	logger      *logrus.Logger
	userHandler *handlers.UserHandler
	productHandler *handlers.ProductHandler
	orderHandler *handlers.OrderHandler
	authMiddleware *middleware.AuthMiddleware
}

// NewRouter creates a new HTTP router
func NewRouter(
	cfg *config.Config,
	logger *logrus.Logger,
	userHandler *handlers.UserHandler,
	productHandler *handlers.ProductHandler,
	orderHandler *handlers.OrderHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()

	return &Router{
		engine:         engine,
		config:         cfg,
		logger:         logger,
		userHandler:    userHandler,
		productHandler: productHandler,
		orderHandler:   orderHandler,
		authMiddleware: authMiddleware,
	}
}

// SetupRoutes configures all routes
func (r *Router) SetupRoutes() {
	// Global middleware
	r.setupGlobalMiddleware()

	// Health check routes
	r.setupHealthRoutes()

	// Metrics and monitoring routes
	r.setupMonitoringRoutes()

	// API routes
	r.setupAPIRoutes()

	// Admin routes
	r.setupAdminRoutes()

	// Documentation routes
	r.setupDocumentationRoutes()
}

// setupGlobalMiddleware configures global middleware
func (r *Router) setupGlobalMiddleware() {
	// Recovery middleware
	r.engine.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		r.logger.WithField("error", recovered).Error("Panic recovered")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	}))

	// Request logging middleware
	r.engine.Use(middleware.RequestLogger(r.logger))

	// CORS middleware
	r.engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Configure based on your needs
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting middleware
	r.engine.Use(middleware.RateLimit())

	// Security headers middleware
	r.engine.Use(middleware.SecurityHeaders())

	// Request ID middleware
	r.engine.Use(middleware.RequestID())

	// Metrics middleware
	r.engine.Use(middleware.PrometheusMetrics())
}

// setupHealthRoutes configures health check routes
func (r *Router) setupHealthRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	r.engine.GET("/health/ready", func(c *gin.Context) {
		// Add readiness checks here (database, redis, etc.)
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})

	r.engine.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
		})
	})
}

// setupMonitoringRoutes configures monitoring and metrics routes
func (r *Router) setupMonitoringRoutes() {
	// Prometheus metrics endpoint
	r.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// pprof endpoints for profiling (only in development)
	if r.config.Environment != "production" {
		pprof.Register(r.engine)
	}
}

// setupAPIRoutes configures API routes
func (r *Router) setupAPIRoutes() {
	api := r.engine.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			// Public routes (no authentication required)
			r.setupPublicRoutes(v1)

			// Protected routes (authentication required)
			r.setupProtectedRoutes(v1)
		}
	}
}

// setupPublicRoutes configures public API routes
func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	// Authentication routes
	auth := rg.Group("/auth")
	{
		auth.POST("/register", r.userHandler.Register)
		auth.POST("/login", r.userHandler.Login)
		auth.POST("/refresh", r.userHandler.RefreshToken)
		auth.POST("/forgot-password", r.userHandler.ForgotPassword)
		auth.POST("/reset-password", r.userHandler.ResetPassword)
		auth.GET("/verify-email/:token", r.userHandler.VerifyEmail)
	}

	// Public product routes
	products := rg.Group("/products")
	{
		products.GET("", r.productHandler.GetProducts)
		products.GET("/:id", r.productHandler.GetProduct)
		products.GET("/search", r.productHandler.SearchProducts)
		products.GET("/categories", r.productHandler.GetCategories)
		products.GET("/category/:slug", r.productHandler.GetProductsByCategory)
		products.GET("/:id/reviews", r.productHandler.GetProductReviews)
		products.GET("/featured", r.productHandler.GetFeaturedProducts)
		products.GET("/trending", r.productHandler.GetTrendingProducts)
	}

	// Public category routes
	categories := rg.Group("/categories")
	{
		categories.GET("", r.productHandler.GetCategories)
		categories.GET("/:slug", r.productHandler.GetCategory)
	}
}

// setupProtectedRoutes configures protected API routes
func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	protected := rg.Group("")
	protected.Use(r.authMiddleware.RequireAuth())

	// User profile routes
	user := protected.Group("/user")
	{
		user.GET("/profile", r.userHandler.GetProfile)
		user.PUT("/profile", r.userHandler.UpdateProfile)
		user.POST("/change-password", r.userHandler.ChangePassword)
		user.POST("/logout", r.userHandler.Logout)
		user.DELETE("/account", r.userHandler.DeleteAccount)

		// User addresses
		addresses := user.Group("/addresses")
		{
			addresses.GET("", r.userHandler.GetAddresses)
			addresses.POST("", r.userHandler.CreateAddress)
			addresses.PUT("/:id", r.userHandler.UpdateAddress)
			addresses.DELETE("/:id", r.userHandler.DeleteAddress)
			addresses.POST("/:id/default", r.userHandler.SetDefaultAddress)
		}

		// User orders
		orders := user.Group("/orders")
		{
			orders.GET("", r.orderHandler.GetUserOrders)
			orders.GET("/:id", r.orderHandler.GetOrder)
			orders.POST("/:id/cancel", r.orderHandler.CancelOrder)
		}

		// User wishlist
		wishlist := user.Group("/wishlist")
		{
			wishlist.GET("", r.userHandler.GetWishlist)
			wishlist.POST("/:productId", r.userHandler.AddToWishlist)
			wishlist.DELETE("/:productId", r.userHandler.RemoveFromWishlist)
		}
	}

	// Cart routes
	cart := protected.Group("/cart")
	{
		cart.GET("", r.orderHandler.GetCart)
		cart.POST("/items", r.orderHandler.AddToCart)
		cart.PUT("/items/:id", r.orderHandler.UpdateCartItem)
		cart.DELETE("/items/:id", r.orderHandler.RemoveFromCart)
		cart.DELETE("", r.orderHandler.ClearCart)
	}

	// Order routes
	orders := protected.Group("/orders")
	{
		orders.POST("", r.orderHandler.CreateOrder)
		orders.GET("/:id", r.orderHandler.GetOrder)
		orders.POST("/:id/payment", r.orderHandler.ProcessPayment)
		orders.GET("/:id/invoice", r.orderHandler.GetInvoice)
	}

	// Review routes
	reviews := protected.Group("/reviews")
	{
		reviews.POST("", r.productHandler.CreateReview)
		reviews.PUT("/:id", r.productHandler.UpdateReview)
		reviews.DELETE("/:id", r.productHandler.DeleteReview)
		reviews.POST("/:id/helpful", r.productHandler.MarkReviewHelpful)
	}
}

// setupAdminRoutes configures admin routes
func (r *Router) setupAdminRoutes() {
	admin := r.engine.Group("/admin")
	admin.Use(r.authMiddleware.RequireAuth())
	admin.Use(r.authMiddleware.RequireRole("admin"))

	// Admin dashboard
	admin.GET("/dashboard", r.getDashboardStats)

	// Admin user management
	users := admin.Group("/users")
	{
		users.GET("", r.userHandler.GetUsers)
		users.GET("/:id", r.userHandler.GetUserByID)
		users.PUT("/:id", r.userHandler.UpdateUser)
		users.DELETE("/:id", r.userHandler.DeleteUser)
		users.POST("/:id/suspend", r.userHandler.SuspendUser)
		users.POST("/:id/activate", r.userHandler.ActivateUser)
	}

	// Admin product management
	products := admin.Group("/products")
	{
		products.POST("", r.productHandler.CreateProduct)
		products.PUT("/:id", r.productHandler.UpdateProduct)
		products.DELETE("/:id", r.productHandler.DeleteProduct)
		products.POST("/:id/activate", r.productHandler.ActivateProduct)
		products.POST("/:id/deactivate", r.productHandler.DeactivateProduct)
		products.GET("/:id/inventory", r.productHandler.GetInventoryMovements)
		products.POST("/:id/inventory", r.productHandler.AdjustInventory)
	}

	// Admin category management
	categories := admin.Group("/categories")
	{
		categories.POST("", r.productHandler.CreateCategory)
		categories.PUT("/:id", r.productHandler.UpdateCategory)
		categories.DELETE("/:id", r.productHandler.DeleteCategory)
	}

	// Admin order management
	orders := admin.Group("/orders")
	{
		orders.GET("", r.orderHandler.GetOrders)
		orders.GET("/:id", r.orderHandler.GetOrder)
		orders.PUT("/:id/status", r.orderHandler.UpdateOrderStatus)
		orders.POST("/:id/ship", r.orderHandler.ShipOrder)
		orders.POST("/:id/refund", r.orderHandler.RefundOrder)
	}

	// Admin review management
	reviews := admin.Group("/reviews")
	{
		reviews.GET("", r.productHandler.GetReviews)
		reviews.POST("/:id/approve", r.productHandler.ApproveReview)
		reviews.POST("/:id/reject", r.productHandler.RejectReview)
	}

	// Admin analytics
	analytics := admin.Group("/analytics")
	{
		analytics.GET("/sales", r.getSalesAnalytics)
		analytics.GET("/products", r.getProductAnalytics)
		analytics.GET("/users", r.getUserAnalytics)
		analytics.GET("/revenue", r.getRevenueAnalytics)
	}

	// Admin system management
	system := admin.Group("/system")
	{
		system.GET("/logs", r.getSystemLogs)
		system.POST("/backup", r.triggerBackup)
		system.GET("/health", r.getSystemHealth)
		system.POST("/cache/clear", r.clearCache)
	}
}

// setupDocumentationRoutes configures documentation routes
func (r *Router) setupDocumentationRoutes() {
	// Swagger documentation
	r.engine.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API documentation
	r.engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/docs/index.html")
	})
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// Dashboard and analytics handlers
func (r *Router) getDashboardStats(c *gin.Context) {
	// Implementation for dashboard statistics
	c.JSON(http.StatusOK, gin.H{
		"message": "Dashboard stats endpoint",
	})
}

func (r *Router) getSalesAnalytics(c *gin.Context) {
	// Implementation for sales analytics
	c.JSON(http.StatusOK, gin.H{
		"message": "Sales analytics endpoint",
	})
}

func (r *Router) getProductAnalytics(c *gin.Context) {
	// Implementation for product analytics
	c.JSON(http.StatusOK, gin.H{
		"message": "Product analytics endpoint",
	})
}

func (r *Router) getUserAnalytics(c *gin.Context) {
	// Implementation for user analytics
	c.JSON(http.StatusOK, gin.H{
		"message": "User analytics endpoint",
	})
}

func (r *Router) getRevenueAnalytics(c *gin.Context) {
	// Implementation for revenue analytics
	c.JSON(http.StatusOK, gin.H{
		"message": "Revenue analytics endpoint",
	})
}

func (r *Router) getSystemLogs(c *gin.Context) {
	// Implementation for system logs
	c.JSON(http.StatusOK, gin.H{
		"message": "System logs endpoint",
	})
}

func (r *Router) triggerBackup(c *gin.Context) {
	// Implementation for triggering backup
	c.JSON(http.StatusOK, gin.H{
		"message": "Backup triggered",
	})
}

func (r *Router) getSystemHealth(c *gin.Context) {
	// Implementation for system health check
	c.JSON(http.StatusOK, gin.H{
		"message": "System health endpoint",
	})
}

func (r *Router) clearCache(c *gin.Context) {
	// Implementation for clearing cache
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache cleared",
	})
}