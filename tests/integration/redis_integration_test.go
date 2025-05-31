package integration

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RedisIntegrationTestSuite struct {
	container testcontainers.Container
	client    *redis.Client
	ctx       context.Context
}

func setupRedisContainer(t *testing.T) *RedisIntegrationTestSuite {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
		DB:   0,
	})

	// Test connection
	_, err = client.Ping(ctx).Result()
	require.NoError(t, err)

	return &RedisIntegrationTestSuite{
		container: container,
		client:    client,
		ctx:       ctx,
	}
}

func (suite *RedisIntegrationTestSuite) tearDown(t *testing.T) {
	if suite.client != nil {
		suite.client.Close()
	}
	if suite.container != nil {
		err := suite.container.Terminate(suite.ctx)
		assert.NoError(t, err)
	}
}

func TestRedisIntegration_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Set and Get String", func(t *testing.T) {
		key := "test:string"
		value := "hello world"

		// Set value
		err := suite.client.Set(suite.ctx, key, value, 0).Err()
		assert.NoError(t, err)

		// Get value
		result, err := suite.client.Get(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set with Expiration", func(t *testing.T) {
		key := "test:expiry"
		value := "expires soon"
		expiration := 2 * time.Second

		// Set value with expiration
		err := suite.client.Set(suite.ctx, key, value, expiration).Err()
		assert.NoError(t, err)

		// Get value immediately
		result, err := suite.client.Get(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// Check TTL
		ttl, err := suite.client.TTL(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Greater(t, ttl, time.Duration(0))
		assert.LessOrEqual(t, ttl, expiration)

		// Wait for expiration
		time.Sleep(expiration + 100*time.Millisecond)

		// Value should be expired
		_, err = suite.client.Get(suite.ctx, key).Result()
		assert.Equal(t, redis.Nil, err)
	})

	t.Run("Delete Key", func(t *testing.T) {
		key := "test:delete"
		value := "to be deleted"

		// Set value
		err := suite.client.Set(suite.ctx, key, value, 0).Err()
		assert.NoError(t, err)

		// Verify it exists
		exists, err := suite.client.Exists(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		// Delete key
		deleted, err := suite.client.Del(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Verify it's gone
		exists, err = suite.client.Exists(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), exists)
	})
}

func TestRedisIntegration_HashOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Hash Set and Get", func(t *testing.T) {
		key := "test:hash"
		field := "name"
		value := "John Doe"

		// Set hash field
		err := suite.client.HSet(suite.ctx, key, field, value).Err()
		assert.NoError(t, err)

		// Get hash field
		result, err := suite.client.HGet(suite.ctx, key, field).Result()
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Hash Multiple Fields", func(t *testing.T) {
		key := "test:user"
		fields := map[string]interface{}{
			"name":  "Jane Doe",
			"email": "jane@example.com",
			"age":   "30",
		}

		// Set multiple fields
		err := suite.client.HMSet(suite.ctx, key, fields).Err()
		assert.NoError(t, err)

		// Get all fields
		result, err := suite.client.HGetAll(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, "Jane Doe", result["name"])
		assert.Equal(t, "jane@example.com", result["email"])
		assert.Equal(t, "30", result["age"])
	})

	t.Run("Hash Delete Field", func(t *testing.T) {
		key := "test:hash:delete"
		
		// Set multiple fields
		err := suite.client.HMSet(suite.ctx, key, map[string]interface{}{
			"field1": "value1",
			"field2": "value2",
			"field3": "value3",
		}).Err()
		assert.NoError(t, err)

		// Delete one field
		deleted, err := suite.client.HDel(suite.ctx, key, "field2").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Verify field is gone
		_, err = suite.client.HGet(suite.ctx, key, "field2").Result()
		assert.Equal(t, redis.Nil, err)

		// Verify other fields still exist
		result, err := suite.client.HGetAll(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "field1")
		assert.Contains(t, result, "field3")
	})
}

