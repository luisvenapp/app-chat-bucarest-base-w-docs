# Documentación Técnica: repository/tokens/tokens.go

## Descripción General

El archivo `tokens.go` define la interfaz del repository para la gestión de tokens de dispositivos móviles utilizados en el sistema de notificaciones push. Proporciona una abstracción limpia para las operaciones de persistencia de tokens, siguiendo el patrón Repository y facilitando la implementación de diferentes estrategias de almacenamiento.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    
    tokensv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1"
)
```

**Análisis de Importaciones:**

- **`context`**: Para manejo de contexto, cancelación y timeouts en operaciones de base de datos
- **`tokensv1`**: Tipos generados de Protocol Buffers específicos para el servicio de tokens

## Interface TokensRepository

```go
type TokensRepository interface {
    SaveToken(ctx context.Context, userId int, room *tokensv1.SaveTokenRequest) error
}
```

**Análisis de la Interface:**

### Propósito de la Interface
- **Abstracción**: Define el contrato para operaciones de tokens sin especificar implementación
- **Flexibilidad**: Permite múltiples implementaciones (SQL, NoSQL, cache, etc.)
- **Testing**: Facilita la creación de mocks para testing unitario
- **Evolución**: Permite agregar nuevos métodos sin romper implementaciones existentes

### Método SaveToken

```go
SaveToken(ctx context.Context, userId int, room *tokensv1.SaveTokenRequest) error
```

**Análisis Detallado:**

#### Signature del Método
- **Nombre**: `SaveToken` - Verbo claro que indica la acción
- **Contexto**: Recibe `context.Context` para control de lifecycle
- **UserID**: `int` para asociar el token con un usuario específico
- **Request**: `*tokensv1.SaveTokenRequest` con los datos del token
- **Retorno**: Solo `error` - operación de escritura simple

#### Parámetros

##### `ctx context.Context`
- **Propósito**: Control de cancelación, timeouts y valores de contexto
- **Uso típico**:
  ```go
  // Con timeout
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  
  err := repo.SaveToken(ctx, userID, tokenRequest)
  ```
- **Beneficios**:
  - **Cancelación**: Permite cancelar operaciones largas
  - **Timeouts**: Evita operaciones que cuelguen indefinidamente
  - **Tracing**: Propagación de trace IDs para observabilidad

##### `userId int`
- **Propósito**: Identificador único del usuario propietario del token
- **Tipo**: `int` para eficiencia en índices de base de datos
- **Uso**: Clave foránea para asociar token con usuario
- **Validación**: Debe ser un ID válido de usuario existente

##### `room *tokensv1.SaveTokenRequest`
- **Propósito**: Datos completos del token a guardar
- **Tipo**: Puntero a estructura generada de protobuf
- **Contenido típico**:
  ```protobuf
  message SaveTokenRequest {
      string device_token = 1;    // Token FCM/APNS
      string platform = 2;        // "ios", "android", "web"
      string app_version = 3;     // Versión de la aplicación
      string device_id = 4;       // ID único del dispositivo
      string language = 5;        // Idioma preferido
      bool notifications_enabled = 6; // Si las notificaciones están habilitadas
  }
  ```

#### Retorno

##### `error`
- **Nil**: Operación exitosa
- **Non-nil**: Error específico que describe el problema
- **Tipos de errores esperados**:
  - Errores de validación de datos
  - Errores de conexión a base de datos
  - Errores de constraint violations (duplicados, etc.)
  - Errores de autorización

## Casos de Uso del Repository

### 1. Registro de Nuevo Token

```go
func RegisterDeviceToken(ctx context.Context, repo TokensRepository, userID int, deviceToken string, platform string) error {
    request := &tokensv1.SaveTokenRequest{
        DeviceToken: deviceToken,
        Platform:    platform,
        AppVersion:  "1.2.3",
        DeviceId:    generateDeviceID(),
        Language:    "es",
        NotificationsEnabled: true,
    }
    
    return repo.SaveToken(ctx, userID, request)
}
```

### 2. Actualización de Token Existente

```go
func UpdateDeviceToken(ctx context.Context, repo TokensRepository, userID int, oldToken, newToken string) error {
    // El repository debe manejar la lógica de actualización
    // (reemplazar token existente o crear nuevo registro)
    request := &tokensv1.SaveTokenRequest{
        DeviceToken: newToken,
        Platform:    "ios",
        AppVersion:  "1.2.4",
        DeviceId:    "same_device_id",
    }
    
    return repo.SaveToken(ctx, userID, request)
}
```

### 3. Registro de Múltiples Dispositivos

```go
func RegisterMultipleDevices(ctx context.Context, repo TokensRepository, userID int, devices []DeviceInfo) error {
    for _, device := range devices {
        request := &tokensv1.SaveTokenRequest{
            DeviceToken: device.Token,
            Platform:    device.Platform,
            AppVersion:  device.AppVersion,
            DeviceId:    device.DeviceID,
        }
        
        if err := repo.SaveToken(ctx, userID, request); err != nil {
            return fmt.Errorf("failed to save token for device %s: %w", device.DeviceID, err)
        }
    }
    return nil
}
```

## Implementaciones Típicas

### 1. SQL Implementation

```go
type SQLTokensRepository struct {
    db *sql.DB
}

