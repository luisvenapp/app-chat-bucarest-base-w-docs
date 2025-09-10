# Documentación Técnica: utils/errors.go

## Descripción General

El archivo `errors.go` define un catálogo centralizado de códigos de error utilizados en toda la aplicación. Implementa un patrón de constantes estructuradas que proporciona consistencia en el manejo de errores, facilita la internacionalización y mejora la mantenibilidad del código.

## Estructura del Archivo

### Definición de la Estructura de Errores

```go
var ERRORS = struct {
    NOT_FOUND                    string
    INVALID_REQUEST_DATA         string
    INVALID_OTP_CODE             string
    EXPIRED_OTP_CODE             string
    INVALID_CREDENTIALS          string
    TOO_MANY_OTP_REQUESTS        string
    EMAIL_ALREADY_IN_EXISTS      string
    PHONE_ALREADY_IN_EXISTS      string
    USER_NOT_FOUND               string
    USER_INACTIVE                string
    USER_BLOCKED                 string
    INVALID_TOKEN                string
    FILE_SIZE_MAX_LIMIT_EXCEEDED string
    FILE_INVALID_FORMAT          string
    GENERIC_ERROR                string
    RECORD_NOT_FOUND             string
    FOLDER_ALREADY_EXISTS        string
    FILE_ALREADY_EXISTS          string
    INTERNAL_SERVER_ERROR        string
    SMS_OTP_PROVIDER_ERROR       string
}{
    NOT_FOUND:                    "not_found",
    INVALID_REQUEST_DATA:         "invalid_request_data",
    INVALID_OTP_CODE:             "invalid_otp_code",
    EXPIRED_OTP_CODE:             "expired_otp_code",
    INVALID_CREDENTIALS:          "invalid_credentials",
    TOO_MANY_OTP_REQUESTS:        "too_many_otp_requests",
    EMAIL_ALREADY_IN_EXISTS:      "email_already_exists",
    PHONE_ALREADY_IN_EXISTS:      "phone_already_exists",
    USER_NOT_FOUND:               "user_not_found",
    USER_INACTIVE:                "user_inactive",
    USER_BLOCKED:                 "user_blocked",
    INVALID_TOKEN:                "invalid_token",
    FILE_SIZE_MAX_LIMIT_EXCEEDED: "file_size_max_limit_exceeded",
    FILE_INVALID_FORMAT:          "file_invalid_format",
    GENERIC_ERROR:                "generic_error",
    RECORD_NOT_FOUND:             "record_not_found",
    FOLDER_ALREADY_EXISTS:        "folder_already_exists",
    FILE_ALREADY_EXISTS:          "file_already_exists",
    INTERNAL_SERVER_ERROR:        "internal_server_error",
    SMS_OTP_PROVIDER_ERROR:       "sms_otp_provider_error",
}
```

## Análisis Detallado de Códigos de Error

### Errores Generales

#### `NOT_FOUND: "not_found"`
- **Propósito**: Recurso solicitado no existe
- **Código HTTP**: 404 Not Found
- **Uso típico**: Cuando se busca un registro por ID que no existe
- **Ejemplo**: Usuario, sala de chat, mensaje no encontrado

#### `INVALID_REQUEST_DATA: "invalid_request_data"`
- **Propósito**: Datos de entrada inválidos o malformados
- **Código HTTP**: 400 Bad Request
- **Uso típico**: Validación de parámetros de entrada
- **Ejemplo**: Campos requeridos faltantes, formato de datos incorrecto

#### `GENERIC_ERROR: "generic_error"`
- **Propósito**: Error genérico cuando no se puede clasificar específicamente
- **Código HTTP**: 500 Internal Server Error
- **Uso típico**: Errores inesperados o no categorizados
- **Ejemplo**: Fallos de sistema no específicos

#### `INTERNAL_SERVER_ERROR: "internal_server_error"`
- **Propósito**: Error interno del servidor
- **Código HTTP**: 500 Internal Server Error
- **Uso típico**: Errores de infraestructura o sistema
- **Ejemplo**: Fallos de base de datos, servicios externos

### Errores de Autenticación y Autorización

#### `INVALID_CREDENTIALS: "invalid_credentials"`
- **Propósito**: Credenciales de acceso incorrectas
- **Código HTTP**: 401 Unauthorized
- **Uso típico**: Login fallido
- **Ejemplo**: Email/password incorrectos

#### `INVALID_TOKEN: "invalid_token"`
- **Propósito**: Token de autenticación inválido o expirado
- **Código HTTP**: 401 Unauthorized
- **Uso típico**: Validación de tokens JWT
- **Ejemplo**: Token malformado, expirado o falsificado

#### `USER_NOT_FOUND: "user_not_found"`
- **Propósito**: Usuario específico no existe
- **Código HTTP**: 404 Not Found
- **Uso típico**: Operaciones específicas de usuario
- **Ejemplo**: Perfil de usuario, configuraciones

#### `USER_INACTIVE: "user_inactive"`
- **Propósito**: Usuario existe pero está inactivo
- **Código HTTP**: 403 Forbidden
- **Uso típico**: Control de estado de cuenta
- **Ejemplo**: Cuenta suspendida temporalmente

