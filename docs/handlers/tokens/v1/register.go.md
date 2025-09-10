# Documentación Técnica: handlers/tokens/v1/register.go

## Descripción General

El archivo `register.go` del servicio de tokens implementa la función de registro para el servicio de gestión de tokens de dispositivos móviles. Siguiendo el mismo patrón que el servicio de chat, proporciona la configuración y exposición del servicio gRPC de tokens a través del framework Vanguard.

## Estructura del Archivo

### Importaciones

```go
import (
    "connectrpc.com/vanguard"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1/tokensv1connect"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/server"
)
```

**Análisis de Importaciones:**

- **`vanguard`**: Framework de ConnectRPC para servicios gRPC-Web
- **`tokensv1connect`**: Interfaz generada del servicio de tokens desde Protocol Buffers
- **`server`**: Módulo del core para opciones de configuración de servicios

## Variable de Configuración

```go
var options = server.ServiceHandlerOptions()
```

**Análisis:**

### Configuración Compartida
- **Consistencia**: Utiliza las mismas opciones que el servicio de chat
- **Centralización**: Configuración gestionada desde el core
- **Uniformidad**: Comportamiento consistente entre servicios

### Middleware Aplicado
Al igual que el servicio de chat, incluye:

#### 1. **Authentication Middleware**
- **Validación de tokens**: Verifica JWT en cada request
- **Autorización**: Asegura que solo usuarios autenticados puedan registrar tokens
- **Error handling**: Retorna errores 401 para requests no autenticados

#### 2. **Logging Middleware**
- **Request logging**: Registra cada request de token
- **Performance tracking**: Mide tiempo de respuesta
- **Error logging**: Registra errores para debugging

#### 3. **Metrics Middleware**
- **Contadores**: Número de tokens registrados
- **Latencia**: Tiempo de procesamiento de requests
- **Errores**: Rate de errores por tipo

## Función RegisterServiceHandler

```go
func RegisterServiceHandler() *vanguard.Service {
    return vanguard.NewService(tokensv1connect.NewTokensServiceHandler(handler, options...))
}
```

**Análisis Detallado:**

### Diferencias con el Servicio de Chat

#### Uso de Variable Global `handler`
```go
tokensv1connect.NewTokensServiceHandler(handler, options...)
```

**Comparación:**
- **Chat Service**: `NewHandler()` - Crea nueva instancia
- **Tokens Service**: `handler` - Usa variable global

**Implicaciones:**

##### Ventajas del Approach de Tokens
- **Singleton**: Una sola instancia del handler
- **Memory Efficiency**: Menor uso de memoria
- **Shared State**: Posibilidad de compartir estado (aunque no se usa)

##### Desventajas del Approach de Tokens
- **Global State**: Dependencia en variable global
- **Testing**: Más difícil de testear con mocks
- **Concurrency**: Potenciales problemas de concurrencia

##### Recomendación de Mejora
```go
func RegisterServiceHandler() *vanguard.Service {
    return vanguard.NewService(
        tokensv1connect.NewTokensServiceHandler(
            NewHandler(), // Crear nueva instancia como en chat
            options...,
        ),
    )
}

func NewHandler() tokensv1connect.TokensServiceHandler {
    return &handlerImpl{
        tokensRepo: tokensrepository.NewSQLTokensRepository(database.DB()),
    }
}
```

### Proceso de Construcción

#### 1. Referencia al Handler Global
```go
handler
```
- **Variable**: Definida en `handler.go` como `var handler tokensv1connect.TokensServiceHandler = &handlerImpl{}`
- **Inicialización**: Se inicializa al cargar el paquete
- **Tipo**: Implementa la interfaz `tokensv1connect.TokensServiceHandler`

#### 2. Creación del Service Handler
```go
tokensv1connect.NewTokensServiceHandler(handler, options...)
```
- **Generated Function**: Función generada desde Protocol Buffers
- **Handler**: Instancia global del handler
- **Options**: Mismas opciones que el servicio de chat
- **Result**: Handler gRPC configurado

#### 3. Creación del Servicio Vanguard
```go
vanguard.NewService(...)
```
- **Wrapper**: Envuelve el handler en servicio Vanguard
- **Protocols**: Soporte para gRPC, gRPC-Web y Connect
- **Routes**: Configuración automática de rutas

## Rutas Expuestas

### Endpoints del Servicio de Tokens

