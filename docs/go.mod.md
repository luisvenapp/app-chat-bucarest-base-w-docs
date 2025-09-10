# 📄 Documentación: go.mod

## 🎯 Propósito
Archivo de configuración de módulo Go que define las dependencias del proyecto y la versión de Go requerida.

## 📋 Información del Módulo

### Identificación
- **Módulo**: `github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go`
- **Versión de Go**: `1.23.0`

## 📦 Dependencias Principales

### 🌐 Framework y Comunicación
- **`connectrpc.com/connect`** - Framework Connect para gRPC/HTTP
- **`google.golang.org/grpc`** - Framework gRPC oficial
- **`google.golang.org/protobuf`** - Biblioteca Protocol Buffers

### 🗄️ Bases de Datos
- **`github.com/scylladb-solutions/gocql/v2`** - Driver para ScyllaDB/Cassandra
- **`github.com/lib/pq`** - Driver PostgreSQL
- **`github.com/Masterminds/squirrel`** - Query builder SQL

### 📡 Streaming y Eventos
- **`github.com/nats-io/nats.go`** - Cliente NATS para messaging
- **`github.com/nats-io/jetstream`** - JetStream para eventos persistentes

### 🔐 Seguridad y Autenticación
- **`golang.org/x/crypto`** - Funciones criptográficas
- **`github.com/golang-jwt/jwt/v5`** - Manejo de tokens JWT

### 🎨 CLI y UI (para chat-cli)
- **`github.com/charmbracelet/bubbletea`** - Framework TUI
- **`github.com/charmbracelet/bubbles`** - Componentes UI
- **`github.com/charmbracelet/lipgloss`** - Estilos para terminal

### 🧪 Testing
- **`github.com/stretchr/testify`** - Framework de testing
- **`github.com/google/uuid`** - Generación de UUIDs

### 🏢 Dependencias Internas Venqis
- **`github.com/Venqis-NolaTech/campaing-app-core-go`** - Core framework interno
- **`github.com/Venqis-NolaTech/campaing-app-auth-api-go`** - Servicio de autenticación

## 🔧 Herramientas de Desarrollo

### Protocol Buffers
- **`buf.build/gen/go/bufbuild/protovalidate`** - Validaciones de protobuf
- **`connectrpc.com/grpchealth`** - Health checks para gRPC

### Utilidades
- **`github.com/kelseyhightower/envconfig`** - Configuración por variables de entorno
- **`github.com/prometheus/client_golang`** - Métricas Prometheus

## 📊 Análisis de Dependencias

### Por Categoría
- **Comunicación**: 25% (gRPC, Connect, HTTP)
- **Bases de Datos**: 20% (PostgreSQL, ScyllaDB, Redis)
- **Streaming**: 15% (NATS, JetStream)
- **Seguridad**: 15% (Crypto, JWT, Auth)
- **UI/CLI**: 10% (Bubble Tea, Lipgloss)
- **Testing/Utils**: 15% (Testify, UUID, Config)

### Dependencias Críticas
1. **Core Framework** - Base de toda la aplicación
2. **Database Drivers** - Acceso a datos
3. **gRPC/Connect** - Comunicación API
4. **NATS** - Eventos en tiempo real
5. **Crypto** - Seguridad de mensajes

## 🔄 Gestión de Versiones

### Estrategia de Versionado
- **Semantic Versioning** para dependencias externas
- **Pinning de versiones** para dependencias críticas
- **Actualizaciones regulares** de dependencias de seguridad

### Compatibilidad
- **Go 1.23+** requerido para features modernas
- **Backward compatibility** mantenida en APIs públicas
- **Breaking changes** documentados en CHANGELOG

## 🚨 Dependencias de Seguridad

### Críticas para Seguridad
- `golang.org/x/crypto` - Encriptación de mensajes
- `github.com/golang-jwt/jwt/v5` - Autenticación
- `github.com/scylladb-solutions/gocql/v2` - Conexiones seguras a BD

### Auditoría Regular
- Escaneo automático de vulnerabilidades
- Actualizaciones de seguridad prioritarias
- Revisión de dependencias transitivas

## 📈 Optimizaciones

### Performance
- Drivers optimizados para bases de datos
- Conexiones pooling automático
- Serialización eficiente con protobuf

### Tamaño del Binario
- Dependencias mínimas necesarias
- Build tags para features opcionales
- Eliminación de dependencias no utilizadas

## 🔍 Comandos Útiles

### Gestión de Dependencias
```bash
go mod tidy          # Limpiar dependencias
go mod verify        # Verificar integridad
go mod download      # Descargar dependencias
go list -m all       # Listar todas las dependencias
```

### Análisis
```bash
go mod graph         # Grafo de dependencias
go mod why <module>  # Por qué se necesita un módulo
go list -u -m all    # Verificar actualizaciones
```

## 💡 Notas de Mantenimiento
- Revisar dependencias mensualmente
- Actualizar Go version según roadmap
- Monitorear CVEs en dependencias críticas
- Documentar cambios breaking en updates