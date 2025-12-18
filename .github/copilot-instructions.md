# Instruções Copilot: Rate Limiter em Go

## Visão Geral do Projeto
Um **middleware de Rate Limiter em Go** performático e configurável que protege serviços web de abuso de requisições. Limita tráfego por endereço IP e token de API com persistência em Redis. Todas as respostas são servidas na porta 8080.

**Arquivos Chave**: [docs/context.md](../docs/context.md), [README.md](../README.md)

## Camadas da Arquitetura

### 1. **Estratégia de Persistência** (Infraestrutura)
- **Interface**: `LimiterStoreStrategy` - define o contrato para qualquer backend de armazenamento
- **Implementação**: `RedisStore` usando biblioteca `go-redis/redis`
- **Operações Chave**: Comandos atômicos do Redis (`INCR`, `EXPIRE`, `GET`) para gerenciamento de contadores
- **Padrão**: Strategy pattern permite trocar Redis por outros backends (in-memory, PostgreSQL)

### 2. **Core Limiter Service** (Lógica de Domínio)
- **Usar**: a biblioteca `golang.org/x/time/rate` para lógica de rate limiting
- **Responsabilidade**: Lógica de rate limiting independente de transporte/middleware
- **Lógica de Decisão**: 
  - Limites de Token sobrepõem limites de IP (se header `API_KEY` presente)
  - Verifica se contagem atual excede limite configurado por segundo
  - Incrementa contador e define expiração no Redis
  - Retorna estado de bloqueio com duração de expiração
- **Método Chave**: Aceita parâmetro `key` genérico (IP ou Token) para suportar ambas estratégias de limitação

### 3. **HTTP Middleware** (Transporte)
- **Responsabilidade**: Extrair contexto da requisição e coordenar resposta
- **Operações**:
  - Extrai IP do cliente de `request.RemoteAddr` 
  - Extrai token do header `API_KEY` (formato: `API_KEY: <TOKEN>`)
  - Invoca serviço Core Limiter
  - Retorna **429 Too Many Requests** com mensagem: `"you have reached the maximum number of requests or actions allowed within a certain time frame"` se bloqueado
  - Caso contrário, chama próximo handler

## Padrão de Configuração
- **Fonte**: Variáveis de ambiente + arquivo `.env` (usar `github.com/joho/godotenv`)
- **Variáveis de Ambiente Obrigatórias**:
  - `REDIS_ADDR` - conexão Redis (ex: `localhost:6379`)
  - `DEFAULT_RATE_LIMIT_IP` - máximo de requisições/segundo por IP (padrão: `5`)
  - `DEFAULT_BLOCK_DURATION_SECONDS` - timeout de bloqueio de IP (padrão: `300`)
  - `API_KEY_<TOKEN>` - limites por token (formato: `LIMIT,BLOCK_SECONDS`, ex: `100,60`)

## Detalhes Críticos de Implementação

### Operações Redis
Use **operações atômicas** para consistência entre requisições concorrentes:
```go
// FAÇA: Incremento atômico com expiração
pipe := client.Pipeline()
pipe.Incr(ctx, key)
pipe.Expire(ctx, key, duration)
pipe.Exec(ctx)

// NÃO FAÇA: Chamadas separadas não-atômicas que podem gerar race condition
count := client.Get(ctx, key)
client.Set(ctx, key, count+1, duration)
```

### Resolução de Prioridade IP
1. Se header `API_KEY` presente → usar limite de Token
2. Se não há Token → usar limite de IP  
3. Retornar primeira regra que corresponder; não combinar limites

### Servidor Web
- Usar biblioteca padrão `net/http` (não Gin/Echo) para flexibilidade de middleware
- Dependências mínimas - foco em reusabilidade do middleware em qualquer framework HTTP Go
- Escutar em `0.0.0.0:8080`

## Estrutura de Pastas
```
.
├── cmd/
│   └── server/
│       └── main.go          # Ponto de entrada, configuração do servidor HTTP
├── internal/
│   ├── limiter/
│   │   ├── core.go          # Lógica do serviço CoreLimiter
│   │   ├── store.go         # Interface LimiterStoreStrategy
│   │   └── redis_store.go   # Implementação Redis
│   ├── middleware/
│   │   └── ratelimit.go     # Middleware HTTP
│   └── config/
│       └── config.go        # Carregamento de variáveis de ambiente
├── docker-compose.yml       # Serviços Redis + app
├── Dockerfile               # Build multi-stage Go
├── go.mod / go.sum
├── .env.example
└── README.md
```

## Fluxo de Desenvolvimento

### Configuração Local
```bash
# 1. Carregar dependências
go mod download

# 2. Iniciar Redis via docker-compose
docker-compose up redis

# 3. Executar servidor com variáveis de ambiente
export REDIS_ADDR=localhost:6379
go run ./cmd/server/main.go
```

### Estratégia de Testes
- **Testes Unitários**: Mock do Redis store, testa isolamento da lógica do limiter
- **Testes de Integração**: Container Redis real, fluxo completo do middleware
- **Testes de Carga**: k6 ou Apache JMeter para validação de performance sob tráfego intenso

### Build & Docker
```bash
# Build multi-stage (stage de build cacheado separadamente do runtime)
docker-compose build
docker-compose up
```

## Restrições e Padrões Chave

| Aspecto | Padrão | Por quê |
|---------|--------|--------|
| **Armazenamento** | Redis apenas (strategy pattern para troca) | Operações atômicas, store distribuído para múltiplas instâncias, alto throughput |
| **Concorrência** | Goroutines + atomicidade do Redis | Mutex não necessário - Redis gerencia coordenação distribuída |
| **Tratamento de Erros** | Fail-open (permite requisições se Redis cair) | Previne que Rate Limiter se torne gargalo |
| **Validação de Token** | Não é responsabilidade do Rate Limiter | Serviço de autenticação externo valida; Rate Limiter usa token como chave identificadora |
| **Logging** | Registrar bloqueios/eventos de alto tráfego | Não registrar valores de token (segurança) |

## Erros Comuns a Evitar
1. **Ops não-atômicas no Redis** - Use pipelines ou scripts Lua para atomicidade multi-comando
2. **Extração de IP** - Tratar header X-Forwarded-For atrás de reverse proxy (verificar config)
3. **Precedência de Token** - Verificar que Token sempre sobrepõe IP no fluxo de decisão
4. **Carregamento de configuração** - Garantir que `.env` seja carregado antes da inicialização
5. **Formato de resposta 429** - Mensagem exata deve corresponder à spec, código de status deve ser 429

## Checklist de Testes
- [ ] Bloquear 6ª requisição no limite de 5 req/s por IP
- [ ] Permitir token com limite maior sobrepor limite de IP
- [ ] Persistir bloqueios entre requisições (estado Redis)
- [ ] Expirar bloqueios após duração configurada
- [ ] Retornar exatamente 429 + mensagem especificada quando bloqueado
- [ ] Tratar Redis ausente graciosamente (fail-open)
- [ ] Extrair header API_KEY case-insensitively se necessário
