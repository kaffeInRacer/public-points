package database

import (
	"online-shop/internal/domain/payment"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) payment.Repository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(p *payment.Payment) error {
	return r.db.Create(p).Error
}

func (r *PaymentRepository) GetByID(id string) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.Where("id = ?", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) GetByOrderID(orderID string) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.Where("order_id = ?", orderID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) GetByExternalID(externalID string) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.Where("external_id = ?", externalID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) Update(p *payment.Payment) error {
	return r.db.Save(p).Error
}

func (r *PaymentRepository) UpdateStatus(paymentID string, status payment.Status, transactionID string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if transactionID != "" {
		updates["transaction_id"] = transactionID
	}
	return r.db.Model(&payment.Payment{}).Where("id = ?", paymentID).Updates(updates).Error
}