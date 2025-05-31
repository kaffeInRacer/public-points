package database

import (
	"online-shop/internal/domain/order"

	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) order.Repository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(o *order.Order) error {
	return r.db.Create(o).Error
}

func (r *OrderRepository) GetByID(id string) (*order.Order, error) {
	var o order.Order
	err := r.db.Preload("Items").Where("id = ?", id).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) GetByUserID(userID string, limit, offset int) ([]*order.Order, error) {
	var orders []*order.Order
	err := r.db.Preload("Items").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&orders).Error
	return orders, err
}

func (r *OrderRepository) Update(o *order.Order) error {
	return r.db.Save(o).Error
}

func (r *OrderRepository) UpdateStatus(orderID string, status order.Status) error {
	return r.db.Model(&order.Order{}).
		Where("id = ?", orderID).
		Update("status", status).Error
}

func (r *OrderRepository) List(limit, offset int) ([]*order.Order, error) {
	var orders []*order.Order
	err := r.db.Preload("Items").
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&orders).Error
	return orders, err
}