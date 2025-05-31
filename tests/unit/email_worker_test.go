package unit

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"online-shop/internal/workers"
	"online-shop/tests/mocks"
)

// MockEmailService implements EmailServiceInterface for testing
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendEmail(to, subject, body string) error {
	args := m.Called(to, subject, body)
	return args.Error(0)
}

func (m *MockEmailService) SendHTMLEmail(to, subject, htmlBody, textBody string) error {
	args := m.Called(to, subject, htmlBody, textBody)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailWithAttachment(to, subject, body string, attachments []string) error {
	args := m.Called(to, subject, body, attachments)
	return args.Error(0)
}

func (m *MockEmailService) SendTemplateEmail(to, subject, templateName string, data interface{}) error {
	args := m.Called(to, subject, templateName, data)
	return args.Error(0)
}

func (m *MockEmailService) ValidateEmail(email string) bool {
	args := m.Called(email)
	return args.Bool(0)
}

func (m *MockEmailService) GetSMTPConfig() mocks.SMTPConfig {
	args := m.Called()
	return args.Get(0).(mocks.SMTPConfig)
}

func (m *MockEmailService) SetSMTPConfig(config mocks.SMTPConfig) {
	m.Called(config)
}

func (m *MockEmailService) TestConnection() error {
	args := m.Called()
	return args.Error(0)
}

func TestEmailWorker_NewEmailWorker(t *testing.T) {
	mockEmailService := &MockEmailService{}
	mockLogger := mocks.NewMockLogger()

	worker := workers.NewEmailWorker(mockEmailService, mockLogger)
	assert.NotNil(t, worker)
}

func TestEmailWorker_ProcessEmailJob(t *testing.T) {
	tests := []struct {
		name           string
		emailData      workers.EmailData
		setupMocks     func(*MockEmailService, *mocks.MockLogger)
		expectedError  bool
		expectedCalls  int
	}{
		{
			name: "successful simple email",
			emailData: workers.EmailData{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
				Type:    "simple",
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				emailService.On("SendEmail", "test@example.com", "Test Subject", "Test Body").Return(nil)
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name: "successful HTML email",
			emailData: workers.EmailData{
				To:       "test@example.com",
				Subject:  "Test Subject",
				Body:     "Test Body",
				HTMLBody: "<h1>Test HTML</h1>",
				Type:     "html",
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				emailService.On("SendHTMLEmail", "test@example.com", "Test Subject", "<h1>Test HTML</h1>", "Test Body").Return(nil)
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name: "successful email with attachments",
			emailData: workers.EmailData{
				To:          "test@example.com",
				Subject:     "Test Subject",
				Body:        "Test Body",
				Type:        "attachment",
				Attachments: []string{"file1.pdf", "file2.jpg"},
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				emailService.On("SendEmailWithAttachment", "test@example.com", "Test Subject", "Test Body", []string{"file1.pdf", "file2.jpg"}).Return(nil)
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name: "successful template email",
			emailData: workers.EmailData{
				To:           "test@example.com",
				Subject:      "Test Subject",
				Type:         "template",
				TemplateName: "welcome",
				TemplateData: map[string]interface{}{"name": "John"},
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				emailService.On("SendTemplateEmail", "test@example.com", "Test Subject", "welcome", map[string]interface{}{"name": "John"}).Return(nil)
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name: "failed email sending",
			emailData: workers.EmailData{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
				Type:    "simple",
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				emailService.On("SendEmail", "test@example.com", "Test Subject", "Test Body").Return(errors.New("SMTP error"))
			},
			expectedError: true,
			expectedCalls: 1,
		},
		{
			name: "invalid email address",
			emailData: workers.EmailData{
				To:      "invalid-email",
				Subject: "Test Subject",
				Body:    "Test Body",
				Type:    "simple",
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				emailService.On("ValidateEmail", "invalid-email").Return(false)
			},
			expectedError: true,
			expectedCalls: 0, // SendEmail should not be called
		},
		{
			name: "missing required fields",
			emailData: workers.EmailData{
				To:   "test@example.com",
				Type: "simple",
				// Missing Subject and Body
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				// No email service calls expected
			},
			expectedError: true,
			expectedCalls: 0,
		},
		{
			name: "unknown email type",
			emailData: workers.EmailData{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
				Type:    "unknown",
			},
			setupMocks: func(emailService *MockEmailService, logger *mocks.MockLogger) {
				// No email service calls expected
			},
			expectedError: true,
			expectedCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmailService := &MockEmailService{}
			mockLogger := mocks.NewMockLogger()

			// Setup mocks
			tt.setupMocks(mockEmailService, mockLogger)

			worker := workers.NewEmailWorker(mockEmailService, mockLogger)
			
			// Create email job
			job := workers.NewEmailJob("test-job", tt.emailData)
			
			// Execute job
			err := job.Execute()

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock calls
			mockEmailService.AssertExpectations(t)
		})
	}
}

func TestEmailWorker_EmailJobRetry(t *testing.T) {
	mockEmailService := &MockEmailService{}
	mockLogger := mocks.NewMockLogger()

	emailData := workers.EmailData{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
		Type:    "simple",
	}

	// Setup mock to fail first two attempts, succeed on third
	mockEmailService.On("SendEmail", "test@example.com", "Test Subject", "Test Body").Return(errors.New("SMTP error")).Times(2)
	mockEmailService.On("SendEmail", "test@example.com", "Test Subject", "Test Body").Return(nil).Once()

	worker := workers.NewEmailWorker(mockEmailService, mockLogger)
	job := workers.NewEmailJob("retry-job", emailData)

	// First attempt - should fail
	err := job.Execute()
	assert.Error(t, err)
	assert.True(t, job.ShouldRetry())
	assert.Equal(t, 0, job.GetRetryCount()) // Retry count incremented in OnFailure

	// Simulate retry mechanism
	job.OnFailure(err)
	assert.Equal(t, 1, job.GetRetryCount())

	// Second attempt - should fail
	err = job.Execute()
	assert.Error(t, err)
	assert.True(t, job.ShouldRetry())

	job.OnFailure(err)
	assert.Equal(t, 2, job.GetRetryCount())

	// Third attempt - should succeed
	err = job.Execute()
	assert.NoError(t, err)

	job.OnSuccess()

	mockEmailService.AssertExpectations(t)
}

func TestEmailWorker_EmailJobProperties(t *testing.T) {
	emailData := workers.EmailData{
		To:       "test@example.com",
		Subject:  "Test Subject",
		Body:     "Test Body",
		Type:     "simple",
		Priority: 5,
	}

	job := workers.NewEmailJob("test-job-123", emailData)

	assert.Equal(t, "test-job-123", job.GetID())
	assert.Equal(t, "email", job.GetType())
	assert.Equal(t, 5, job.GetPriority())
	assert.Equal(t, 0, job.GetRetryCount())
	assert.Equal(t, 3, job.GetMaxRetries()) // Default max retries
	assert.True(t, job.ShouldRetry())
}

func TestEmailWorker_EmailValidation(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		isValid  bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			isValid: true,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			isValid: true,
		},
		{
			name:    "valid email with plus",
			email:   "user+tag@example.com",
			isValid: true,
		},
		{
			name:    "invalid email - no @",
			email:   "invalid-email",
			isValid: false,
		},
		{
			name:    "invalid email - no domain",
			email:   "user@",
			isValid: false,
		},
		{
			name:    "invalid email - no user",
			email:   "@example.com",
			isValid: false,
		},
		{
			name:    "empty email",
			email:   "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmailService := &MockEmailService{}
			mockLogger := mocks.NewMockLogger()

			mockEmailService.On("ValidateEmail", tt.email).Return(tt.isValid)

			worker := workers.NewEmailWorker(mockEmailService, mockLogger)
			
			// Test through email job validation
			emailData := workers.EmailData{
				To:      tt.email,
				Subject: "Test",
				Body:    "Test",
				Type:    "simple",
			}

			job := workers.NewEmailJob("validation-test", emailData)
			err := job.Execute()

			if tt.isValid {
				// If email is valid, execution might still fail due to other reasons
				// but validation should pass
				mockEmailService.On("SendEmail", tt.email, "Test", "Test").Return(nil).Maybe()
			} else {
				// If email is invalid, execution should fail
				assert.Error(t, err)
			}

			mockEmailService.AssertExpectations(t)
		})
	}
}

func TestEmailWorker_ConcurrentEmailProcessing(t *testing.T) {
	mockEmailService := &MockEmailService{}
	mockLogger := mocks.NewMockLogger()

	// Setup mock to handle multiple concurrent calls
	mockEmailService.On("SendEmail", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	worker := workers.NewEmailWorker(mockEmailService, mockLogger)

	const numJobs = 10
	jobs := make([]*workers.EmailJob, numJobs)
	results := make(chan error, numJobs)

	// Create jobs
	for i := 0; i < numJobs; i++ {
		emailData := workers.EmailData{
			To:      fmt.Sprintf("test%d@example.com", i),
			Subject: fmt.Sprintf("Test Subject %d", i),
			Body:    fmt.Sprintf("Test Body %d", i),
			Type:    "simple",
		}
		jobs[i] = workers.NewEmailJob(fmt.Sprintf("concurrent-job-%d", i), emailData)
	}

	// Execute jobs concurrently
	for _, job := range jobs {
		go func(j *workers.EmailJob) {
			results <- j.Execute()
		}(job)
	}

	// Collect results
	for i := 0; i < numJobs; i++ {
		err := <-results
		assert.NoError(t, err)
	}

	// Verify all calls were made
	mockEmailService.AssertExpectations(t)
	assert.Equal(t, numJobs, len(mockEmailService.Calls))
}

func TestEmailWorker_EmailJobTimeout(t *testing.T) {
	mockEmailService := &MockEmailService{}
	mockLogger := mocks.NewMockLogger()

	// Setup mock to simulate slow email sending
	mockEmailService.On("SendEmail", "test@example.com", "Test Subject", "Test Body").Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(200 * time.Millisecond) // Simulate slow operation
	})

	worker := workers.NewEmailWorker(mockEmailService, mockLogger)

	emailData := workers.EmailData{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
		Type:    "simple",
		Timeout: 100 * time.Millisecond, // Shorter than the mock delay
	}

	job := workers.NewEmailJob("timeout-job", emailData)

	start := time.Now()
	err := job.Execute()
	duration := time.Since(start)

	// Should timeout and return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Less(t, duration, 150*time.Millisecond) // Should timeout before mock completes
}

