package storage

import (
	"context"
	"time"
)

// Storage é a interface que define o contrato para o nosso mecanismo de persistência.
// Qualquer implementação de armazenamento (Redis, em memória, etc.) deve satisfazer esta interface.
// Isso permite que a lógica do rate limiter seja desacoplada do armazenamento subjacente.
type Storage interface {
	// Increment incrementa o contador de requisições para uma chave específica (IP ou token)
	// e retorna o novo valor. A chave deve expirar após a janela de tempo definida (window).
	// Esta operação deve ser atômica.
	Increment(ctx context.Context, key string, window time.Duration) (int, error)

	// SetBlock bloqueia uma chave por um período específico (duration).
	// Enquanto a chave estiver bloqueada, novas requisições devem ser negadas.
	SetBlock(ctx context.Context, key string, duration time.Duration) error

	// IsBlocked verifica se uma chave está atualmente bloqueada.
	// Retorna true se estiver bloqueada, junto com o tempo restante do bloqueio (TTL).
	IsBlocked(ctx context.Context, key string) (bool, time.Duration, error)
}