#### gRPC Nativo
```
POST /tokens.v1.TokensService/SaveToken
```

#### gRPC-Web
```
POST /tokens.v1.TokensService/SaveToken (Content-Type: application/grpc-web)
```

#### Connect Protocol
```
POST /tokens.v1.TokensService/SaveToken (Content-Type: application/json)
```

### Ejemplo de Request/Response

#### Request JSON (Connect Protocol)
```json
{
    "device_token": "fcm_token_abc123...",
    "platform": "android",
    "app_version": "1.2.3",
    "device_id": "device_unique_id",
    "language": "es",
    "notifications_enabled": true
}
```

#### Response JSON
```json
{
    "success": true
}
```

## Integración con el Sistema

### Registro en handlers/handlers.go

```go
var RegisterServicesFns = []server.RegisterServiceFn{
    chatv1handler.RegisterServiceHandler,
    tokensv1handler.RegisterServiceHandler, // Esta función
}
```

### Flujo de Registro

```mermaid
graph TD
    A[main.go] --> B[server.NewServer()]
    B --> C[handlers.RegisterServicesFns]
    C --> D[tokensv1handler.RegisterServiceHandler()]
    D --> E[tokensv1connect.NewTokensServiceHandler()]
    E --> F[Apply options...]
    F --> G[vanguard.NewService()]
    G --> H[Register routes]
    
    I[Global handler variable] --> E
    J[server.ServiceHandlerOptions()] --> F
```

## Casos de Uso del Servicio

### 1. Registro de Token FCM (Android)

```bash
curl -X POST http://localhost:8080/tokens.v1.TokensService/SaveToken \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt_token>" \
  -d '{
    "device_token": "fcm_token_android_123",
    "platform": "android",
    "app_version": "1.2.3",
    "device_id": "android_device_456"
  }'
```

### 2. Registro de Token APNS (iOS)

```bash
curl -X POST http://localhost:8080/tokens.v1.TokensService/SaveToken \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt_token>" \
  -d '{
    "device_token": "apns_token_ios_789",
    "platform": "ios",
    "app_version": "1.2.3",
    "device_id": "ios_device_012"
  }'
```

### 3. Registro de Token Web Push

```bash
curl -X POST http://localhost:8080/tokens.v1.TokensService/SaveToken \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt_token>" \
  -d '{
    "device_token": "web_push_token_345",
    "platform": "web",
    "app_version": "1.2.3",
    "device_id": "web_browser_678"
  }'
```

## Testing

### Unit Tests

```go
func TestRegisterServiceHandler(t *testing.T) {
    // Test que la función retorna un servicio válido
    service := RegisterServiceHandler()
    
    assert.NotNil(t, service)
    assert.IsType(t, &vanguard.Service{}, service)
}

func TestGlobalHandlerExists(t *testing.T) {
    // Test que el handler global está inicializado
    assert.NotNil(t, handler)
    assert.Implements(t, (*tokensv1connect.TokensServiceHandler)(nil), handler)
}
```

### Integration Tests

```go
func TestTokensServiceIntegration(t *testing.T) {
    // Setup test server
    service := RegisterServiceHandler()
    mux := http.NewServeMux()
    service.RegisterRoutes(mux)
    
    server := httptest.NewServer(mux)
    defer server.Close()
    
    // Create client
    client := tokensv1connect.NewTokensServiceClient(
        http.DefaultClient,
        server.URL,
    )
    
    // Test SaveToken endpoint
    ctx := context.Background()
    req := &tokensv1.SaveTokenRequest{
        DeviceToken: "test_token",
        Platform:    "test",
        AppVersion:  "1.0.0",
        DeviceId:    "test_device",
    }
    
    // Should fail with auth error (no token provided)
    _, err := client.SaveToken(ctx, connect.NewRequest(req))
    assert.Error(t, err)
    
    connectErr := err.(*connect.Error)
    assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}
```

### Mock Testing

```go
func TestWithMockHandler(t *testing.T) {
    // Backup original handler
    originalHandler := handler
    defer func() { handler = originalHandler }()
    
    // Set mock handler
    mockHandler := &MockTokensServiceHandler{}
    handler = mockHandler
    
    // Configure mock expectations
    mockHandler.On("SaveToken", mock.Anything, mock.Anything).
        Return(&tokensv1.SaveTokenResponse{Success: true}, nil)
    
    // Test service registration with mock
    service := RegisterServiceHandler()
    assert.NotNil(t, service)
    
    // Test would continue with actual service calls...
}
```

