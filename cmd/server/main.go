package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marfebr/go_ratelimit/internal/config"
	"github.com/marfebr/go_ratelimit/internal/limiter"
	"github.com/marfebr/go_ratelimit/internal/middleware"
)

func main() {
	// Carrega configuração
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	log.Printf("Configuração carregada:")
	log.Printf("  Redis: %s", cfg.RedisAddr)
	log.Printf("  Rate Limit IP: %d req/s", cfg.DefaultRateLimitIP)
	log.Printf("  Block Duration IP: %d segundos", cfg.DefaultBlockDurationIP)
	log.Printf("  Tokens configurados: %d", len(cfg.TokenLimits))

	// Inicializa Redis Store
	redisStore, err := limiter.NewRedisStore(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	defer redisStore.Close()

	log.Println("Conectado ao Redis com sucesso")

	// Cria Core Limiter
	coreLimiter := limiter.NewCoreLimiter(redisStore)
	defer coreLimiter.Close()

	// Configura rotas
	mux := http.NewServeMux()
	
	// Endpoint de teste
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Rate Limiter funcionando!", "status": "ok"}`))
	})

	// Endpoint de health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	// Aplica middleware de rate limiting
	handler := middleware.RateLimitMiddleware(coreLimiter, cfg)(mux)

	// Configura servidor
	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: handler,
	}

	// Captura sinais de shutdown e encerra graciosamente
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Encerrando servidor...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Erro ao encerrar servidor graciosamente: %v", err)
		}
	}()

	// Inicia servidor
	log.Printf("Servidor iniciado na porta 8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}

	log.Println("Servidor encerrado")
}
