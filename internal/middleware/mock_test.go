package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/marfebr/go_ratelimit/internal/limiter"
)

// mockStore implementa limiter.LimiterStoreStrategy para testes do middleware
type mockStore struct {
	mu         sync.Mutex
	counters   map[string]int64
	expiries   map[string]time.Time
	shouldFail bool
}

func newMockStore() *mockStore {
	return &mockStore{
		counters: make(map[string]int64),
		expiries: make(map[string]time.Time),
	}
}

func (m *mockStore) Increment(ctx context.Context, key string, expiry time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return 0, fmt.Errorf("mock error")
	}

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

func (m *mockStore) GetCount(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return 0, fmt.Errorf("mock error")
	}

	if exp, exists := m.expiries[key]; exists {
		if time.Now().After(exp) {
			delete(m.counters, key)
			delete(m.expiries, key)
			return 0, nil
		}
	}

	return m.counters[key], nil
}

func (m *mockStore) Close() error {
	return nil
}

var _ limiter.LimiterStoreStrategy = (*mockStore)(nil)

func (m *mockStore) Exists(ctx context.Context, key string) (bool, error) {
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

func (m *mockStore) SetExpiring(ctx context.Context, key string, _ string, expiry time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldFail {
		return fmt.Errorf("mock error")
	}
	m.expiries[key] = time.Now().Add(expiry)
	return nil
}
