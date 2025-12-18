package limiter

import (
	"context"
	"time"
)

// LimiterStoreStrategy define a interface para persistência do rate limiter
// Permite trocar facilmente o backend (Redis, in-memory, PostgreSQL, etc.)
type LimiterStoreStrategy interface {
	// Increment incrementa o contador para a chave e define expiração
	// Retorna o valor atual após incremento
	Increment(ctx context.Context, key string, expiry time.Duration) (int64, error)

	// GetCount retorna o contador atual para a chave
	GetCount(ctx context.Context, key string) (int64, error)

	// Exists verifica se a chave existe (e não expirou)
	Exists(ctx context.Context, key string) (bool, error)

	// SetExpiring seta um valor com expiração
	SetExpiring(ctx context.Context, key string, value string, expiry time.Duration) error

	// Close fecha a conexão com o backend
	Close() error
}