func TestRedisIntegration_ListOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("List Push and Pop", func(t *testing.T) {
		key := "test:list"

		// Push values to list
		err := suite.client.LPush(suite.ctx, key, "first", "second", "third").Err()
		assert.NoError(t, err)

		// Check list length
		length, err := suite.client.LLen(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(3), length)

		// Pop values from list
		value, err := suite.client.RPop(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, "first", value) // LPUSH adds to front, RPOP removes from back

		value, err = suite.client.RPop(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, "second", value)

		// Check remaining length
		length, err = suite.client.LLen(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), length)
	})

	t.Run("List Range", func(t *testing.T) {
		key := "test:list:range"

		// Push multiple values
		values := []interface{}{"item1", "item2", "item3", "item4", "item5"}
		err := suite.client.RPush(suite.ctx, key, values...).Err()
		assert.NoError(t, err)

		// Get range of values
		result, err := suite.client.LRange(suite.ctx, key, 1, 3).Result()
		assert.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, []string{"item2", "item3", "item4"}, result)

		// Get all values
		all, err := suite.client.LRange(suite.ctx, key, 0, -1).Result()
		assert.NoError(t, err)
		assert.Len(t, all, 5)
	})
}

func TestRedisIntegration_SetOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Set Add and Members", func(t *testing.T) {
		key := "test:set"

		// Add members to set
		err := suite.client.SAdd(suite.ctx, key, "member1", "member2", "member3").Err()
		assert.NoError(t, err)

		// Get all members
		members, err := suite.client.SMembers(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Len(t, members, 3)
		assert.Contains(t, members, "member1")
		assert.Contains(t, members, "member2")
		assert.Contains(t, members, "member3")

		// Check if member exists
		exists, err := suite.client.SIsMember(suite.ctx, key, "member1").Result()
		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = suite.client.SIsMember(suite.ctx, key, "nonexistent").Result()
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Set Operations", func(t *testing.T) {
		set1 := "test:set1"
		set2 := "test:set2"

		// Create two sets
		err := suite.client.SAdd(suite.ctx, set1, "a", "b", "c").Err()
		assert.NoError(t, err)

		err = suite.client.SAdd(suite.ctx, set2, "b", "c", "d").Err()
		assert.NoError(t, err)

		// Intersection
		intersection, err := suite.client.SInter(suite.ctx, set1, set2).Result()
		assert.NoError(t, err)
		assert.Len(t, intersection, 2)
		assert.Contains(t, intersection, "b")
		assert.Contains(t, intersection, "c")

		// Union
		union, err := suite.client.SUnion(suite.ctx, set1, set2).Result()
		assert.NoError(t, err)
		assert.Len(t, union, 4)
		assert.Contains(t, union, "a")
		assert.Contains(t, union, "b")
		assert.Contains(t, union, "c")
		assert.Contains(t, union, "d")

		// Difference
		diff, err := suite.client.SDiff(suite.ctx, set1, set2).Result()
		assert.NoError(t, err)
		assert.Len(t, diff, 1)
		assert.Contains(t, diff, "a")
	})
}

func TestRedisIntegration_SortedSetOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Sorted Set Add and Range", func(t *testing.T) {
		key := "test:zset"

		// Add members with scores
		members := []*redis.Z{
			{Score: 1, Member: "one"},
			{Score: 2, Member: "two"},
			{Score: 3, Member: "three"},
		}
		err := suite.client.ZAdd(suite.ctx, key, members...).Err()
		assert.NoError(t, err)

		// Get range by rank
		result, err := suite.client.ZRange(suite.ctx, key, 0, -1).Result()
		assert.NoError(t, err)
		assert.Equal(t, []string{"one", "two", "three"}, result)

		// Get range with scores
		resultWithScores, err := suite.client.ZRangeWithScores(suite.ctx, key, 0, -1).Result()
		assert.NoError(t, err)
		assert.Len(t, resultWithScores, 3)
		assert.Equal(t, float64(1), resultWithScores[0].Score)
		assert.Equal(t, "one", resultWithScores[0].Member)
	})

	t.Run("Sorted Set Score Operations", func(t *testing.T) {
		key := "test:zset:scores"

		// Add members
		err := suite.client.ZAdd(suite.ctx, key, &redis.Z{Score: 10, Member: "player1"}).Err()
		assert.NoError(t, err)

		err = suite.client.ZAdd(suite.ctx, key, &redis.Z{Score: 20, Member: "player2"}).Err()
		assert.NoError(t, err)

		// Get score
		score, err := suite.client.ZScore(suite.ctx, key, "player1").Result()
		assert.NoError(t, err)
		assert.Equal(t, float64(10), score)

		// Increment score
		newScore, err := suite.client.ZIncrBy(suite.ctx, key, 5, "player1").Result()
		assert.NoError(t, err)
		assert.Equal(t, float64(15), newScore)

		// Get rank
		rank, err := suite.client.ZRank(suite.ctx, key, "player1").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), rank) // Still lowest score

		rank, err = suite.client.ZRank(suite.ctx, key, "player2").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rank) // Highest score
	})
}

