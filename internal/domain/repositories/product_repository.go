package repositories

import (
	"context"
	"online-shop/internal/domain/entities"
	"github.com/google/uuid"
)

// ProductRepository defines the interface for product data operations
type ProductRepository interface {
	Create(ctx context.Context, product *entities.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Product, error)
	Update(ctx context.Context, product *entities.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*entities.Product, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*entities.Product, error)
	GetByCategory(ctx context.Context, category string, limit, offset int) ([]*entities.Product, error)
}