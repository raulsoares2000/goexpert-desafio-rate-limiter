package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"RateLimiter/configs"
	corelimiter "RateLimiter/internal/limiter"
)

// --- Mock do Storage (Copiado para este teste) ---
// MockStorage é uma implementação em memória da interface Storage.
// Usamos ele para controlar o comportamento do RateLimiter sem depender do Redis.
type MockStorage struct {
	counts  map[string]int
	blocked map[string]time.Time
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		counts:  make(map[string]int),
		blocked: make(map[string]time.Time),
	}
}

func (ms *MockStorage) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	ms.counts[key]++
	return ms.counts[key], nil
}

func (ms *MockStorage) SetBlock(ctx context.Context, key string, duration time.Duration) error {
	ms.blocked[key] = time.Now().Add(duration)
	return nil
}

func (ms *MockStorage) IsBlocked(ctx context.Context, key string) (bool, time.Duration, error) {
	expireTime, exists := ms.blocked[key]
	if !exists || time.Now().After(expireTime) {
		delete(ms.blocked, key) // Limpa bloqueios expirados
		return false, 0, nil
	}
	return true, time.Until(expireTime), nil
}

// --- Testes do Middleware ---
func TestRateLimiterMiddleware(t *testing.T) {
	// Handler final que será chamado se o middleware deixar a requisição passar.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("Deve permitir requisição por IP que está abaixo do limite", func(t *testing.T) {
		// 1. Setup
		mockStorage := NewMockStorage()
		cfg := &configs.Config{DefaultLimitByIP: 5}
		// Criamos um RateLimiter REAL, mas com o nosso storage FAKE.
		rateLimiter := corelimiter.NewRateLimiter(mockStorage, cfg)

		// Criamos a cadeia de handlers para o teste.
		middleware := RateLimiterMiddleware(rateLimiter)
		handlerToTest := middleware(nextHandler)

		// 2. Execução
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		// 3. Verificação
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler retornou status errado: recebido %v esperado %v", status, http.StatusOK)
		}
	})

	t.Run("Deve bloquear requisição por IP que excede o limite", func(t *testing.T) {
		// 1. Setup
		mockStorage := NewMockStorage()
		cfg := &configs.Config{
			DefaultLimitByIP:   1, // Limite de apenas 1 requisição
			BlockTimeInSeconds: 60,
		}
		rateLimiter := corelimiter.NewRateLimiter(mockStorage, cfg)
		middleware := RateLimiterMiddleware(rateLimiter)
		handlerToTest := middleware(nextHandler)

		// 2. Execução
		// Primeira requisição (deve passar)
		req1 := httptest.NewRequest("GET", "/", nil)
		req1.RemoteAddr = "192.0.2.1:12345"
		handlerToTest.ServeHTTP(httptest.NewRecorder(), req1)

		// Segunda requisição (deve ser bloqueada)
		req2 := httptest.NewRequest("GET", "/", nil)
		req2.RemoteAddr = "192.0.2.1:12345"
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req2)

		// 3. Verificação
		if status := rr.Code; status != http.StatusTooManyRequests {
			t.Errorf("Handler retornou status errado: recebido %v esperado %v", status, http.StatusTooManyRequests)
		}

		expectedBody := "you have reached the maximum number of requests or actions allowed within a certain time frame"
		if body := rr.Body.String(); body != expectedBody {
			t.Errorf("Handler retornou corpo errado: recebido '%v' esperado '%v'", body, expectedBody)
		}
	})

	t.Run("Deve priorizar o Token sobre o IP e permitir a requisição", func(t *testing.T) {
		// 1. Setup
		mockStorage := NewMockStorage()
		cfg := &configs.Config{
			DefaultLimitByIP:    0, // Limite de IP super restrito
			DefaultLimitByToken: 5, // Limite de Token permissivo
		}
		rateLimiter := corelimiter.NewRateLimiter(mockStorage, cfg)
		middleware := RateLimiterMiddleware(rateLimiter)
		handlerToTest := middleware(nextHandler)

		// 2. Execução
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		req.Header.Set("API_KEY", "my-token") // Requisição com token
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		// 3. Verificação
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler com token foi bloqueado indevidamente: recebido status %v esperado %v", status, http.StatusOK)
		}

		// Verificação extra: o contador foi incrementado para o token, não para o IP
		if count := mockStorage.counts["my-token"]; count != 1 {
			t.Errorf("Contador do token deveria ser 1, mas foi %d", count)
		}
		if count := mockStorage.counts["192.0.2.1"]; count != 0 {
			t.Errorf("Contador do IP deveria ser 0, mas foi %d", count)
		}
	})
}
