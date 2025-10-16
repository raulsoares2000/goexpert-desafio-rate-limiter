package configs

import (
	"github.com/spf13/viper"
)

// Config armazena todas as configurações da aplicação.
// As tags `mapstructure` são usadas pelo Viper para mapear
// as chaves do .env para os campos da struct de forma automática.
type Config struct {
	// Configs do Servidor Web
	WebServerPort string `mapstructure:"WEB_SERVER_PORT"`

	// Configs do Redis
	RedisAddr string `mapstructure:"REDIS_ADDR"`

	// Configs do Rate Limiter
	DefaultLimitByIP    int    `mapstructure:"DEFAULT_LIMIT_BY_IP"`
	DefaultLimitByToken int    `mapstructure:"DEFAULT_LIMIT_BY_TOKEN"`
	BlockTimeInSeconds  int    `mapstructure:"BLOCK_TIME_IN_SECONDS"`
	TokenLimits         string `mapstructure:"TOKEN_LIMITS"` // Será processado depois na lógica do limiter

	// Configs de Banco de Dados (para futura implementação da Strategy)
	DBDriver   string `mapstructure:"DB_DRIVER"`
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
}

// LoadConfig carrega as configurações do arquivo .env ou das variáveis de ambiente.
func LoadConfig(path string) (*Config, error) {
	var cfg *Config

	viper.AddConfigPath(path)
	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.SetConfigFile(".env")

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}

	return cfg, nil
}
