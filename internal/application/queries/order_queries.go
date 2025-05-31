package queries

import (
	"online-shop/internal/domain/order"
)

type GetOrderQuery struct {
	OrderID string `json:"order_id" validate:"required"`
}

type GetUserOrdersQuery struct {
	UserID string `json:"user_id" validate:"required"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ListOrdersQuery struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type GetOrderQueryHandler struct {
	orderRepo order.Repository
}

func NewGetOrderQueryHandler(orderRepo order.Repository) *GetOrderQueryHandler {
	return &GetOrderQueryHandler{orderRepo: orderRepo}
}

func (h *GetOrderQueryHandler) Handle(query GetOrderQuery) (*order.Order, error) {
	return h.orderRepo.GetByID(query.OrderID)
}

type GetUserOrdersQueryHandler struct {
	orderRepo order.Repository
}

func NewGetUserOrdersQueryHandler(orderRepo order.Repository) *GetUserOrdersQueryHandler {
	return &GetUserOrdersQueryHandler{orderRepo: orderRepo}
}

func (h *GetUserOrdersQueryHandler) Handle(query GetUserOrdersQuery) ([]*order.Order, error) {
	if query.Limit <= 0 {
		query.Limit = 10
	}
	return h.orderRepo.GetByUserID(query.UserID, query.Limit, query.Offset)
}

type ListOrdersQueryHandler struct {
	orderRepo order.Repository
}

func NewListOrdersQueryHandler(orderRepo order.Repository) *ListOrdersQueryHandler {
	return &ListOrdersQueryHandler{orderRepo: orderRepo}
}

func (h *ListOrdersQueryHandler) Handle(query ListOrdersQuery) ([]*order.Order, error) {
	if query.Limit <= 0 {
		query.Limit = 20
	}
	return h.orderRepo.List(query.Limit, query.Offset)
}