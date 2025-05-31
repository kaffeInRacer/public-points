package commands

import "errors"

var (
	// User errors
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")

	// Product errors
	ErrProductNotFound     = errors.New("product not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrInvalidProductData  = errors.New("invalid product data")

	// Order errors
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled")
	ErrInvalidOrderData    = errors.New("invalid order data")

	// Payment errors
	ErrPaymentNotFound     = errors.New("payment not found")
	ErrPaymentFailed       = errors.New("payment failed")
	ErrPaymentExpired      = errors.New("payment expired")
	ErrInvalidPaymentData  = errors.New("invalid payment data")

	// General errors
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrValidationFailed   = errors.New("validation failed")
)