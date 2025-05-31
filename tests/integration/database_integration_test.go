package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"online-shop/tests/mocks"
)

// TestUser represents a test user model
type TestUser struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TestProduct represents a test product model
type TestProduct struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"not null"`
	Description string
	Price       float64   `gorm:"not null"`
	Stock       int       `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TestOrder represents a test order model
type TestOrder struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	User      TestUser  `gorm:"foreignKey:UserID"`
	Total     float64   `gorm:"not null"`
	Status    string    `gorm:"not null;default:'pending'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DatabaseIntegrationTestSuite struct {
	container testcontainers.Container
	db        *gorm.DB
	ctx       context.Context
}

func setupPostgresContainer(t *testing.T) *DatabaseIntegrationTestSuite {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Auto-migrate test models
	err = db.AutoMigrate(&TestUser{}, &TestProduct{}, &TestOrder{})
	require.NoError(t, err)

	return &DatabaseIntegrationTestSuite{
		container: container,
		db:        db,
		ctx:       ctx,
	}
}

func (suite *DatabaseIntegrationTestSuite) tearDown(t *testing.T) {
	if suite.container != nil {
		err := suite.container.Terminate(suite.ctx)
		assert.NoError(t, err)
	}
}

func TestDatabaseIntegration_CRUD_Operations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupPostgresContainer(t)
	defer suite.tearDown(t)

	t.Run("Create User", func(t *testing.T) {
		user := TestUser{
			Email: "test@example.com",
			Name:  "Test User",
		}

		result := suite.db.Create(&user)
		assert.NoError(t, result.Error)
		assert.NotZero(t, user.ID)
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("Read User", func(t *testing.T) {
		// Create user first
		user := TestUser{
			Email: "read@example.com",
			Name:  "Read User",
		}
		suite.db.Create(&user)

		// Read user
		var foundUser TestUser
		result := suite.db.First(&foundUser, user.ID)
		assert.NoError(t, result.Error)
		assert.Equal(t, user.Email, foundUser.Email)
		assert.Equal(t, user.Name, foundUser.Name)
	})

	t.Run("Update User", func(t *testing.T) {
		// Create user first
		user := TestUser{
			Email: "update@example.com",
			Name:  "Update User",
		}
		suite.db.Create(&user)

		// Update user
		result := suite.db.Model(&user).Update("Name", "Updated User")
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)

		// Verify update
		var updatedUser TestUser
		suite.db.First(&updatedUser, user.ID)
		assert.Equal(t, "Updated User", updatedUser.Name)
	})

	t.Run("Delete User", func(t *testing.T) {
		// Create user first
		user := TestUser{
			Email: "delete@example.com",
			Name:  "Delete User",
		}
		suite.db.Create(&user)

		// Delete user
		result := suite.db.Delete(&user)
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)

		// Verify deletion
		var deletedUser TestUser
		result = suite.db.First(&deletedUser, user.ID)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})
}

func TestDatabaseIntegration_Relationships(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupPostgresContainer(t)
	defer suite.tearDown(t)

	t.Run("User Order Relationship", func(t *testing.T) {
		// Create user
		user := TestUser{
			Email: "order@example.com",
			Name:  "Order User",
		}
		suite.db.Create(&user)

		// Create order
		order := TestOrder{
			UserID: user.ID,
			Total:  99.99,
			Status: "pending",
		}
		suite.db.Create(&order)

		// Load order with user
		var foundOrder TestOrder
		result := suite.db.Preload("User").First(&foundOrder, order.ID)
		assert.NoError(t, result.Error)
		assert.Equal(t, user.ID, foundOrder.UserID)
		assert.Equal(t, user.Email, foundOrder.User.Email)
		assert.Equal(t, user.Name, foundOrder.User.Name)
	})

	t.Run("User Multiple Orders", func(t *testing.T) {
		// Create user
		user := TestUser{
			Email: "multiorder@example.com",
			Name:  "Multi Order User",
		}
		suite.db.Create(&user)

		// Create multiple orders
		orders := []TestOrder{
			{UserID: user.ID, Total: 50.00, Status: "completed"},
			{UserID: user.ID, Total: 75.50, Status: "pending"},
			{UserID: user.ID, Total: 120.00, Status: "shipped"},
		}

		for _, order := range orders {
			suite.db.Create(&order)
		}

		// Find all orders for user
		var userOrders []TestOrder
		result := suite.db.Where("user_id = ?", user.ID).Find(&userOrders)
		assert.NoError(t, result.Error)
		assert.Len(t, userOrders, 3)

		// Verify totals
		var totalAmount float64
		for _, order := range userOrders {
			totalAmount += order.Total
		}
		assert.Equal(t, 245.50, totalAmount)
	})
}

func TestDatabaseIntegration_Transactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupPostgresContainer(t)
	defer suite.tearDown(t)

	t.Run("Successful Transaction", func(t *testing.T) {
		err := suite.db.Transaction(func(tx *gorm.DB) error {
			// Create user
			user := TestUser{
				Email: "transaction@example.com",
				Name:  "Transaction User",
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}

			// Create product
			product := TestProduct{
				Name:  "Transaction Product",
				Price: 29.99,
				Stock: 10,
			}
			if err := tx.Create(&product).Error; err != nil {
				return err
			}

			// Create order
			order := TestOrder{
				UserID: user.ID,
				Total:  product.Price,
				Status: "completed",
			}
			if err := tx.Create(&order).Error; err != nil {
				return err
			}

			// Update product stock
			if err := tx.Model(&product).Update("Stock", product.Stock-1).Error; err != nil {
				return err
			}

			return nil
		})

		assert.NoError(t, err)

		// Verify all records were created
		var user TestUser
		assert.NoError(t, suite.db.Where("email = ?", "transaction@example.com").First(&user).Error)

		var product TestProduct
		assert.NoError(t, suite.db.Where("name = ?", "Transaction Product").First(&product).Error)
		assert.Equal(t, 9, product.Stock) // Stock should be decremented

		var order TestOrder
		assert.NoError(t, suite.db.Where("user_id = ?", user.ID).First(&order).Error)
	})

	t.Run("Failed Transaction Rollback", func(t *testing.T) {
		initialUserCount := int64(0)
		suite.db.Model(&TestUser{}).Count(&initialUserCount)

		err := suite.db.Transaction(func(tx *gorm.DB) error {
			// Create user
			user := TestUser{
				Email: "rollback@example.com",
				Name:  "Rollback User",
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}

			// Create product
			product := TestProduct{
				Name:  "Rollback Product",
				Price: 39.99,
				Stock: 5,
			}
			if err := tx.Create(&product).Error; err != nil {
				return err
			}

			// Simulate error
			return fmt.Errorf("simulated transaction error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated transaction error")

		// Verify rollback - no new records should exist
		finalUserCount := int64(0)
		suite.db.Model(&TestUser{}).Count(&finalUserCount)
		assert.Equal(t, initialUserCount, finalUserCount)

		var user TestUser
		result := suite.db.Where("email = ?", "rollback@example.com").First(&user)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})
}

func TestDatabaseIntegration_QueryOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupPostgresContainer(t)
	defer suite.tearDown(t)

	// Create test data
	users := make([]TestUser, 100)
	for i := 0; i < 100; i++ {
		users[i] = TestUser{
			Email: fmt.Sprintf("user%d@example.com", i),
			Name:  fmt.Sprintf("User %d", i),
		}
	}
	suite.db.CreateInBatches(users, 10)

	products := make([]TestProduct, 50)
	for i := 0; i < 50; i++ {
		products[i] = TestProduct{
			Name:        fmt.Sprintf("Product %d", i),
			Description: fmt.Sprintf("Description for product %d", i),
			Price:       float64(10 + i),
			Stock:       100 - i,
		}
	}
	suite.db.CreateInBatches(products, 10)

	t.Run("Pagination", func(t *testing.T) {
		var paginatedUsers []TestUser
		result := suite.db.Limit(10).Offset(20).Find(&paginatedUsers)
		assert.NoError(t, result.Error)
		assert.Len(t, paginatedUsers, 10)
	})

	t.Run("Filtering and Sorting", func(t *testing.T) {
		var expensiveProducts []TestProduct
		result := suite.db.Where("price > ?", 30).Order("price DESC").Find(&expensiveProducts)
		assert.NoError(t, result.Error)
		assert.Greater(t, len(expensiveProducts), 0)

		// Verify sorting
		for i := 1; i < len(expensiveProducts); i++ {
			assert.GreaterOrEqual(t, expensiveProducts[i-1].Price, expensiveProducts[i].Price)
		}
	})

	t.Run("Aggregation", func(t *testing.T) {
		var totalUsers int64
		result := suite.db.Model(&TestUser{}).Count(&totalUsers)
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(100), totalUsers)

		var avgPrice float64
		result = suite.db.Model(&TestProduct{}).Select("AVG(price)").Scan(&avgPrice)
		assert.NoError(t, result.Error)
		assert.Greater(t, avgPrice, float64(0))
	})

	t.Run("Batch Operations", func(t *testing.T) {
		// Batch update
		result := suite.db.Model(&TestProduct{}).Where("stock < ?", 20).Update("stock", gorm.Expr("stock + ?", 50))
		assert.NoError(t, result.Error)
		assert.Greater(t, result.RowsAffected, int64(0))

		// Verify batch update
		var lowStockCount int64
		suite.db.Model(&TestProduct{}).Where("stock < ?", 20).Count(&lowStockCount)
		assert.Equal(t, int64(0), lowStockCount) // Should be 0 after update
	})
}

func TestDatabaseIntegration_ConnectionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupPostgresContainer(t)
	defer suite.tearDown(t)

	// Get underlying SQL DB to configure connection pool
	sqlDB, err := suite.db.DB()
	require.NoError(t, err)

	// Configure connection pool
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	t.Run("Concurrent Database Operations", func(t *testing.T) {
		const numGoroutines = 20
		const operationsPerGoroutine = 10

		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				var err error
				for j := 0; j < operationsPerGoroutine; j++ {
					user := TestUser{
						Email: fmt.Sprintf("concurrent%d_%d@example.com", goroutineID, j),
						Name:  fmt.Sprintf("Concurrent User %d_%d", goroutineID, j),
					}

					if createErr := suite.db.Create(&user).Error; createErr != nil {
						err = createErr
						break
					}

					var foundUser TestUser
					if findErr := suite.db.First(&foundUser, user.ID).Error; findErr != nil {
						err = findErr
						break
					}

					if updateErr := suite.db.Model(&foundUser).Update("Name", foundUser.Name+" Updated").Error; updateErr != nil {
						err = updateErr
						break
					}
				}
				results <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err)
		}

		// Verify total records created
		var totalUsers int64
		suite.db.Model(&TestUser{}).Count(&totalUsers)
		expectedUsers := int64(numGoroutines * operationsPerGoroutine)
		assert.GreaterOrEqual(t, totalUsers, expectedUsers)
	})

	t.Run("Connection Pool Stats", func(t *testing.T) {
		stats := sqlDB.Stats()
		
		assert.LessOrEqual(t, stats.OpenConnections, 10) // Max open connections
		assert.GreaterOrEqual(t, stats.OpenConnections, 0)
		assert.LessOrEqual(t, stats.Idle, 5) // Max idle connections
		assert.GreaterOrEqual(t, stats.Idle, 0)
	})
}

func TestDatabaseIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupPostgresContainer(t)
	defer suite.tearDown(t)

	t.Run("Duplicate Key Error", func(t *testing.T) {
		user1 := TestUser{
			Email: "duplicate@example.com",
			Name:  "User 1",
		}
		result := suite.db.Create(&user1)
		assert.NoError(t, result.Error)

		user2 := TestUser{
			Email: "duplicate@example.com", // Same email
			Name:  "User 2",
		}
		result = suite.db.Create(&user2)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "duplicate")
	})

	t.Run("Foreign Key Constraint Error", func(t *testing.T) {
		order := TestOrder{
			UserID: 99999, // Non-existent user ID
			Total:  50.00,
			Status: "pending",
		}
		result := suite.db.Create(&order)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "foreign key")
	})

	t.Run("Record Not Found", func(t *testing.T) {
		var user TestUser
		result := suite.db.First(&user, 99999) // Non-existent ID
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})

	t.Run("Invalid SQL Query", func(t *testing.T) {
		var users []TestUser
		result := suite.db.Where("invalid_column = ?", "value").Find(&users)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "column")
	})
}

func TestDatabaseIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupPostgresContainer(t)
	defer suite.tearDown(t)

	t.Run("Bulk Insert Performance", func(t *testing.T) {
		const batchSize = 1000
		users := make([]TestUser, batchSize)
		for i := 0; i < batchSize; i++ {
			users[i] = TestUser{
				Email: fmt.Sprintf("bulk%d@example.com", i),
				Name:  fmt.Sprintf("Bulk User %d", i),
			}
		}

		start := time.Now()
		result := suite.db.CreateInBatches(users, 100)
		duration := time.Since(start)

		assert.NoError(t, result.Error)
		assert.Less(t, duration, 5*time.Second) // Should complete within 5 seconds

		// Verify all records were created
		var count int64
		suite.db.Model(&TestUser{}).Where("email LIKE ?", "bulk%@example.com").Count(&count)
		assert.Equal(t, int64(batchSize), count)
	})

	t.Run("Query Performance with Index", func(t *testing.T) {
		// Create index on email column (should already exist from unique constraint)
		// This test verifies that queries using indexed columns are fast

		start := time.Now()
		var user TestUser
		result := suite.db.Where("email = ?", "bulk500@example.com").First(&user)
		duration := time.Since(start)

		assert.NoError(t, result.Error)
		assert.Less(t, duration, 100*time.Millisecond) // Should be very fast with index
	})
}