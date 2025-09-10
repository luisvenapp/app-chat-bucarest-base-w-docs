# Documentación Técnica: go.mod

## Descripción General

El archivo `go.mod` define el módulo Go y gestiona todas las dependencias del proyecto Campaign App Chat Messages API. Especifica la versión de Go requerida, las dependencias directas necesarias para la funcionalidad del chat, y las dependencias indirectas que son requeridas por las librerías utilizadas.

## Estructura del Archivo

### Definición del Módulo

```go
module github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go

go 1.23.0
```

**Análisis:**

#### Module Path
- **Organización**: `github.com/Venqis-NolaTech` - Organización en GitHub
- **Proyecto**: `campaing-app-chat-messages-api-go` - Nombre específico del servicio
- **Convención**: Sigue convenciones estándar de Go modules
- **Importación**: Otros módulos pueden importar usando este path

#### Versión de Go
- **Versión**: `1.23.0` - Versión específica de Go requerida
- **Características**: Acceso a features más recientes de Go
- **Compatibilidad**: Garantiza que el código funcione con esta versión o superior

## Dependencias Directas (require)

### Validación y Protocol Buffers

#### `buf.build/go/protovalidate v0.14.0`
- **Propósito**: Validación de mensajes Protocol Buffers
- **Uso**: Validación automática de requests gRPC
- **Características**: Reglas de validación declarativas en archivos .proto

### Conectividad gRPC

#### `connectrpc.com/connect v1.18.1`
- **Propósito**: Framework principal para servicios gRPC-Web
- **Características**:
  - Soporte para gRPC, gRPC-Web y Connect protocols
  - Compatibilidad con navegadores web
  - Streaming bidireccional
  - Middleware support

#### `connectrpc.com/vanguard v0.3.0`
- **Propósito**: Router y multiplexer para servicios Connect
- **Funcionalidades**:
  - Routing automático de servicios
  - Soporte para múltiples protocolos
  - Middleware integration
  - Service discovery

### Base de Datos

#### `github.com/Masterminds/squirrel v1.5.4`
- **Propósito**: Query builder para SQL
- **Ventajas**:
  - Construcción segura de queries
  - Prevención de SQL injection
  - Soporte para múltiples dialectos SQL
  - Fluent interface

#### `github.com/scylladb-solutions/gocql/v2 v2.0.0`
- **Propósito**: Driver para Cassandra/ScyllaDB
- **Uso**: Base de datos NoSQL para mensajes de alto volumen
- **Características**:
  - Connection pooling
  - Cluster awareness
  - Prepared statements

### Servicios Internos

#### `github.com/Venqis-NolaTech/campaing-app-auth-api-go v0.0.0-20250823192912-a3b46d00a320`
- **Propósito**: Cliente para servicio de autenticación
- **Funcionalidades**:
  - Validación de tokens JWT
  - Gestión de sesiones
  - Información de usuarios

#### `github.com/Venqis-NolaTech/campaing-app-core-go v0.0.0-20250829203835-b38cb12075e0`
- **Propósito**: Librerías compartidas del core
- **Incluye**:
  - Configuración centralizada
  - Utilidades de base de datos
  - Middleware común
  - Gestión de eventos

#### `github.com/Venqis-NolaTech/campaing-app-notifications-api-go v0.0.0-20250901150123-1fc6fb6fffb3`
- **Propósito**: Cliente para servicio de notificaciones
- **Uso**: Envío de notificaciones push

### CLI y UI Terminal

#### `github.com/charmbracelet/bubbles v0.21.0`
- **Propósito**: Componentes UI para terminal
- **Uso**: CLI interactivo de chat
- **Componentes**: Input fields, viewports, spinners

#### `github.com/charmbracelet/bubbletea v1.3.6`
- **Propósito**: Framework TUI (Terminal User Interface)
- **Características**:
  - Event-driven architecture
  - Component-based UI
  - Cross-platform terminal support