func (r *SQLTokensRepository) SaveToken(ctx context.Context, userId int, req *tokensv1.SaveTokenRequest) error {
    query := `
        INSERT INTO device_tokens (user_id, device_token, platform, app_version, device_id, language, notifications_enabled, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        ON CONFLICT (user_id, device_id) 
        DO UPDATE SET 
            device_token = EXCLUDED.device_token,
            platform = EXCLUDED.platform,
            app_version = EXCLUDED.app_version,
            language = EXCLUDED.language,
            notifications_enabled = EXCLUDED.notifications_enabled,
            updated_at = NOW()
    `
    
    _, err := r.db.ExecContext(ctx, query, 
        userId, 
        req.DeviceToken, 
        req.Platform, 
        req.AppVersion, 
        req.DeviceId,
        req.Language,
        req.NotificationsEnabled,
    )
    
    return err
}
```

### 2. Cache-First Implementation

```go
type CachedTokensRepository struct {
    cache  cache.Cache
    sqlRepo TokensRepository
}

func (r *CachedTokensRepository) SaveToken(ctx context.Context, userId int, req *tokensv1.SaveTokenRequest) error {
    // Guardar en base de datos
    if err := r.sqlRepo.SaveToken(ctx, userId, req); err != nil {
        return err
    }
    
    // Invalidar cache del usuario
    cacheKey := fmt.Sprintf("user_tokens:%d", userId)
    r.cache.Delete(cacheKey)
    
    return nil
}
```

### 3. Batch Implementation

```go
type BatchTokensRepository struct {
    base TokensRepository
    batch []TokenSaveOperation
    mutex sync.Mutex
}

type TokenSaveOperation struct {
    UserID  int
    Request *tokensv1.SaveTokenRequest
}

func (r *BatchTokensRepository) SaveToken(ctx context.Context, userId int, req *tokensv1.SaveTokenRequest) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    r.batch = append(r.batch, TokenSaveOperation{
        UserID:  userId,
        Request: req,
    })
    
    // Flush batch si alcanza cierto tamaño
    if len(r.batch) >= 100 {
        return r.flushBatch(ctx)
    }
    
    return nil
}
```

## Consideraciones de Diseño

### 1. Simplicidad de la Interface
- **Un solo método**: Interface minimalista y enfocada
- **Operación atómica**: SaveToken maneja tanto inserción como actualización
- **Error handling**: Retorno simple de error

### 2. Flexibilidad de Implementación
- **Upsert logic**: Implementaciones pueden decidir si insertar o actualizar
- **Validation**: Cada implementación puede agregar validaciones específicas
- **Performance**: Optimizaciones específicas por tipo de storage

### 3. Extensibilidad Futura
```go
// Posibles extensiones futuras de la interface
type TokensRepository interface {
    SaveToken(ctx context.Context, userId int, req *tokensv1.SaveTokenRequest) error
    
    // Métodos que podrían agregarse:
    // GetUserTokens(ctx context.Context, userId int) ([]*Token, error)
    // DeleteToken(ctx context.Context, userId int, deviceId string) error
    // GetTokensByPlatform(ctx context.Context, platform string) ([]*Token, error)
    // UpdateTokenStatus(ctx context.Context, token string, active bool) error
}
```

## Patrones de Diseño Implementados

### 1. Repository Pattern
- **Abstracción**: Separa lógica de negocio de persistencia
- **Encapsulación**: Oculta detalles de implementación de storage
- **Testabilidad**: Facilita testing con mocks

### 2. Interface Segregation Principle
- **Específica**: Interface enfocada solo en operaciones de tokens
- **Cohesiva**: Todos los métodos están relacionados
- **Minimal**: Solo los métodos necesarios

### 3. Dependency Inversion
- **Abstracción**: Dependencia en interface, no en implementación concreta
- **Flexibilidad**: Permite cambiar implementaciones sin afectar clientes

## Testing

### Mock Implementation

```go
type MockTokensRepository struct {
    mock.Mock
}

