# Documentación Técnica: main.go

## Descripción General

El archivo `main.go` es el punto de entrada principal de la aplicación **Campaign App Chat Messages API**. Este archivo implementa la inicialización y configuración del servidor gRPC que maneja los servicios de mensajería y chat de la aplicación.

## Estructura del Archivo

### Importaciones

```go
import (
    "embed"
    "log"
    
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/catalogs"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/handlers"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/config"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/server"
)
```

**Análisis de Importaciones:**

- **`embed`**: Paquete estándar de Go que permite embebir archivos estáticos en el binario compilado
- **`log`**: Paquete estándar para logging básico
- **Módulos internos**:
  - `catalogs`: Contiene configuraciones y constantes de la aplicación
  - `handlers`: Implementa los manejadores de servicios gRPC
  - `proto`: Contiene las definiciones de Protocol Buffers generadas
- **Módulos del core**: Utilidades compartidas para configuración y servidor

### Variables Globales

```go
var (
    //go:embed proto
    protoFilesFs embed.FS
    
    address = config.GetString("server.address")
)
```

**Análisis Detallado:**

#### `protoFilesFs embed.FS`
- **Propósito**: Embebe todo el directorio `proto` en el binario compilado
- **Directiva `//go:embed proto`**: Instruye al compilador de Go para incluir todos los archivos del directorio `proto` en el sistema de archivos embebido
- **Ventajas**:
  - Los archivos .proto están disponibles en tiempo de ejecución sin dependencias externas
  - Facilita la distribución de documentación de API
  - Permite servir archivos proto para descarga por clientes

#### `address = config.GetString("server.address")`
- **Propósito**: Obtiene la dirección del servidor desde la configuración
- **Configuración dinámica**: Permite cambiar la dirección sin recompilar
- **Formato esperado**: Típicamente `"host:puerto"` (ej: `"0.0.0.0:8080"`)

### Función Principal

```go
func main() {
    server.InitEnvironment()
    
    server.InitRedis()
    server.InitNats()
    
    srv := server.NewServer(
        server.WithProdMode(catalogs.IsProd),
        server.WithDebugRoute(catalogs.SpecialRoutes.DebugRoute),
        server.WithSwagger(catalogs.SpecialRoutes.SwaggerRoute, proto.SwaggerJsonDoc),
        server.WithServices(handlers.RegisterServicesFns),
        server.WithProtosDownload(protoFilesFs, catalogs.SpecialRoutes.ProtosDownload, "proto"),
    )
    
    log.Printf("Initializing gRPC server on address: %s\n", address)
    if err := srv.Listen(address); err != nil {
        log.Fatal(err)
    }
}
```

**Análisis Paso a Paso:**

#### 1. Inicialización del Entorno
```go
server.InitEnvironment()
```
- **Función**: Configura variables de entorno y configuración base
- **Responsabilidades**:
  - Carga archivos de configuración
  - Establece logging
  - Configura timezone y localización

#### 2. Inicialización de Servicios Externos
```go
server.InitRedis()
server.InitNats()
```

**`server.InitRedis()`**:
- **Propósito**: Establece conexión con Redis para caché y sesiones
- **Uso en la aplicación**:
  - Caché de salas de chat
  - Almacenamiento de sesiones de usuario
  - Datos temporales de mensajería

**`server.InitNats()`**:
- **Propósito**: Inicializa conexión con NATS para mensajería en tiempo real
- **Uso en la aplicación**:
  - Streaming de mensajes en tiempo real
  - Eventos de salas de chat
  - Notificaciones push

#### 3. Configuración del Servidor

```go
srv := server.NewServer(
    server.WithProdMode(catalogs.IsProd),
    server.WithDebugRoute(catalogs.SpecialRoutes.DebugRoute),
    server.WithSwagger(catalogs.SpecialRoutes.SwaggerRoute, proto.SwaggerJsonDoc),
    server.WithServices(handlers.RegisterServicesFns),
    server.WithProtosDownload(protoFilesFs, catalogs.SpecialRoutes.ProtosDownload, "proto"),
)
```

