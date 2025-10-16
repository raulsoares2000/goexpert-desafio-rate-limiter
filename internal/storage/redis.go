package storage

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

// RedisStorage é a implementação da ‘interface’ Storage que utiliza o Redis como backend.
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage cria e retorna uma nova instância de RedisStorage.
// Ele estabelece a conexão com o Redis e verifica se está ativa.
func NewRedisStorage(addr string) (*RedisStorage, error) {
	// Cria um cliente Redis com o endereço fornecido.
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Usa Ping para verificar se a conexão com o Redis foi bem-sucedida.
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("não foi possível conectar ao Redis: %w", err)
	}

	return &RedisStorage{client: client}, nil
}

// Increment incrementa o contador de requisições para uma chave no Redis.
// A operação é atômica e a chave expira após a janela de tempo definida.
func (rs *RedisStorage) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	// Usamos um prefixo para organizar as chaves de contagem no Redis.
	requestKey := fmt.Sprintf("requests:%s", key)

	// Usamos um pipeline para garantir que os comandos INCR e EXPIRE
	// sejam executados de forma atômica (um após o outro, sem interrupções).
	pipe := rs.client.TxPipeline()

	// Incrementa a chave. Se a chave não existir, ela é criada com valor 1.
	countCmd := pipe.Incr(ctx, requestKey)

	// Define o tempo de expiração para a chave. Isso só funciona se a chave for nova.
	// Se a chave já existir, o EXPIRE não faria nada, por isso o pipeline é importante.
	pipe.Expire(ctx, requestKey, window)

	// Executa os comandos na pipeline.
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	// Retorna o valor atual do contador.
	count, err := countCmd.Result()
	return int(count), err
}

// SetBlock cria uma chave no Redis para sinalizar que um IP/‘Token’ está bloqueado.
func (rs *RedisStorage) SetBlock(ctx context.Context, key string, duration time.Duration) error {
	// Usamos um prefixo diferente para as chaves de bloqueio.
	blockedKey := fmt.Sprintf("blocked:%s", key)

	// Cria a chave de bloqueio com um valor qualquer ("1") e o tempo de expiração.
	// O comando Set do Redis com expiração é atômico.
	return rs.client.Set(ctx, blockedKey, "1", duration).Err()
}

// IsBlocked verifica se a chave de bloqueio para um IP/Token existe no Redis.
func (rs *RedisStorage) IsBlocked(ctx context.Context, key string) (bool, time.Duration, error) {
	blockedKey := fmt.Sprintf("blocked:%s", key)

	// Pega o tempo de vida restante (TTL - Time To Live) da chave de bloqueio.
	ttl, err := rs.client.TTL(ctx, blockedKey).Result()
	if err != nil {
		return false, 0, err
	}

	// Se o TTL for -2, significa que a chave não existe, portanto não está bloqueado.
	// Qualquer valor maior que 0 significa que está bloqueado.
	if ttl > 0 {
		return true, ttl, nil
	}

	return false, 0, nil
}
