package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marfebr/go_ratelimit/internal/config"
	"github.com/marfebr/go_ratelimit/internal/limiter"
)

func TestRateLimitMiddleware_AllowedRequest(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	coreLimiter := limiter.NewCoreLimiter(mockStore)
	cfg := &config.Config{
		DefaultRateLimitIP:     10,
		DefaultBlockDurationIP: 300,
		TokenLimits:            make(map[string]config.TokenLimit),
	}

	handler := RateLimitMiddleware(coreLimiter, cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Execute
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Status esperado 200, obtido %d", w.Code)
	}

	if w.Body.String() != "success" {
		t.Errorf("Body esperado 'success', obtido '%s'", w.Body.String())
	}
}

func TestRateLimitMiddleware_BlockedRequest(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	coreLimiter := limiter.NewCoreLimiter(mockStore)
	cfg := &config.Config{
		DefaultRateLimitIP:     3,
		DefaultBlockDurationIP: 300,
		TokenLimits:            make(map[string]config.TokenLimit),
	}

	handler := RateLimitMiddleware(coreLimiter, cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Execute - faz 3 requisições permitidas
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.2:12345"

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Requisição %d deveria ser permitida, status %d", i+1, w.Code)
		}
	}

	// 4ª requisição deve ser bloqueada
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Status esperado 429, obtido %d", w.Code)
	}

	expectedMsg := `{"message": "you have reached the maximum number of requests or actions allowed within a certain time frame"}`
	if w.Body.String() != expectedMsg {
		t.Errorf("Mensagem incorreta. Esperado: %s, Obtido: %s", expectedMsg, w.Body.String())
	}
}

func TestRateLimitMiddleware_TokenOverridesIP(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	coreLimiter := limiter.NewCoreLimiter(mockStore)
	cfg := &config.Config{
		DefaultRateLimitIP:     2, // IP limitado a 2
		DefaultBlockDurationIP: 300,
		TokenLimits: map[string]config.TokenLimit{
			"premium": {Limit: 100, BlockDurationSecs: 60}, // Token permite 100
		},
	}

	handler := RateLimitMiddleware(coreLimiter, cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Execute - faz 5 requisições com token (deveria permitir pois limite do token é 100)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.3:12345"
		req.Header.Set("API_KEY", "premium")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Requisição %d com token deveria ser permitida, status %d", i+1, w.Code)
		}
	}

	// Agora testa sem token - deveria bloquear na 3ª requisição (limite IP = 2)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.4:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Requisição %d sem token deveria ser permitida", i+1)
		}
	}

	// 3ª requisição sem token deve bloquear
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.4:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3ª requisição sem token deveria ser bloqueada, status %d", w.Code)
	}
}

func TestRateLimitMiddleware_UnconfiguredToken(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	coreLimiter := limiter.NewCoreLimiter(mockStore)
	cfg := &config.Config{
		DefaultRateLimitIP:     5,
		DefaultBlockDurationIP: 300,
		TokenLimits:            make(map[string]config.TokenLimit),
	}

	handler := RateLimitMiddleware(coreLimiter, cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Execute - token não configurado deve usar limite de IP
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.5:12345"
	req.Header.Set("API_KEY", "unconfigured_token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Assert - deveria permitir (primeira requisição)
	if w.Code != http.StatusOK {
		t.Errorf("Token não configurado deveria usar limite de IP, status %d", w.Code)
	}
}

func TestExtractIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	ip := extractIP(req)

	if ip != "192.168.1.100" {
		t.Errorf("IP esperado '192.168.1.100', obtido '%s'", ip)
	}
}

func TestExtractIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")

	ip := extractIP(req)

	if ip != "203.0.113.1" {
		t.Errorf("IP esperado '203.0.113.1', obtido '%s'", ip)
	}
}

func TestExtractIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.2")

	ip := extractIP(req)

	if ip != "203.0.113.2" {
		t.Errorf("IP esperado '203.0.113.2', obtido '%s'", ip)
	}
}

func TestExtractIP_Priority(t *testing.T) {
	// X-Forwarded-For tem prioridade sobre X-Real-IP
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("X-Real-IP", "203.0.113.2")

	ip := extractIP(req)

	if ip != "203.0.113.1" {
		t.Errorf("X-Forwarded-For deveria ter prioridade, obtido '%s'", ip)
	}
}
