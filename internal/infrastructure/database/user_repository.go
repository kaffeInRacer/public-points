package database

import (
	"online-shop/internal/domain/user"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) user.Repository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *user.User) error {
	return r.db.Create(u).Error
}

func (r *UserRepository) GetByID(id string) (*user.User, error) {
	var u user.User
	err := r.db.Where("id = ?", id).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(email string) (*user.User, error) {
	var u user.User
	err := r.db.Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Update(u *user.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&user.User{}).Error
}

func (r *UserRepository) List(limit, offset int) ([]*user.User, error) {
	var users []*user.User
	err := r.db.Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}