# Documentación Técnica: catalogs/catalogs.go

## Descripción General

El archivo `catalogs.go` define las configuraciones centrales y constantes de la aplicación de chat. Actúa como un catálogo de configuraciones que son utilizadas a lo largo de toda la aplicación para mantener consistencia en rutas, modos de operación y direcciones de servicios.

## Estructura del Archivo

### Importaciones

```go
import (
    "os"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/config"
)
```

**Análisis de Importaciones:**

- **`os`**: Paquete estándar para acceso a variables de entorno del sistema operativo
- **`config`**: Módulo del core para manejo centralizado de configuraciones

### Variables Globales

#### Variable de Modo de Producción

```go
var (
    IsProd = os.Getenv("MODE") == "PROD"
)
```

**Análisis Detallado:**

- **Propósito**: Determina si la aplicación está ejecutándose en modo producción
- **Evaluación**: Compara la variable de entorno `MODE` con el string `"PROD"`
- **Tipo**: `bool` - `true` si está en producción, `false` en caso contrario
- **Uso en la aplicación**:
  - Configuración de logging (más restrictivo en producción)
  - Habilitación/deshabilitación de endpoints de debug
  - Optimizaciones de rendimiento
  - Configuración de CORS y seguridad

**Valores posibles de `MODE`:**
- `"PROD"` → `IsProd = true`
- `"DEV"`, `"DEVELOPMENT"`, `"TEST"`, etc. → `IsProd = false`
- No definida → `IsProd = false`

#### Estructura de Rutas Especiales

```go
SpecialRoutes = struct {
    DebugRoute     string
    SwaggerRoute   string
    ProtosDownload string
}{
    DebugRoute:     "/api/chat/debug",
    SwaggerRoute:   "/api/chat/swagger",
    ProtosDownload: "/api/chat/protos_download",
}
```

**Análisis de la Estructura:**

##### Definición de Tipo Anónimo
```go
struct {
    DebugRoute     string
    SwaggerRoute   string
    ProtosDownload string
}
```

- **Patrón de Diseño**: Struct anónimo para agrupar rutas relacionadas
- **Ventajas**:
  - Organización lógica de constantes
  - Acceso mediante dot notation
  - Inmutabilidad una vez inicializada
  - Autocompletado en IDEs

##### Rutas Definidas

**`DebugRoute: "/api/chat/debug"`**
- **Propósito**: Endpoint para información de diagnóstico y debugging
- **Funcionalidad típica**:
  - Estado de conexiones a bases de datos
  - Métricas de memoria y CPU
  - Información de configuración (sin datos sensibles)
  - Estado de servicios externos (Redis, NATS)
- **Seguridad**: Solo disponible en modo desarrollo
- **Formato de respuesta**: JSON con información del sistema

**`SwaggerRoute: "/api/chat/swagger"`**
- **Propósito**: Documentación interactiva de la API
- **Funcionalidad**:
  - Interfaz web para explorar endpoints
  - Documentación generada desde archivos .proto
  - Capacidad de testing directo desde el navegador
  - Esquemas de request/response
- **Acceso**: Disponible en desarrollo y producción
- **Tecnología**: Swagger UI integrado

**`ProtosDownload: "/api/chat/protos_download"`**
- **Propósito**: Descarga de archivos Protocol Buffers
- **Funcionalidad**:
  - Acceso a archivos .proto originales
  - Facilita generación de clientes en diferentes lenguajes
  - Versionado de esquemas de API
- **Formato**: Archivos .proto comprimidos o individuales
- **Uso**: Integración con herramientas de generación de código

### Funciones

#### Función ClientAddress

```go
func ClientAddress() string {
    return config.GetString("grpc.clientAddresses.chat-messages")
}
```

**Análisis Detallado:**

##### Propósito
- **Función**: Obtiene la dirección del cliente gRPC para el servicio de chat-messages
- **Uso**: Configuración de conexiones entre microservicios
- **Patrón**: Centralización de configuración de direcciones

