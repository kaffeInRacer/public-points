package product

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	CategoryID  string    `json:"category_id"`
	Category    *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	MerchantID  string    `json:"merchant_id"`
	Images      []string  `json:"images" gorm:"type:text[]"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Category struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    *string   `json:"parent_id"`
	Parent      *Category `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusDeleted  Status = "deleted"
)

type SearchFilter struct {
	Query      string
	CategoryID string
	MinPrice   float64
	MaxPrice   float64
	MerchantID string
	Status     Status
	Limit      int
	Offset     int
}

type Repository interface {
	Create(product *Product) error
	GetByID(id string) (*Product, error)
	Update(product *Product) error
	Delete(id string) error
	List(filter SearchFilter) ([]*Product, error)
	Search(query string, limit, offset int) ([]*Product, error)
	UpdateStock(productID string, quantity int) error
}

type CategoryRepository interface {
	Create(category *Category) error
	GetByID(id string) (*Category, error)
	Update(category *Category) error
	Delete(id string) error
	List(limit, offset int) ([]*Category, error)
}

type Service interface {
	CreateProduct(name, description string, price float64, stock int, categoryID, merchantID string, images []string) (*Product, error)
	GetProduct(id string) (*Product, error)
	UpdateProduct(id string, updates map[string]interface{}) (*Product, error)
	DeleteProduct(id string) error
	SearchProducts(filter SearchFilter) ([]*Product, error)
	UpdateStock(productID string, quantity int) error
}

func NewProduct(name, description string, price float64, stock int, categoryID, merchantID string, images []string) (*Product, error) {
	if name == "" || price <= 0 || stock < 0 {
		return nil, errors.New("invalid product data")
	}

	return &Product{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
		CategoryID:  categoryID,
		MerchantID:  merchantID,
		Images:      images,
		Status:      StatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func NewCategory(name, description string, parentID *string) (*Category, error) {
	if name == "" {
		return nil, errors.New("category name is required")
	}

	return &Category{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		ParentID:    parentID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (p *Product) IsAvailable() bool {
	return p.Status == StatusActive && p.Stock > 0
}

func (p *Product) ReduceStock(quantity int) error {
	if p.Stock < quantity {
		return errors.New("insufficient stock")
	}
	p.Stock -= quantity
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Product) IncreaseStock(quantity int) {
	p.Stock += quantity
	p.UpdatedAt = time.Now()
}