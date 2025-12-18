package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockStore implementa LimiterStoreStrategy para testes
type MockStore struct {
	mu         sync.Mutex
	counters   map[string]int64
	expiries   map[string]time.Time
	shouldFail bool
}

// NewMockStore cria uma nova instância de MockStore
func NewMockStore() *MockStore {
	return &MockStore{
		counters: make(map[string]int64),
		expiries: make(map[string]time.Time),
	}
}

// SetShouldFail configura o mock para simular falhas
func (m *MockStore) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// Increment incrementa o contador
func (m *MockStore) Increment(ctx context.Context, key string, expiry time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return 0, fmt.Errorf("mock error: simulated failure")
	}

	// Verifica se expirou
	if exp, exists := m.expiries[key]; exists {
		if time.Now().After(exp) {
			delete(m.counters, key)
			delete(m.expiries, key)
		}
	}

	m.counters[key]++
	m.expiries[key] = time.Now().Add(expiry)

	return m.counters[key], nil
}

// GetCount retorna o contador atual
func (m *MockStore) GetCount(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return 0, fmt.Errorf("mock error: simulated failure")
	}

	// Verifica se expirou
	if exp, exists := m.expiries[key]; exists {
		if time.Now().After(exp) {
			delete(m.counters, key)
			delete(m.expiries, key)
			return 0, nil
		}
	}

	return m.counters[key], nil
}

// Close não faz nada no mock
func (m *MockStore) Close() error {
	return nil
}

// Reset limpa todos os contadores
func (m *MockStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters = make(map[string]int64)
	m.expiries = make(map[string]time.Time)
}

// Exists verifica existência de chave não expirada
func (m *MockStore) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.expiries[key]
	if !ok {
		return false, nil
	}
	if time.Now().After(exp) {
		delete(m.expiries, key)
		return false, nil
	}
	return true, nil
}

// SetExpiring configura expiração para chave (valor ignorado no mock)
func (m *MockStore) SetExpiring(ctx context.Context, key string, _ string, expiry time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldFail {
		return fmt.Errorf("mock error: simulated failure")
	}
	m.expiries[key] = time.Now().Add(expiry)
	return nil
}
