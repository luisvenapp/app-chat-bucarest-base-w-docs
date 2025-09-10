# Documentación Técnica: handlers/chat/v1/register.go

## Descripción General

El archivo `register.go` implementa la función de registro del servicio de chat en el framework Vanguard. Actúa como el punto de entrada para configurar y exponer el servicio gRPC de chat, aplicando las opciones de configuración necesarias y creando la instancia del servicio que será registrada en el servidor principal.

## Estructura del Archivo

### Importaciones

```go
import (
    "connectrpc.com/vanguard"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1/chatv1connect"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/server"
)
```

**Análisis de Importaciones:**

- **`vanguard`**: Framework de ConnectRPC para servicios gRPC-Web
- **`chatv1connect`**: Interfaz generada del servicio de chat desde Protocol Buffers
- **`server`**: Módulo del core que proporciona opciones de configuración de servicios

## Variable de Configuración

```go
var options = server.ServiceHandlerOptions()
```

**Análisis Detallado:**

### Propósito de las Opciones
- **Configuración centralizada**: Obtiene opciones estándar del core
- **Consistencia**: Asegura configuración uniforme entre servicios
- **Flexibilidad**: Permite personalización desde configuración externa

### Opciones Típicas Incluidas
```go
// Ejemplo de opciones que podrían estar incluidas
type ServiceHandlerOptions struct {
    // Middleware de autenticación
    AuthMiddleware []connect.UnaryInterceptorFunc
    
    // Middleware de logging
    LoggingMiddleware []connect.UnaryInterceptorFunc
    
    // Middleware de métricas
    MetricsMiddleware []connect.UnaryInterceptorFunc
    
    // Configuración de CORS
    CORSConfig *CORSConfig
    
    // Timeouts y límites
    RequestTimeout time.Duration
    MaxRequestSize int64
    
    // Configuración de compresión
    CompressionConfig *CompressionConfig
}
```

### Middleware Aplicado
Las opciones típicamente incluyen:

#### 1. **Authentication Middleware**
```go
func authMiddleware() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Validar token de autenticación
            if err := validateAuth(req); err != nil {
                return nil, connect.NewError(connect.CodeUnauthenticated, err)
            }
            return next(ctx, req)
        }
    }
}
```

#### 2. **Logging Middleware**
```go
func loggingMiddleware() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            start := time.Now()
            resp, err := next(ctx, req)
            duration := time.Since(start)
            
            log.Info("Request processed",
                "method", req.Spec().Procedure,
                "duration", duration,
                "error", err,
            )
            return resp, err
        }
    }
}
```

#### 3. **Metrics Middleware**
```go
func metricsMiddleware() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            start := time.Now()
            resp, err := next(ctx, req)
            duration := time.Since(start)
            
            // Registrar métricas
            requestDuration.WithLabelValues(
                req.Spec().Procedure,
                getStatusCode(err),
            ).Observe(duration.Seconds())
            
            return resp, err
        }
    }
}
```

## Función RegisterServiceHandler

```go
func RegisterServiceHandler() *vanguard.Service {
    return vanguard.NewService(chatv1connect.NewChatServiceHandler(NewHandler(), options...))
}
```

**Análisis Detallado:**

### Signature de la Función
```go
func RegisterServiceHandler() *vanguard.Service
```
- **Nombre**: `RegisterServiceHandler` - Convención estándar para funciones de registro
- **Retorno**: `*vanguard.Service` - Servicio configurado listo para registro
- **Visibilidad**: Función pública para uso desde el paquete handlers principal

### Proceso de Construcción

#### 1. Creación del Handler
```go
NewHandler()
```
- **Factory Function**: Crea nueva instancia del handler de chat
- **Inicialización**: Configura todas las dependencias necesarias
- **Stateless**: Cada instancia es independiente

