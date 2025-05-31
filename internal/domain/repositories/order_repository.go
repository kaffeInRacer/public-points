package repositories

import (
	"context"
	"online-shop/internal/domain/entities"
	"github.com/google/uuid"
)

// OrderRepository defines the interface for order data operations
type OrderRepository interface {
	Create(ctx context.Context, order *entities.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Order, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Order, error)
	Update(ctx context.Context, order *entities.Order) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*entities.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}