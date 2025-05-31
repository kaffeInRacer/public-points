package main

import (
	"log"
	"online-shop/internal/application/commands"
	"online-shop/internal/application/queries"
	"online-shop/internal/infrastructure/database"
	"online-shop/internal/infrastructure/elasticsearch"
	"online-shop/internal/infrastructure/payment"
	"online-shop/internal/infrastructure/redis"
	"online-shop/internal/interfaces/http/handlers"
	"online-shop/internal/interfaces/http/middleware"
	"online-shop/pkg/config"
	"online-shop/pkg/jwt"
	"online-shop/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize logger
	logger.Init()
	log := logger.GetLogger()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	// Initialize database
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}

	// Initialize Redis
	redisClient := redis.NewClient(&cfg.Redis)
	cacheService := redis.NewCacheService(redisClient)

	// Initialize Elasticsearch
	esClient, err := elasticsearch.NewClient(&cfg.Elasticsearch)
	if err != nil {
		log.Fatal("Failed to connect to Elasticsearch: ", err)
	}
	searchService := elasticsearch.NewSearchService(esClient)

	// Create Elasticsearch index
	if err := searchService.CreateIndex(nil); err != nil {
		log.Warn("Failed to create Elasticsearch index: ", err)
	}

	// Initialize repositories
	userRepo := database.NewUserRepository(db.DB)
	productRepo := database.NewProductRepository(db.DB)
	categoryRepo := database.NewCategoryRepository(db.DB)
	orderRepo := database.NewOrderRepository(db.DB)
	paymentRepo := database.NewPaymentRepository(db.DB)

	// Initialize payment provider
	midtransProvider := payment.NewMidtransProvider(&cfg.Midtrans)
	paymentService := payment.NewPaymentService(midtransProvider, paymentRepo)

	// Initialize JWT manager
	jwtManager := jwt.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.ExpiryHours)

	// Initialize command handlers
	registerHandler := commands.NewRegisterUserCommandHandler(userRepo)
	loginHandler := commands.NewLoginUserCommandHandler(userRepo)
	updateProfileHandler := commands.NewUpdateUserProfileCommandHandler(userRepo)
	changePasswordHandler := commands.NewChangePasswordCommandHandler(userRepo)
	createOrderHandler := commands.NewCreateOrderCommandHandler(orderRepo, productRepo)
	cancelOrderHandler := commands.NewCancelOrderCommandHandler(orderRepo, productRepo)

	// Initialize query handlers
	getUserProfileHandler := queries.NewGetUserProfileQueryHandler(userRepo)
	getProductHandler := queries.NewGetProductQueryHandler(productRepo)
	searchProductsHandler := queries.NewSearchProductsQueryHandler(productRepo)
	listCategoriesHandler := queries.NewListCategoriesQueryHandler(categoryRepo)
	getOrderHandler := queries.NewGetOrderQueryHandler(orderRepo)
	getUserOrdersHandler := queries.NewGetUserOrdersQueryHandler(orderRepo)

	// Initialize HTTP handlers
	userHandler := handlers.NewUserHandler(
		registerHandler,
		loginHandler,
		updateProfileHandler,
		changePasswordHandler,
		getUserProfileHandler,
		jwtManager,
	)

	productHandler := handlers.NewProductHandler(
		getProductHandler,
		searchProductsHandler,
		listCategoriesHandler,
	)

	orderHandler := handlers.NewOrderHandler(
		createOrderHandler,
		cancelOrderHandler,
		getOrderHandler,
		getUserOrdersHandler,
	)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Setup Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")

	// User routes
	users := api.Group("/users")
	{
		users.POST("/register", userHandler.Register)
		users.POST("/login", userHandler.Login)
		users.GET("/profile", authMiddleware.RequireAuth(), userHandler.GetProfile)
		users.PUT("/profile", authMiddleware.RequireAuth(), userHandler.UpdateProfile)
		users.PUT("/password", authMiddleware.RequireAuth(), userHandler.ChangePassword)
	}

	// Product routes
	products := api.Group("/products")
	{
		products.GET("/search", productHandler.SearchProducts)
		products.GET("/:id", productHandler.GetProduct)
		products.GET("/categories", productHandler.ListCategories)
	}

	// Order routes
	orders := api.Group("/orders")
	orders.Use(authMiddleware.RequireAuth())
	{
		orders.POST("", orderHandler.CreateOrder)
		orders.GET("", orderHandler.GetUserOrders)
		orders.GET("/:id", orderHandler.GetOrder)
		orders.PUT("/:id/cancel", orderHandler.CancelOrder)
	}

	// Payment webhook (no auth required)
	api.POST("/payments/webhook", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if err := paymentService.HandleWebhook(data); err != nil {
			log.Error("Payment webhook error: ", err)
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Info("Starting server on ", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}