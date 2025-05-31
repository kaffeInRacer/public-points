package queries

import (
	"online-shop/internal/domain/product"
)

type GetProductQuery struct {
	ProductID string `json:"product_id" validate:"required"`
}

type SearchProductsQuery struct {
	Query      string  `json:"query"`
	CategoryID string  `json:"category_id"`
	MinPrice   float64 `json:"min_price"`
	MaxPrice   float64 `json:"max_price"`
	MerchantID string  `json:"merchant_id"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
}

type ListCategoriesQuery struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type GetProductQueryHandler struct {
	productRepo product.Repository
}

func NewGetProductQueryHandler(productRepo product.Repository) *GetProductQueryHandler {
	return &GetProductQueryHandler{productRepo: productRepo}
}

func (h *GetProductQueryHandler) Handle(query GetProductQuery) (*product.Product, error) {
	return h.productRepo.GetByID(query.ProductID)
}

type SearchProductsQueryHandler struct {
	productRepo product.Repository
}

func NewSearchProductsQueryHandler(productRepo product.Repository) *SearchProductsQueryHandler {
	return &SearchProductsQueryHandler{productRepo: productRepo}
}

func (h *SearchProductsQueryHandler) Handle(query SearchProductsQuery) ([]*product.Product, error) {
	if query.Limit <= 0 {
		query.Limit = 20
	}

	filter := product.SearchFilter{
		Query:      query.Query,
		CategoryID: query.CategoryID,
		MinPrice:   query.MinPrice,
		MaxPrice:   query.MaxPrice,
		MerchantID: query.MerchantID,
		Status:     product.StatusActive,
		Limit:      query.Limit,
		Offset:     query.Offset,
	}

	return h.productRepo.List(filter)
}

type ListCategoriesQueryHandler struct {
	categoryRepo product.CategoryRepository
}

func NewListCategoriesQueryHandler(categoryRepo product.CategoryRepository) *ListCategoriesQueryHandler {
	return &ListCategoriesQueryHandler{categoryRepo: categoryRepo}
}

func (h *ListCategoriesQueryHandler) Handle(query ListCategoriesQuery) ([]*product.Category, error) {
	if query.Limit <= 0 {
		query.Limit = 50
	}
	return h.categoryRepo.List(query.Limit, query.Offset)
}