func TestEmailWorker_EmailJobPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		expected int
	}{
		{
			name:     "high priority",
			priority: 10,
			expected: 10,
		},
		{
			name:     "normal priority",
			priority: 5,
			expected: 5,
		},
		{
			name:     "low priority",
			priority: 1,
			expected: 1,
		},
		{
			name:     "default priority",
			priority: 0,
			expected: 1, // Should default to 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emailData := workers.EmailData{
				To:       "test@example.com",
				Subject:  "Test",
				Body:     "Test",
				Type:     "simple",
				Priority: tt.priority,
			}

			job := workers.NewEmailJob("priority-test", emailData)
			assert.Equal(t, tt.expected, job.GetPriority())
		})
	}
}

func TestEmailWorker_SMTPConfiguration(t *testing.T) {
	mockEmailService := &MockEmailService{}
	mockLogger := mocks.NewMockLogger()

	config := mocks.SMTPConfig{
		Host:       "smtp.example.com",
		Port:       587,
		Username:   "user@example.com",
		Password:   "password",
		FromEmail:  "noreply@example.com",
		FromName:   "Test Service",
		UseTLS:     true,
		UseSSL:     false,
		Timeout:    30,
		MaxRetries: 3,
		RetryDelay: 5,
	}

	mockEmailService.On("GetSMTPConfig").Return(config)
	mockEmailService.On("SetSMTPConfig", config).Return()
	mockEmailService.On("TestConnection").Return(nil)

	worker := workers.NewEmailWorker(mockEmailService, mockLogger)

	// Test getting configuration
	retrievedConfig := mockEmailService.GetSMTPConfig()
	assert.Equal(t, config, retrievedConfig)

	// Test setting configuration
	mockEmailService.SetSMTPConfig(config)

	// Test connection
	err := mockEmailService.TestConnection()
	assert.NoError(t, err)

	mockEmailService.AssertExpectations(t)
}

func TestEmailWorker_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockEmailService)
		expectedError string
	}{
		{
			name: "SMTP connection error",
			setupMock: func(emailService *MockEmailService) {
				emailService.On("SendEmail", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("SMTP connection failed"))
			},
			expectedError: "SMTP connection failed",
		},
		{
			name: "authentication error",
			setupMock: func(emailService *MockEmailService) {
				emailService.On("SendEmail", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("authentication failed"))
			},
			expectedError: "authentication failed",
		},
		{
			name: "rate limit error",
			setupMock: func(emailService *MockEmailService) {
				emailService.On("SendEmail", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("rate limit exceeded"))
			},
			expectedError: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmailService := &MockEmailService{}
			mockLogger := mocks.NewMockLogger()

			tt.setupMock(mockEmailService)

			worker := workers.NewEmailWorker(mockEmailService, mockLogger)

			emailData := workers.EmailData{
				To:      "test@example.com",
				Subject: "Test",
				Body:    "Test",
				Type:    "simple",
			}

			job := workers.NewEmailJob("error-test", emailData)
			err := job.Execute()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)

			mockEmailService.AssertExpectations(t)
		})
	}
}