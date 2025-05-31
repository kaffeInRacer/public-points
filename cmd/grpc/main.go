package main

import (
	"fmt"
	"log"
	"net"
	"online-shop/internal/infrastructure/redis"
	"online-shop/internal/infrastructure/database"
	grpcServices "online-shop/internal/infrastructure/grpc"
	"online-shop/internal/infrastructure/payment"
	"online-shop/internal/infrastructure/elasticsearch"
	userPb "online-shop/online-shop/proto/user"
	productPb "online-shop/online-shop/proto/product"
	orderPb "online-shop/online-shop/proto/order"
	"online-shop/pkg/config"
	"online-shop/pkg/jwt"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Initialize logger
	logr, _ := zap.NewProduction()
	defer logr.Sync()
	logr.Info("Starting Online Shop gRPC Server...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logr.Error("Failed to load config", zap.Error(err))
		// Use default config
		cfg = &config.Config{
			GRPC: config.GRPCConfig{
				Host: "0.0.0.0",
				Port: "12001",
			},
			Database: config.DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "online_shop",
				SSLMode:  "disable",
			},
			Redis: config.RedisConfig{
				Host:     "localhost",
				Port:     "6379",
				Password: "",
				DB:       0,
			},
			Elasticsearch: config.ElasticsearchConfig{
				URL: "http://localhost:9200",
			},
			JWT: config.JWTConfig{
				SecretKey:   "your-secret-key",
				ExpiryHours: 24,
			},
			Midtrans: config.MidtransConfig{
				ServerKey:   "your-server-key",
				ClientKey:   "your-client-key",
				Environment: "sandbox",
			},
		}
	}

	// Initialize database connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logr.Error("Failed to connect to database", zap.Error(err))
		// Continue without database for now
	}

	// Initialize Redis client
	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	redisClient := redis.NewRedisClient(redisAddr, logr)

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewClient(&cfg.Elasticsearch)
	if err != nil {
		logr.Error("Failed to connect to Elasticsearch", zap.Error(err))
		// Continue without Elasticsearch for now
	}
	
	// Initialize search service
	var searchService *elasticsearch.SearchService
	if esClient != nil {
		searchService = elasticsearch.NewSearchService(esClient)
	}

	// Initialize JWT service
	jwtService := jwt.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.ExpiryHours)

	// Initialize payment provider
	paymentProvider := payment.NewMidtransProvider(&cfg.Midtrans)

	// Initialize repositories (only if database is available)
	var userRepo *database.UserRepository
	var productRepo *database.ProductRepository
	var categoryRepo *database.CategoryRepository
	var orderRepo *database.OrderRepository
	var paymentRepo *database.PaymentRepository

	if db != nil {
		userRepo = database.NewUserRepository(db).(*database.UserRepository)
		productRepo = database.NewProductRepository(db).(*database.ProductRepository)
		categoryRepo = database.NewCategoryRepository(db).(*database.CategoryRepository)
		orderRepo = database.NewOrderRepository(db).(*database.OrderRepository)
		paymentRepo = database.NewPaymentRepository(db).(*database.PaymentRepository)
	}

	// Create gRPC server
	server := grpc.NewServer()

	// Initialize and register gRPC services
	if userRepo != nil {
		userService := grpcServices.NewUserServiceServer(userRepo, redisClient, jwtService, logr)
		userPb.RegisterUserServiceServer(server, userService)
		logr.Info("UserService registered")
	}

	if productRepo != nil && categoryRepo != nil {
		productService := grpcServices.NewProductServiceServer(productRepo, categoryRepo, redisClient, searchService, logr)
		productPb.RegisterProductServiceServer(server, productService)
		logr.Info("ProductService registered")
	}

	if orderRepo != nil && productRepo != nil && userRepo != nil && paymentRepo != nil {
		orderService := grpcServices.NewOrderServiceServer(orderRepo, productRepo, userRepo, paymentRepo, redisClient, paymentProvider, logr)
		orderPb.RegisterOrderServiceServer(server, orderService)
		logr.Info("OrderService registered")
	}

	// Register reflection service for debugging
	reflection.Register(server)

	// Start listening
	address := fmt.Sprintf("%s:%s", cfg.GRPC.Host, cfg.GRPC.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	logr.Info("gRPC Server starting on", zap.String("address", address))
	logr.Info("gRPC reflection enabled for debugging")

	// Start server
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}