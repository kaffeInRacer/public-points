package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in headers
		requestID := c.GetHeader(RequestIDHeader)
		
		// If not, generate a new one
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set the request ID in the context and response header
		c.Set("RequestID", requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID returns the request ID from the context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("RequestID"); exists {
		return requestID.(string)
	}
	return ""
}