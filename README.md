# Desafio Técnico 1
## Objetivo
Desenvolver um rate limiter em Go que possa ser configurado para limitar o número máximo de requisições por segundo com base em um endereço IP específico ou em um token de acesso.

## Descrição
O objetivo deste desafio é criar um rate limiter em Go que possa ser utilizado para controlar o tráfego de requisições para um serviço web. O rate limiter deve ser capaz de limitar o número de requisições com base em dois critérios:

- Endereço IP: O rate limiter deve restringir o número de requisições recebidas de um único endereço IP dentro de um intervalo de tempo definido.
- Token de Acesso: O rate limiter deve também poderá limitar as requisições baseadas em um token de acesso único, permitindo diferentes limites de tempo de expiração para diferentes tokens. O Token deve ser informado no header no seguinte formato:
- API_KEY: <TOKEN>
As configurações de limite do token de acesso devem se sobrepor as do IP. Ex: Se o limite por IP é de 10 req/s e a de um determinado token é de 100 req/s, o rate limiter deve utilizar as informações do token.
Requisitos:

O rate limiter deve poder trabalhar como um middleware que é injetado ao servidor web
O rate limiter deve permitir a configuração do número máximo de requisições permitidas por segundo.
O rate limiter deve ter ter a opção de escolher o tempo de bloqueio do IP ou do Token caso a quantidade de requisições tenha sido excedida.
As configurações de limite devem ser realizadas via variáveis de ambiente ou em um arquivo “.env” na pasta raiz.
Deve ser possível configurar o rate limiter tanto para limitação por IP quanto por token de acesso.
O sistema deve responder adequadamente quando o limite é excedido:
Código HTTP: 429
Mensagem: you have reached the maximum number of requests or actions allowed within a certain time frame
Todas as informações de "limiter” devem ser armazenadas e consultadas de um banco de dados Redis. Você pode utilizar docker-compose para subir o Redis.
Crie uma “strategy” que permita trocar facilmente o Redis por outro mecanismo de persistência.
A lógica do limiter deve estar separada do middleware.
Exemplos:

Limitação por IP: Suponha que o rate limiter esteja configurado para permitir no máximo 5 requisições por segundo por IP. Se o IP 192.168.1.1 enviar 6 requisições em um segundo, a sexta requisição deve ser bloqueada.
Limitação por Token: Se um token abc123 tiver um limite configurado de 10 requisições por segundo e enviar 11 requisições nesse intervalo, a décima primeira deve ser bloqueada.
Nos dois casos acima, as próximas requisições poderão ser realizadas somente quando o tempo total de expiração ocorrer. Ex: Se o tempo de expiração é de 5 minutos, determinado IP poderá realizar novas requisições somente após os 5 minutos.
## Dicas:

Teste seu rate limiter sob diferentes condições de carga para garantir que ele funcione conforme esperado em situações de alto tráfego.
## Entrega:

O código-fonte completo da implementação.
Documentação explicando como o rate limiter funciona e como ele pode ser configurado.
Testes automatizados demonstrando a eficácia e a robustez do rate limiter.
Utilize docker/docker-compose para que possamos realizar os testes de sua aplicação.
O servidor web deve responder na porta 8080.

## Testes de Carga (k6)

Um script k6 está disponível em [scripts/k6/rate-test.js](scripts/k6/rate-test.js) para validar limites por IP e por Token sob tráfego.

Pré-requisitos: Docker (para executar k6 via container) e a aplicação rodando em 8080.

Exemplos de execução:

```bash
# Sem token (valida limite por IP)
docker run --rm -it \
	-e BASE_URL=http://host.docker.internal:8080 \
	-e DURATION=30s -e RPS=15 \
	-v "$PWD/scripts/k6":/scripts grafana/k6:latest run /scripts/rate-test.js

# Com token (defina API_KEY_<TOKEN> no .env da app)
docker run --rm -it \
	-e BASE_URL=http://host.docker.internal:8080 \
	-e API_TOKEN=abc123 -e DURATION=30s -e RPS=100 \
	-v "$PWD/scripts/k6":/scripts grafana/k6:latest run /scripts/rate-test.js
```

Observações:
- Ajuste `RPS` para ficar acima do limite desejado e observar respostas 429.
- Em Linux, substitua `host.docker.internal` por o IP da máquina/`http://172.17.0.1:8080` se necessário.

