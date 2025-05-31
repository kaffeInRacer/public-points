package workers

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"online-shop/internal/infrastructure/queue"
	"online-shop/internal/utils"
	"online-shop/pkg/config"
)

// EmailWorker handles email processing
type EmailWorker struct {
	config    *config.Config
	logger    *logrus.Logger
	templates map[string]*template.Template
}

// NewEmailWorker creates a new email worker
func NewEmailWorker(cfg *config.Config, logger *logrus.Logger) *EmailWorker {
	worker := &EmailWorker{
		config:    cfg,
		logger:    logger,
		templates: make(map[string]*template.Template),
	}

	// Load email templates
	worker.loadTemplates()

	return worker
}

// ProcessMessage processes an email message
func (w *EmailWorker) ProcessMessage(message queue.Message) error {
	w.logger.Info("Processing email message", logrus.Fields{"message_id": message.ID})

	// Parse email data
	var emailData queue.EmailMessage
	if err := utils.MapToStruct(message.Payload, &emailData); err != nil {
		return fmt.Errorf("failed to parse email data: %w", err)
	}

	// Send email
	if err := w.sendEmail(emailData); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	w.logger.Info("Email sent successfully",
		logrus.Fields{
			"message_id": message.ID,
			"to":         emailData.To,
			"subject":    emailData.Subject,
		})

	return nil
}