#### 2. Creación del Service Handler
```go
chatv1connect.NewChatServiceHandler(NewHandler(), options...)
```
- **Generated Function**: Función generada desde Protocol Buffers
- **Handler**: Instancia del handler que implementa la lógica de negocio
- **Options**: Spread de opciones de configuración
- **Result**: Handler gRPC configurado con middleware

#### 3. Creación del Servicio Vanguard
```go
vanguard.NewService(...)
```
- **Wrapper**: Envuelve el handler en un servicio Vanguard
- **Protocol Support**: Soporte para gRPC, gRPC-Web y Connect
- **Routing**: Configuración automática de rutas
- **Middleware**: Aplicación de middleware configurado

### Flujo de Registro

```mermaid
graph TD
    A[RegisterServiceHandler()] --> B[NewHandler()]
    B --> C[chatv1connect.NewChatServiceHandler()]
    C --> D[Apply options...]
    D --> E[vanguard.NewService()]
    E --> F[Return configured service]
    
    G[server.ServiceHandlerOptions()] --> D
    H[Auth Middleware] --> G
    I[Logging Middleware] --> G
    J[Metrics Middleware] --> G
```

## Integración con el Sistema

### Uso en handlers/handlers.go

```go
var RegisterServicesFns = []server.RegisterServiceFn{
    chatv1handler.RegisterServiceHandler,  // Esta función
    tokensv1handler.RegisterServiceHandler,
}
```

### Uso en main.go

```go
srv := server.NewServer(
    server.WithServices(handlers.RegisterServicesFns),
    // otras opciones...
)
```

### Proceso Completo de Registro

1. **main.go** llama a `server.NewServer()`
2. **server.NewServer()** itera sobre `RegisterServicesFns`
3. **RegisterServiceHandler()** se ejecuta
4. **NewHandler()** crea instancia del handler
5. **chatv1connect.NewChatServiceHandler()** aplica opciones
6. **vanguard.NewService()** crea servicio final
7. **Servidor** registra el servicio en el router

## Configuración de Rutas

### Rutas Generadas Automáticamente

El servicio registrado expone automáticamente las siguientes rutas:

#### gRPC Nativo
```
POST /chat.v1.ChatService/CreateRoom
POST /chat.v1.ChatService/GetRooms
POST /chat.v1.ChatService/SendMessage
POST /chat.v1.ChatService/StreamMessages
// ... todas las demás operaciones
```

#### gRPC-Web
```
POST /chat.v1.ChatService/CreateRoom (Content-Type: application/grpc-web)
POST /chat.v1.ChatService/GetRooms (Content-Type: application/grpc-web)
// ... con soporte para navegadores web
```

#### Connect Protocol
```
POST /chat.v1.ChatService/CreateRoom (Content-Type: application/json)
POST /chat.v1.ChatService/GetRooms (Content-Type: application/json)
// ... con JSON sobre HTTP
```

## Patrones de Diseño Implementados

### 1. Factory Pattern
```go
func RegisterServiceHandler() *vanguard.Service {
    // Crea y configura el servicio
}
```
- **Encapsulación**: Oculta complejidad de creación
- **Configuración**: Aplica configuración estándar
- **Reutilización**: Función reutilizable para testing

### 2. Dependency Injection
```go
chatv1connect.NewChatServiceHandler(NewHandler(), options...)
```
- **Handler**: Inyecta implementación específica
- **Options**: Inyecta configuración externa
- **Flexibilidad**: Permite diferentes configuraciones

### 3. Decorator Pattern (Implícito)
```go
options...
```
- **Middleware**: Cada opción puede agregar funcionalidad
- **Composición**: Múltiples decoradores aplicados en secuencia
- **Transparencia**: Handler original no se modifica

## Testing

### Unit Tests