func (m *MockTokensRepository) SaveToken(ctx context.Context, userId int, req *tokensv1.SaveTokenRequest) error {
    args := m.Called(ctx, userId, req)
    return args.Error(0)
}
```

### Test Examples

```go
func TestSaveToken(t *testing.T) {
    tests := []struct {
        name          string
        userID        int
        request       *tokensv1.SaveTokenRequest
        expectedError error
    }{
        {
            name:   "successful save",
            userID: 123,
            request: &tokensv1.SaveTokenRequest{
                DeviceToken: "valid_token",
                Platform:    "ios",
                AppVersion:  "1.0.0",
                DeviceId:    "device_123",
            },
            expectedError: nil,
        },
        {
            name:   "invalid user ID",
            userID: -1,
            request: &tokensv1.SaveTokenRequest{
                DeviceToken: "valid_token",
                Platform:    "android",
            },
            expectedError: errors.New("invalid user ID"),
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &MockTokensRepository{}
            mockRepo.On("SaveToken", mock.Anything, tt.userID, tt.request).Return(tt.expectedError)
            
            err := mockRepo.SaveToken(context.Background(), tt.userID, tt.request)
            
            if tt.expectedError != nil {
                assert.Error(t, err)
                assert.Equal(t, tt.expectedError.Error(), err.Error())
            } else {
                assert.NoError(t, err)
            }
            
            mockRepo.AssertExpectations(t)
        })
    }
}
```

## Consideraciones de Performance

### 1. Batch Operations
```go
// Para múltiples tokens del mismo usuario
func SaveMultipleTokens(ctx context.Context, repo TokensRepository, userID int, tokens []*tokensv1.SaveTokenRequest) error {
    // Implementación secuencial simple
    for _, token := range tokens {
        if err := repo.SaveToken(ctx, userID, token); err != nil {
            return err
        }
    }
    return nil
}
```

### 2. Connection Pooling
- **Database connections**: Reutilización de conexiones
- **Context timeout**: Timeouts apropiados para operaciones
- **Retry logic**: Manejo de errores transitorios

### 3. Caching Strategy
```go
// Cache de tokens para consultas frecuentes
type CachedTokensRepository struct {
    base  TokensRepository
    cache map[string]*tokensv1.SaveTokenRequest
    ttl   time.Duration
}
```

## Seguridad

### 1. Validation
```go
func (r *SQLTokensRepository) SaveToken(ctx context.Context, userId int, req *tokensv1.SaveTokenRequest) error {
    // Validar datos de entrada
    if userId <= 0 {
        return errors.New("invalid user ID")
    }
    
    if req.DeviceToken == "" {
        return errors.New("device token is required")
    }
    
    if !isValidPlatform(req.Platform) {
        return errors.New("invalid platform")
    }
    
    // Continuar con guardado...
}
```

### 2. SQL Injection Prevention
```go
// Usar parámetros preparados
query := `INSERT INTO device_tokens (user_id, device_token) VALUES ($1, $2)`
_, err := r.db.ExecContext(ctx, query, userId, req.DeviceToken)
```

### 3. Rate Limiting
```go
// Limitar número de tokens por usuario
func (r *SQLTokensRepository) SaveToken(ctx context.Context, userId int, req *tokensv1.SaveTokenRequest) error {
    // Verificar límite de tokens por usuario
    count, err := r.getUserTokenCount(ctx, userId)
    if err != nil {
        return err
    }
    
    if count >= maxTokensPerUser {
        return errors.New("maximum tokens per user exceeded")
    }
    
    // Continuar con guardado...
}
```

## Mejores Prácticas Implementadas

1. **Context Propagation**: Uso de context para control de lifecycle
2. **Error Handling**: Retorno explícito de errores
3. **Type Safety**: Uso de tipos generados desde protobuf
4. **Interface Design**: Interface simple y enfocada
5. **Extensibility**: Diseño que permite extensiones futuras
6. **Testing Support**: Estructura que facilita mocking

## Áreas de Mejora Identificadas

1. **Batch Operations**: Agregar soporte para operaciones en lote
2. **Query Methods**: Agregar métodos para consultar tokens existentes
3. **Cleanup Methods**: Métodos para eliminar tokens obsoletos
4. **Status Management**: Manejo de estado de tokens (activo/inactivo)
5. **Metrics**: Instrumentación para observabilidad

Este archivo define una base sólida para la gestión de tokens de dispositivos, proporcionando una abstracción limpia que facilita la implementación de diferentes estrategias de almacenamiento y la evolución futura del sistema de notificaciones.