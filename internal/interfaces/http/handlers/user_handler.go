package handlers

import (
	"net/http"
	"online-shop/internal/application/commands"
	"online-shop/internal/application/queries"
	"online-shop/pkg/jwt"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	registerHandler       *commands.RegisterUserCommandHandler
	loginHandler          *commands.LoginUserCommandHandler
	updateProfileHandler  *commands.UpdateUserProfileCommandHandler
	changePasswordHandler *commands.ChangePasswordCommandHandler
	getProfileHandler     *queries.GetUserProfileQueryHandler
	jwtManager            *jwt.JWTManager
}

func NewUserHandler(
	registerHandler *commands.RegisterUserCommandHandler,
	loginHandler *commands.LoginUserCommandHandler,
	updateProfileHandler *commands.UpdateUserProfileCommandHandler,
	changePasswordHandler *commands.ChangePasswordCommandHandler,
	getProfileHandler *queries.GetUserProfileQueryHandler,
	jwtManager *jwt.JWTManager,
) *UserHandler {
	return &UserHandler{
		registerHandler:       registerHandler,
		loginHandler:          loginHandler,
		updateProfileHandler:  updateProfileHandler,
		changePasswordHandler: changePasswordHandler,
		getProfileHandler:     getProfileHandler,
		jwtManager:            jwtManager,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var cmd commands.RegisterUserCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.registerHandler.Handle(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":  user,
		"token": token,
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var cmd commands.LoginUserCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.loginHandler.Handle(cmd)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user,
		"token": token,
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := queries.GetUserProfileQuery{UserID: userID.(string)}
	user, err := h.getProfileHandler.Handle(query)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := commands.UpdateUserProfileCommand{
		UserID:  userID.(string),
		Updates: updates,
	}

	user, err := h.updateProfileHandler.Handle(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := commands.ChangePasswordCommand{
		UserID:      userID.(string),
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	if err := h.changePasswordHandler.Handle(cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// RefreshToken handles token refresh requests
func (h *UserHandler) RefreshToken(c *gin.Context) {
	// TODO: Implement token refresh logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Token refresh not implemented yet"})
}

// ForgotPassword handles forgot password requests
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	// TODO: Implement forgot password logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Forgot password not implemented yet"})
}

// ResetPassword handles password reset requests
func (h *UserHandler) ResetPassword(c *gin.Context) {
	// TODO: Implement password reset logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Password reset not implemented yet"})
}

// VerifyEmail handles email verification requests
func (h *UserHandler) VerifyEmail(c *gin.Context) {
	// TODO: Implement email verification logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Email verification not implemented yet"})
}

// Logout handles user logout requests
func (h *UserHandler) Logout(c *gin.Context) {
	// TODO: Implement logout logic (invalidate token, etc.)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// DeleteAccount handles account deletion requests
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	// TODO: Implement account deletion logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Account deletion not implemented yet"})
}

// GetAddresses handles getting user addresses
func (h *UserHandler) GetAddresses(c *gin.Context) {
	// TODO: Implement get addresses logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get addresses not implemented yet"})
}

// CreateAddress handles creating a new address
func (h *UserHandler) CreateAddress(c *gin.Context) {
	// TODO: Implement create address logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Create address not implemented yet"})
}

// UpdateAddress handles updating an address
func (h *UserHandler) UpdateAddress(c *gin.Context) {
	// TODO: Implement update address logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update address not implemented yet"})
}

// DeleteAddress handles deleting an address
func (h *UserHandler) DeleteAddress(c *gin.Context) {
	// TODO: Implement delete address logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Delete address not implemented yet"})
}

// SetDefaultAddress handles setting default address
func (h *UserHandler) SetDefaultAddress(c *gin.Context) {
	// TODO: Implement set default address logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Set default address not implemented yet"})
}

// GetWishlist handles getting user wishlist
func (h *UserHandler) GetWishlist(c *gin.Context) {
	// TODO: Implement get wishlist logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get wishlist not implemented yet"})
}

// AddToWishlist handles adding item to wishlist
func (h *UserHandler) AddToWishlist(c *gin.Context) {
	// TODO: Implement add to wishlist logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Add to wishlist not implemented yet"})
}

// RemoveFromWishlist handles removing item from wishlist
func (h *UserHandler) RemoveFromWishlist(c *gin.Context) {
	// TODO: Implement remove from wishlist logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Remove from wishlist not implemented yet"})
}

// GetUsers handles getting all users (admin)
func (h *UserHandler) GetUsers(c *gin.Context) {
	// TODO: Implement get all users logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get users not implemented yet"})
}

// GetUserByID handles getting user by ID (admin)
func (h *UserHandler) GetUserByID(c *gin.Context) {
	// TODO: Implement get user by ID logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get user by ID not implemented yet"})
}

// CreateUser handles creating a new user (admin)
func (h *UserHandler) CreateUser(c *gin.Context) {
	// TODO: Implement create user logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Create user not implemented yet"})
}

// DeleteUser handles deleting a user (admin)
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// TODO: Implement delete user logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Delete user not implemented yet"})
}

// UpdateUser handles updating a user (admin)
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// TODO: Implement update user logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update user not implemented yet"})
}

// SuspendUser handles suspending a user (admin)
func (h *UserHandler) SuspendUser(c *gin.Context) {
	// TODO: Implement suspend user logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Suspend user not implemented yet"})
}

// ActivateUser handles activating a user (admin)
func (h *UserHandler) ActivateUser(c *gin.Context) {
	// TODO: Implement activate user logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Activate user not implemented yet"})
}