```go
func TestRegisterServiceHandler(t *testing.T) {
    // Test que la función retorna un servicio válido
    service := RegisterServiceHandler()
    
    assert.NotNil(t, service)
    assert.IsType(t, &vanguard.Service{}, service)
}

func TestServiceHandlerOptions(t *testing.T) {
    // Test que las opciones se aplican correctamente
    opts := server.ServiceHandlerOptions()
    
    assert.NotNil(t, opts)
    // Verificar que contiene middleware esperado
}
```

### Integration Tests

```go
func TestServiceRegistration(t *testing.T) {
    // Test de integración completa
    service := RegisterServiceHandler()
    
    // Crear servidor de test
    mux := http.NewServeMux()
    service.RegisterRoutes(mux)
    
    server := httptest.NewServer(mux)
    defer server.Close()
    
    // Test que las rutas están registradas
    client := chatv1connect.NewChatServiceClient(
        http.DefaultClient,
        server.URL,
    )
    
    // Hacer request de test
    ctx := context.Background()
    req := &chatv1.GetRoomsRequest{}
    
    _, err := client.GetRooms(ctx, connect.NewRequest(req))
    // Verificar que el endpoint responde (aunque falle por auth)
    assert.Error(t, err) // Esperamos error de auth
    
    connectErr := err.(*connect.Error)
    assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}
```

### Mock para Testing

```go
func TestWithMockHandler(t *testing.T) {
    // Crear mock del handler
    mockHandler := &MockChatServiceHandler{}
    
    // Configurar expectativas
    mockHandler.On("GetRooms", mock.Anything, mock.Anything).
        Return(&chatv1.GetRoomsResponse{}, nil)
    
    // Crear servicio con mock
    service := vanguard.NewService(
        chatv1connect.NewChatServiceHandler(mockHandler, options...),
    )
    
    // Test del servicio
    // ...
}
```

## Consideraciones de Performance

### 1. Singleton Pattern para Options
```go
var options = server.ServiceHandlerOptions()
```
- **Ventaja**: Options se calculan una sola vez
- **Memory**: Reutilización de configuración
- **Consistency**: Mismas opciones para todas las instancias

### 2. Handler Creation
```go
NewHandler()
```
- **Per-Service**: Nueva instancia por servicio registrado
- **Stateless**: No mantiene estado entre requests
- **Thread-Safe**: Cada instancia es independiente

### 3. Middleware Efficiency
- **Chain**: Middleware se ejecuta en cadena
- **Overhead**: Cada middleware agrega latencia mínima
- **Optimization**: Middleware optimizado en el core

## Configuración Avanzada

### Custom Options

```go
func RegisterServiceHandlerWithCustomOptions(customOpts ...connect.HandlerOption) *vanguard.Service {
    allOpts := append(server.ServiceHandlerOptions(), customOpts...)
    return vanguard.NewService(
        chatv1connect.NewChatServiceHandler(NewHandler(), allOpts...),
    )
}
```

### Environment-Specific Configuration

```go
func RegisterServiceHandler() *vanguard.Service {
    opts := server.ServiceHandlerOptions()
    
    // Agregar opciones específicas del entorno
    if config.GetBool("debug.enabled") {
        opts = append(opts, withDebugMiddleware())
    }
    
    if config.GetBool("metrics.enabled") {
        opts = append(opts, withMetricsMiddleware())
    }
    
    return vanguard.NewService(
        chatv1connect.NewChatServiceHandler(NewHandler(), opts...),
    )
}
```

## Mejores Prácticas Implementadas

1. **Separation of Concerns**: Registro separado de implementación
2. **Configuration Management**: Opciones centralizadas
3. **Factory Pattern**: Creación encapsulada del servicio
4. **Dependency Injection**: Inyección de handler y opciones
5. **Consistency**: Uso de convenciones estándar
6. **Testability**: Estructura que facilita testing
7. **Flexibility**: Permite personalización de opciones

Este archivo, aunque simple, es crucial para la exposición del servicio de chat, proporcionando una interfaz limpia y configurada que integra perfectamente con el framework del servidor y las herramientas de observabilidad.