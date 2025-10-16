# Rate Limiter em Go

Este projeto é uma implementação de um *Rate Limiter* (limitador de taxa) desenvolvido em Go. Ele foi projetado para ser utilizado como um middleware em um servidor web, controlando o tráfego de requisições para proteger os serviços contra uso abusivo, ataques de negação de serviço (DoS) ou sobrecarga.

O sistema é capaz de limitar o número de requisições com base no endereço IP do cliente ou em um Token de Acesso (API Key) fornecido, utilizando Redis para persistir os dados de contagem e bloqueio de forma eficiente.

## ✨ Principais Funcionalidades

* **Limitação por Endereço IP:** Restringe o número de requisições por segundo de um único IP.
* **Limitação por Token de Acesso:** Permite limites de requisição customizados para diferentes tokens de acesso (API Keys).
* **Precedência de Token:** As configurações de limite por token sempre se sobrepõem às de IP.
* **Configuração Flexível:** Todas as configurações são gerenciadas através de um arquivo `.env`, permitindo fácil alteração sem modificar o código.
* **Armazenamento em Redis:** Utiliza o Redis para um controle de estado rápido, distribuído e persistente.
* **Padrão de Projeto Strategy:** A lógica de persistência é desacoplada através de uma interface (`Storage`), permitindo que o Redis seja facilmente trocado por outro banco de dados no futuro.
* **Arquitetura Desacoplada:** A lógica central do *rate limiter* é separada do middleware HTTP, tornando-a reutilizável e mais fácil de testar.
* **Containerização Completa:** A aplicação e suas dependências (Redis) são totalmente gerenciadas com Docker e Docker Compose, garantindo um ambiente de desenvolvimento e produção consistente e de fácil configuração.

## 🛠️ Tecnologias Utilizadas

* **Linguagem:** Go
* **Banco de Dados:** Redis
* **Containerização:** Docker & Docker Compose
* **Roteador HTTP:** [Chi](https://github.com/go-chi/chi)
* **Gerenciamento de Configuração:** [GodotEnv](https://github.com/joho/godotenv)

## 🚀 Como Executar

O projeto é totalmente containerizado, então tudo o que você precisa ter instalado é o Docker e o Docker Compose.

### 1. Pré-requisitos

* [Docker](https://www.docker.com/get-started)
* [Docker Compose](https://docs.docker.com/compose/install/)

### 2. Configuração do Ambiente

1.  **Clone o repositório:**
    ```bash
    git clone <URL_DO_SEU_REPOSITORIO>
    cd RateLimiter
    ```

2.  **Arquivo de Configuração:**
    Para fins educativos, o arquivo de configuração `.env` já está incluído no repositório. Ele contém todas as variáveis necessárias para a aplicação e é carregado automaticamente pelo Docker Compose.

    Você pode inspecionar ou alterar os valores no arquivo `.env` antes de prosseguir. A configuração padrão é:

    ```dotenv
    # Arquivo: .env

    # Configurações do Servidor Web
    WEB_SERVER_PORT=8080

    # Configurações do Redis
    # O host 'redis' é o nome do serviço definido no docker-compose.yml
    REDIS_ADDR=redis:6379

    # Configurações do Rate Limiter
    DEFAULT_LIMIT_BY_IP=5
    DEFAULT_LIMIT_BY_TOKEN=10
    BLOCK_TIME_IN_SECONDS=60
    # Formato: TOKEN_1:LIMITE_1,TOKEN_2:LIMITE_2
    TOKEN_LIMITS=abc123:100,xyz987:200

    # Configurações do Banco de Dados (para futura implementação da Strategy)
    DB_DRIVER=mysql
    DB_HOST=localhost
    DB_PORT=3306
    DB_USER=root
    DB_PASSWORD=password
    DB_NAME=mydb
    ```

### 3. Subindo a Aplicação

Com o repositório clonado, execute o seguinte comando na raiz do projeto:

```bash
docker-compose up --build
```
O comando ```docker-compose up``` irá construir a imagem da aplicação Go, baixar a imagem do Redis e iniciar os dois containers. A flag ```--build``` garante que a imagem da sua aplicação seja construída do zero.

Após a execução, você verá os logs dos serviços, e a aplicação estará disponível em ```http://localhost:8080```.

Para parar todos os serviços, pressione ```Ctrl + C``` no terminal onde o compose está rodando, e depois execute ```docker-compose down```.

## ⚙️ Uso e Exemplos

Para testar o *rate limiter*, você pode usar uma ferramenta como o `curl`.

### Teste de Limite por IP

Envie requisições repetidas para o servidor. Conforme a configuração padrão (`DEFAULT_LIMIT_BY_IP=5`), a 6ª requisição dentro de um segundo será bloqueada.
```bash
# Este loop enviará 7 requisições
for i in $(seq 1 7); do curl http://localhost:8080/; echo ""; done
```

**Resposta esperada:** As 5 primeiras requisições retornarão `Hello, World!`, e as subsequentes retornarão a mensagem de bloqueio com status HTTP 429.

### Teste de Limite por Token

Envie requisições com o header `API_KEY`. Se um token não tiver um limite específico, o padrão de 10 requisições/segundo será usado. Se um token `abc123` tiver um limite de 10 requisições por segundo, a 11ª requisição será bloqueada.
```bash
# Este loop enviará 12 requisições com um token genérico
for i in $(seq 1 12); do curl -H "API_KEY: um-token-qualquer" http://localhost:8080/; echo ""; done
```

**Resposta esperada:** As 10 primeiras passarão, e as seguintes serão bloqueadas com status HTTP 429.

## ✅ Testes Automatizados

O projeto conta com uma suíte de testes de unidade e integração para garantir sua robustez e eficácia.

### Testes de Unidade

Estes testes validam a lógica de negócio principal (`limiter`) e o middleware de forma isolada, sem dependências externas.

Para executá-los, basta rodar o seguinte comando na raiz do projeto:
```bash
go test ./...
```

### Testes de Integração

Estes testes validam o fluxo completo da aplicação, incluindo a integração com um banco de dados Redis real (executado em um container de teste separado).

1. Inicie o ambiente de teste:
```bash
docker-compose -f docker-compose.test.yml up -d
```

2. Execute os testes:
```bash
go test ./...
```

3. Desligue o ambiente de teste:
```bash
docker-compose -f docker-compose.test.yml down
```

## 📂 Estrutura do Projeto
```
RateLimiter/
├── cmd/server/         # Ponto de entrada da aplicação (função main)
├── configs/            # Lógica de carregamento de configuração
├── internal/
│   ├── limiter/        # Lógica de negócio central do rate limiter
│   ├── middleware/     # Middleware HTTP para integração com o servidor web
│   └── storage/        # Implementação da persistência (interface e Redis)
├── .env                # Arquivo de configuração (local)
├── Dockerfile          # Instruções para construir a imagem da aplicação Go
├── docker-compose.yml  # Orquestrador para o ambiente de desenvolvimento
├── docker-compose.test.yml # Orquestrador para o ambiente de testes de integração
└── go.mod              # Gerenciador de dependências do Go
```