#### `USER_BLOCKED: "user_blocked"`
- **Propósito**: Usuario bloqueado permanentemente
- **Código HTTP**: 403 Forbidden
- **Uso típico**: Control de acceso por políticas
- **Ejemplo**: Usuario baneado por violaciones

### Errores de OTP (One-Time Password)

#### `INVALID_OTP_CODE: "invalid_otp_code"`
- **Propósito**: Código OTP incorrecto
- **Código HTTP**: 400 Bad Request
- **Uso típico**: Verificación de códigos de autenticación
- **Ejemplo**: Código SMS/Email incorrecto

#### `EXPIRED_OTP_CODE: "expired_otp_code"`
- **Propósito**: Código OTP válido pero expirado
- **Código HTTP**: 400 Bad Request
- **Uso típico**: Verificación temporal de códigos
- **Ejemplo**: Código válido pero fuera del tiempo límite

#### `TOO_MANY_OTP_REQUESTS: "too_many_otp_requests"`
- **Propósito**: Demasiadas solicitudes de OTP
- **Código HTTP**: 429 Too Many Requests
- **Uso típico**: Rate limiting para prevenir abuso
- **Ejemplo**: Múltiples solicitudes de código en poco tiempo

#### `SMS_OTP_PROVIDER_ERROR: "sms_otp_provider_error"`
- **Propósito**: Error en el proveedor de SMS
- **Código HTTP**: 503 Service Unavailable
- **Uso típico**: Fallos en servicios externos de SMS
- **Ejemplo**: Twilio, AWS SNS no disponible

### Errores de Duplicación

#### `EMAIL_ALREADY_IN_EXISTS: "email_already_exists"`
- **Propósito**: Email ya registrado en el sistema
- **Código HTTP**: 409 Conflict
- **Uso típico**: Registro de nuevos usuarios
- **Ejemplo**: Intento de registro con email existente

#### `PHONE_ALREADY_IN_EXISTS: "phone_already_exists"`
- **Propósito**: Número de teléfono ya registrado
- **Código HTTP**: 409 Conflict
- **Uso típico**: Registro o actualización de perfil
- **Ejemplo**: Teléfono ya asociado a otra cuenta

### Errores de Archivos

#### `FILE_SIZE_MAX_LIMIT_EXCEEDED: "file_size_max_limit_exceeded"`
- **Propósito**: Archivo excede el tamaño máximo permitido
- **Código HTTP**: 413 Payload Too Large
- **Uso típico**: Upload de archivos
- **Ejemplo**: Imagen de perfil, archivos adjuntos

#### `FILE_INVALID_FORMAT: "file_invalid_format"`
- **Propósito**: Formato de archivo no soportado
- **Código HTTP**: 400 Bad Request
- **Uso típico**: Validación de tipos de archivo
- **Ejemplo**: Subir .exe cuando solo se permiten imágenes

#### `FILE_ALREADY_EXISTS: "file_already_exists"`
- **Propósito**: Archivo ya existe en el destino
- **Código HTTP**: 409 Conflict
- **Uso típico**: Operaciones de archivo
- **Ejemplo**: Subir archivo con nombre duplicado

#### `FOLDER_ALREADY_EXISTS: "folder_already_exists"`
- **Propósito**: Directorio ya existe
- **Código HTTP**: 409 Conflict
- **Uso típico**: Creación de directorios
- **Ejemplo**: Crear carpeta con nombre existente

### Errores de Datos

#### `RECORD_NOT_FOUND: "record_not_found"`
- **Propósito**: Registro específico no encontrado en base de datos
- **Código HTTP**: 404 Not Found
- **Uso típico**: Operaciones de base de datos
- **Ejemplo**: Buscar registro por criterios específicos

## Patrones de Diseño Implementados

### 1. Constant Pool Pattern

**Implementación:**
```go
var ERRORS = struct {
    // Campos como constantes
}{
    // Valores inicializados
}
```

**Ventajas:**
- **Inmutabilidad**: Los valores no pueden ser modificados
- **Centralización**: Todos los errores en un lugar
- **Type Safety**: Acceso mediante dot notation
- **Autocompletado**: IDEs pueden sugerir códigos disponibles

### 2. Namespace Pattern

**Uso:**
```go
// Acceso namespaced
errors.New(ERRORS.INVALID_TOKEN)
errors.New(ERRORS.USER_NOT_FOUND)
```

**Beneficios:**
- **Organización**: Evita contaminación del namespace global
- **Claridad**: Origen claro de las constantes
- **Mantenimiento**: Fácil localizar y modificar

### 3. String-based Error Codes

**Ventajas sobre códigos numéricos:**
- **Legibilidad**: Códigos autodescriptivos
- **Debugging**: Fácil identificar errores en logs
- **Internacionalización**: Mapeo directo a mensajes localizados
- **API Consistency**: Códigos estables entre versiones

## Uso en la Aplicación

### En Handlers

