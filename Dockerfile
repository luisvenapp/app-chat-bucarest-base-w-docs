# --- Etapa 1: Builder ---
# Usamos una imagen oficial de Go con Alpine para mantener el tamaño bajo.
# Fijar una versión específica (ej. 1.23) asegura compilaciones reproducibles.
FROM golang:1.23-alpine AS builder

# 1. INSTALAR HERRAMIENTAS
# Solo necesitamos git y el cliente de openssh.
RUN apk add --no-cache git openssh-client

# 2. CONFIGURAR EL ENTORNO
# Establece el directorio de trabajo dentro del contenedor.
WORKDIR /app

# 3. CONFIGURAR GO Y GIT PARA MÓDULOS PRIVADOS
# Le dice a Go que los repos de tu organización son privados y no deben buscarse en proxies públicos.
ENV GOPRIVATE="github.com/Venqis-NolaTech/*"
# Le indica a la herramienta 'go' que use git directamente para estos repos.
ENV GONOPROXY="github.com/Venqis-NolaTech/*"
# La magia de Git: Reemplaza cualquier intento de clonar por HTTPS con SSH.
# Esto es crucial porque 'go get' por defecto intenta usar HTTPS.
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/

# 4. DESCARGAR DEPENDENCIAS (APROVECHANDO LA CACHÉ)
# Copiamos solo los archivos de dependencias primero. Si no cambian, Docker
# usará la capa de caché de este paso, ahorrando mucho tiempo.
COPY go.mod go.sum ./

# 5. EL PASO CLAVE: DESCARGA SEGURA CON MONTAJE SSH
# Este es el núcleo de la solución.
# --mount=type=ssh: Monta de forma segura el socket del agente SSH del host. La clave NUNCA entra al contenedor.
# --mount=type=cache: Usa una caché persistente de Docker para los módulos descargados.
RUN --mount=type=ssh,id=default \
    --mount=type=cache,target=/go/pkg/mod \
    sh -c "mkdir -p -m 0700 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts && go mod download"

# 6. COPIAR Y COMPILAR EL CÓDIGO FUENTE
# Ahora que las dependencias están resueltas, copiamos el resto del código.
COPY . .

# Compilamos la aplicación creando un binario estático y sin información de depuración.
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main .


# --- Etapa 2: Runner ---
# Empezamos desde una imagen mínima para la etapa final. 'scratch' es la más pequeña
# posible, pero 'alpine' es una excelente opción si necesitas certificados SSL o un shell.
FROM alpine:3.20 AS runner

# Instala solo los paquetes absolutamente necesarios para la ejecución.
# ca-certificates es esencial para hacer llamadas HTTPS desde tu aplicación.
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Copia ÚNICAMENTE el binario compilado de la etapa 'builder'.
# La imagen final no contiene código fuente, herramientas de compilación ni claves SSH.
COPY --from=builder /app/main /app/main

RUN mkdir -p /app/config

# Expone el puerto que tu aplicación escucha.
EXPOSE 8080

# Define el usuario no-root para ejecutar la aplicación (Buena práctica de seguridad)
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# El comando para iniciar tu aplicación.
CMD ["/app/main", "--config-file=/app/config/config.json"]
