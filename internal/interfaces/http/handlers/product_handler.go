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