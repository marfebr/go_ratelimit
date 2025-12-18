package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	// Setup
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("DEFAULT_RATE_LIMIT_IP", "10")
	os.Setenv("DEFAULT_BLOCK_DURATION_SECONDS", "600")
	os.Setenv("API_KEY_token1", "100,60")
	os.Setenv("API_KEY_token2", "200,120")
	defer func() {
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("DEFAULT_RATE_LIMIT_IP")
		os.Unsetenv("DEFAULT_BLOCK_DURATION_SECONDS")
		os.Unsetenv("API_KEY_token1")
		os.Unsetenv("API_KEY_token2")
	}()

	// Execute
	cfg, err := LoadConfig()

	// Assert
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("RedisAddr esperado 'localhost:6379', obtido '%s'", cfg.RedisAddr)
	}

	if cfg.DefaultRateLimitIP != 10 {
		t.Errorf("DefaultRateLimitIP esperado 10, obtido %d", cfg.DefaultRateLimitIP)
	}

	if cfg.DefaultBlockDurationIP != 600 {
		t.Errorf("DefaultBlockDurationIP esperado 600, obtido %d", cfg.DefaultBlockDurationIP)
	}

	if len(cfg.TokenLimits) != 2 {
		t.Errorf("Esperado 2 tokens configurados, obtido %d", len(cfg.TokenLimits))
	}

	// Verifica token1
	if limit, exists := cfg.TokenLimits["token1"]; exists {
		if limit.Limit != 100 {
			t.Errorf("Token1 limite esperado 100, obtido %d", limit.Limit)
		}
		if limit.BlockDurationSecs != 60 {
			t.Errorf("Token1 block duration esperado 60, obtido %d", limit.BlockDurationSecs)
		}
	} else {
		t.Error("Token1 não encontrado")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Setup - apenas REDIS_ADDR obrigatório
	os.Setenv("REDIS_ADDR", "localhost:6379")
	defer os.Unsetenv("REDIS_ADDR")

	// Execute
	cfg, err := LoadConfig()

	// Assert
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}

	if cfg.DefaultRateLimitIP != 5 {
		t.Errorf("Padrão DefaultRateLimitIP esperado 5, obtido %d", cfg.DefaultRateLimitIP)
	}

	if cfg.DefaultBlockDurationIP != 300 {
		t.Errorf("Padrão DefaultBlockDurationIP esperado 300, obtido %d", cfg.DefaultBlockDurationIP)
	}
}

func TestLoadConfig_MissingRedisAddr(t *testing.T) {
	// Setup
	os.Unsetenv("REDIS_ADDR")

	// Execute
	_, err := LoadConfig()

	// Assert
	if err == nil {
		t.Error("Esperado erro quando REDIS_ADDR não está configurado")
	}
}

func TestLoadConfig_InvalidRateLimit(t *testing.T) {
	// Setup
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("DEFAULT_RATE_LIMIT_IP", "invalid")
	defer func() {
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("DEFAULT_RATE_LIMIT_IP")
	}()

	// Execute
	_, err := LoadConfig()

	// Assert
	if err == nil {
		t.Error("Esperado erro com valor inválido para DEFAULT_RATE_LIMIT_IP")
	}
}

func TestLoadConfig_InvalidTokenFormat(t *testing.T) {
	// Setup
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("API_KEY_badtoken", "100") // formato inválido (falta block duration)
	defer func() {
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("API_KEY_badtoken")
	}()

	// Execute
	_, err := LoadConfig()

	// Assert
	if err == nil {
		t.Error("Esperado erro com formato inválido de token")
	}
}

func TestGetTokenLimit(t *testing.T) {
	cfg := &Config{
		TokenLimits: map[string]TokenLimit{
			"token1": {Limit: 100, BlockDurationSecs: 60},
		},
	}

	// Test existing token
	limit, exists := cfg.GetTokenLimit("token1")
	if !exists {
		t.Error("Token1 deveria existir")
	}
	if limit.Limit != 100 {
		t.Errorf("Limite esperado 100, obtido %d", limit.Limit)
	}

	// Test non-existing token
	_, exists = cfg.GetTokenLimit("nonexistent")
	if exists {
		t.Error("Token inexistente não deveria ser encontrado")
	}
}
