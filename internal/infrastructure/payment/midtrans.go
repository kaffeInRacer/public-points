package payment

import (
	"fmt"
	"online-shop/internal/domain/payment"
	"online-shop/pkg/config"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

type MidtransProvider struct {
	client snap.Client
	config *config.MidtransConfig
}

func NewMidtransProvider(cfg *config.MidtransConfig) *MidtransProvider {
	var env midtrans.EnvironmentType
	if cfg.Environment == "production" {
		env = midtrans.Production
	} else {
		env = midtrans.Sandbox
	}

	client := snap.Client{}
	client.New(cfg.ServerKey, env)

	return &MidtransProvider{
		client: client,
		config: cfg,
	}
}

func (p *MidtransProvider) CreatePayment(pay *payment.Payment) (*payment.PaymentResponse, error) {
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  pay.ID,
			GrossAmt: int64(pay.Amount),
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: "Customer", // You might want to get this from user data
			Email: "customer@example.com",
		},
		EnabledPayments: snap.AllSnapPaymentType,
		Expiry: &snap.ExpiryDetails{
			StartTime: time.Now().Format("2006-01-02 15:04:05 +0700"),
			Unit:      "hours",
			Duration:  24,
		},
	}

	snapResp, err := p.client.CreateTransaction(req)
	if err != nil {
		return nil, err
	}

	return &payment.PaymentResponse{
		PaymentURL:    snapResp.RedirectURL,
		ExternalID:    pay.ID,
		TransactionID: snapResp.Token,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}, nil
}

func (p *MidtransProvider) GetPaymentStatus(externalID string) (*payment.PaymentStatus, error) {
	// In a real implementation, you would call Midtrans API to get transaction status
	// For now, we'll return a mock response
	return &payment.PaymentStatus{
		Status:        payment.StatusPending,
		TransactionID: "",
		ProcessedAt:   nil,
	}, nil
}

func (p *MidtransProvider) RefundPayment(externalID string, amount float64) error {
	// Implement refund logic using Midtrans API
	return fmt.Errorf("refund not implemented")
}

type PaymentService struct {
	provider payment.PaymentProvider
	repo     payment.Repository
}

func NewPaymentService(provider payment.PaymentProvider, repo payment.Repository) *PaymentService {
	return &PaymentService{
		provider: provider,
		repo:     repo,
	}
}

func (s *PaymentService) CreatePayment(orderID, userID string, amount float64, method payment.Method) (*payment.Payment, error) {
	pay := payment.NewPayment(orderID, userID, amount, method)

	// Create payment with provider
	response, err := s.provider.CreatePayment(pay)
	if err != nil {
		return nil, err
	}

	// Update payment with provider response
	pay.PaymentURL = response.PaymentURL
	pay.ExternalID = response.ExternalID
	pay.TransactionID = response.TransactionID
	pay.ExpiresAt = response.ExpiresAt

	// Save to database
	if err := s.repo.Create(pay); err != nil {
		return nil, err
	}

	return pay, nil
}

func (s *PaymentService) ProcessPayment(paymentID string) (*payment.Payment, error) {
	pay, err := s.repo.GetByID(paymentID)
	if err != nil {
		return nil, err
	}

	// Get status from provider
	status, err := s.provider.GetPaymentStatus(pay.ExternalID)
	if err != nil {
		return nil, err
	}

	// Update payment status
	switch status.Status {
	case payment.StatusPaid:
		pay.MarkAsPaid(status.TransactionID)
	case payment.StatusFailed:
		pay.MarkAsFailed()
	case payment.StatusExpired:
		pay.MarkAsExpired()
	}

	// Save updated payment
	if err := s.repo.Update(pay); err != nil {
		return nil, err
	}

	return pay, nil
}

func (s *PaymentService) HandleWebhook(data map[string]interface{}) error {
	// Extract transaction data from webhook
	orderID, ok := data["order_id"].(string)
	if !ok {
		return fmt.Errorf("invalid webhook data: missing order_id")
	}

	transactionStatus, ok := data["transaction_status"].(string)
	if !ok {
		return fmt.Errorf("invalid webhook data: missing transaction_status")
	}

	// Get payment by external ID (order_id in Midtrans)
	pay, err := s.repo.GetByExternalID(orderID)
	if err != nil {
		return err
	}

	// Update payment status based on webhook
	var status payment.Status
	switch transactionStatus {
	case "capture", "settlement":
		status = payment.StatusPaid
	case "pending":
		status = payment.StatusPending
	case "deny", "cancel", "expire":
		status = payment.StatusFailed
	default:
		return fmt.Errorf("unknown transaction status: %s", transactionStatus)
	}

	transactionID := ""
	if tid, ok := data["transaction_id"].(string); ok {
		transactionID = tid
	}

	return s.repo.UpdateStatus(pay.ID, status, transactionID)
}

func (s *PaymentService) RefundPayment(paymentID string, amount float64) error {
	pay, err := s.repo.GetByID(paymentID)
	if err != nil {
		return err
	}

	if !pay.CanBeRefunded() {
		return fmt.Errorf("payment cannot be refunded")
	}

	// Process refund with provider
	if err := s.provider.RefundPayment(pay.ExternalID, amount); err != nil {
		return err
	}

	// Update payment status
	return s.repo.UpdateStatus(pay.ID, payment.StatusRefunded, "")
}

func (s *PaymentService) GetPayment(id string) (*payment.Payment, error) {
	return s.repo.GetByID(id)
}