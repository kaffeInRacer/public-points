package database

import (
	"fmt"
	"online-shop/internal/domain/order"
	"online-shop/internal/domain/payment"
	"online-shop/internal/domain/product"
	"online-shop/internal/domain/user"
	"online-shop/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Database{DB: db}, nil
}

func (d *Database) Migrate() error {
	return d.DB.AutoMigrate(
		&user.User{},
		&product.Category{},
		&product.Product{},
		&order.Order{},
		&order.OrderItem{},
		&payment.Payment{},
	)
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}