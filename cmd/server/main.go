package main

import (
	"log"
	"net/http"

	"RateLimiter/configs"
	corelimiter "RateLimiter/internal/limiter"
	"RateLimiter/internal/middleware"
	"RateLimiter/internal/storage"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	// 1. Carrega as configurações da aplicação.
	// Usamos "." para indicar que o .env está na pasta raiz do projeto.
	cfg, err := configs.LoadConfig(".")
	if err != nil {
		// A função LoadConfig já usa panic, mas mantemos o log para clareza.
		log.Fatalf("Erro ao carregar a configuração: %v", err)
	}

	// 2. Inicializa a camada de armazenamento (storage).
	// Aqui estamos criando a implementação com Redis.
	strg, err := storage.NewRedisStorage(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Erro ao inicializar o storage com Redis: %v", err)
	}

	// 3. Inicializa a lógica central do rate limiter.
	// Injetamos o storage e as configurações.
	rateLimiter := corelimiter.NewRateLimiter(strg, cfg)

	// 4. Cria um novo roteador usando o chi.
	router := chi.NewRouter()

	// 5. Aplica os middlewares. A ordem é importante.
	// Logger: para registrar cada requisição no console.
	router.Use(chimiddleware.Logger)
	// Recoverer: para evitar que a aplicação quebre em caso de pânico em um handler.
	router.Use(chimiddleware.Recoverer)
	// Nosso middleware customizado de Rate Limit.
	router.Use(middleware.RateLimiterMiddleware(rateLimiter))

	// 6. Define uma rota de teste.
	// Todas as requisições para esta rota passarão primeiro pelos middlewares acima.
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// 7. Inicia o servidor web.
	log.Printf("Servidor iniciado e ouvindo na porta %s", cfg.WebServerPort)
	if err := http.ListenAndServe(":"+cfg.WebServerPort, router); err != nil {
		log.Fatalf("Não foi possível iniciar o servidor: %v", err)
	}
}