#### `github.com/charmbracelet/lipgloss v1.1.0`
- **Propósito**: Styling para interfaces de terminal
- **Funcionalidades**:
  - CSS-like styling
  - Layout management
  - Color support

### Utilidades

#### `github.com/google/uuid v1.6.0`
- **Propósito**: Generación de UUIDs
- **Uso**: Identificadores únicos para eventos, mensajes, etc.
- **Estándar**: RFC 4122 compliant

#### `github.com/nats-io/nats.go v1.44.0`
- **Propósito**: Cliente para NATS messaging system
- **Características**:
  - Pub/Sub messaging
  - JetStream support
  - Clustering
  - Streaming

### Criptografía y Texto

#### `golang.org/x/crypto v0.41.0`
- **Propósito**: Funciones criptográficas extendidas
- **Uso**: Encriptación de mensajes, derivación de claves
- **Incluye**: scrypt, bcrypt, etc.

#### `golang.org/x/net v0.43.0`
- **Propósito**: Extensiones de networking
- **Uso**: HTTP/2, gRPC transport

#### `golang.org/x/text v0.28.0`
- **Propósito**: Procesamiento de texto internacional
- **Uso**: Normalización Unicode, remoción de acentos
- **Características**: i18n support

### Protocol Buffers y gRPC

#### `google.golang.org/genproto/googleapis/api v0.0.0-20250728155136-f173205681a0`
- **Propósito**: Tipos comunes de Google APIs
- **Uso**: Annotations, field behavior

#### `google.golang.org/protobuf v1.36.8`
- **Propósito**: Runtime de Protocol Buffers para Go
- **Características**:
  - Serialización eficiente
  - Reflection support
  - JSON marshaling

## Dependencias Indirectas (require - indirect)

### Validación y Expresiones

#### `cel.dev/expr v0.24.0`
- **Propósito**: Common Expression Language
- **Uso**: Evaluación de expresiones en validaciones

#### `github.com/google/cel-go v0.25.0`
- **Propósito**: Implementación Go de CEL
- **Uso**: Backend para protovalidate

### Parsing y Compilación

#### `github.com/antlr4-go/antlr/v4 v4.13.0`
- **Propósito**: Parser generator runtime
- **Uso**: Parsing de expresiones CEL

### Terminal y UI

#### `github.com/atotto/clipboard v0.1.4`
- **Propósito**: Acceso al clipboard del sistema
- **Uso**: Funcionalidades de copy/paste en CLI

#### `github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f`
- **Propósito**: Input handling para Windows
- **Uso**: Compatibilidad cross-platform del CLI

### Caching y Redis

#### `github.com/redis/go-redis/v9 v9.12.0`
- **Propósito**: Cliente Redis
- **Uso**: Caching de salas y mensajes
- **Características**: Pipeline, clustering, pub/sub

#### `github.com/cespare/xxhash/v2 v2.3.0`
- **Propósito**: Hash function rápida
- **Uso**: Hashing interno de Redis

### HTTP y Routing

#### `github.com/go-chi/chi/v5 v5.1.0`
- **Propósito**: HTTP router
- **Uso**: Routing de endpoints HTTP

### JSON y Serialización

#### `github.com/goccy/go-json v0.10.5`
- **Propósito**: JSON encoder/decoder optimizado
- **Ventajas**: Mejor performance que encoding/json estándar

### Autenticación

#### `github.com/golang-jwt/jwt/v5 v5.2.2`
- **Propósito**: JSON Web Tokens
- **Uso**: Validación y generación de tokens JWT

### Base de Datos

#### `github.com/lib/pq v1.10.9`
- **Propósito**: Driver PostgreSQL
- **Uso**: Conexión a base de datos principal

#### `github.com/jmoiron/sqlx v1.4.0`
- **Propósito**: Extensiones para database/sql
- **Características**: Named parameters, struct scanning

### Compresión

#### `github.com/klauspost/compress v1.18.0`
- **Propósito**: Algoritmos de compresión
- **Uso**: Compresión de datos en NATS JetStream

### NATS

