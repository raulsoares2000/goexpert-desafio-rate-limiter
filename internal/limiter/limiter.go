package limiter

import (
	"context"
	"strconv"
	"strings"
	"time"

	"RateLimiter/configs"
	"RateLimiter/internal/storage"
)

// Constantes para definir o tipo de chave do limiter
const (
	TypeIP    = "IP"
	TypeToken = "TOKEN"
)

// RateLimiter é a estrutura central que contém a lógica de limitação.
// Ele é desacoplado de qualquer camada de transporte (como HTTP).
type RateLimiter struct {
	storage        storage.Storage
	limitByIP      int
	limitByToken   int
	blockTime      time.Duration
	tokenLimitsMap map[string]int
}

// NewRateLimiter cria e configura uma nova instância do RateLimiter.
func NewRateLimiter(st storage.Storage, cfg *configs.Config) *RateLimiter {
	// Processa a string de limites de 'token' do arquivo de configuração
	// e a transforma num mapa para acesso rápido.
	tokenLimitsMap := make(map[string]int)
	if cfg.TokenLimits != "" {
		pairs := strings.Split(cfg.TokenLimits, ",")
		for _, pair := range pairs {
			parts := strings.Split(strings.TrimSpace(pair), ":")
			if len(parts) == 2 {
				limit, err := strconv.Atoi(parts[1])
				if err == nil {
					tokenLimitsMap[parts[0]] = limit
				}
			}
		}
	}

	return &RateLimiter{
		storage:        st,
		limitByIP:      cfg.DefaultLimitByIP,
		limitByToken:   cfg.DefaultLimitByToken,
		blockTime:      time.Duration(cfg.BlockTimeInSeconds) * time.Second,
		tokenLimitsMap: tokenLimitsMap,
	}
}

// Allow verifica se uma requisição para um determinado identificador deve ser permitida.
// Retorna 'true' se permitida, 'false' se bloqueada.
func (rl *RateLimiter) Allow(ctx context.Context, keyType string, identifier string) (bool, error) {
	// 1. Primeira verificação: o identificador já está bloqueado?
	isBlocked, _, err := rl.storage.IsBlocked(ctx, identifier)
	if err != nil {
		// Se houver um erro ao consultar o storage, por segurança, bloqueamos a requisição.
		return false, err
	}
	if isBlocked {
		return false, nil // Bloqueado, nega a requisição imediatamente.
	}

	// 2. Determinar qual limite aplicar com base no tipo de chave.
	limit := rl.getLimitForKey(keyType, identifier)

	// 3. Incrementar o contador de requisições no storage.
	// A janela de tempo é de 1 segundo, pois o limite é por segundo.
	count, err := rl.storage.Increment(ctx, identifier, 1*time.Second)
	if err != nil {
		return false, err
	}

	// 4. Tomar a decisão: o contador ultrapassou o limite?
	if count > limit {
		// Se ultrapassou, bloqueia o identificador pelo tempo configurado.
		if err := rl.storage.SetBlock(ctx, identifier, rl.blockTime); err != nil {
			return false, err
		}
		return false, nil // Bloqueia esta requisição.
	}

	// Se chegou até aqui, a requisição está dentro do limite.
	return true, nil
}

// getLimitForKey é um método auxiliar que retorna o limite correto para a chave.
func (rl *RateLimiter) getLimitForKey(keyType string, identifier string) int {
	if keyType == TypeToken {
		// Verifica se existe um limite customizado para este token específico.
		if limit, ok := rl.tokenLimitsMap[identifier]; ok {
			return limit // Usa o limite específico do token.
		}
		// Se não, usa o limite padrão para ‘tokens’.
		return rl.limitByToken
	}

	// Para qualquer outro caso (IP), usa o limite padrão de IP.
	return rl.limitByIP
}
