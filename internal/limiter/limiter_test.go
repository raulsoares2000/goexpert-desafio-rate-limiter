package limiter

import (
	"context"
	"testing"
	"time"

	"RateLimiter/configs"
)

// --- Mock do Storage ---
// MockStorage é uma implementação em memória da interface Storage para uso em testes.
type MockStorage struct {
	counts  map[string]int
	blocked map[string]time.Time
}

// NewMockStorage cria um novo mock de storage.
func NewMockStorage() *MockStorage {
	return &MockStorage{
		counts:  make(map[string]int),
		blocked: make(map[string]time.Time),
	}
}

// Implementação dos métodos da interface Storage para o mock.
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
		return false, 0, nil
	}
	return true, time.Until(expireTime), nil
}

// --- Testes do RateLimiter ---
func TestRateLimiter(t *testing.T) {
	// 1. Configuração inicial para os testes
	mockStorage := NewMockStorage()

	// Criamos uma configuração fake para os testes
	cfg := &configs.Config{
		DefaultLimitByIP:    5,
		DefaultLimitByToken: 10,
		BlockTimeInSeconds:  60,
		TokenLimits:         "abc123:2,xyz987:200", // Token com limite super baixo para teste
	}

	// Criamos a instância do RateLimiter com o mock
	rateLimiter := NewRateLimiter(mockStorage, cfg)
	ctx := context.Background()

	// 2. Execução dos cenários de teste
	t.Run("Deve permitir requisições por IP abaixo do limite", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			allowed, err := rateLimiter.Allow(ctx, TypeIP, "192.168.1.1")
			if err != nil {
				t.Fatalf("Erro inesperado: %v", err)
			}
			if !allowed {
				t.Fatalf("Requisição %d foi bloqueada indevidamente", i+1)
			}
		}
	})

	t.Run("Deve bloquear requisição por IP que excede o limite", func(t *testing.T) {
		// A 6ª requisição para este IP
		allowed, err := rateLimiter.Allow(ctx, TypeIP, "192.168.1.1")
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}
		if allowed {
			t.Fatal("Requisição que excedeu o limite foi permitida indevidamente")
		}
	})

	t.Run("Deve continuar bloqueando um IP que já foi bloqueado", func(t *testing.T) {
		allowed, err := rateLimiter.Allow(ctx, TypeIP, "192.168.1.1")
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}
		if allowed {
			t.Fatal("IP bloqueado foi permitido indevidamente")
		}
	})

	t.Run("Deve usar o limite específico do token", func(t *testing.T) {
		// Limite para 'abc123' é 2.
		// Primeira e segunda requisição devem passar.
		rateLimiter.Allow(ctx, TypeToken, "abc123")
		rateLimiter.Allow(ctx, TypeToken, "abc123")

		// Terceira requisição deve ser bloqueada.
		allowed, err := rateLimiter.Allow(ctx, TypeToken, "abc123")
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}
		if allowed {
			t.Fatal("Token com limite específico foi permitido indevidamente após exceder o limite")
		}
	})
}
