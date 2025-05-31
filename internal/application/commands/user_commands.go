package commands

import (
	"online-shop/internal/domain/user"
)

type RegisterUserCommand struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Phone     string `json:"phone"`
}

type LoginUserCommand struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateUserProfileCommand struct {
	UserID    string                 `json:"user_id" validate:"required"`
	Updates   map[string]interface{} `json:"updates"`
}

type ChangePasswordCommand struct {
	UserID      string `json:"user_id" validate:"required"`
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type DeactivateUserCommand struct {
	UserID string `json:"user_id" validate:"required"`
}

type RegisterUserCommandHandler struct {
	userRepo user.Repository
}

func NewRegisterUserCommandHandler(userRepo user.Repository) *RegisterUserCommandHandler {
	return &RegisterUserCommandHandler{userRepo: userRepo}
}

func (h *RegisterUserCommandHandler) Handle(cmd RegisterUserCommand) (*user.User, error) {
	// Check if user already exists
	existingUser, _ := h.userRepo.GetByEmail(cmd.Email)
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Create new user
	newUser, err := user.NewUser(cmd.Email, cmd.Password, cmd.FirstName, cmd.LastName, cmd.Phone)
	if err != nil {
		return nil, err
	}

	// Save user
	if err := h.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

type LoginUserCommandHandler struct {
	userRepo user.Repository
}

func NewLoginUserCommandHandler(userRepo user.Repository) *LoginUserCommandHandler {
	return &LoginUserCommandHandler{userRepo: userRepo}
}

func (h *LoginUserCommandHandler) Handle(cmd LoginUserCommand) (*user.User, error) {
	// Get user by email
	existingUser, err := h.userRepo.GetByEmail(cmd.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Validate password
	if err := existingUser.ValidatePassword(cmd.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !existingUser.IsActive() {
		return nil, ErrUserInactive
	}

	return existingUser, nil
}

type UpdateUserProfileCommandHandler struct {
	userRepo user.Repository
}

func NewUpdateUserProfileCommandHandler(userRepo user.Repository) *UpdateUserProfileCommandHandler {
	return &UpdateUserProfileCommandHandler{userRepo: userRepo}
}

func (h *UpdateUserProfileCommandHandler) Handle(cmd UpdateUserProfileCommand) (*user.User, error) {
	// Get user
	existingUser, err := h.userRepo.GetByID(cmd.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Update fields
	for key, value := range cmd.Updates {
		switch key {
		case "first_name":
			if v, ok := value.(string); ok {
				existingUser.FirstName = v
			}
		case "last_name":
			if v, ok := value.(string); ok {
				existingUser.LastName = v
			}
		case "phone":
			if v, ok := value.(string); ok {
				existingUser.Phone = v
			}
		}
	}

	// Save user
	if err := h.userRepo.Update(existingUser); err != nil {
		return nil, err
	}

	return existingUser, nil
}

type ChangePasswordCommandHandler struct {
	userRepo user.Repository
}

func NewChangePasswordCommandHandler(userRepo user.Repository) *ChangePasswordCommandHandler {
	return &ChangePasswordCommandHandler{userRepo: userRepo}
}

func (h *ChangePasswordCommandHandler) Handle(cmd ChangePasswordCommand) error {
	// Get user
	existingUser, err := h.userRepo.GetByID(cmd.UserID)
	if err != nil {
		return ErrUserNotFound
	}

	// Validate old password
	if err := existingUser.ValidatePassword(cmd.OldPassword); err != nil {
		return ErrInvalidCredentials
	}

	// Update password
	if err := existingUser.UpdatePassword(cmd.NewPassword); err != nil {
		return err
	}

	// Save user
	return h.userRepo.Update(existingUser)
}