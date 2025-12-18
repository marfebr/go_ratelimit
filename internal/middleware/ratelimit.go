package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/marfebr/go_ratelimit/internal/config"
	"github.com/marfebr/go_ratelimit/internal/limiter"
)

// RateLimitMiddleware cria um middleware de rate limiting
func RateLimitMiddleware(coreLimiter *limiter.CoreLimiter, cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()

			// Extrai token do header API_KEY
			apiKey := r.Header.Get("API_KEY")
			
			var key string
			var limit int
			var blockDuration time.Duration

			// Prioridade: Token > IP
			if apiKey != "" {
				// Verifica se existe configuração para este token
				if tokenLimit, exists := cfg.GetTokenLimit(apiKey); exists {
					key = "token:" + apiKey
					limit = tokenLimit.Limit
					blockDuration = time.Duration(tokenLimit.BlockDurationSecs) * time.Second
				} else {
					// Token não configurado, usa limite de IP
					key = "ip:" + extractIP(r)
					limit = cfg.DefaultRateLimitIP
					blockDuration = time.Duration(cfg.DefaultBlockDurationIP) * time.Second
				}
			} else {
				// Sem token, usa limite de IP
				key = "ip:" + extractIP(r)
				limit = cfg.DefaultRateLimitIP
				blockDuration = time.Duration(cfg.DefaultBlockDurationIP) * time.Second
			}

			// Verifica rate limit
			status, err := coreLimiter.Allow(ctx, key, limit, blockDuration)
			if err != nil {
				// Fail-open: em caso de erro, permite requisição
				next.ServeHTTP(w, r)
				return
			}

			// Se bloqueado, retorna 429
			if !status.Allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"message": "you have reached the maximum number of requests or actions allowed within a certain time frame"}`))
				return
			}

			// Permite requisição
			next.ServeHTTP(w, r)
		})
	}
}

// extractIP extrai o endereço IP da requisição
func extractIP(r *http.Request) string {
	// Verifica header X-Forwarded-For (comum em reverse proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Pega o primeiro IP da lista
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Verifica X-Real-IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback para RemoteAddr
	ip := r.RemoteAddr
	// Remove porta se presente
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
