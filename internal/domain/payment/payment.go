package payment

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	OrderID         string    `json:"order_id"`
	UserID          string    `json:"user_id"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	Method          Method    `json:"method"`
	Status          Status    `json:"status"`
	TransactionID   string    `json:"transaction_id"`
	ExternalID      string    `json:"external_id"`
	PaymentURL      string    `json:"payment_url"`
	ExpiresAt       time.Time `json:"expires_at"`
	ProcessedAt     *time.Time `json:"processed_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Method string

const (
	MethodCreditCard   Method = "credit_card"
	MethodBankTransfer Method = "bank_transfer"
	MethodEWallet      Method = "e_wallet"
	MethodVirtualAccount Method = "virtual_account"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusRefunded  Status = "refunded"
	StatusExpired   Status = "expired"
)

type Repository interface {
	Create(payment *Payment) error
	GetByID(id string) (*Payment, error)
	GetByOrderID(orderID string) (*Payment, error)
	GetByExternalID(externalID string) (*Payment, error)
	Update(payment *Payment) error
	UpdateStatus(paymentID string, status Status, transactionID string) error
}

type Service interface {
	CreatePayment(orderID, userID string, amount float64, method Method) (*Payment, error)
	ProcessPayment(paymentID string) (*Payment, error)
	HandleWebhook(data map[string]interface{}) error
	RefundPayment(paymentID string, amount float64) error
	GetPayment(id string) (*Payment, error)
}

type PaymentProvider interface {
	CreatePayment(payment *Payment) (*PaymentResponse, error)
	GetPaymentStatus(externalID string) (*PaymentStatus, error)
	RefundPayment(externalID string, amount float64) error
}

type PaymentResponse struct {
	PaymentURL    string
	ExternalID    string
	TransactionID string
	ExpiresAt     time.Time
}

type PaymentStatus struct {
	Status        Status
	TransactionID string
	ProcessedAt   *time.Time
}

func NewPayment(orderID, userID string, amount float64, method Method) *Payment {
	return &Payment{
		ID:         uuid.New().String(),
		OrderID:    orderID,
		UserID:     userID,
		Amount:     amount,
		Currency:   "IDR",
		Method:     method,
		Status:     StatusPending,
		ExpiresAt:  time.Now().Add(24 * time.Hour), // 24 hours expiry
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func (p *Payment) MarkAsPaid(transactionID string) {
	p.Status = StatusPaid
	p.TransactionID = transactionID
	now := time.Now()
	p.ProcessedAt = &now
	p.UpdatedAt = now
}

func (p *Payment) MarkAsFailed() {
	p.Status = StatusFailed
	p.UpdatedAt = time.Now()
}

func (p *Payment) MarkAsExpired() {
	p.Status = StatusExpired
	p.UpdatedAt = time.Now()
}

func (p *Payment) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

func (p *Payment) IsPaid() bool {
	return p.Status == StatusPaid
}

func (p *Payment) CanBeRefunded() bool {
	return p.Status == StatusPaid
}