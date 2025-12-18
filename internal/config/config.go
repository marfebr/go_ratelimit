package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config armazena as configurações da aplicação
type Config struct {
	RedisAddr              string
	DefaultRateLimitIP     int
	DefaultBlockDurationIP int
	TokenLimits            map[string]TokenLimit
}

// TokenLimit define limite e duração de bloqueio para um token específico
type TokenLimit struct {
	Limit             int
	BlockDurationSecs int
}

// LoadConfig carrega configurações de variáveis de ambiente e .env
func LoadConfig() (*Config, error) {
	// Tenta carregar .env (ignora erro se não existir)
	_ = godotenv.Load()

	cfg := &Config{
		TokenLimits: make(map[string]TokenLimit),
	}

	// Redis Address (obrigatório)
	cfg.RedisAddr = os.Getenv("REDIS_ADDR")
	if cfg.RedisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR não configurado")
	}

	// Default Rate Limit IP
	rateLimitStr := os.Getenv("DEFAULT_RATE_LIMIT_IP")
	if rateLimitStr == "" {
		cfg.DefaultRateLimitIP = 5 // padrão
	} else {
		limit, err := strconv.Atoi(rateLimitStr)
		if err != nil {
			return nil, fmt.Errorf("DEFAULT_RATE_LIMIT_IP inválido: %w", err)
		}
		cfg.DefaultRateLimitIP = limit
	}

	// Default Block Duration IP
	blockDurationStr := os.Getenv("DEFAULT_BLOCK_DURATION_SECONDS")
	if blockDurationStr == "" {
		cfg.DefaultBlockDurationIP = 300 // padrão 5 minutos
	} else {
		duration, err := strconv.Atoi(blockDurationStr)
		if err != nil {
			return nil, fmt.Errorf("DEFAULT_BLOCK_DURATION_SECONDS inválido: %w", err)
		}
		cfg.DefaultBlockDurationIP = duration
	}

	// Carrega limites de tokens (API_KEY_<TOKEN>=LIMIT,BLOCK_SECONDS)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "API_KEY_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			tokenKey := strings.TrimPrefix(parts[0], "API_KEY_")
			valueParts := strings.Split(parts[1], ",")
			if len(valueParts) != 2 {
				return nil, fmt.Errorf("formato inválido para %s (esperado: LIMIT,BLOCK_SECONDS)", parts[0])
			}

			limit, err := strconv.Atoi(strings.TrimSpace(valueParts[0]))
			if err != nil {
				return nil, fmt.Errorf("limite inválido para %s: %w", parts[0], err)
			}

			blockDuration, err := strconv.Atoi(strings.TrimSpace(valueParts[1]))
			if err != nil {
				return nil, fmt.Errorf("block duration inválido para %s: %w", parts[0], err)
			}

			cfg.TokenLimits[tokenKey] = TokenLimit{
				Limit:             limit,
				BlockDurationSecs: blockDuration,
			}
		}
	}

	return cfg, nil
}

// GetTokenLimit retorna o limite configurado para um token específico
func (c *Config) GetTokenLimit(token string) (TokenLimit, bool) {
	limit, exists := c.TokenLimits[token]
	return limit, exists
}
