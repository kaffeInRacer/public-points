package handlers

import (
	"net/http"
	"online-shop/internal/application/queries"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	getProductHandler     *queries.GetProductQueryHandler
	searchProductsHandler *queries.SearchProductsQueryHandler
	listCategoriesHandler *queries.ListCategoriesQueryHandler
}

func NewProductHandler(
	getProductHandler *queries.GetProductQueryHandler,
	searchProductsHandler *queries.SearchProductsQueryHandler,
	listCategoriesHandler *queries.ListCategoriesQueryHandler,
) *ProductHandler {
	return &ProductHandler{
		getProductHandler:     getProductHandler,
		searchProductsHandler: searchProductsHandler,
		listCategoriesHandler: listCategoriesHandler,
	}
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	query := queries.GetProductQuery{ProductID: productID}
	product, err := h.getProductHandler.Handle(query)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (h *ProductHandler) SearchProducts(c *gin.Context) {
	query := queries.SearchProductsQuery{
		Query:      c.Query("q"),
		CategoryID: c.Query("category_id"),
		MerchantID: c.Query("merchant_id"),
	}

	if minPrice := c.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			query.MinPrice = price
		}
	}

	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			query.MaxPrice = price
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			query.Offset = o
		}
	}

	products, err := h.searchProductsHandler.Handle(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"total":    len(products),
	})
}

func (h *ProductHandler) ListCategories(c *gin.Context) {
	query := queries.ListCategoriesQuery{}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			query.Offset = o
		}
	}

	categories, err := h.listCategoriesHandler.Handle(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// GetProducts handles getting all products with pagination
func (h *ProductHandler) GetProducts(c *gin.Context) {
	// TODO: Implement get all products logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get products not implemented yet"})
}

// GetCategories handles getting all categories
func (h *ProductHandler) GetCategories(c *gin.Context) {
	// Delegate to ListCategories for now
	h.ListCategories(c)
}

// GetProductsByCategory handles getting products by category
func (h *ProductHandler) GetProductsByCategory(c *gin.Context) {
	// TODO: Implement get products by category logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get products by category not implemented yet"})
}

// GetProductReviews handles getting product reviews
func (h *ProductHandler) GetProductReviews(c *gin.Context) {
	// TODO: Implement get product reviews logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get product reviews not implemented yet"})
}

// GetFeaturedProducts handles getting featured products
func (h *ProductHandler) GetFeaturedProducts(c *gin.Context) {
	// TODO: Implement get featured products logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get featured products not implemented yet"})
}

// GetTrendingProducts handles getting trending products
func (h *ProductHandler) GetTrendingProducts(c *gin.Context) {
	// TODO: Implement get trending products logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get trending products not implemented yet"})
}

// GetCategory handles getting a specific category
func (h *ProductHandler) GetCategory(c *gin.Context) {
	// TODO: Implement get category logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get category not implemented yet"})
}

// CreateReview handles creating a product review
func (h *ProductHandler) CreateReview(c *gin.Context) {
	// TODO: Implement create review logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Create review not implemented yet"})
}

// UpdateReview handles updating a product review
func (h *ProductHandler) UpdateReview(c *gin.Context) {
	// TODO: Implement update review logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update review not implemented yet"})
}

// DeleteReview handles deleting a product review
func (h *ProductHandler) DeleteReview(c *gin.Context) {
	// TODO: Implement delete review logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Delete review not implemented yet"})
}

// GetReviews handles getting all reviews (admin)
func (h *ProductHandler) GetReviews(c *gin.Context) {
	// TODO: Implement get all reviews logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get reviews not implemented yet"})
}

// CreateProduct handles creating a new product (admin)
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	// TODO: Implement create product logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Create product not implemented yet"})
}

// UpdateProduct handles updating a product (admin)
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	// TODO: Implement update product logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update product not implemented yet"})
}

// DeleteProduct handles deleting a product (admin)
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	// TODO: Implement delete product logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Delete product not implemented yet"})
}

// GetInventoryMovements handles getting inventory movements (admin)
func (h *ProductHandler) GetInventoryMovements(c *gin.Context) {
	// TODO: Implement get inventory movements logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get inventory movements not implemented yet"})
}

// CreateCategory handles creating a new category (admin)
func (h *ProductHandler) CreateCategory(c *gin.Context) {
	// TODO: Implement create category logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Create category not implemented yet"})
}

// UpdateCategory handles updating a category (admin)
func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	// TODO: Implement update category logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update category not implemented yet"})
}

// DeleteCategory handles deleting a category (admin)
func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	// TODO: Implement delete category logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Delete category not implemented yet"})
}

// MarkReviewHelpful handles marking a review as helpful
func (h *ProductHandler) MarkReviewHelpful(c *gin.Context) {
	// TODO: Implement mark review helpful logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Mark review helpful not implemented yet"})
}

// ActivateProduct handles activating a product (admin)
func (h *ProductHandler) ActivateProduct(c *gin.Context) {
	// TODO: Implement activate product logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Activate product not implemented yet"})
}

// DeactivateProduct handles deactivating a product (admin)
func (h *ProductHandler) DeactivateProduct(c *gin.Context) {
	// TODO: Implement deactivate product logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Deactivate product not implemented yet"})
}

// AdjustInventory handles adjusting product inventory (admin)
func (h *ProductHandler) AdjustInventory(c *gin.Context) {
	// TODO: Implement adjust inventory logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Adjust inventory not implemented yet"})
}

// ApproveReview handles approving a review (admin)
func (h *ProductHandler) ApproveReview(c *gin.Context) {
	// TODO: Implement approve review logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Approve review not implemented yet"})
}

// RejectReview handles rejecting a review (admin)
func (h *ProductHandler) RejectReview(c *gin.Context) {
	// TODO: Implement reject review logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Reject review not implemented yet"})
}