#### `github.com/nats-io/nkeys v0.4.11`
- **Propósito**: Cryptographic keys para NATS
- **Uso**: Autenticación segura con NATS

#### `github.com/nats-io/nuid v1.0.1`
- **Propósito**: Unique identifiers para NATS
- **Uso**: IDs únicos para mensajes NATS

## Análisis de Arquitectura

### Patrones de Dependencias

#### 1. **Microservices Architecture**
```
auth-api-go ←→ chat-messages-api-go ←→ notifications-api-go
                        ↓
                    core-go (shared)
```

#### 2. **Database Strategy**
- **PostgreSQL**: Datos relacionales (usuarios, salas)
- **Cassandra/ScyllaDB**: Mensajes de alto volumen
- **Redis**: Caching y sesiones

#### 3. **Communication Patterns**
- **gRPC**: Comunicación entre servicios
- **NATS**: Eventos en tiempo real
- **HTTP**: APIs públicas y debugging

### Versionado de Dependencias

#### Dependencias Internas
- **Commit-based**: Usan commits específicos para control preciso
- **Ejemplo**: `v0.0.0-20250823192912-a3b46d00a320`
- **Ventaja**: Control exacto de versiones en desarrollo

#### Dependencias Externas
- **Semantic Versioning**: Usan versiones semánticas
- **Ejemplo**: `v1.18.1`, `v0.21.0`
- **Ventaja**: Compatibilidad y estabilidad

## Consideraciones de Seguridad

### Dependencias Criptográficas
- **golang.org/x/crypto**: Funciones criptográficas seguras
- **github.com/golang-jwt/jwt/v5**: JWT con algoritmos seguros
- **Actualizaciones**: Mantener actualizadas para patches de seguridad

### Validación de Entrada
- **buf.build/go/protovalidate**: Validación automática
- **cel.dev/expr**: Expresiones de validación complejas
- **Prevención**: SQL injection, XSS, etc.

## Performance y Optimización

### Librerías Optimizadas
- **github.com/goccy/go-json**: JSON más rápido
- **github.com/klauspost/compress**: Compresión eficiente
- **github.com/cespare/xxhash/v2**: Hashing rápido

### Connection Pooling
- **database/sql**: Pool automático para PostgreSQL
- **gocql**: Pool para Cassandra
- **redis**: Pool para Redis

## Gestión de Dependencias

### Comandos Útiles

```bash
# Actualizar dependencias
go mod tidy

# Verificar dependencias
go mod verify

# Ver dependencias
go mod graph

# Actualizar dependencia específica
go get github.com/some/package@latest

# Downgrade de dependencia
go get github.com/some/package@v1.2.3
```

### Estrategias de Actualización

#### 1. **Dependencias Críticas**
- Actualizar con cuidado
- Testing exhaustivo
- Rollback plan

#### 2. **Dependencias de Desarrollo**
- Actualizar más frecuentemente
- Menos riesgo en producción

#### 3. **Dependencias de Seguridad**
- Actualizar inmediatamente
- Monitorear vulnerabilidades

## Mejores Prácticas Implementadas

1. **Version Pinning**: Versiones específicas para reproducibilidad
2. **Minimal Dependencies**: Solo dependencias necesarias
3. **Security Updates**: Dependencias actualizadas regularmente
4. **Compatibility**: Versiones compatibles entre dependencias
5. **Documentation**: Dependencias bien documentadas
6. **Testing**: Dependencias probadas en CI/CD

## Monitoreo de Dependencias

### Herramientas Recomendadas

```bash
# Verificar vulnerabilidades
go list -json -m all | nancy sleuth

# Auditoría de dependencias
govulncheck ./...

# Análisis de licencias
go-licenses check ./...
```

### Automatización

```yaml
# GitHub Actions para monitoreo
name: Dependency Check
on: [push, pull_request]
jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: go mod verify
      - run: govulncheck ./...
```

Este archivo `go.mod` refleja una arquitectura bien diseñada con dependencias cuidadosamente seleccionadas para construir un sistema de chat robusto, seguro y escalable.