package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

type RedisCache struct {
	client *redis.Client
}

func New(addr string, password string) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		}),
	}
}

// Set записывает ключ-значение в кэш с временем жизни 4 дня
func (r *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	// Устанавливаем ключ-значение в кэш

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	err = r.client.Set(key, jsonData, ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisCache) GetJSON(key string, dest interface{}) (bool, error) {
	// Получаем JSON-строку из Redis
	val, err := r.client.Get(key).Result()
	if err == redis.Nil {
		// Ключ не найден в кэше
		return false, nil
	} else if err != nil {
		// Произошла другая ошибка Redis
		return false, fmt.Errorf("failed to get key '%s' from Redis: %w", key, err)
	}

	// Демаршалируем JSON-строку в предоставленную структуру
	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal JSON from key '%s': %w", key, err)
	}

	// Успешно найдено и демаршалировано
	return true, nil
}
