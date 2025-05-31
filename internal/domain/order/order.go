package order

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID          string      `json:"id" gorm:"primaryKey"`
	UserID      string      `json:"user_id"`
	Items       []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
	TotalAmount float64     `json:"total_amount"`
	Status      Status      `json:"status"`
	PaymentID   string      `json:"payment_id"`
	ShippingAddress Address `json:"shipping_address" gorm:"embedded;embeddedPrefix:shipping_"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID        string  `json:"id" gorm:"primaryKey"`
	OrderID   string  `json:"order_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Subtotal  float64 `json:"subtotal"`
}

type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type Status string

const (
	StatusPending    Status = "pending"
	StatusConfirmed  Status = "confirmed"
	StatusProcessing Status = "processing"
	StatusShipped    Status = "shipped"
	StatusDelivered  Status = "delivered"
	StatusCancelled  Status = "cancelled"
	StatusRefunded   Status = "refunded"
)

type Repository interface {
	Create(order *Order) error
	GetByID(id string) (*Order, error)
	GetByUserID(userID string, limit, offset int) ([]*Order, error)
	Update(order *Order) error
	UpdateStatus(orderID string, status Status) error
	List(limit, offset int) ([]*Order, error)
}

type Service interface {
	CreateOrder(userID string, items []CreateOrderItem, shippingAddress Address) (*Order, error)
	GetOrder(id string) (*Order, error)
	GetUserOrders(userID string, limit, offset int) ([]*Order, error)
	UpdateOrderStatus(orderID string, status Status) error
	CancelOrder(orderID, userID string) error
}

type CreateOrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

func NewOrder(userID string, items []CreateOrderItem, shippingAddress Address) (*Order, error) {
	if userID == "" || len(items) == 0 {
		return nil, errors.New("invalid order data")
	}

	orderID := uuid.New().String()
	var orderItems []OrderItem
	var totalAmount float64

	for _, item := range items {
		if item.Quantity <= 0 || item.Price <= 0 {
			return nil, errors.New("invalid order item")
		}

		subtotal := float64(item.Quantity) * item.Price
		orderItem := OrderItem{
			ID:        uuid.New().String(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Subtotal:  subtotal,
		}
		orderItems = append(orderItems, orderItem)
		totalAmount += subtotal
	}

	return &Order{
		ID:              orderID,
		UserID:          userID,
		Items:           orderItems,
		TotalAmount:     totalAmount,
		Status:          StatusPending,
		ShippingAddress: shippingAddress,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

func (o *Order) CanBeCancelled() bool {
	return o.Status == StatusPending || o.Status == StatusConfirmed
}

func (o *Order) Cancel() error {
	if !o.CanBeCancelled() {
		return errors.New("order cannot be cancelled")
	}
	o.Status = StatusCancelled
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) UpdateStatus(status Status) {
	o.Status = status
	o.UpdatedAt = time.Now()
}

func (o *Order) IsCompleted() bool {
	return o.Status == StatusDelivered
}

func (o *Order) IsCancelled() bool {
	return o.Status == StatusCancelled || o.Status == StatusRefunded
}