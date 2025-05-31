package grpc

import (
	"context"
	"fmt"
	"time"

	productDomain "online-shop/internal/domain/product"
	"online-shop/internal/infrastructure/redis"
	"online-shop/internal/infrastructure/database"
	"online-shop/internal/infrastructure/elasticsearch"
	pb "online-shop/online-shop/proto/product"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductServiceServer struct {
	pb.UnimplementedProductServiceServer
	productRepo   *database.ProductRepository
	categoryRepo  *database.CategoryRepository
	cacheClient   *redis.RedisClient
	searchClient  *elasticsearch.SearchService
	logger        *zap.Logger
}

func NewProductServiceServer(
	productRepo *database.ProductRepository,
	categoryRepo *database.CategoryRepository,
	cacheClient *redis.RedisClient,
	searchClient *elasticsearch.SearchService,
	logger *zap.Logger,
) *ProductServiceServer {
	return &ProductServiceServer{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		cacheClient:  cacheClient,
		searchClient: searchClient,
		logger:       logger,
	}
}

func (s *ProductServiceServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	s.logger.Info("Create product request", zap.String("name", req.Name), zap.String("merchant_id", req.MerchantId))

	// Validate input
	if req.Name == "" || req.Price <= 0 || req.CategoryId == "" {
		return nil, status.Error(codes.InvalidArgument, "Name, price, and category ID are required")
	}

	// Check if category exists
	category, err := s.categoryRepo.GetByID(req.CategoryId)
	if err != nil || category == nil {
		return nil, status.Error(codes.NotFound, "Category not found")
	}

	// Create product entity
	productEntity := &productDomain.Product{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       int(req.Stock),
		CategoryID:  req.CategoryId,
		MerchantID:  req.MerchantId,
		Images:      req.Images,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to database
	if err := s.productRepo.Create(productEntity); err != nil {
		s.logger.Error("Failed to create product", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to create product")
	}

	// Index in Elasticsearch
	if err := s.searchClient.IndexProduct(ctx, productEntity); err != nil {
		s.logger.Warn("Failed to index product in Elasticsearch", zap.Error(err))
	}

	// Cache product
	productKey := fmt.Sprintf("product:%s", productEntity.ID)
	if err := s.cacheClient.Set(productKey, productEntity, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to cache product", zap.Error(err))
	}

	// Invalidate products list cache
	s.invalidateProductsCache()

	s.logger.Info("Product created successfully", zap.String("product_id", productEntity.ID), zap.String("name", productEntity.Name))

	return &pb.CreateProductResponse{
		Product: s.entityToProto(productEntity, category),
	}, nil
}

func (s *ProductServiceServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	s.logger.Info("Get product request", zap.String("product_id", req.ProductId))

	// Try to get from cache first
	productKey := fmt.Sprintf("product:%s", req.ProductId)
	var productEntity *productDomain.Product
	
	if err := s.cacheClient.Get(productKey, &productEntity); err != nil {
		// Cache miss, get from database
		var err error
		productEntity, err = s.productRepo.GetByID(req.ProductId)
		if err != nil || productEntity == nil {
			return nil, status.Error(codes.NotFound, "Product not found")
		}

		// Cache the product
		if err := s.cacheClient.Set(productKey, productEntity, 24*time.Hour); err != nil {
			s.logger.Warn("Failed to cache product", zap.Error(err))
		}
	}

	// Get category
	category, err := s.categoryRepo.GetByID(productEntity.CategoryID)
	if err != nil {
		s.logger.Warn("Failed to get category", zap.String("category_id", productEntity.CategoryID), zap.Error(err))
	}

	return &pb.GetProductResponse{
		Product: s.entityToProto(productEntity, category),
	}, nil
}

func (s *ProductServiceServer) GetProducts(ctx context.Context, req *pb.GetProductsRequest) (*pb.GetProductsResponse, error) {
	s.logger.Info("Get products request", zap.Int32("limit", req.Limit), zap.Int32("offset", req.Offset))

	// Set default values
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("products:list:%d:%d:%s:%s", limit, offset, req.SortBy, req.SortOrder)
	var cachedResult struct {
		Products []*productDomain.Product `json:"products"`
		Total    int64             `json:"total"`
	}

	if err := s.cacheClient.Get(cacheKey, &cachedResult); err == nil {
		// Cache hit
		protoProducts := make([]*pb.Product, len(cachedResult.Products))
		for i, product := range cachedResult.Products {
			category, _ := s.categoryRepo.GetByID(product.CategoryID)
			protoProducts[i] = s.entityToProto(product, category)
		}

		return &pb.GetProductsResponse{
			Products: protoProducts,
			Total:    cachedResult.Total,
		}, nil
	}

	// Cache miss, get from database
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	filter := productDomain.SearchFilter{
		Limit:  limit,
		Offset: offset,
	}
	
	products, err := s.productRepo.List(filter)
	if err != nil {
		s.logger.Error("Failed to get products", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to get products")
	}
	
	total := int64(len(products)) // For simplicity, using length as total

	// Cache the result
	cachedResult.Products = products
	cachedResult.Total = int64(total)
	if err := s.cacheClient.Set(cacheKey, cachedResult, 10*time.Minute); err != nil {
		s.logger.Warn("Failed to cache products list", zap.Error(err))
	}

	// Convert to proto
	protoProducts := make([]*pb.Product, len(products))
	for i, product := range products {
		category, _ := s.categoryRepo.GetByID(product.CategoryID)
		protoProducts[i] = s.entityToProto(product, category)
	}

	return &pb.GetProductsResponse{
		Products: protoProducts,
		Total:    int64(cachedResult.Total),
	}, nil
}

func (s *ProductServiceServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	s.logger.Info("Update product request", zap.String("product_id", req.ProductId))

	// Get product from database
	product, err := s.productRepo.GetByID(req.ProductId)
	if err != nil || product == nil {
		return nil, status.Error(codes.NotFound, "Product not found")
	}

	// Update fields
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Stock >= 0 {
		product.Stock = int(req.Stock)
	}
	if len(req.Images) > 0 {
		product.Images = req.Images
	}
	product.UpdatedAt = time.Now()

	// Save to database
	if err := s.productRepo.Update(product); err != nil {
		s.logger.Error("Failed to update product", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to update product")
	}

	// Update in Elasticsearch
	if err := s.searchClient.IndexProduct(ctx, product); err != nil {
		s.logger.Warn("Failed to update product in Elasticsearch", zap.Error(err))
	}

	// Update cache
	productKey := fmt.Sprintf("product:%s", product.ID)
	if err := s.cacheClient.Set(productKey, product, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to update product cache", zap.Error(err))
	}

	// Invalidate products list cache
	s.invalidateProductsCache()

	// Get category
	category, err := s.categoryRepo.GetByID(product.CategoryID)
	if err != nil {
		s.logger.Warn("Failed to get category", zap.String("category_id", product.CategoryID), zap.Error(err))
	}

	s.logger.Info("Product updated successfully", zap.String("product_id", product.ID))

	return &pb.UpdateProductResponse{
		Product: s.entityToProto(product, category),
	}, nil
}

func (s *ProductServiceServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	s.logger.Info("Delete product request", zap.String("product_id", req.ProductId))

	// Check if product exists
	product, err := s.productRepo.GetByID(req.ProductId)
	if err != nil || product == nil {
		return nil, status.Error(codes.NotFound, "Product not found")
	}

	// Delete from database
	if err := s.productRepo.Delete(req.ProductId); err != nil {
		s.logger.Error("Failed to delete product", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to delete product")
	}

	// Delete from Elasticsearch
	if err := s.searchClient.DeleteProduct(ctx, req.ProductId); err != nil {
		s.logger.Warn("Failed to delete product from Elasticsearch", zap.Error(err))
	}

	// Delete from cache
	productKey := fmt.Sprintf("product:%s", req.ProductId)
	if err := s.cacheClient.Delete(productKey); err != nil {
		s.logger.Warn("Failed to delete product from cache", zap.Error(err))
	}

	// Invalidate products list cache
	s.invalidateProductsCache()

	s.logger.Info("Product deleted successfully", zap.String("product_id", req.ProductId))

	return &pb.DeleteProductResponse{
		Success: true,
	}, nil
}

func (s *ProductServiceServer) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	s.logger.Info("Search products request", zap.String("query", req.Query))

	// Set default values
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// Try cache first for search results
	cacheKey := fmt.Sprintf("search:%s:%s:%f:%f:%s:%d:%d", 
		req.Query, req.CategoryId, req.MinPrice, req.MaxPrice, req.MerchantId, limit, offset)
	
	var cachedResult struct {
		Products []*productDomain.Product `json:"products"`
		Total    int64             `json:"total"`
	}

	if err := s.cacheClient.Get(cacheKey, &cachedResult); err == nil {
		// Cache hit
		protoProducts := make([]*pb.Product, len(cachedResult.Products))
		for i, product := range cachedResult.Products {
			category, _ := s.categoryRepo.GetByID(product.CategoryID)
			protoProducts[i] = s.entityToProto(product, category)
		}

		return &pb.SearchProductsResponse{
			Products: protoProducts,
			Total:    cachedResult.Total,
		}, nil
	}

	// Search in Elasticsearch
	searchQuery := elasticsearch.SearchQuery{
		Query:      req.Query,
		CategoryID: req.CategoryId,
		MinPrice:   req.MinPrice,
		MaxPrice:   req.MaxPrice,
		MerchantID: req.MerchantId,
		From:       offset,
		Size:       limit,
	}

	searchResult, err := s.searchClient.SearchProducts(ctx, searchQuery)
	if err != nil {
		s.logger.Error("Failed to search products", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to search products")
	}

	// Convert ProductDocuments to Product entities for caching
	products := make([]*productDomain.Product, len(searchResult.Products))
	for i, doc := range searchResult.Products {
		products[i] = &productDomain.Product{
			ID:          doc.ID,
			Name:        doc.Name,
			Description: doc.Description,
			Price:       doc.Price,
			CategoryID:  doc.CategoryID,
			MerchantID:  doc.MerchantID,
			Images:      doc.Images,
			Status:      productDomain.Status(doc.Status),
		}
	}

	// Cache the search result
	cachedResult.Products = products
	cachedResult.Total = searchResult.Total
	if err := s.cacheClient.Set(cacheKey, cachedResult, 5*time.Minute); err != nil {
		s.logger.Warn("Failed to cache search results", zap.Error(err))
	}

	// Convert to proto
	protoProducts := make([]*pb.Product, len(products))
	for i, product := range products {
		category, _ := s.categoryRepo.GetByID(product.CategoryID)
		protoProducts[i] = s.entityToProto(product, category)
	}

	return &pb.SearchProductsResponse{
		Products: protoProducts,
		Total:    int64(cachedResult.Total),
	}, nil
}

func (s *ProductServiceServer) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	s.logger.Info("List categories request")

	// Set default values
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// Try cache first
	cacheKey := fmt.Sprintf("categories:list:%d:%d", limit, offset)
	var categories []*productDomain.Category

	if err := s.cacheClient.Get(cacheKey, &categories); err != nil {
		// Cache miss, get from database
		var err error
		categories, err = s.categoryRepo.List(limit, offset)
		if err != nil {
			s.logger.Error("Failed to get categories", zap.Error(err))
			return nil, status.Error(codes.Internal, "Failed to get categories")
		}

		// Cache the result
		if err := s.cacheClient.Set(cacheKey, categories, 30*time.Minute); err != nil {
			s.logger.Warn("Failed to cache categories", zap.Error(err))
		}
	}

	// Convert to proto
	protoCategories := make([]*pb.Category, len(categories))
	for i, category := range categories {
		protoCategories[i] = s.categoryEntityToProto(category)
	}

	return &pb.ListCategoriesResponse{
		Categories: protoCategories,
	}, nil
}

func (s *ProductServiceServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
	s.logger.Info("Update stock request", zap.String("product_id", req.ProductId), zap.Int32("stock", req.Stock))

	// Get product from database
	product, err := s.productRepo.GetByID(req.ProductId)
	if err != nil || product == nil {
		return &pb.UpdateStockResponse{
			Success: false,
			Message: "Product not found",
		}, nil
	}

	// Update stock
	product.Stock = int(req.Stock)
	product.UpdatedAt = time.Now()

	// Save to database
	if err := s.productRepo.Update(product); err != nil {
		s.logger.Error("Failed to update product stock", zap.Error(err))
		return &pb.UpdateStockResponse{
			Success: false,
			Message: "Failed to update stock",
		}, nil
	}

	// Update in Elasticsearch
	if err := s.searchClient.IndexProduct(ctx, product); err != nil {
		s.logger.Warn("Failed to update product in Elasticsearch", zap.Error(err))
	}

	// Update cache
	productKey := fmt.Sprintf("product:%s", product.ID)
	if err := s.cacheClient.Set(productKey, product, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to update product cache", zap.Error(err))
	}

	// Invalidate products list cache
	s.invalidateProductsCache()

	// Get category
	category, err := s.categoryRepo.GetByID(product.CategoryID)
	if err != nil {
		s.logger.Warn("Failed to get category", zap.String("category_id", product.CategoryID), zap.Error(err))
	}

	s.logger.Info("Product stock updated successfully", zap.String("product_id", product.ID), zap.Int("new_stock", product.Stock))

	return &pb.UpdateStockResponse{
		Success: true,
		Message: "Stock updated successfully",
		Product: s.entityToProto(product, category),
	}, nil
}

func (s *ProductServiceServer) GetProductsByCategory(ctx context.Context, req *pb.GetProductsByCategoryRequest) (*pb.GetProductsByCategoryResponse, error) {
	s.logger.Info("Get products by category request", zap.String("category_id", req.CategoryId))

	// Set default values
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// Try cache first
	cacheKey := fmt.Sprintf("products:category:%s:%d:%d", req.CategoryId, limit, offset)
	var cachedResult struct {
		Products []*productDomain.Product `json:"products"`
		Total    int64             `json:"total"`
	}

	if err := s.cacheClient.Get(cacheKey, &cachedResult); err == nil {
		// Cache hit
		protoProducts := make([]*pb.Product, len(cachedResult.Products))
		for i, product := range cachedResult.Products {
			category, _ := s.categoryRepo.GetByID(product.CategoryID)
			protoProducts[i] = s.entityToProto(product, category)
		}

		return &pb.GetProductsByCategoryResponse{
			Products: protoProducts,
			Total:    cachedResult.Total,
		}, nil
	}

	// Get from database
	filter := productDomain.SearchFilter{
		CategoryID: req.CategoryId,
		Limit:      limit,
		Offset:     offset,
	}
	products, err := s.productRepo.List(filter)
	if err != nil {
		s.logger.Error("Failed to get products by category", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to get products by category")
	}

	// Cache the result
	cachedResult.Products = products
	cachedResult.Total = int64(len(products))
	if err := s.cacheClient.Set(cacheKey, cachedResult, 15*time.Minute); err != nil {
		s.logger.Warn("Failed to cache products by category", zap.Error(err))
	}

	// Convert to proto
	protoProducts := make([]*pb.Product, len(products))
	for i, product := range products {
		category, _ := s.categoryRepo.GetByID(product.CategoryID)
		protoProducts[i] = s.entityToProto(product, category)
	}

	return &pb.GetProductsByCategoryResponse{
		Products: protoProducts,
		Total:    cachedResult.Total,
	}, nil
}

func (s *ProductServiceServer) entityToProto(product *productDomain.Product, category *productDomain.Category) *pb.Product {
	protoProduct := &pb.Product{
		Id:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       int32(product.Stock),
		CategoryId:  product.CategoryID,
		MerchantId:  product.MerchantID,
		Images:      product.Images,
		Status:      string(product.Status),
		CreatedAt:   timestamppb.New(product.CreatedAt),
		UpdatedAt:   timestamppb.New(product.UpdatedAt),
	}

	if category != nil {
		protoProduct.Category = s.categoryEntityToProto(category)
	}

	return protoProduct
}

func (s *ProductServiceServer) categoryEntityToProto(category *productDomain.Category) *pb.Category {
	return &pb.Category{
		Id:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		ParentId:    func() string { if category.ParentID != nil { return *category.ParentID }; return "" }(),
		CreatedAt:   timestamppb.New(category.CreatedAt),
		UpdatedAt:   timestamppb.New(category.UpdatedAt),
	}
}

func (s *ProductServiceServer) invalidateProductsCache() {
	// Delete all products list cache entries
	pattern := "products:list:*"
	if err := s.cacheClient.DeletePattern(pattern); err != nil {
		s.logger.Warn("Failed to invalidate products cache", zap.Error(err))
	}

	// Delete products by category cache entries
	pattern = "products:category:*"
	if err := s.cacheClient.DeletePattern(pattern); err != nil {
		s.logger.Warn("Failed to invalidate products by category cache", zap.Error(err))
	}
}