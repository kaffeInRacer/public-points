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