**Análisis de Opciones del Servidor:**

##### `server.WithProdMode(catalogs.IsProd)`
- **Propósito**: Configura el modo de producción basado en variable de entorno
- **Comportamiento**:
  - **Modo Producción**: Desactiva logs de debug, optimiza rendimiento
  - **Modo Desarrollo**: Habilita logs detallados, validaciones adicionales

##### `server.WithDebugRoute(catalogs.SpecialRoutes.DebugRoute)`
- **Ruta**: `/api/chat/debug`
- **Funcionalidad**: Endpoint para diagnósticos y debugging
- **Información expuesta**:
  - Estado de conexiones
  - Métricas de rendimiento
  - Información de configuración

##### `server.WithSwagger(catalogs.SpecialRoutes.SwaggerRoute, proto.SwaggerJsonDoc)`
- **Ruta**: `/api/chat/swagger`
- **Propósito**: Sirve documentación interactiva de la API
- **Contenido**: Documentación generada automáticamente desde archivos .proto

##### `server.WithServices(handlers.RegisterServicesFns)`
- **Propósito**: Registra todos los servicios gRPC de la aplicación
- **Servicios incluidos**:
  - `ChatService`: Manejo de mensajes y salas
  - `TokensService`: Gestión de tokens de dispositivos

##### `server.WithProtosDownload(protoFilesFs, catalogs.SpecialRoutes.ProtosDownload, "proto")`
- **Ruta**: `/api/chat/protos_download`
- **Funcionalidad**: Permite descargar archivos .proto para generación de clientes
- **Uso**: Facilita la integración con diferentes lenguajes de programación

#### 4. Inicio del Servidor

```go
log.Printf("Initializing gRPC server on address: %s\n", address)
if err := srv.Listen(address); err != nil {
    log.Fatal(err)
}
```

**Análisis del Proceso de Inicio:**

- **Logging**: Registra la dirección donde se inicia el servidor
- **`srv.Listen(address)`**: Inicia el servidor en la dirección configurada
- **Manejo de errores**: Si falla el inicio, termina la aplicación con `log.Fatal(err)`

## Flujo de Ejecución

1. **Inicialización**: Configura entorno, Redis y NATS
2. **Configuración**: Crea servidor con todas las opciones necesarias
3. **Registro de servicios**: Registra handlers de chat y tokens
4. **Inicio**: Comienza a escuchar en la dirección configurada
5. **Operación**: Maneja requests gRPC de forma concurrente

## Dependencias Críticas

### Servicios Externos Requeridos
- **Redis**: Para caché y sesiones
- **NATS**: Para mensajería en tiempo real
- **PostgreSQL**: Base de datos principal (inicializada en otros módulos)
- **Cassandra/ScyllaDB**: Base de datos de mensajes (opcional)

### Configuración Requerida
- `server.address`: Dirección del servidor
- Variables de entorno para Redis y NATS
- Configuración de base de datos

## Consideraciones de Seguridad

1. **Modo Producción**: Desactiva endpoints de debug en producción
2. **Configuración Externa**: Todas las configuraciones sensibles vienen de variables de entorno
3. **Logging Seguro**: No expone información sensible en logs

## Escalabilidad y Rendimiento

1. **Servidor Concurrente**: gRPC maneja múltiples conexiones simultáneas
2. **Caché Redis**: Reduce carga en base de datos
3. **Streaming NATS**: Maneja eventos en tiempo real eficientemente
4. **Archivos Embebidos**: Reduce I/O para servir documentación

## Monitoreo y Debugging

1. **Endpoint Debug**: `/api/chat/debug` para diagnósticos
2. **Logs Estructurados**: Información detallada de inicio
3. **Documentación Swagger**: `/api/chat/swagger` para testing
4. **Descarga de Protos**: `/api/chat/protos_download` para integración

Este archivo representa el núcleo de la aplicación, orquestando todos los componentes necesarios para proporcionar un servicio de chat robusto y escalable.