```go
func (h *handler) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        return nil, connect.NewError(connect.CodeUnauthenticated, errors.New(ERRORS.INVALID_TOKEN))
    }
    
    user, err := h.repository.GetUser(ctx, userID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, connect.NewError(connect.CodeNotFound, errors.New(ERRORS.USER_NOT_FOUND))
        }
        return nil, connect.NewError(connect.CodeInternal, errors.New(ERRORS.INTERNAL_SERVER_ERROR))
    }
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### En Repository Layer

```go
func (r *repository) CreateUser(ctx context.Context, user *User) error {
    _, err := r.db.ExecContext(ctx, "INSERT INTO users (email, phone) VALUES ($1, $2)", user.Email, user.Phone)
    if err != nil {
        if isUniqueViolation(err, "email") {
            return errors.New(ERRORS.EMAIL_ALREADY_IN_EXISTS)
        }
        if isUniqueViolation(err, "phone") {
            return errors.New(ERRORS.PHONE_ALREADY_IN_EXISTS)
        }
        return errors.New(ERRORS.INTERNAL_SERVER_ERROR)
    }
    return nil
}
```

### En Validation Layer

```go
func ValidateFileUpload(file *multipart.FileHeader) error {
    // Validar tamaño
    if file.Size > MaxFileSize {
        return errors.New(ERRORS.FILE_SIZE_MAX_LIMIT_EXCEEDED)
    }
    
    // Validar formato
    if !isValidFormat(file.Header.Get("Content-Type")) {
        return errors.New(ERRORS.FILE_INVALID_FORMAT)
    }
    
    return nil
}
```

## Internacionalización

### Mapeo de Códigos a Mensajes

```go
// i18n/messages.go
var Messages = map[string]map[string]string{
    "en": {
        "not_found":           "Resource not found",
        "invalid_token":       "Invalid or expired token",
        "user_not_found":      "User not found",
        "email_already_exists": "Email already registered",
    },
    "es": {
        "not_found":           "Recurso no encontrado",
        "invalid_token":       "Token inválido o expirado",
        "user_not_found":      "Usuario no encontrado",
        "email_already_exists": "Email ya registrado",
    },
}

func GetMessage(code, lang string) string {
    if messages, ok := Messages[lang]; ok {
        if message, ok := messages[code]; ok {
            return message
        }
    }
    return Messages["en"][code] // Fallback a inglés
}
```

### Uso con Internacionalización

```go
func (h *handler) handleError(err error, lang string) *connect.Error {
    errorCode := err.Error()
    message := i18n.GetMessage(errorCode, lang)
    
    switch errorCode {
    case ERRORS.NOT_FOUND:
        return connect.NewError(connect.CodeNotFound, errors.New(message))
    case ERRORS.INVALID_TOKEN:
        return connect.NewError(connect.CodeUnauthenticated, errors.New(message))
    default:
        return connect.NewError(connect.CodeInternal, errors.New(message))
    }
}
```

## Testing

### Unit Tests para Códigos de Error

```go
func TestErrorCodes(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected string
    }{
        {"not found", ERRORS.NOT_FOUND, "not_found"},
        {"invalid token", ERRORS.INVALID_TOKEN, "invalid_token"},
        {"user not found", ERRORS.USER_NOT_FOUND, "user_not_found"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.code != tt.expected {
                t.Errorf("expected %s, got %s", tt.expected, tt.code)
            }
        })
    }
}
```

### Integration Tests

```go
func TestErrorHandling(t *testing.T) {
    // Test que el error correcto se retorna
    handler := NewHandler()
    
    req := &connect.Request[GetUserRequest]{}
    // No incluir token de autenticación
    
    _, err := handler.GetUser(context.Background(), req)
    
    connectErr := err.(*connect.Error)
    assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
    assert.Contains(t, connectErr.Message(), ERRORS.INVALID_TOKEN)
}
```

## Monitoreo y Métricas

### Tracking de Errores

```go
// metrics/errors.go
var errorCounter = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "api_errors_total",
        Help: "Total number of API errors by type",
    },
    []string{"error_code", "endpoint"},
)

func TrackError(errorCode, endpoint string) {
    errorCounter.WithLabelValues(errorCode, endpoint).Inc()
}
```

### Uso en Handlers

```go
func (h *handler) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        metrics.TrackError(ERRORS.INVALID_TOKEN, "GetUser")
        return nil, connect.NewError(connect.CodeUnauthenticated, errors.New(ERRORS.INVALID_TOKEN))
    }
    
    // ... resto de la lógica
}
```

## Mejores Prácticas Implementadas

1. **Consistencia**: Todos los errores siguen el mismo patrón de naming
2. **Inmutabilidad**: Estructura de solo lectura previene modificaciones accidentales
3. **Centralización**: Un solo lugar para todos los códigos de error
4. **Autodocumentación**: Nombres descriptivos que explican el error
5. **Extensibilidad**: Fácil agregar nuevos códigos de error
6. **Internacionalización**: Códigos preparados para múltiples idiomas
7. **Type Safety**: Acceso type-safe a través de la estructura

Este archivo es fundamental para el manejo consistente de errores en toda la aplicación, proporcionando una base sólida para la comunicación de errores tanto internamente como hacia los clientes de la API.