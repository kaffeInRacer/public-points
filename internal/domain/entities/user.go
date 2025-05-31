package entities

import (
	"time"
	"github.com/google/uuid"
)

// User represents a user entity
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	FirstName string    `json:"first_name" gorm:"not null"`
	LastName  string    `json:"last_name" gorm:"not null"`
	Role      string    `json:"role" gorm:"default:'customer'"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserAddress represents a user address entity
type UserAddress struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	Street    string    `json:"street" gorm:"not null"`
	City      string    `json:"city" gorm:"not null"`
	State     string    `json:"state" gorm:"not null"`
	ZipCode   string    `json:"zip_code" gorm:"not null"`
	Country   string    `json:"country" gorm:"not null"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}

// TableName returns the table name for UserAddress
func (UserAddress) TableName() string {
	return "user_addresses"
}