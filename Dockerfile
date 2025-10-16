# --- Estágio de Build ---
# Usamos uma imagem oficial do Go com o Alpine Linux, que é leve.
FROM golang:1.25-alpine AS builder

# Define o diretório de trabalho dentro do container.
WORKDIR /app

# Copia os arquivos de gerenciamento de dependências primeiro.
# Isso aproveita o cache do Docker. Se esses arquivos não mudarem,
# o Docker não baixará as dependências novamente.
COPY go.mod go.sum ./

# Baixa todas as dependências do projeto.
RUN go mod download

# Copia todo o resto do código-fonte do projeto.
COPY . .

# Compila a aplicação.
# CGO_ENABLED=0 cria um binário estático, sem depender de bibliotecas do sistema.
# GOOS=linux garante que o executável seja para Linux (o sistema do nosso container final).
RUN CGO_ENABLED=0 GOOS=linux go build -o /server cmd/server/main.go

# --- Estágio de Produção ---
# Começamos com uma imagem Alpine zerada, que é extremamente pequena (~5MB).
FROM alpine:latest AS production

# Define o diretório de trabalho.
WORKDIR /app

# Copia o arquivo .env da raiz do projeto para a raiz da aplicação no container.
COPY .env .
# Copia APENAS o binário compilado do estágio 'builder'.
# Nada de código-fonte ou ferramentas de compilação irão para a imagem final.
COPY --from=builder /server .

# Expõe a porta 8080, informando ao Docker que o container
# escutará nesta porta em tempo de execução.
EXPOSE 8080

# Define o comando que será executado quando o container iniciar.
CMD ["/app/server"]