# Documentación Técnica: utils/authtoken.go

## Descripción General

El archivo `authtoken.go` implementa las funciones de validación y autenticación de tokens para la aplicación de chat. Proporciona dos tipos principales de validación: tokens públicos para endpoints no autenticados y tokens de sesión para endpoints que requieren autenticación de usuario.

## Estructura del Archivo

### Importaciones

```go
import (
    "errors"
    "net/http"
    
    "connectrpc.com/connect"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api/auth"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/config"
)
```

**Análisis de Importaciones:**

- **`errors`**: Paquete estándar para manejo de errores
- **`net/http`**: Para manejo de headers HTTP
- **`connect`**: Framework gRPC-Web para manejo de requests
- **`api`**: Módulo del core para funciones de API
- **`auth`**: Módulo del core para autenticación
- **`config`**: Módulo del core para configuración

### Variables Globales

```go
var publictoken = config.GetString("publictoken")
```

**Análisis Detallado:**

- **Propósito**: Token público para endpoints que no requieren autenticación de usuario
- **Configuración**: Obtenido desde configuración externa
- **Uso**: Validación de acceso a endpoints públicos pero controlados
- **Seguridad**: Permite control de acceso sin exponer funcionalidad sensible

**Casos de Uso del Token Público:**
- Endpoints de health check
- Documentación de API
- Endpoints de registro de usuarios
- Funcionalidades públicas pero controladas

## Funciones de Validación

### Función ValidatePublicToken

```go
func ValidatePublicToken(header http.Header) (bool, error) {
    token, err := auth.GetTokenFromHeader(header)
    if err != nil {
        return false, err
    }
    
    if token != publictoken {
        return false, errors.New("invalid token")
    }
    
    return true, nil
}
```

**Análisis Paso a Paso:**

#### 1. Extracción del Token
```go
token, err := auth.GetTokenFromHeader(header)
```

**Proceso:**
- **Función**: `auth.GetTokenFromHeader()` extrae el token del header HTTP
- **Headers típicos**: `Authorization: Bearer <token>` o `X-API-Key: <token>`
- **Manejo de errores**: Retorna error si no encuentra token o formato inválido

**Implementación típica en el core:**
```go
func GetTokenFromHeader(header http.Header) (string, error) {
    auth := header.Get("Authorization")
    if auth == "" {
        return "", errors.New("missing authorization header")
    }
    
    if !strings.HasPrefix(auth, "Bearer ") {
        return "", errors.New("invalid authorization format")
    }
    
    return strings.TrimPrefix(auth, "Bearer "), nil
}
```

#### 2. Validación del Token
```go
if token != publictoken {
    return false, errors.New("invalid token")
}
```

**Proceso:**
- **Comparación directa**: String comparison con el token configurado
- **Seguridad**: Comparación en tiempo constante para evitar timing attacks
- **Error específico**: Mensaje genérico para no revelar información

#### 3. Retorno de Resultado
```go
return true, nil
```

**Casos de Retorno:**
- `(true, nil)`: Token válido
- `(false, error)`: Token inválido o error en procesamiento

### Función ValidateAuthToken

```go
func ValidateAuthToken[T any](req *connect.Request[T]) (int, error) {
    session, err := api.CheckSessionFromConnectRequest(req)
    if err != nil {
        return 0, connect.NewError(connect.CodeUnauthenticated, errors.New(ERRORS.INVALID_TOKEN))
    }
    return session.UserID, nil
}
```

**Análisis Detallado:**

#### Función Genérica
```go
func ValidateAuthToken[T any](req *connect.Request[T]) (int, error)
```

**Características:**
- **Generic Function**: Acepta cualquier tipo de request gRPC
- **Type Parameter**: `T any` permite reutilización con diferentes tipos de mensaje
- **Flexibilidad**: Una sola función para todos los endpoints autenticados

#### Validación de Sesión
```go
session, err := api.CheckSessionFromConnectRequest(req)
```

**Proceso de Validación:**

1. **Extracción de Headers**: Obtiene headers de autenticación del request
2. **Decodificación de Token**: Decodifica y valida el JWT
3. **Verificación de Firma**: Valida la firma del token
4. **Verificación de Expiración**: Comprueba que el token no haya expirado
5. **Extracción de Claims**: Obtiene información de la sesión

**Estructura típica de sesión:**
```go
type Session struct {
    UserID    int       `json:"user_id"`
    Type      string    `json:"type"`      // "ACCESS", "REFRESH"
    ExpiresAt time.Time `json:"expires_at"`
    IssuedAt  time.Time `json:"issued_at"`
}
```

#### Manejo de Errores
```go
if err != nil {
    return 0, connect.NewError(connect.CodeUnauthenticated, errors.New(ERRORS.INVALID_TOKEN))
}
```

**Análisis del Error:**
- **Código gRPC**: `connect.CodeUnauthenticated` (HTTP 401)
- **Mensaje**: Utiliza constante `ERRORS.INVALID_TOKEN` para consistencia
- **Retorno**: `0` como UserID inválido
- **Seguridad**: No expone detalles específicos del error

