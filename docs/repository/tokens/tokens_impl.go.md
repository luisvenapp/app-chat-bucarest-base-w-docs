# Documentación Técnica: repository/tokens/tokens_impl.go

## Descripción General

El archivo `tokens_impl.go` implementa la interfaz `TokensRepository` utilizando PostgreSQL como backend de almacenamiento. Proporciona la funcionalidad concreta para persistir tokens de dispositivos móviles, utilizando transacciones para garantizar consistencia y el patrón Query Builder para construcción segura de consultas SQL.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    "database/sql"
    
    sq "github.com/Masterminds/squirrel"
    tokensv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1"
    dbpq "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/postgres"
)
```

**Análisis de Importaciones:**

- **`context`**: Para manejo de contexto, cancelación y timeouts
- **`database/sql`**: Interfaz estándar de Go para bases de datos SQL
- **`squirrel`**: Query builder para construcción segura de consultas SQL
- **`tokensv1`**: Tipos generados de Protocol Buffers para tokens
- **`dbpq`**: Módulo del core para utilidades de PostgreSQL

## Estructura SQLTokensRepository

```go
type SQLTokensRepository struct {
    db *sql.DB
}
```

**Análisis de la Estructura:**

### Campo `db *sql.DB`
- **Propósito**: Conexión a la base de datos PostgreSQL
- **Tipo**: Puntero a la interfaz estándar de SQL de Go
- **Uso**: Ejecutar consultas y transacciones
- **Thread Safety**: `sql.DB` es thread-safe por diseño

### Características del Diseño
- **Simplicidad**: Estructura mínima con una sola dependencia
- **Stateless**: No mantiene estado entre operaciones
- **Dependency Injection**: Recibe conexión DB en el constructor
- **Interface Compliance**: Implementa `TokensRepository`

## Función Constructor

```go
func NewSQLTokensRepository(db *sql.DB) TokensRepository {
    return &SQLTokensRepository{
        db: db,
    }
}
```

**Análisis del Constructor:**

### Patrón Factory
- **Input**: Conexión de base de datos
- **Output**: Interfaz `TokensRepository`
- **Encapsulación**: Oculta implementación concreta
- **Dependency Injection**: Inyecta dependencia externa

### Ventajas del Approach
- **Testability**: Fácil de testear con mocks
- **Flexibility**: Permite diferentes implementaciones
- **Decoupling**: Desacopla interfaz de implementación

## Función SaveToken

```go
func (r *SQLTokensRepository) SaveToken(ctx context.Context, userId int, room *tokensv1.SaveTokenRequest) error {
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    query := dbpq.QueryBuilder().
        Insert("public.messaging_token").
        SetMap(sq.Eq{
            "token":            room.Token,
            "platform":         room.Platform,
            "platform_version": room.PlatformVersion,
            "device":           room.Device,
            "lang":             room.Lang,
            "is_voip":          room.IsVoip,
            "debug":            room.Debug,
            "user_id":          userId,
            "created_at":       sq.Expr("NOW()"),
        }).
        Suffix("RETURNING id")
    
    queryString, args, err := query.ToSql()
    if err != nil {
        return err
    }
    
    rows, err := tx.QueryContext(ctx, queryString, args...)
    if err != nil {
        return err
    }
    rows.Close()
    
    err = tx.Commit()
    if err != nil {
        return err
    }
    
    return nil
}
```

**Análisis Detallado:**

### 1. Gestión de Transacciones

#### Inicio de Transacción
```go
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()
```

**Características:**
- **Context-aware**: Usa `BeginTx` con contexto para cancelación
- **Isolation level**: `nil` usa el nivel por defecto (READ COMMITTED)
- **Defer rollback**: Garantiza rollback en caso de error o panic
- **Error handling**: Retorna inmediatamente si falla el inicio

**Ventajas del Uso de Transacciones:**
- **Atomicidad**: Operación todo-o-nada
- **Consistencia**: Estado consistente en caso de error
- **Isolation**: Aislamiento de operaciones concurrentes
- **Durability**: Cambios persistentes solo al hacer commit

### 2. Construcción de Query con Squirrel

```go
query := dbpq.QueryBuilder().
    Insert("public.messaging_token").
    SetMap(sq.Eq{
        "token":            room.Token,
        "platform":         room.Platform,
        "platform_version": room.PlatformVersion,
        "device":           room.Device,
        "lang":             room.Lang,
        "is_voip":          room.IsVoip,
        "debug":            room.Debug,
        "user_id":          userId,
        "created_at":       sq.Expr("NOW()"),
    }).
    Suffix("RETURNING id")