## Consideraciones de Arquitectura

### Comparación de Patrones

#### Chat Service Pattern (Recomendado)
```go
func RegisterServiceHandler() *vanguard.Service {
    return vanguard.NewService(
        chatv1connect.NewChatServiceHandler(NewHandler(), options...)
    )
}
```

**Ventajas:**
- **Isolation**: Cada registro crea nueva instancia
- **Testing**: Fácil de testear con dependency injection
- **Flexibility**: Permite diferentes configuraciones por instancia

#### Tokens Service Pattern (Actual)
```go
var handler tokensv1connect.TokensServiceHandler = &handlerImpl{}

func RegisterServiceHandler() *vanguard.Service {
    return vanguard.NewService(
        tokensv1connect.NewTokensServiceHandler(handler, options...)
    )
}
```

**Ventajas:**
- **Simplicity**: Menos código
- **Memory**: Una sola instancia

**Desventajas:**
- **Global State**: Dependencia en variable global
- **Testing**: Más difícil de testear
- **Flexibility**: Menos flexible para configuraciones diferentes

### Refactoring Recomendado

```go
// Eliminar variable global
// var handler tokensv1connect.TokensServiceHandler = &handlerImpl{}

// Agregar función factory
func NewHandler() tokensv1connect.TokensServiceHandler {
    return &handlerImpl{
        tokensRepo: tokensrepository.NewSQLTokensRepository(database.DB()),
    }
}

// Actualizar función de registro
func RegisterServiceHandler() *vanguard.Service {
    return vanguard.NewService(
        tokensv1connect.NewTokensServiceHandler(NewHandler(), options...)
    )
}
```

## Monitoreo y Observabilidad

### Métricas Específicas del Servicio

```go
var (
    tokensRegistered = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "tokens_service_registrations_total",
            Help: "Total number of device token registrations",
        },
        []string{"platform", "status"},
    )
    
    tokenRegistrationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "tokens_service_registration_duration_seconds",
            Help: "Time taken to register device tokens",
        },
        []string{"platform"},
    )
)
```

### Logging Específico

```go
func (h *handlerImpl) SaveToken(ctx context.Context, req *connect.Request[tokensv1.SaveTokenRequest]) (*connect.Response[tokensv1.SaveTokenResponse], error) {
    log.Info("Token registration request",
        "platform", req.Msg.Platform,
        "app_version", req.Msg.AppVersion,
        "user_id", getUserIDFromContext(ctx),
    )
    
    // ... lógica del handler
}
```

## Seguridad

### Validación de Entrada

```go
func validateTokenRequest(req *tokensv1.SaveTokenRequest) error {
    if req.DeviceToken == "" {
        return errors.New("device_token is required")
    }
    
    if !isValidPlatform(req.Platform) {
        return errors.New("invalid platform")
    }
    
    if len(req.DeviceToken) > maxTokenLength {
        return errors.New("device_token too long")
    }
    
    return nil
}
```

### Rate Limiting

```go
func rateLimitMiddleware() connect.UnaryInterceptorFunc {
    limiter := rate.NewLimiter(rate.Every(time.Minute), 10) // 10 requests per minute
    
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            if !limiter.Allow() {
                return nil, connect.NewError(connect.CodeResourceExhausted, 
                    errors.New("rate limit exceeded"))
            }
            return next(ctx, req)
        }
    }
}
```

## Mejores Prácticas Implementadas

1. **Consistent Configuration**: Uso de opciones centralizadas
2. **Service Registration**: Patrón estándar de registro
3. **Protocol Support**: Soporte para múltiples protocolos
4. **Middleware Integration**: Integración con middleware del core
5. **Error Handling**: Manejo consistente de errores

## Áreas de Mejora Identificadas

1. **Eliminate Global State**: Refactorizar para eliminar variable global
2. **Factory Pattern**: Implementar patrón factory como en chat service
3. **Dependency Injection**: Mejorar inyección de dependencias
4. **Testing Support**: Facilitar testing con mocks
5. **Configuration Flexibility**: Permitir configuraciones específicas

Este archivo, aunque funcional, se beneficiaría de seguir más de cerca el patrón implementado en el servicio de chat para mayor consistencia y mantenibilidad.