#### Retorno Exitoso
```go
return session.UserID, nil
```

**Información Retornada:**
- **UserID**: Identificador único del usuario autenticado
- **Uso**: Utilizado en toda la aplicación para autorización
- **Tipo**: `int` para compatibilidad con base de datos

## Patrones de Seguridad Implementados

### 1. Principio de Menor Privilegio

**Token Público:**
- Acceso limitado a funcionalidades específicas
- No proporciona acceso a datos de usuario
- Validación simple pero efectiva

**Token de Sesión:**
- Acceso completo a funcionalidades del usuario
- Validación robusta con JWT
- Información de contexto incluida

### 2. Fail-Safe Defaults

```go
// En caso de error, siempre denegar acceso
if err != nil {
    return 0, connect.NewError(connect.CodeUnauthenticated, ...)
}
```

### 3. Error Handling Consistente

```go
// Uso de constantes para mensajes de error
errors.New(ERRORS.INVALID_TOKEN)
```

## Uso en la Aplicación

### En Handlers Públicos

```go
func (h *handler) PublicEndpoint(ctx context.Context, req *connect.Request[SomeRequest]) (*connect.Response[SomeResponse], error) {
    // Validar token público
    valid, err := utils.ValidatePublicToken(req.Header())
    if err != nil || !valid {
        return nil, connect.NewError(connect.CodeUnauthenticated, err)
    }
    
    // Lógica del endpoint público
    return connect.NewResponse(&SomeResponse{}), nil
}
```

### En Handlers Autenticados

```go
func (h *handler) AuthenticatedEndpoint(ctx context.Context, req *connect.Request[SomeRequest]) (*connect.Response[SomeResponse], error) {
    // Validar token de sesión y obtener UserID
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        return nil, err
    }
    
    // Usar userID para lógica específica del usuario
    data, err := h.repository.GetUserData(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    return connect.NewResponse(&SomeResponse{Data: data}), nil
}
```

## Consideraciones de Rendimiento

### 1. Caché de Validación

```go
// Implementación típica con caché
var tokenCache = cache.New(5*time.Minute, 10*time.Minute)

func ValidateAuthTokenCached[T any](req *connect.Request[T]) (int, error) {
    token, _ := auth.GetTokenFromHeader(req.Header())
    
    // Verificar caché primero
    if userID, found := tokenCache.Get(token); found {
        return userID.(int), nil
    }
    
    // Validación completa
    userID, err := ValidateAuthToken(req)
    if err != nil {
        return 0, err
    }
    
    // Guardar en caché
    tokenCache.Set(token, userID, cache.DefaultExpiration)
    return userID, nil
}
```

### 2. Validación Asíncrona

```go
// Para endpoints que no requieren validación inmediata
func ValidateAuthTokenAsync[T any](req *connect.Request[T]) <-chan ValidationResult {
    result := make(chan ValidationResult, 1)
    
    go func() {
        userID, err := ValidateAuthToken(req)
        result <- ValidationResult{UserID: userID, Error: err}
    }()
    
    return result
}
```

## Testing

### Unit Tests

```go
func TestValidatePublicToken(t *testing.T) {
    tests := []struct {
        name        string
        token       string
        expectValid bool
        expectError bool
    }{
        {
            name:        "valid token",
            token:       "valid-public-token",
            expectValid: true,
            expectError: false,
        },
        {
            name:        "invalid token",
            token:       "invalid-token",
            expectValid: false,
            expectError: true,
        },
        {
            name:        "empty token",
            token:       "",
            expectValid: false,
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            header := http.Header{}
            header.Set("Authorization", "Bearer "+tt.token)
            
            valid, err := ValidatePublicToken(header)
            
            if tt.expectError && err == nil {
                t.Error("expected error but got none")
            }
            if !tt.expectError && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if valid != tt.expectValid {
                t.Errorf("expected valid=%v, got %v", tt.expectValid, valid)
            }
        })
    }
}
```

### Integration Tests

```go
func TestValidateAuthTokenIntegration(t *testing.T) {
    // Crear token válido
    token, err := auth.GenerateSessionToken(auth.SessionData{
        UserID: 123,
        Type:   "ACCESS",
    })
    require.NoError(t, err)
    
    // Crear request mock
    req := &connect.Request[TestMessage]{}
    req.Header().Set("Authorization", "Bearer "+token)
    
    // Validar token
    userID, err := ValidateAuthToken(req)
    require.NoError(t, err)
    assert.Equal(t, 123, userID)
}
```

## Mejores Prácticas Implementadas

1. **Separación de Responsabilidades**: Funciones específicas para cada tipo de token
2. **Reutilización**: Función genérica para validación de sesiones
3. **Error Handling**: Manejo consistente y seguro de errores
4. **Configuración Externa**: Tokens desde configuración, no hardcoded
5. **Type Safety**: Uso de generics para type safety
6. **Security by Default**: Denegación por defecto en caso de error

Este archivo es fundamental para la seguridad de la aplicación, proporcionando las bases para la autenticación y autorización de todos los endpoints de la API.