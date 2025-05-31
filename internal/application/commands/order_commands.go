package commands

import (
	"online-shop/internal/domain/order"
	"online-shop/internal/domain/product"
)

type CreateOrderCommand struct {
	UserID          string                `json:"user_id" validate:"required"`
	Items           []CreateOrderItemCmd  `json:"items" validate:"required,min=1"`
	ShippingAddress order.Address         `json:"shipping_address" validate:"required"`
}

type CreateOrderItemCmd struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

type UpdateOrderStatusCommand struct {
	OrderID string       `json:"order_id" validate:"required"`
	Status  order.Status `json:"status" validate:"required"`
}

type CancelOrderCommand struct {
	OrderID string `json:"order_id" validate:"required"`
	UserID  string `json:"user_id" validate:"required"`
}

type CreateOrderCommandHandler struct {
	orderRepo   order.Repository
	productRepo product.Repository
}

func NewCreateOrderCommandHandler(orderRepo order.Repository, productRepo product.Repository) *CreateOrderCommandHandler {
	return &CreateOrderCommandHandler{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

func (h *CreateOrderCommandHandler) Handle(cmd CreateOrderCommand) (*order.Order, error) {
	var orderItems []order.CreateOrderItem

	// Validate products and calculate prices
	for _, item := range cmd.Items {
		prod, err := h.productRepo.GetByID(item.ProductID)
		if err != nil {
			return nil, ErrProductNotFound
		}

		if !prod.IsAvailable() {
			return nil, ErrProductNotFound
		}

		if prod.Stock < item.Quantity {
			return nil, ErrInsufficientStock
		}

		orderItems = append(orderItems, order.CreateOrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     prod.Price,
		})
	}

	// Create order
	newOrder, err := order.NewOrder(cmd.UserID, orderItems, cmd.ShippingAddress)
	if err != nil {
		return nil, err
	}

	// Save order
	if err := h.orderRepo.Create(newOrder); err != nil {
		return nil, err
	}

	// Update product stock
	for _, item := range cmd.Items {
		if err := h.productRepo.UpdateStock(item.ProductID, -item.Quantity); err != nil {
			// TODO: Implement compensation logic or use saga pattern
			return nil, err
		}
	}

	return newOrder, nil
}

type UpdateOrderStatusCommandHandler struct {
	orderRepo order.Repository
}

func NewUpdateOrderStatusCommandHandler(orderRepo order.Repository) *UpdateOrderStatusCommandHandler {
	return &UpdateOrderStatusCommandHandler{orderRepo: orderRepo}
}

func (h *UpdateOrderStatusCommandHandler) Handle(cmd UpdateOrderStatusCommand) error {
	existingOrder, err := h.orderRepo.GetByID(cmd.OrderID)
	if err != nil {
		return ErrOrderNotFound
	}

	existingOrder.UpdateStatus(cmd.Status)
	return h.orderRepo.Update(existingOrder)
}

type CancelOrderCommandHandler struct {
	orderRepo   order.Repository
	productRepo product.Repository
}

func NewCancelOrderCommandHandler(orderRepo order.Repository, productRepo product.Repository) *CancelOrderCommandHandler {
	return &CancelOrderCommandHandler{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

func (h *CancelOrderCommandHandler) Handle(cmd CancelOrderCommand) error {
	existingOrder, err := h.orderRepo.GetByID(cmd.OrderID)
	if err != nil {
		return ErrOrderNotFound
	}

	// Check if user owns the order
	if existingOrder.UserID != cmd.UserID {
		return ErrUnauthorized
	}

	// Check if order can be cancelled
	if !existingOrder.CanBeCancelled() {
		return ErrOrderCannotBeCancelled
	}

	// Cancel order
	if err := existingOrder.Cancel(); err != nil {
		return err
	}

	// Restore product stock
	for _, item := range existingOrder.Items {
		if err := h.productRepo.UpdateStock(item.ProductID, item.Quantity); err != nil {
			// TODO: Implement compensation logic
			return err
		}
	}

	return h.orderRepo.Update(existingOrder)
}