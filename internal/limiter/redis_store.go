package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implementa LimiterStoreStrategy usando Redis
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore cria uma nova instância de RedisStore
func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Testa conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisStore{client: client}, nil
}

// Increment incrementa atomicamente o contador e define expiração
func (r *RedisStore) Increment(ctx context.Context, key string, expiry time.Duration) (int64, error) {
	// Pipeline garante operações atômicas
	pipe := r.client.Pipeline()
	
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiry)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incrCmd.Val(), nil
}

// GetCount retorna o contador atual
func (r *RedisStore) GetCount(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// Exists verifica se a chave existe
func (r *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SetExpiring seta um valor com expiração
func (r *RedisStore) SetExpiring(ctx context.Context, key string, value string, expiry time.Duration) error {
	return r.client.Set(ctx, key, value, expiry).Err()
}

// Close fecha a conexão com o Redis
func (r *RedisStore) Close() error {
	return r.client.Close()
}
