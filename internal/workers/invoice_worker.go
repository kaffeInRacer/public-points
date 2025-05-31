package workers

import (
	"bytes"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	"online-shop/internal/infrastructure/queue"
	"online-shop/internal/utils"
	"online-shop/pkg/config"
)

// InvoiceWorker handles invoice generation and email sending
type InvoiceWorker struct {
	config      *config.Config
	logger      *logrus.Logger
	emailWorker *EmailWorker
	rabbitmq    *queue.RabbitMQ
}

// NewInvoiceWorker creates a new invoice worker
func NewInvoiceWorker(cfg *config.Config, logger *logrus.Logger) *InvoiceWorker {
	return &InvoiceWorker{
		config:      cfg,
		logger:      logger,
		emailWorker: NewEmailWorker(cfg, logger),
	}
}

// SetRabbitMQ sets the RabbitMQ connection for sending emails
func (w *InvoiceWorker) SetRabbitMQ(rabbitmq *queue.RabbitMQ) {
	w.rabbitmq = rabbitmq
}

// ProcessMessage processes an invoice message
func (w *InvoiceWorker) ProcessMessage(message queue.Message) error {
	w.logger.Info("Processing invoice message", logrus.Fields{"message_id": message.ID})

	// Parse invoice data
	var invoiceData queue.InvoiceMessage
	if err := utils.MapToStruct(message.Payload, &invoiceData); err != nil {
		return fmt.Errorf("failed to parse invoice data: %w", err)
	}

	// Generate invoice
	invoice, err := w.generateInvoice(invoiceData)
	if err != nil {
		return fmt.Errorf("failed to generate invoice: %w", err)
	}

	// Send invoice via email
	if err := w.sendInvoiceEmail(invoiceData, invoice); err != nil {
		return fmt.Errorf("failed to send invoice email: %w", err)
	}

	w.logger.Info("Invoice processed and sent successfully",
		logrus.Fields{
			"message_id":  message.ID,
			"order_id":    invoiceData.OrderID,
			"user_email":  invoiceData.UserEmail,
		})

	return nil
}

// generateInvoice generates an invoice document
func (w *InvoiceWorker) generateInvoice(data queue.InvoiceMessage) (*Invoice, error) {
	invoice := &Invoice{
		OrderID:     data.OrderID,
		OrderNumber: data.OrderNumber,
		UserEmail:   data.UserEmail,
		Date:        time.Now(),
		Items:       make([]InvoiceLineItem, len(data.Items)),
		TotalAmount: data.TotalAmount,
	}

	// Convert items
	for i, item := range data.Items {
		invoice.Items[i] = InvoiceLineItem{
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			TotalPrice:  item.TotalPrice,
		}
	}

	// Calculate subtotal and taxes
	invoice.Subtotal = w.calculateSubtotal(invoice.Items)
	invoice.TaxAmount = w.calculateTax(invoice.Subtotal)
	invoice.ShippingAmount = w.calculateShipping(invoice.Items)

	// Generate invoice number
	invoice.InvoiceNumber = w.generateInvoiceNumber(data.OrderNumber)

	return invoice, nil
}

// sendInvoiceEmail sends the invoice via email
func (w *InvoiceWorker) sendInvoiceEmail(data queue.InvoiceMessage, invoice *Invoice) error {
	// Prepare email data
	emailData := map[string]interface{}{
		"OrderNumber":    data.OrderNumber,
		"InvoiceNumber":  invoice.InvoiceNumber,
		"Date":           invoice.Date.Format("January 2, 2006"),
		"Items":          invoice.Items,
		"Subtotal":       invoice.Subtotal,
		"TaxAmount":      invoice.TaxAmount,
		"ShippingAmount": invoice.ShippingAmount,
		"TotalAmount":    invoice.TotalAmount,
		"CustomerEmail":  data.UserEmail,
	}

	// Create email message
	emailMessage := queue.EmailMessage{
		To:       data.UserEmail,
		Subject:  fmt.Sprintf("Invoice for Order #%s", data.OrderNumber),
		Template: "invoice",
		Data:     emailData,
		Priority: 2, // High priority for invoices
	}

	// Send email directly or queue it
	if w.rabbitmq != nil {
		// Queue the email for processing
		return w.rabbitmq.PublishEmail(nil, emailMessage)
	} else {
		// Send email directly
		return w.emailWorker.sendEmail(emailMessage)
	}
}