```

**Análisis del Query Builder:**

#### `dbpq.QueryBuilder()`
- **Propósito**: Obtiene builder configurado para PostgreSQL
- **Configuración**: Incluye placeholder style ($1, $2, etc.)
- **Consistencia**: Configuración estándar para toda la aplicación

#### `Insert("public.messaging_token")`
- **Tabla**: `public.messaging_token` (esquema explícito)
- **Operación**: INSERT statement
- **Seguridad**: Nombre de tabla escapado automáticamente

#### `SetMap(sq.Eq{...})`
- **Propósito**: Define columnas y valores para insertar
- **Type Safety**: Usa map para asociar columnas con valores
- **SQL Injection**: Protegido por parámetros preparados

#### Mapeo de Campos

##### Campos del Request
```go
"token":            room.Token,            // Token FCM/APNS
"platform":         room.Platform,         // "ios", "android", "web"
"platform_version": room.PlatformVersion,  // Versión del OS
"device":           room.Device,           // Modelo del dispositivo
"lang":             room.Lang,             // Idioma preferido
"is_voip":          room.IsVoip,           // Soporte VoIP
"debug":            room.Debug,            // Modo debug
```

##### Campos Adicionales
```go
"user_id":          userId,                // ID del usuario autenticado
"created_at":       sq.Expr("NOW()"),     // Timestamp de creación
```

#### `Suffix("RETURNING id")`
- **Propósito**: Retorna el ID generado del registro insertado
- **PostgreSQL Feature**: Cláusula RETURNING específica de PostgreSQL
- **Uso**: Permite obtener el ID sin consulta adicional

### 3. Generación y Ejecución de SQL

```go
queryString, args, err := query.ToSql()
if err != nil {
    return err
}
```

**Proceso:**
- **Generación**: Convierte builder a SQL string y argumentos
- **Validación**: Verifica que el query sea válido
- **Parámetros**: Genera lista de argumentos para prepared statement

**SQL Generado (ejemplo):**
```sql
INSERT INTO public.messaging_token (
    token, platform, platform_version, device, lang, 
    is_voip, debug, user_id, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW()
) RETURNING id
```

### 4. Ejecución de la Query

```go
rows, err := tx.QueryContext(ctx, queryString, args...)
if err != nil {
    return err
}
rows.Close()
```

**Análisis:**
- **QueryContext**: Ejecuta query con contexto para cancelación
- **Transacción**: Usa la transacción iniciada anteriormente
- **Prepared Statement**: Usa parámetros para prevenir SQL injection
- **Resource Management**: Cierra rows inmediatamente

**Nota sobre rows.Close():**
- **Necesario**: Aunque no leemos los resultados, debemos cerrar rows
- **Resource Leak**: Sin close(), puede causar leak de conexiones
- **Best Practice**: Siempre cerrar rows después de QueryContext

### 5. Commit de la Transacción

```go
err = tx.Commit()
if err != nil {
    return err
}

