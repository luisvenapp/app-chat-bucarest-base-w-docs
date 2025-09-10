# 📄 Documentación: Dockerfile

## 🎯 Propósito
Configuración de contenedor Docker optimizada para aplicaciones Go con soporte para repositorios privados y build multi-etapa.

## 🏗️ Arquitectura Multi-Stage

### 📦 Etapa 1: Builder
```dockerfile
FROM golang:1.23-alpine AS builder
```

#### Características
- **Base**: Alpine Linux para tamaño mínimo
- **Go Version**: 1.23 para features modernas
- **Propósito**: Compilación del código fuente

#### Herramientas Instaladas
- `git` - Control de versiones
- `openssh-client` - Acceso a repositorios privados

### 🚀 Etapa 2: Runner
```dockerfile
FROM alpine:3.20 AS runner
```

#### Características
- **Base**: Alpine 3.20 ultra-ligero
- **Propósito**: Ejecución de la aplicación
- **Tamaño**: ~10MB base + binario

## 🔐 Configuración de Repositorios Privados

### Variables de Entorno
```dockerfile
ENV GOPRIVATE="github.com/Venqis-NolaTech/*"
ENV GONOPROXY="github.com/Venqis-NolaTech/*"
```

### Configuración Git
```dockerfile
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/
```

#### Propósito
- **GOPRIVATE**: Evita proxy público para repos privados
- **GONOPROXY**: Fuerza descarga directa
- **Git Config**: Redirige HTTPS a SSH para autenticación

## 🔑 Gestión de SSH Keys

### Montaje Seguro
```dockerfile
RUN --mount=type=ssh,id=default \
    --mount=type=cache,target=/go/pkg/mod \
    sh -c "mkdir -p -m 0700 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts && go mod download"
```

#### Características de Seguridad
- **SSH Mount**: Keys nunca entran al contenedor
- **Known Hosts**: Verificación de identidad de GitHub
- **Permisos**: 0700 para directorio SSH
- **Cache**: Persistencia de módulos descargados

## 📦 Proceso de Build

### Descarga de Dependencias
```dockerfile
COPY go.mod go.sum ./
RUN --mount=type=ssh,id=default \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download
```

### Compilación
```dockerfile
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main .
```

#### Optimizaciones
- **CGO_ENABLED=0**: Binario estático
- **GOOS=linux**: Target específico
- **-ldflags="-s -w"**: Elimina símbolos de debug
- **Cache**: Reutiliza builds anteriores

## 🛡️ Configuración de Seguridad

### Usuario No-Root
```dockerfile
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
```

### Permisos Mínimos
- Usuario dedicado para la aplicación
- Sin privilegios de administrador
- Grupo específico para la app

## 📁 Estructura del Contenedor

### Directorios
```dockerfile
WORKDIR /app
RUN mkdir -p /app/config
```

### Archivos
- `/app/main` - Binario de la aplicación
- `/app/config/` - Directorio de configuración

## 🌐 Configuración de Red

### Puertos Expuestos
```dockerfile
EXPOSE 8080
```

### Certificados SSL
```dockerfile
RUN apk add --no-cache ca-certificates tzdata curl
```

#### Paquetes Instalados
- **ca-certificates**: Para conexiones HTTPS
- **tzdata**: Información de zonas horarias
- **curl**: Health checks y debugging

## 🚀 Comando de Inicio

```dockerfile
CMD ["/app/main", "--config-file=/app/config/config.json"]
```

### Características
- **Configuración**: Archivo JSON externo
- **Flexibilidad**: Parámetros configurables
- **Logging**: Salida estándar para contenedores

## 📊 Optimizaciones de Performance

### Cache Layers
1. **Dependencias Go**: Cache de `go mod download`
2. **Build Cache**: Cache de compilación
3. **Base Images**: Reutilización de layers

### Tamaño del Imagen
- **Builder**: ~500MB (temporal)
- **Final**: ~20MB (solo runtime)
- **Reducción**: 95% de tamaño

## 🔧 Comandos de Build

### Build Local
```bash
docker build -t chat-api .
```

### Build con SSH
```bash
docker build --ssh default -t chat-api .
```

### Build Multi-Platform
```bash
docker buildx build --platform linux/amd64,linux/arm64 -t chat-api .
```

## 🐳 Docker Compose Integration

### Variables de Entorno
```yaml
environment:
  - POSTGRES_HOST=postgres
  - SCYLLA_HOSTS=scylla-node1:9042
  - REDIS_HOST=redis
  - NATS_URL=nats://nats:4222
```

### Volúmenes
```yaml
volumes:
  - ./config:/app/config:ro
```

## 🔍 Health Checks

### Configuración
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1
```

## 🚨 Troubleshooting

### Problemas Comunes
1. **SSH Keys**: Verificar configuración de SSH agent
2. **Permisos**: Verificar permisos de archivos
3. **Network**: Verificar conectividad a repositorios
4. **Cache**: Limpiar cache si hay problemas

### Debugging
```bash
# Build con output detallado
docker build --progress=plain --no-cache -t chat-api .

# Inspeccionar imagen
docker run -it --entrypoint /bin/sh chat-api

# Verificar logs
docker logs <container-id>
```

## 💡 Mejores Prácticas Implementadas

### Seguridad
- ✅ Usuario no-root
- ✅ Imagen base mínima
- ✅ Secrets no persistidos
- ✅ Verificación de host keys

### Performance
- ✅ Multi-stage build
- ✅ Cache de dependencias
- ✅ Binario estático
- ✅ Imagen optimizada

### Mantenibilidad
- ✅ Layers ordenados por frecuencia de cambio
- ✅ Configuración externa
- ✅ Health checks
- ✅ Logging estructurado