## Configuração

Variáveis de ambiente suportadas (podem ser definidas via `.env`):
- `REDIS_ADDR`: endereço do Redis (ex.: `localhost:6379`).
- `DEFAULT_RATE_LIMIT_IP`: limite padrão de requisições por segundo por IP (ex.: `5`).
- `DEFAULT_BLOCK_DURATION_SECONDS`: duração do bloqueio em segundos (ex.: `300`).
- `API_KEY_<TOKEN>`: limites específicos por token no formato `LIMITE,BLOQUEIO_SEGUNDOS` (ex.: `API_KEY_abc123=100,60`).

Exemplo de `.env` (veja também [.env.example](.env.example)):

```env
REDIS_ADDR=localhost:6379
DEFAULT_RATE_LIMIT_IP=5
DEFAULT_BLOCK_DURATION_SECONDS=300
# API_KEY_abc123=100,60
```

## Como Executar

### Local (Go + Redis)

```bash
# Baixar dependências
go mod download

# Subir Redis via docker compose
docker compose up -d redis

# Configurar variáveis (ou usar .env)
export REDIS_ADDR=localhost:6379

# Rodar servidor
go run ./cmd/server/main.go
```

### Docker Compose (App + Redis)

```bash
docker compose up --build
```

Aplicação escuta em `0.0.0.0:8080`. Endpoints úteis:
- `/`: resposta JSON simples para testes.
- `/health`: checagem de saúde.

## Exemplos com curl

Abaixo alguns exemplos práticos para validar limites por IP e por Token.

### Sem Token (limite por IP)

Pré-requisitos: Redis e servidor rodando em 8080. Exemplo rápido:

```bash
# Suba o Redis
docker compose up -d redis

# Em outro terminal, rode o servidor com limites padrão (ex.: 5 req/s, bloqueio 60s)
export REDIS_ADDR=localhost:6379
export DEFAULT_RATE_LIMIT_IP=5
export DEFAULT_BLOCK_DURATION_SECONDS=60
go run ./cmd/server/main.go
```

Teste simples: faça várias requisições rápidas para exceder o limite por IP.

```bash
# Envie 10 requisições rápidas e veja os status HTTP
for i in {1..10}; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/; done
# Esperado: primeiras ~5 retornam 200; demais começam a retornar 429

# Para inspecionar o corpo da resposta 429 (mensagem exigida pelo desafio)
curl -i http://localhost:8080/
# Mensagem esperada: "you have reached the maximum number of requests or actions allowed within a certain time frame"
```

Se quiser aumentar a taxa de requisições por segundo, rode em paralelo:

```bash
seq 1 30 | xargs -n1 -P20 -I{} curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/
```

### Com Token (precede o limite por IP)

Defina um limite específico para um token via variável de ambiente `API_KEY_<TOKEN>` no formato `LIMITE,BLOQUEIO_SEGUNDOS`. Exemplo: token `abc123` com 100 req/s e bloqueio de 60s.

```bash
# No mesmo terminal do servidor (ou antes de iniciar), exporte o token
export API_KEY_abc123=100,60

# Reinicie o servidor se necessário para carregar o novo ambiente
go run ./cmd/server/main.go

# Agora faça requisições com o header do token
for i in {1..120}; do curl -s -o /dev/null -w "%{http_code}\n" -H "API_KEY: abc123" http://localhost:8080/; done
# Esperado: ~100 respostas 200 dentro de 1s; o excedente 429

# Corpo da resposta 429
curl -i -H "API_KEY: abc123" http://localhost:8080/
# Mensagem esperada: "you have reached the maximum number of requests or actions allowed within a certain time frame"
```

Observações:
- O limite é por janela de 1 segundo; o bloqueio dura `DEFAULT_BLOCK_DURATION_SECONDS` (ou o valor por token).
- O token sempre sobrepõe o limite por IP quando presente.
- Após o bloqueio expirar, novas requisições voltam a ser permitidas.

## Testes Automatizados

### Unitários rápidos
```bash
go test -short ./...
```

### Integração (Redis real via Testcontainers)
```bash
go test -v -timeout=5m -run Integration ./internal/limiter
```

### Todos os testes (unit + integração)
```bash
go test -v ./internal/...
```