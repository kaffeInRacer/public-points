package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLogger returns a middleware that logs HTTP requests
func RequestLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID
		requestID := GetRequestID(c)

		// Build log fields
		fields := logrus.Fields{
			"request_id":  requestID,
			"method":      c.Request.Method,
			"path":        path,
			"query":       raw,
			"status":      c.Writer.Status(),
			"latency":     latency,
			"ip":          c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"body_size":   c.Writer.Size(),
		}

		// Add user ID if available
		if userID, exists := c.Get("UserID"); exists {
			fields["user_id"] = userID
		}

		// Add error if any
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Log based on status code
		switch {
		case c.Writer.Status() >= 500:
			logger.WithFields(fields).Error("HTTP Request")
		case c.Writer.Status() >= 400:
			logger.WithFields(fields).Warn("HTTP Request")
		default:
			logger.WithFields(fields).Info("HTTP Request")
		}
	}
}