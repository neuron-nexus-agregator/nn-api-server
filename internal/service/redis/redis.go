package redis

import (
	"encoding/json"
	"fmt"
	"strconv"
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

func (r *RedisCache) SetViews(id string, value int64, ttl time.Duration) error {
	return r.Set("views:"+id, value, ttl)
}

func (r *RedisCache) IncViews(id string) error {

	exists, err := r.client.Exists("views:" + id).Result()
	if err != nil {
		return fmt.Errorf("failed to check Redis key existence: %w", err)
	}

	if exists == 0 {
		// Если ключа нет, инициализируем его значением из базы
		return r.SetViews(id, 1, 24*time.Hour)
	}

	return r.client.Incr("views:" + id).Err()
}

func (r *RedisCache) GetAllViews() (map[int64]int64, error) {
	keys, err := r.client.Keys("views:*").Result()
	if err != nil {
		return nil, err
	}

	results := make(map[int64]int64)
	for _, key := range keys {
		val, err := r.client.Get(key).Result()
		if err != nil {
			continue
		}
		key_int64, err := strconv.ParseInt(key[7:], 10, 64)
		if err != nil {
			continue
		}
		val_int64, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		results[key_int64] = val_int64

		//set value to 0
		err = r.client.Del(key).Err()
		if err != nil {
			continue
		}
	}
	return results, nil
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
