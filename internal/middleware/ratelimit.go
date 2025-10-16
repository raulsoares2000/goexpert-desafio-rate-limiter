package middleware

import (
	// Damos um alias 'corelimiter' para o pacote para evitar conflito
	// com o nome da variável 'limiter' na função abaixo.
	corelimiter "RateLimiter/internal/limiter"
	"net"
	"net/http"
)

// RateLimiterMiddleware cria o nosso middleware.
// O parâmetro continua se chamando 'limiter', mas agora não há mais conflito.
func RateLimiterMiddleware(limiter *corelimiter.RateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var identifier string
			var keyType string

			// 1. Tenta identificar o requisitante pelo Token de Acesso.
			token := r.Header.Get("API_KEY")
			if token != "" {
				identifier = token
				// Usamos o alias para acessar a constante do pacote.
				keyType = corelimiter.TypeToken
			} else {
				// 2. Se não houver token, usa o endereço de IP.
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				identifier = ip
				// Usamos o alias para acessar a constante do pacote.
				keyType = corelimiter.TypeIP
			}

			// 3. Consulta a lógica do limiter (a variável 'limiter').
			allowed, err := limiter.Allow(r.Context(), keyType, identifier)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// 4. Age com base na decisão do limiter.
			if !allowed {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("you have reached the maximum number of requests or actions allowed within a certain time frame"))
				return
			}

			// Se for permitida, passa a requisição para o próximo handler.
			next.ServeHTTP(w, r)
		})
	}
}