// generateInvoiceNumber generates a unique invoice number
func (w *InvoiceWorker) generateInvoiceNumber(orderNumber string) string {
	timestamp := time.Now().Format("20060102")
	return fmt.Sprintf("INV-%s-%s", timestamp, orderNumber)
}

// calculateSubtotal calculates the subtotal of all items
func (w *InvoiceWorker) calculateSubtotal(items []InvoiceLineItem) float64 {
	var subtotal float64
	for _, item := range items {
		subtotal += item.TotalPrice
	}
	return subtotal
}

// calculateTax calculates tax amount (assuming 10% tax rate)
func (w *InvoiceWorker) calculateTax(subtotal float64) float64 {
	taxRate := 0.10 // 10% tax rate
	return subtotal * taxRate
}

// calculateShipping calculates shipping amount
func (w *InvoiceWorker) calculateShipping(items []InvoiceLineItem) float64 {
	// Simple shipping calculation - $5 base + $1 per item
	baseShipping := 5.0
	perItemShipping := 1.0
	
	totalItems := 0
	for _, item := range items {
		totalItems += item.Quantity
	}
	
	return baseShipping + (float64(totalItems) * perItemShipping)
}

// generatePDFInvoice generates a PDF version of the invoice
func (w *InvoiceWorker) generatePDFInvoice(invoice *Invoice) ([]byte, error) {
	// This is a placeholder for PDF generation
	// In a real implementation, you would use a library like gofpdf or wkhtmltopdf
	
	var buf bytes.Buffer
	
	// Generate simple text-based invoice for now
	buf.WriteString(fmt.Sprintf("INVOICE\n"))
	buf.WriteString(fmt.Sprintf("Invoice Number: %s\n", invoice.InvoiceNumber))
	buf.WriteString(fmt.Sprintf("Order Number: %s\n", invoice.OrderNumber))
	buf.WriteString(fmt.Sprintf("Date: %s\n", invoice.Date.Format("January 2, 2006")))
	buf.WriteString(fmt.Sprintf("Customer: %s\n\n", invoice.UserEmail))
	
	buf.WriteString("ITEMS:\n")
	buf.WriteString("----------------------------------------\n")
	for _, item := range invoice.Items {
		buf.WriteString(fmt.Sprintf("%-20s %dx $%.2f = $%.2f\n", 
			item.ProductName, item.Quantity, item.UnitPrice, item.TotalPrice))
	}
	buf.WriteString("----------------------------------------\n")
	buf.WriteString(fmt.Sprintf("Subtotal: $%.2f\n", invoice.Subtotal))
	buf.WriteString(fmt.Sprintf("Tax: $%.2f\n", invoice.TaxAmount))
	buf.WriteString(fmt.Sprintf("Shipping: $%.2f\n", invoice.ShippingAmount))
	buf.WriteString(fmt.Sprintf("TOTAL: $%.2f\n", invoice.TotalAmount))
	
	return buf.Bytes(), nil
}

// saveInvoiceToStorage saves the invoice to persistent storage
func (w *InvoiceWorker) saveInvoiceToStorage(invoice *Invoice) error {
	// This is a placeholder for saving to storage (database, file system, S3, etc.)
	w.logger.Info("Saving invoice to storage", 
		zap.String("invoice_number", invoice.InvoiceNumber),
		zap.String("order_id", invoice.OrderID),
	)
	
	// In a real implementation, you would save to your preferred storage
	// For example:
	// - Save to database
	// - Upload to S3
	// - Save to local file system
	
	return nil
}

// Invoice represents an invoice document
type Invoice struct {
	InvoiceNumber  string             `json:"invoice_number"`
	OrderID        string             `json:"order_id"`
	OrderNumber    string             `json:"order_number"`
	UserEmail      string             `json:"user_email"`
	Date           time.Time          `json:"date"`
	Items          []InvoiceLineItem  `json:"items"`
	Subtotal       float64            `json:"subtotal"`
	TaxAmount      float64            `json:"tax_amount"`
	ShippingAmount float64            `json:"shipping_amount"`
	TotalAmount    float64            `json:"total_amount"`
}

// InvoiceLineItem represents a line item in an invoice
type InvoiceLineItem struct {
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}