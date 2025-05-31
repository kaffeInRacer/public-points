package entities

import (
	"time"
	"github.com/google/uuid"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Order represents an order entity
type Order struct {
	ID         uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID   `json:"user_id" gorm:"type:uuid;not null"`
	User       User        `json:"user" gorm:"foreignKey:UserID"`
	Items      []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
	TotalPrice float64     `json:"total_price" gorm:"not null"`
	Status     string      `json:"status" gorm:"default:'pending'"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// OrderItem represents an order item entity
type OrderItem struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrderID   uuid.UUID `json:"order_id" gorm:"type:uuid;not null"`
	ProductID uuid.UUID `json:"product_id" gorm:"type:uuid;not null"`
	Product   Product   `json:"product" gorm:"foreignKey:ProductID"`
	Quantity  int       `json:"quantity" gorm:"not null"`
	Price     float64   `json:"price" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the table name for Order
func (Order) TableName() string {
	return "orders"
}

// TableName returns the table name for OrderItem
func (OrderItem) TableName() string {
	return "order_items"
}