return nil
```

**Proceso:**
- **Commit**: Confirma todos los cambios de la transacción
- **Error handling**: Retorna error si el commit falla
- **Success**: Retorna nil si todo es exitoso

## Esquema de Base de Datos

### Tabla messaging_token

```sql
CREATE TABLE public.messaging_token (
    id SERIAL PRIMARY KEY,
    token VARCHAR(500) NOT NULL,
    platform VARCHAR(20) NOT NULL,
    platform_version VARCHAR(50),
    device VARCHAR(100),
    lang VARCHAR(10),
    is_voip BOOLEAN DEFAULT FALSE,
    debug BOOLEAN DEFAULT FALSE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Índices para performance
CREATE INDEX idx_messaging_token_user_id ON public.messaging_token(user_id);
CREATE INDEX idx_messaging_token_platform ON public.messaging_token(platform);
CREATE UNIQUE INDEX idx_messaging_token_user_device ON public.messaging_token(user_id, device);
```

**Análisis del Esquema:**

#### Campos Principales
- **id**: Primary key auto-incremental
- **token**: Token del dispositivo (FCM/APNS)
- **platform**: Plataforma del dispositivo
- **user_id**: Foreign key al usuario

#### Campos Opcionales
- **platform_version**: Versión del sistema operativo
- **device**: Modelo del dispositivo
- **lang**: Idioma preferido del usuario
- **is_voip**: Soporte para notificaciones VoIP
- **debug**: Indica si es token de desarrollo

#### Timestamps
- **created_at**: Fecha de creación
- **updated_at**: Fecha de última actualización

## Casos de Uso

### 1. Registro de Token FCM (Android)

```go
request := &tokensv1.SaveTokenRequest{
    Token:           "fcm_token_abc123...",
    Platform:        "android",
    PlatformVersion: "14",
    Device:          "Samsung Galaxy S24",
    Lang:            "es",
    IsVoip:          false,
    Debug:           false,
}

err := repo.SaveToken(ctx, userID, request)
```

### 2. Registro de Token APNS (iOS)

```go
request := &tokensv1.SaveTokenRequest{
    Token:           "apns_token_def456...",
    Platform:        "ios", 
    PlatformVersion: "17.2",
    Device:          "iPhone 15 Pro",
    Lang:            "en",
    IsVoip:          true,
    Debug:           false,
}

err := repo.SaveToken(ctx, userID, request)
```

### 3. Token de Desarrollo

```go
request := &tokensv1.SaveTokenRequest{
    Token:           "dev_token_789...",
    Platform:        "ios",
    PlatformVersion: "17.2",
    Device:          "iPhone Simulator",
    Lang:            "en",
    IsVoip:          false,
    Debug:           true, // Token de desarrollo
}

err := repo.SaveToken(ctx, userID, request)
```

## Consideraciones de Diseño

### 1. Manejo de Duplicados

**Problema Actual:**
- La implementación actual permite duplicados
- Múltiples tokens para el mismo usuario/dispositivo

**Solución Recomendada:**
```go
query := dbpq.QueryBuilder().
    Insert("public.messaging_token").
    SetMap(sq.Eq{
        // ... campos
    }).
    Suffix(`
        ON CONFLICT (user_id, device) 
        DO UPDATE SET 
            token = EXCLUDED.token,
            platform_version = EXCLUDED.platform_version,
            lang = EXCLUDED.lang,
            is_voip = EXCLUDED.is_voip,
            debug = EXCLUDED.debug,
            updated_at = NOW()
        RETURNING id
    `)
```

### 2. Limpieza de Tokens Obsoletos

```go
func (r *SQLTokensRepository) CleanupExpiredTokens(ctx context.Context, days int) error {
    query := dbpq.QueryBuilder().
        Delete("public.messaging_token").
        Where(sq.Lt{"created_at": sq.Expr("NOW() - INTERVAL ? DAY", days)})
    
    queryString, args, err := query.ToSql()
    if err != nil {
        return err
    }
    
    _, err = r.db.ExecContext(ctx, queryString, args...)
    return err
}
```

### 3. Consulta de Tokens por Usuario

```go
func (r *SQLTokensRepository) GetUserTokens(ctx context.Context, userID int) ([]*Token, error) {
    query := dbpq.QueryBuilder().
        Select("id", "token", "platform", "device", "created_at").
        From("public.messaging_token").
        Where(sq.Eq{"user_id": userID}).
        OrderBy("created_at DESC")
    
    queryString, args, err := query.ToSql()
    if err != nil {
        return nil, err
    }
    
    rows, err := r.db.QueryContext(ctx, queryString, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var tokens []*Token
    for rows.Next() {
        var token Token
        err := rows.Scan(&token.ID, &token.Token, &token.Platform, &token.Device, &token.CreatedAt)
        if err != nil {
            return nil, err
        }
        tokens = append(tokens, &token)
    }
    
    return tokens, rows.Err()
}
```

## Testing

### Unit Tests

```go
func TestSaveToken(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    repo := NewSQLTokensRepository(db)
    
    request := &tokensv1.SaveTokenRequest{
        Token:           "test_token_123",
        Platform:        "ios",
        PlatformVersion: "17.0",
        Device:          "iPhone 15",
        Lang:            "en",
        IsVoip:          true,
        Debug:           false,
    }
    
    err := repo.SaveToken(context.Background(), 123, request)
    assert.NoError(t, err)
    
    // Verificar que el token fue guardado
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM public.messaging_token WHERE user_id = $1", 123).Scan(&count)
    assert.NoError(t, err)
    assert.Equal(t, 1, count)
}

func TestSaveTokenTransactionRollback(t *testing.T) {
    // Test que verifica rollback en caso de error
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    repo := NewSQLTokensRepository(db)
    
    // Request con datos inválidos que causarán error
    request := &tokensv1.SaveTokenRequest{
        Token:    strings.Repeat("x", 1000), // Token muy largo
        Platform: "invalid_platform",
    }
    
    err := repo.SaveToken(context.Background(), 999999, request) // User ID inexistente
    assert.Error(t, err)
    
    // Verificar que no se guardó nada
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM public.messaging_token").Scan(&count)
    assert.NoError(t, err)
    assert.Equal(t, 0, count)
}
```

### Integration Tests

```go
func TestSaveTokenIntegration(t *testing.T) {
    // Test de integración completa
    db := setupRealDB(t)
    defer cleanupRealDB(t, db)
    
    repo := NewSQLTokensRepository(db)
    
    // Crear usuario de prueba
    userID := createTestUser(t, db)
    
    request := &tokensv1.SaveTokenRequest{
        Token:           generateValidFCMToken(),
        Platform:        "android",
        PlatformVersion: "14",
        Device:          "Pixel 8",
        Lang:            "es",
        IsVoip:          false,
        Debug:           false,
    }
    
    err := repo.SaveToken(context.Background(), userID, request)
    assert.NoError(t, err)
    
    // Verificar datos guardados
    var savedToken, savedPlatform string
    err = db.QueryRow(`
        SELECT token, platform 
        FROM public.messaging_token 
        WHERE user_id = $1
    `, userID).Scan(&savedToken, &savedPlatform)
    
    assert.NoError(t, err)
    assert.Equal(t, request.Token, savedToken)
    assert.Equal(t, request.Platform, savedPlatform)
}
```

## Consideraciones de Performance

### 1. Índices de Base de Datos
```sql
-- Índices recomendados
CREATE INDEX CONCURRENTLY idx_messaging_token_user_id ON public.messaging_token(user_id);
CREATE INDEX CONCURRENTLY idx_messaging_token_platform ON public.messaging_token(platform);
CREATE INDEX CONCURRENTLY idx_messaging_token_created_at ON public.messaging_token(created_at);
```

### 2. Connection Pooling
- **sql.DB**: Maneja pool de conexiones automáticamente
- **Configuración**: Ajustar `MaxOpenConns`, `MaxIdleConns`
- **Monitoring**: Monitorear uso de conexiones

### 3. Prepared Statements
- **Squirrel**: Genera prepared statements automáticamente
- **Performance**: Mejor performance para queries repetidas
- **Security**: Previene SQL injection

## Mejores Prácticas Implementadas

1. **Transaction Management**: Uso apropiado de transacciones
2. **SQL Injection Prevention**: Uso de parámetros preparados
3. **Resource Management**: Cierre apropiado de rows
4. **Error Handling**: Manejo robusto de errores
5. **Context Awareness**: Respeto a cancelación y timeouts
6. **Query Builder**: Construcción segura de queries
7. **Interface Compliance**: Implementación completa de la interfaz

## Áreas de Mejora Identificadas

1. **Duplicate Handling**: Implementar UPSERT para evitar duplicados
2. **Batch Operations**: Soporte para inserción en lote
3. **Token Validation**: Validación de formato de tokens
4. **Cleanup Operations**: Funciones para limpiar tokens obsoletos
5. **Query Methods**: Métodos adicionales para consultar tokens
6. **Metrics**: Instrumentación para observabilidad

Este archivo proporciona una implementación sólida y segura para la persistencia de tokens de dispositivos, siguiendo las mejores prácticas de Go y PostgreSQL.