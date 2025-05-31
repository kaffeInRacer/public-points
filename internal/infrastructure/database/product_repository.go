package database

import (
	"online-shop/internal/domain/product"

	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) product.Repository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(p *product.Product) error {
	return r.db.Create(p).Error
}

func (r *ProductRepository) GetByID(id string) (*product.Product, error) {
	var p product.Product
	err := r.db.Preload("Category").Where("id = ?", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) Update(p *product.Product) error {
	return r.db.Save(p).Error
}

func (r *ProductRepository) Delete(id string) error {
	return r.db.Model(&product.Product{}).Where("id = ?", id).Update("status", product.StatusDeleted).Error
}

func (r *ProductRepository) List(filter product.SearchFilter) ([]*product.Product, error) {
	var products []*product.Product
	query := r.db.Preload("Category")

	if filter.Query != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+filter.Query+"%", "%"+filter.Query+"%")
	}

	if filter.CategoryID != "" {
		query = query.Where("category_id = ?", filter.CategoryID)
	}

	if filter.MinPrice > 0 {
		query = query.Where("price >= ?", filter.MinPrice)
	}

	if filter.MaxPrice > 0 {
		query = query.Where("price <= ?", filter.MaxPrice)
	}

	if filter.MerchantID != "" {
		query = query.Where("merchant_id = ?", filter.MerchantID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	err := query.Limit(filter.Limit).Offset(filter.Offset).Find(&products).Error
	return products, err
}

func (r *ProductRepository) Search(query string, limit, offset int) ([]*product.Product, error) {
	var products []*product.Product
	err := r.db.Preload("Category").
		Where("name ILIKE ? OR description ILIKE ?", "%"+query+"%", "%"+query+"%").
		Where("status = ?", product.StatusActive).
		Limit(limit).Offset(offset).Find(&products).Error
	return products, err
}

func (r *ProductRepository) UpdateStock(productID string, quantity int) error {
	return r.db.Model(&product.Product{}).
		Where("id = ?", productID).
		Update("stock", gorm.Expr("stock + ?", quantity)).Error
}

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) product.CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(c *product.Category) error {
	return r.db.Create(c).Error
}

func (r *CategoryRepository) GetByID(id string) (*product.Category, error) {
	var c product.Category
	err := r.db.Preload("Parent").Where("id = ?", id).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) Update(c *product.Category) error {
	return r.db.Save(c).Error
}

func (r *CategoryRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&product.Category{}).Error
}

func (r *CategoryRepository) List(limit, offset int) ([]*product.Category, error) {
	var categories []*product.Category
	err := r.db.Preload("Parent").Limit(limit).Offset(offset).Find(&categories).Error
	return categories, err
}