package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger returns a middleware that logs HTTP requests
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
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
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", raw),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		}

		// Add user ID if available
		if userID, exists := c.Get("UserID"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		// Add error if any
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case c.Writer.Status() >= 500:
			logger.Error("HTTP Request", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("HTTP Request", fields...)
		default:
			logger.Info("HTTP Request", fields...)
		}
	}
}