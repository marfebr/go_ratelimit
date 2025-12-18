package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// setupRedisContainer inicia um container Redis para testes de integração
func setupRedisContainer(t *testing.T) (*redis.RedisContainer, string) {
	ctx := context.Background()

	redisContainer, err := redis.Run(ctx,
		"redis:7-alpine",
		redis.WithSnapshotting(10, 1),
		redis.WithLogLevel(redis.LogLevelVerbose),
	)
	if err != nil {
		t.Fatalf("Erro ao iniciar container Redis: %v", err)
	}

	// Cleanup automático
	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("Erro ao terminar container Redis: %v", err)
		}
	})

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("Erro ao obter endpoint do Redis: %v", err)
	}

	return redisContainer, endpoint
}

func TestRedisStore_Integration_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração em modo short")
	}

	// Setup
	_, endpoint := setupRedisContainer(t)
	
	store, err := NewRedisStore(endpoint)
	if err != nil {
		t.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test:integration:key1"
	expiry := 5 * time.Second

	// Test Increment
	count1, err := store.Increment(ctx, key, expiry)
	if err != nil {
		t.Fatalf("Erro ao incrementar: %v", err)
	}
	if count1 != 1 {
		t.Errorf("Primeiro incremento esperado 1, obtido %d", count1)
	}

	// Test GetCount
	currentCount, err := store.GetCount(ctx, key)
	if err != nil {
		t.Fatalf("Erro ao obter count: %v", err)
	}
	if currentCount != 1 {
		t.Errorf("GetCount esperado 1, obtido %d", currentCount)
	}

	// Test multiple increments
	count2, err := store.Increment(ctx, key, expiry)
	if err != nil {
		t.Fatalf("Erro ao incrementar segunda vez: %v", err)
	}
	if count2 != 2 {
		t.Errorf("Segundo incremento esperado 2, obtido %d", count2)
	}
}

func TestRedisStore_Integration_Expiry(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração em modo short")
	}

	// Setup
	_, endpoint := setupRedisContainer(t)
	
	store, err := NewRedisStore(endpoint)
	if err != nil {
		t.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test:integration:expiry"
	expiry := 2 * time.Second

	// Incrementa contador
	count, err := store.Increment(ctx, key, expiry)
	if err != nil {
		t.Fatalf("Erro ao incrementar: %v", err)
	}
	if count != 1 {
		t.Errorf("Count esperado 1, obtido %d", count)
	}

	// Verifica que existe
	currentCount, err := store.GetCount(ctx, key)
	if err != nil {
		t.Fatalf("Erro ao obter count: %v", err)
	}
	if currentCount != 1 {
		t.Errorf("Count antes da expiração esperado 1, obtido %d", currentCount)
	}

	// Aguarda expiração
	time.Sleep(3 * time.Second)

	// Verifica que expirou
	expiredCount, err := store.GetCount(ctx, key)
	if err != nil {
		t.Fatalf("Erro ao obter count após expiração: %v", err)
	}
	if expiredCount != 0 {
		t.Errorf("Count após expiração esperado 0, obtido %d", expiredCount)
	}
}

func TestRedisStore_Integration_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração em modo short")
	}

	// Setup
	_, endpoint := setupRedisContainer(t)
	
	store, err := NewRedisStore(endpoint)
	if err != nil {
		t.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test:integration:concurrent"
	expiry := 10 * time.Second
	numGoroutines := 10

	// Canal para sincronizar goroutines
	done := make(chan bool, numGoroutines)

	// Executa incrementos concorrentes
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := store.Increment(ctx, key, expiry)
			if err != nil {
				t.Errorf("Erro em incremento concorrente: %v", err)
			}
			done <- true
		}()
	}

	// Aguarda todas as goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verifica count final
	finalCount, err := store.GetCount(ctx, key)
	if err != nil {
		t.Fatalf("Erro ao obter count final: %v", err)
	}
	if finalCount != int64(numGoroutines) {
		t.Errorf("Count final esperado %d, obtido %d", numGoroutines, finalCount)
	}
}

func TestRedisStore_Integration_MultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração em modo short")
	}

	// Setup
	_, endpoint := setupRedisContainer(t)
	
	store, err := NewRedisStore(endpoint)
	if err != nil {
		t.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	expiry := 10 * time.Second

	keys := []string{
		"test:integration:key1",
		"test:integration:key2",
		"test:integration:key3",
	}

	// Incrementa diferentes keys com diferentes valores
	for i, key := range keys {
		for j := 0; j <= i; j++ {
			_, err := store.Increment(ctx, key, expiry)
			if err != nil {
				t.Fatalf("Erro ao incrementar key %s: %v", key, err)
			}
		}
	}

	// Verifica isolamento entre keys
	for i, key := range keys {
		count, err := store.GetCount(ctx, key)
		if err != nil {
			t.Fatalf("Erro ao obter count para key %s: %v", key, err)
		}
		expected := int64(i + 1)
		if count != expected {
			t.Errorf("Key %s: count esperado %d, obtido %d", key, expected, count)
		}
	}
}

func TestCoreLimiter_Integration_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração em modo short")
	}

	// Setup
	_, endpoint := setupRedisContainer(t)
	
	store, err := NewRedisStore(endpoint)
	if err != nil {
		t.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	defer store.Close()

	limiter := NewCoreLimiter(store)
	defer limiter.Close()

	ctx := context.Background()
	key := "test:integration:limiter"
	limit := 5
	blockDuration := 3 * time.Second

	// Fase 1: Dentro do limite
	for i := 1; i <= limit; i++ {
		status, err := limiter.Allow(ctx, key, limit, blockDuration)
		if err != nil {
			t.Fatalf("Erro na requisição %d: %v", i, err)
		}
		if !status.Allowed {
			t.Errorf("Requisição %d deveria ser permitida", i)
		}
		if status.CurrentCount != int64(i) {
			t.Errorf("Requisição %d: count esperado %d, obtido %d", i, i, status.CurrentCount)
		}
	}

	// Fase 2: Excede limite
	status, err := limiter.Allow(ctx, key, limit, blockDuration)
	if err != nil {
		t.Fatalf("Erro ao exceder limite: %v", err)
	}
	if status.Allowed {
		t.Error("Requisição após limite deveria ser bloqueada")
	}
	if status.CurrentCount != 6 {
		t.Errorf("Count após limite esperado 6, obtido %d", status.CurrentCount)
	}

	// Fase 3: Ainda bloqueado
	status2, err := limiter.Allow(ctx, key, limit, blockDuration)
	if err != nil {
		t.Fatalf("Erro ao verificar bloqueio: %v", err)
	}
	if status2.Allowed {
		t.Error("Deveria continuar bloqueado")
	}

	// Fase 4: Aguarda expiração
	t.Log("Aguardando expiração do bloqueio...")
	time.Sleep(4 * time.Second)

	// Fase 5: Deveria permitir novamente
	status3, err := limiter.Allow(ctx, key, limit, blockDuration)
	if err != nil {
		t.Fatalf("Erro após expiração: %v", err)
	}
	if !status3.Allowed {
		t.Error("Deveria permitir após expiração")
	}
	if status3.CurrentCount != 1 {
		t.Errorf("Count após expiração esperado 1, obtido %d", status3.CurrentCount)
	}
}

func TestRedisStore_Integration_ConnectionFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração em modo short")
	}

	// Tenta conectar em um endereço inválido
	_, err := NewRedisStore("localhost:9999")
	if err == nil {
		t.Error("Esperado erro ao conectar em endereço inválido")
	}
}