##### Configuración
- **Clave de configuración**: `"grpc.clientAddresses.chat-messages"`
- **Estructura jerárquica**:
  ```yaml
  grpc:
    clientAddresses:
      chat-messages: "localhost:8080"
      auth: "localhost:8081"
      notifications: "localhost:8082"
  ```

##### Casos de Uso
1. **Comunicación entre microservicios**:
   ```go
   address := catalogs.ClientAddress()
   conn, err := grpc.Dial(address, grpc.WithInsecure())
   ```

2. **Configuración de clientes**:
   ```go
   client := chatv1client.NewChatServiceClient(catalogs.ClientAddress())
   ```

3. **Load balancing y service discovery**:
   - Puede retornar direcciones de load balancer
   - Integración con sistemas como Consul o Kubernetes

##### Flexibilidad de Configuración
- **Desarrollo**: `"localhost:8080"`
- **Testing**: `"chat-service:8080"`
- **Producción**: `"chat-messages.internal.company.com:443"`
- **Kubernetes**: `"chat-messages-service.default.svc.cluster.local:8080"`

## Patrones de Diseño Implementados

### 1. Singleton Pattern (Implícito)
- Las variables globales actúan como singletons
- Una sola instancia de configuración en toda la aplicación
- Acceso global y consistente

### 2. Configuration Centralization
- Todas las configuraciones de rutas en un solo lugar
- Facilita mantenimiento y cambios
- Reduce duplicación de código

### 3. Environment-based Configuration
- Comportamiento diferente según entorno
- Configuración externa sin recompilación
- Separación de concerns

## Uso en la Aplicación

### En main.go
```go
srv := server.NewServer(
    server.WithProdMode(catalogs.IsProd),
    server.WithDebugRoute(catalogs.SpecialRoutes.DebugRoute),
    server.WithSwagger(catalogs.SpecialRoutes.SwaggerRoute, proto.SwaggerJsonDoc),
    server.WithProtosDownload(protoFilesFs, catalogs.SpecialRoutes.ProtosDownload, "proto"),
)
```

### En handlers
```go
if !catalogs.IsProd {
    // Logging adicional en desarrollo
    log.Debug("Detailed debug information")
}
```

### En clientes
```go
conn, err := grpc.Dial(catalogs.ClientAddress(), opts...)
```

## Consideraciones de Seguridad

### 1. Modo Producción
- `IsProd` controla exposición de información sensible
- Endpoints de debug deshabilitados en producción
- Logging más restrictivo

### 2. Configuración Externa
- Direcciones de servicios desde configuración externa
- No hardcoding de URLs en código
- Facilita rotación de servicios

### 3. Validación de Rutas
- Rutas predefinidas evitan inyección de paths
- Consistencia en naming conventions
- Facilita auditoría de endpoints

## Escalabilidad y Mantenimiento

### 1. Extensibilidad
```go
// Fácil agregar nuevas rutas
SpecialRoutes = struct {
    DebugRoute     string
    SwaggerRoute   string
    ProtosDownload string
    HealthRoute    string  // Nueva ruta
    MetricsRoute   string  // Nueva ruta
}{
    // ... rutas existentes
    HealthRoute:   "/api/chat/health",
    MetricsRoute:  "/api/chat/metrics",
}
```

### 2. Versionado de API
```go
// Soporte para múltiples versiones
SpecialRoutes = struct {
    V1 struct {
        DebugRoute   string
        SwaggerRoute string
    }
    V2 struct {
        DebugRoute   string
        SwaggerRoute string
    }
}
```

### 3. Configuración Dinámica
- `ClientAddress()` permite cambios sin restart
- Integración con sistemas de configuración distribuida
- Hot-reload de configuraciones

## Mejores Prácticas Implementadas

1. **Inmutabilidad**: Variables inicializadas una vez
2. **Naming Conventions**: Nombres descriptivos y consistentes
3. **Separación de Concerns**: Configuración separada de lógica de negocio
4. **Environment Awareness**: Comportamiento adaptado al entorno
5. **Centralización**: Un solo punto de verdad para configuraciones

Este archivo es fundamental para la configuración y comportamiento de la aplicación, proporcionando un punto central para todas las configuraciones relacionadas con rutas y modos de operación.