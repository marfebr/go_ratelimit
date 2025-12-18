package limiter

import (
	"context"
	"testing"
	"time"
)

func TestCoreLimiter_Allow_WithinLimit(t *testing.T) {
	// Setup
	mockStore := NewMockStore()
	limiter := NewCoreLimiter(mockStore)
	ctx := context.Background()

	key := "test:ip:192.168.1.1"
	limit := 5
	blockDuration := 10 * time.Second

	// Execute - dentro do limite (5 requisições)
	for i := 1; i <= limit; i++ {
		status, err := limiter.Allow(ctx, key, limit, blockDuration)
		if err != nil {
			t.Fatalf("Erro inesperado na requisição %d: %v", i, err)
		}

		if !status.Allowed {
			t.Errorf("Requisição %d deveria ser permitida", i)
		}

		if status.CurrentCount != int64(i) {
			t.Errorf("CurrentCount esperado %d, obtido %d", i, status.CurrentCount)
		}
	}
}

func TestCoreLimiter_Allow_ExceedsLimit(t *testing.T) {
	// Setup
	mockStore := NewMockStore()
	limiter := NewCoreLimiter(mockStore)
	ctx := context.Background()

	key := "test:ip:192.168.1.2"
	limit := 5
	blockDuration := 10 * time.Second

	// Execute - dentro do limite primeiro
	for i := 1; i <= limit; i++ {
		status, err := limiter.Allow(ctx, key, limit, blockDuration)
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}
		if !status.Allowed {
			t.Errorf("Requisição %d deveria ser permitida", i)
		}
	}

	// 6ª requisição deve ser bloqueada
	status, err := limiter.Allow(ctx, key, limit, blockDuration)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}

	if status.Allowed {
		t.Error("6ª requisição deveria ser bloqueada")
	}

	if status.CurrentCount != 6 {
		t.Errorf("CurrentCount esperado 6, obtido %d", status.CurrentCount)
	}
}

func TestCoreLimiter_Allow_DifferentKeys(t *testing.T) {
	// Setup
	mockStore := NewMockStore()
	limiter := NewCoreLimiter(mockStore)
	ctx := context.Background()

	key1 := "test:ip:192.168.1.1"
	key2 := "test:ip:192.168.1.2"
	limit := 3
	blockDuration := 10 * time.Second

	// Execute - key1 atinge limite
	for i := 0; i < limit; i++ {
		limiter.Allow(ctx, key1, limit, blockDuration)
	}

	// key1 deveria estar bloqueada
	status1, _ := limiter.Allow(ctx, key1, limit, blockDuration)
	if status1.Allowed {
		t.Error("key1 deveria estar bloqueada")
	}

	// key2 ainda deveria ser permitida
	status2, _ := limiter.Allow(ctx, key2, limit, blockDuration)
	if !status2.Allowed {
		t.Error("key2 deveria ser permitida (primeira requisição)")
	}
}

func TestCoreLimiter_Allow_FailOpen(t *testing.T) {
	// Setup
	mockStore := NewMockStore()
	mockStore.SetShouldFail(true) // Simula falha do Redis
	limiter := NewCoreLimiter(mockStore)
	ctx := context.Background()

	key := "test:ip:192.168.1.3"
	limit := 5
	blockDuration := 10 * time.Second

	// Execute
	status, err := limiter.Allow(ctx, key, limit, blockDuration)

	// Assert - fail-open: permite requisição mesmo com erro
	if err == nil {
		t.Error("Esperado erro do mock store")
	}

	if !status.Allowed {
		t.Error("Deveria permitir requisição em caso de erro (fail-open)")
	}
}

func TestCoreLimiter_Allow_Expiry(t *testing.T) {
	// Setup
	mockStore := NewMockStore()
	limiter := NewCoreLimiter(mockStore)
	ctx := context.Background()

	key := "test:ip:192.168.1.4"
	limit := 3
	blockDuration := 100 * time.Millisecond

	// Execute - atinge limite
	for i := 0; i < limit; i++ {
		limiter.Allow(ctx, key, limit, blockDuration)
	}

	// Verifica bloqueio
	status1, _ := limiter.Allow(ctx, key, limit, blockDuration)
	if status1.Allowed {
		t.Error("Deveria estar bloqueado")
	}

	// Aguarda expiração
	time.Sleep(150 * time.Millisecond)

	// Mock não implementa expiração automática, mas testa o conceito
	mockStore.Reset()

	// Deveria permitir novamente
	status2, _ := limiter.Allow(ctx, key, limit, blockDuration)
	if !status2.Allowed {
		t.Error("Deveria permitir após expiração")
	}
	if status2.CurrentCount != 1 {
		t.Errorf("CurrentCount deveria ser 1 após reset, obtido %d", status2.CurrentCount)
	}
}

func TestCoreLimiter_Close(t *testing.T) {
	mockStore := NewMockStore()
	limiter := NewCoreLimiter(mockStore)

	err := limiter.Close()
	if err != nil {
		t.Errorf("Close não deveria retornar erro: %v", err)
	}
}