// sendEmail sends an email using SMTP
func (w *EmailWorker) sendEmail(email queue.EmailMessage) error {
	// Render email content
	body, err := w.renderTemplate(email.Template, email.Data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Prepare email message
	msg := w.buildEmailMessage(email.To, email.Subject, body)

	// SMTP server address
	addr := fmt.Sprintf("%s:%d", w.config.SMTP.Host, w.config.SMTP.Port)

	// SMTP authentication (only if username is provided)
	var auth smtp.Auth
	if w.config.SMTP.Username != "" {
		auth = smtp.PlainAuth("",
			w.config.SMTP.Username,
			w.config.SMTP.Password,
			w.config.SMTP.Host,
		)
	}

	// Send email with or without TLS
	if w.config.SMTP.UseTLS {
		err = w.sendEmailWithTLS(addr, auth, w.config.SMTP.From, []string{email.To}, []byte(msg))
	} else {
		err = smtp.SendMail(addr, auth, w.config.SMTP.From, []string{email.To}, []byte(msg))
	}

	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

// sendEmailWithTLS sends email using TLS connection
func (w *EmailWorker) sendEmailWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName: w.config.SMTP.Host,
	}

	// Connect to SMTP server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, w.config.SMTP.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate if auth is provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	if _, err := writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// renderTemplate renders an email template with data
func (w *EmailWorker) renderTemplate(templateName string, data map[string]interface{}) (string, error) {
	tmpl, exists := w.templates[templateName]
	if !exists {
		// Use default template if specific template not found
		return w.renderDefaultTemplate(templateName, data)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// renderDefaultTemplate renders a default template
func (w *EmailWorker) renderDefaultTemplate(templateName string, data map[string]interface{}) (string, error) {
	switch templateName {
	case "welcome":
		return w.renderWelcomeTemplate(data)
	case "order_confirmation":
		return w.renderOrderConfirmationTemplate(data)
	case "invoice":
		return w.renderInvoiceTemplate(data)
	case "password_reset":
		return w.renderPasswordResetTemplate(data)
	default:
		return w.renderGenericTemplate(data)
	}
}

// Template renderers
func (w *EmailWorker) renderWelcomeTemplate(data map[string]interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Welcome to Online Shop</title>
</head>
<body>
    <h1>Welcome {{.FirstName}}!</h1>
    <p>Thank you for joining our online shop. We're excited to have you as a customer.</p>
    <p>You can now browse our products and start shopping.</p>
    <p>Best regards,<br>The Online Shop Team</p>
</body>
</html>`

	t, err := template.New("welcome").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (w *EmailWorker) renderOrderConfirmationTemplate(data map[string]interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Order Confirmation</title>
</head>
<body>
    <h1>Order Confirmation</h1>
    <p>Dear {{.CustomerName}},</p>
    <p>Thank you for your order! Your order #{{.OrderNumber}} has been confirmed.</p>
    <p><strong>Order Details:</strong></p>
    <ul>
        {{range .Items}}
        <li>{{.ProductName}} - Quantity: {{.Quantity}} - ${{.TotalPrice}}</li>
        {{end}}
    </ul>
    <p><strong>Total: ${{.TotalAmount}}</strong></p>
    <p>We'll send you another email when your order ships.</p>
    <p>Best regards,<br>The Online Shop Team</p>
</body>
</html>`

	t, err := template.New("order_confirmation").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (w *EmailWorker) renderInvoiceTemplate(data map[string]interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Invoice</title>
    <style>
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .total { font-weight: bold; }
    </style>
</head>
<body>
    <h1>Invoice</h1>
    <p><strong>Order Number:</strong> {{.OrderNumber}}</p>
    <p><strong>Date:</strong> {{.Date}}</p>
    
    <table>
        <thead>
            <tr>
                <th>Product</th>
                <th>Quantity</th>
                <th>Unit Price</th>
                <th>Total</th>
            </tr>
        </thead>
        <tbody>
            {{range .Items}}
            <tr>
                <td>{{.ProductName}}</td>
                <td>{{.Quantity}}</td>
                <td>${{.UnitPrice}}</td>
                <td>${{.TotalPrice}}</td>
            </tr>
            {{end}}
        </tbody>
        <tfoot>
            <tr class="total">
                <td colspan="3">Total Amount</td>
                <td>${{.TotalAmount}}</td>
            </tr>
        </tfoot>
    </table>
    
    <p>Thank you for your business!</p>
</body>
</html>`

	t, err := template.New("invoice").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (w *EmailWorker) renderPasswordResetTemplate(data map[string]interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Password Reset</title>
</head>
<body>
    <h1>Password Reset Request</h1>
    <p>Dear {{.FirstName}},</p>
    <p>You requested a password reset for your account.</p>
    <p>Click the link below to reset your password:</p>
    <p><a href="{{.ResetLink}}">Reset Password</a></p>
    <p>This link will expire in 24 hours.</p>
    <p>If you didn't request this, please ignore this email.</p>
    <p>Best regards,<br>The Online Shop Team</p>
</body>
</html>`

	t, err := template.New("password_reset").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (w *EmailWorker) renderGenericTemplate(data map[string]interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Subject}}</title>
</head>
<body>
    <h1>{{.Subject}}</h1>
    <p>{{.Message}}</p>
    <p>Best regards,<br>The Online Shop Team</p>
</body>
</html>`

	t, err := template.New("generic").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// buildEmailMessage builds the email message with headers
func (w *EmailWorker) buildEmailMessage(to, subject, body string) string {
	msg := fmt.Sprintf("From: %s\r\n", w.config.SMTP.From)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += body

	return msg
}

// loadTemplates loads email templates from files
func (w *EmailWorker) loadTemplates() {
	templateDir := "templates/email"
	
	// Check if template directory exists
	if _, err := filepath.Glob(templateDir + "/*.html"); err != nil {
		w.logger.Warn("Email template directory not found, using default templates")
		return
	}

	// Load templates from files
	templates := []string{"welcome", "order_confirmation", "invoice", "password_reset"}
	
	for _, name := range templates {
		templatePath := filepath.Join(templateDir, name+".html")
		if tmpl, err := template.ParseFiles(templatePath); err == nil {
			w.templates[name] = tmpl
			w.logger.Info("Loaded email template", logrus.Fields{"template": name})
		} else {
			w.logger.Warn("Failed to load email template", 
				logrus.Fields{
					"template": name,
					"error":    err.Error(),
				})
		}
	}
}

