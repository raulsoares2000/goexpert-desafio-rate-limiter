// Arquivo: RateLimiter/cmd/server/main_test.go
package main

import (
	"RateLimiter/configs"
	corelimiter "RateLimiter/internal/limiter"
	"RateLimiter/internal/middleware"
	"RateLimiter/internal/storage"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
)

var testRedisClient *redis.Client

// TestMain é uma função especial que roda ANTES de todos os testes neste pacote.
// É o lugar perfeito para configurar o ambiente de teste (como a conexão com o DB)
// e para limpar depois.
func TestMain(m *testing.M) {
	// Endereço do nosso Redis de teste, iniciado pelo docker-compose.test.yml
	redisTestAddr := "localhost:6380"

	testRedisClient = redis.NewClient(&redis.Options{
		Addr: redisTestAddr,
	})

	// Garante que a conexão está funcionando antes de rodar os testes
	if err := testRedisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Não foi possível conectar ao Redis de teste em %s: %v", redisTestAddr, err)
	}

	// Roda todos os testes do pacote
	exitCode := m.Run()

	// Encerra o processo de teste com o código de saída correto
	os.Exit(exitCode)
}

func TestIntegrationRateLimiter(t *testing.T) {
	// Garante que o Redis esteja limpo antes de cada sub-teste
	t.Cleanup(func() {
		testRedisClient.FlushAll(context.Background())
	})

	// 1. Setup da aplicação para o teste
	// Criamos uma configuração fake, mas o endereço do Redis é real.
	cfg := &configs.Config{
		WebServerPort:       "8080", // Irrelevante, pois o servidor de teste usa porta aleatória
		RedisAddr:           "localhost:6380",
		DefaultLimitByIP:    2, // Limite baixo para facilitar o teste
		DefaultLimitByToken: 3,
		BlockTimeInSeconds:  60,
	}

	// Montamos toda a nossa aplicação, exatamente como no main.go real
	storage, _ := storage.NewRedisStorage(cfg.RedisAddr)
	rateLimiter := corelimiter.NewRateLimiter(storage, cfg)
	router := chi.NewRouter()
	router.Use(middleware.RateLimiterMiddleware(rateLimiter))
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	// httptest.NewServer inicia um servidor HTTP real em uma porta disponível do sistema.
	server := httptest.NewServer(router)
	defer server.Close() // Garante que o servidor seja desligado ao final do teste

	// 2. Cenário de Teste: Excedendo o limite de IP
	t.Run("Deve bloquear requisição por IP após exceder o limite", func(t *testing.T) {
		// As duas primeiras requisições devem passar (limite = 2)
		for i := 0; i < 2; i++ {
			res, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("Erro ao fazer a requisição: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("Esperado status 200 OK, recebido: %d", res.StatusCode)
			}
		}

		// A terceira requisição deve ser bloqueada
		res, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Erro ao fazer a requisição: %v", err)
		}
		if res.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("Esperado status 429 Too Many Requests, recebido: %d", res.StatusCode)
		}
	})
}
