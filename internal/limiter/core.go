package limiter

import (
	"context"
	"fmt"
	"time"
)

// CoreLimiter implementa a lógica principal de rate limiting
type CoreLimiter struct {
	store LimiterStoreStrategy
}

// NewCoreLimiter cria uma nova instância do CoreLimiter
func NewCoreLimiter(store LimiterStoreStrategy) *CoreLimiter {
	return &CoreLimiter{
		store: store,
	}
}

// BlockStatus representa o resultado da verificação de rate limit
type BlockStatus struct {
	Allowed       bool
	CurrentCount  int64
	Limit         int
	BlockDuration time.Duration
}

// Allow verifica se a requisição deve ser permitida
// key: identificador único (IP ou Token)
// limit: número máximo de requisições permitidas por segundo
// blockDuration: tempo de bloqueio após exceder o limite
func (c *CoreLimiter) Allow(ctx context.Context, key string, limit int, blockDuration time.Duration) (*BlockStatus, error) {
	// Chaves: contador de 1s e flag de bloqueio por duração
	counterKey := "rl:cnt:" + key
	blockKey := "rl:blk:" + key

	// Se estiver bloqueado, retorna 429
	exists, err := c.store.Exists(ctx, blockKey)
	if err != nil {
		// Fail-open
		return &BlockStatus{Allowed: true, CurrentCount: 0, Limit: limit, BlockDuration: blockDuration}, fmt.Errorf("erro ao verificar bloqueio: %w", err)
	}
	if exists {
		return &BlockStatus{Allowed: false, CurrentCount: 0, Limit: limit, BlockDuration: blockDuration}, nil
	}

	// Incrementa contador com janela de 1 segundo
	count, err := c.store.Increment(ctx, counterKey, time.Second)
	if err != nil {
		// Fail-open
		return &BlockStatus{Allowed: true, CurrentCount: 0, Limit: limit, BlockDuration: blockDuration}, fmt.Errorf("erro ao incrementar contador: %w", err)
	}

	if count > int64(limit) {
		// Seta bloqueio pela duração configurada
		if err := c.store.SetExpiring(ctx, blockKey, "1", blockDuration); err != nil {
			// Fail-open
			return &BlockStatus{Allowed: true, CurrentCount: count, Limit: limit, BlockDuration: blockDuration}, fmt.Errorf("erro ao setar bloqueio: %w", err)
		}
		return &BlockStatus{Allowed: false, CurrentCount: count, Limit: limit, BlockDuration: blockDuration}, nil
	}

	return &BlockStatus{Allowed: true, CurrentCount: count, Limit: limit, BlockDuration: blockDuration}, nil
}

// Close fecha a conexão com o store
func (c *CoreLimiter) Close() error {
	return c.store.Close()
}
