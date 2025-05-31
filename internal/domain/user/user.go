package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"uniqueIndex"`
	Password  string    `json:"-"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     string    `json:"phone"`
	Role      Role      `json:"role"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Role string

const (
	RoleCustomer Role = "customer"
	RoleAdmin    Role = "admin"
	RoleMerchant Role = "merchant"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusSuspended Status = "suspended"
)

type Repository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	List(limit, offset int) ([]*User, error)
}

type Service interface {
	Register(email, password, firstName, lastName, phone string) (*User, error)
	Login(email, password string) (*User, error)
	GetProfile(userID string) (*User, error)
	UpdateProfile(userID string, updates map[string]interface{}) (*User, error)
	ChangePassword(userID, oldPassword, newPassword string) error
	DeactivateAccount(userID string) error
}

func NewUser(email, password, firstName, lastName, phone string) (*User, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        uuid.New().String(),
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: firstName,
		LastName:  lastName,
		Phone:     phone,
		Role:      RoleCustomer,
		Status:    StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (u *User) ValidatePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

func (u *User) UpdatePassword(newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}