func TestRedisIntegration_PubSub(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Publish Subscribe", func(t *testing.T) {
		channel := "test:channel"
		message := "hello subscribers"

		// Subscribe to channel
		pubsub := suite.client.Subscribe(suite.ctx, channel)
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(suite.ctx)
		assert.NoError(t, err)

		// Publish message
		err = suite.client.Publish(suite.ctx, channel, message).Err()
		assert.NoError(t, err)

		// Receive message
		msg, err := pubsub.ReceiveMessage(suite.ctx)
		assert.NoError(t, err)
		assert.Equal(t, channel, msg.Channel)
		assert.Equal(t, message, msg.Payload)
	})

	t.Run("Pattern Subscribe", func(t *testing.T) {
		pattern := "test:*"
		channel1 := "test:channel1"
		channel2 := "test:channel2"
		message := "pattern message"

		// Subscribe to pattern
		pubsub := suite.client.PSubscribe(suite.ctx, pattern)
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(suite.ctx)
		assert.NoError(t, err)

		// Publish to matching channels
		err = suite.client.Publish(suite.ctx, channel1, message).Err()
		assert.NoError(t, err)

		err = suite.client.Publish(suite.ctx, channel2, message).Err()
		assert.NoError(t, err)

		// Receive messages
		for i := 0; i < 2; i++ {
			msg, err := pubsub.ReceiveMessage(suite.ctx)
			assert.NoError(t, err)
			assert.Equal(t, message, msg.Payload)
			assert.Contains(t, []string{channel1, channel2}, msg.Channel)
		}
	})
}

func TestRedisIntegration_Transactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Multi Exec Transaction", func(t *testing.T) {
		key1 := "test:tx:key1"
		key2 := "test:tx:key2"

		// Execute transaction
		pipe := suite.client.TxPipeline()
		pipe.Set(suite.ctx, key1, "value1", 0)
		pipe.Set(suite.ctx, key2, "value2", 0)
		pipe.Incr(suite.ctx, "test:tx:counter")

		results, err := pipe.Exec(suite.ctx)
		assert.NoError(t, err)
		assert.Len(t, results, 3)

		// Verify all commands succeeded
		for _, result := range results {
			assert.NoError(t, result.Err())
		}

		// Verify values were set
		val1, err := suite.client.Get(suite.ctx, key1).Result()
		assert.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := suite.client.Get(suite.ctx, key2).Result()
		assert.NoError(t, err)
		assert.Equal(t, "value2", val2)

		counter, err := suite.client.Get(suite.ctx, "test:tx:counter").Result()
		assert.NoError(t, err)
		assert.Equal(t, "1", counter)
	})

	t.Run("Watch Multi Exec", func(t *testing.T) {
		key := "test:watch:key"
		
		// Set initial value
		err := suite.client.Set(suite.ctx, key, "initial", 0).Err()
		assert.NoError(t, err)

		// Start watching the key
		err = suite.client.Watch(suite.ctx, func(tx *redis.Tx) error {
			// Get current value
			val, err := tx.Get(suite.ctx, key).Result()
			if err != nil {
				return err
			}

			// Modify value in transaction
			_, err = tx.TxPipelined(suite.ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(suite.ctx, key, val+"_modified", 0)
				return nil
			})
			return err
		}, key)
		assert.NoError(t, err)

		// Verify value was modified
		result, err := suite.client.Get(suite.ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, "initial_modified", result)
	})
}

func TestRedisIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Bulk Operations Performance", func(t *testing.T) {
		const numOperations = 1000

		// Test bulk SET operations
		start := time.Now()
		pipe := suite.client.Pipeline()
		for i := 0; i < numOperations; i++ {
			pipe.Set(suite.ctx, fmt.Sprintf("bulk:key:%d", i), fmt.Sprintf("value:%d", i), 0)
		}
		_, err := pipe.Exec(suite.ctx)
		setDuration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, setDuration, 5*time.Second) // Should complete within 5 seconds

		// Test bulk GET operations
		start = time.Now()
		pipe = suite.client.Pipeline()
		for i := 0; i < numOperations; i++ {
			pipe.Get(suite.ctx, fmt.Sprintf("bulk:key:%d", i))
		}
		results, err := pipe.Exec(suite.ctx)
		getDuration := time.Since(start)

		assert.NoError(t, err)
		assert.Len(t, results, numOperations)
		assert.Less(t, getDuration, 5*time.Second) // Should complete within 5 seconds

		t.Logf("Bulk SET %d operations: %v", numOperations, setDuration)
		t.Logf("Bulk GET %d operations: %v", numOperations, getDuration)
	})

	t.Run("Concurrent Operations", func(t *testing.T) {
		const numGoroutines = 10
		const operationsPerGoroutine = 100

		results := make(chan error, numGoroutines)

		start := time.Now()
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				var err error
				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("concurrent:%d:%d", goroutineID, j)
					value := fmt.Sprintf("value:%d:%d", goroutineID, j)

					if setErr := suite.client.Set(suite.ctx, key, value, 0).Err(); setErr != nil {
						err = setErr
						break
					}

					if _, getErr := suite.client.Get(suite.ctx, key).Result(); getErr != nil {
						err = getErr
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

		duration := time.Since(start)
		totalOperations := numGoroutines * operationsPerGoroutine * 2 // SET + GET
		t.Logf("Concurrent %d operations: %v", totalOperations, duration)
		assert.Less(t, duration, 10*time.Second)
	})
}

func TestRedisIntegration_ConnectionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	// Create client with specific pool settings
	host, _ := suite.container.Host(suite.ctx)
	port, _ := suite.container.MappedPort(suite.ctx, "6379")

	poolClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port.Port()),
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})
	defer poolClient.Close()

	t.Run("Pool Statistics", func(t *testing.T) {
		// Perform some operations to use connections
		for i := 0; i < 20; i++ {
			err := poolClient.Set(suite.ctx, fmt.Sprintf("pool:test:%d", i), "value", 0).Err()
			assert.NoError(t, err)
		}

		// Get pool stats
		stats := poolClient.PoolStats()
		assert.LessOrEqual(t, stats.TotalConns, uint32(10)) // Max pool size
		assert.GreaterOrEqual(t, stats.IdleConns, uint32(0))
		assert.GreaterOrEqual(t, stats.StaleConns, uint32(0))

		t.Logf("Pool Stats - Total: %d, Idle: %d, Stale: %d", 
			stats.TotalConns, stats.IdleConns, stats.StaleConns)
	})
}

func TestRedisIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupRedisContainer(t)
	defer suite.tearDown(t)

	t.Run("Key Not Found", func(t *testing.T) {
		_, err := suite.client.Get(suite.ctx, "nonexistent:key").Result()
		assert.Equal(t, redis.Nil, err)
	})

	t.Run("Wrong Type Operation", func(t *testing.T) {
		key := "test:string:key"
		
		// Set a string value
		err := suite.client.Set(suite.ctx, key, "string value", 0).Err()
		assert.NoError(t, err)

		// Try to use list operation on string key
		err = suite.client.LPush(suite.ctx, key, "list value").Err()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "WRONGTYPE")
	})

	t.Run("Invalid Command", func(t *testing.T) {
		// Try to execute invalid command
		_, err := suite.client.Do(suite.ctx, "INVALID_COMMAND").Result()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})

	t.Run("Connection Timeout", func(t *testing.T) {
		// Create client with very short timeout
		host, _ := suite.container.Host(suite.ctx)
		port, _ := suite.container.MappedPort(suite.ctx, "6379")

		timeoutClient := redis.NewClient(&redis.Options{
			Addr:        fmt.Sprintf("%s:%s", host, port.Port()),
			DB:          0,
			ReadTimeout: 1 * time.Nanosecond, // Extremely short timeout
		})
		defer timeoutClient.Close()

		// This should timeout
		_, err := timeoutClient.Get(suite.ctx, "any:key").Result()
		assert.Error(t, err)
		// Note: The exact error type may vary, but it should be a timeout-related error
	})
}