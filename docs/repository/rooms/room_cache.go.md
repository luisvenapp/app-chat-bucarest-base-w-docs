# Documentación Técnica: repository/rooms/room_cache.go

## Descripción General

El archivo `room_cache.go` implementa un sistema de caché sofisticado para salas de chat y mensajes, utilizando Redis como backend de almacenamiento. Proporciona funcionalidades avanzadas como invalidación selectiva, actualización atómica de caché y gestión de locks distribuidos para garantizar consistencia en entornos concurrentes.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"
    
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/cache"
)
```

**Análisis de Importaciones:**

- **`context`**: Para manejo de contexto y cancelación
- **`encoding/json`**: Serialización de estructuras para almacenamiento en caché
- **`fmt`**: Formateo de strings para claves de caché
- **`sync`**: Primitivas de sincronización para locks locales
- **`time`**: Gestión de TTL y timestamps
- **`chatv1`**: Tipos de Protocol Buffers para salas y mensajes
- **`cache`**: Módulo del core que abstrae operaciones de Redis

## Estructuras de Datos

### Estructuras de Respuesta Cacheada

```go
type CachedRoomResponse struct {
    Data *chatv1.Room `json:"data"`
}

type CachedMessageResponse struct {
    Data *chatv1.MessageData `json:"data"`
}
```

**Análisis de las Estructuras:**

#### `CachedRoomResponse`
- **Propósito**: Wrapper para serialización JSON de salas
- **Campo Data**: Contiene la sala completa con todos sus metadatos
- **JSON Tag**: Permite serialización/deserialización consistente

#### `CachedMessageResponse`
- **Propósito**: Wrapper para serialización JSON de mensajes
- **Campo Data**: Contiene el mensaje completo con metadatos
- **Uso**: Cache de mensajes individuales para acceso rápido

### Sistema de Locks Distribuidos

```go
var (
    roomCacheLocks   = make(map[string]*sync.Mutex)
    roomCacheLocksMu sync.Mutex
)
```

**Análisis del Sistema de Locks:**

#### `roomCacheLocks map[string]*sync.Mutex`
- **Propósito**: Map de locks por roomID para evitar race conditions
- **Clave**: roomID como string
- **Valor**: Mutex específico para esa sala
- **Uso**: Sincronización de operaciones de caché por sala

#### `roomCacheLocksMu sync.Mutex`
- **Propósito**: Protege el map de locks contra acceso concurrente
- **Patrón**: Lock para proteger estructura de locks
- **Necesidad**: El map en sí no es thread-safe

### Función getRoomLock

```go
func getRoomLock(roomID string) *sync.Mutex {
    roomCacheLocksMu.Lock()
    defer roomCacheLocksMu.Unlock()
    
    lock, exists := roomCacheLocks[roomID]
    if !exists {
        lock = &sync.Mutex{}
        roomCacheLocks[roomID] = lock
    }
    return lock
}
```

**Análisis Detallado:**

#### Patrón Lazy Initialization
- **Verificación**: Comprueba si ya existe lock para la sala
- **Creación**: Crea nuevo lock si no existe
- **Almacenamiento**: Guarda el lock en el map para reutilización
- **Retorno**: Retorna lock específico para la sala

#### Thread Safety
- **Lock global**: Protege acceso al map de locks
- **Defer unlock**: Garantiza liberación del lock global
- **Atomicidad**: Operación atómica de verificación y creación

#### Memory Management
- **Crecimiento**: El map puede crecer indefinidamente
- **Mejora potencial**: Implementar limpieza de locks no utilizados

```go
// Mejora sugerida para limpieza de locks
func cleanupUnusedLocks() {
    roomCacheLocksMu.Lock()
    defer roomCacheLocksMu.Unlock()
    
    for roomID, lock := range roomCacheLocks {
        if lock.TryLock() {
            // Si podemos obtener el lock inmediatamente, no está en uso
            lock.Unlock()
            delete(roomCacheLocks, roomID)
        }
    }
}
```

## Funciones de Caché de Salas

### Función GetCachedRoom

```go
func GetCachedRoom(ctx context.Context, cacheKey string) (*chatv1.Room, bool) {
    cacheValue, err := cache.Get(ctx, cacheKey)
    if err != nil || cacheValue == "" {
        return nil, false
    }
    
    var cachedResponse CachedRoomResponse
    if err := json.Unmarshal([]byte(cacheValue), &cachedResponse); err != nil {
        return nil, false
    }
    
    return cachedResponse.Data, true
}
```

**Análisis del Proceso:**

#### 1. Recuperación del Caché
```go
cacheValue, err := cache.Get(ctx, cacheKey)
if err != nil || cacheValue == "" {
    return nil, false
}
```
- **Operación**: Consulta Redis usando la clave proporcionada
- **Error handling**: Retorna false si hay error o valor vacío
- **Context**: Respeta cancelación y timeouts del contexto

#### 2. Deserialización
```go
var cachedResponse CachedRoomResponse
if err := json.Unmarshal([]byte(cacheValue), &cachedResponse); err != nil {
    return nil, false
}
```
- **JSON parsing**: Convierte string JSON a estructura
- **Error handling**: Retorna false si la deserialización falla
- **Type safety**: Usa estructura tipada para garantizar formato

#### 3. Extracción de Datos
```go
return cachedResponse.Data, true
```
- **Unwrapping**: Extrae la sala del wrapper
- **Success indicator**: Retorna true para indicar éxito

### Función SetCachedRoom

```go
func SetCachedRoom(ctx context.Context, roomId string, cacheKey string, data *chatv1.Room) {
    cachedResponse := CachedRoomResponse{
        Data: data,
    }
    
    cacheData, err := json.Marshal(cachedResponse)
    if err == nil {
        cache.Set(ctx, cacheKey, string(cacheData), 1*time.Hour)
        setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", roomId)
        cache.SAdd(ctx, setKey, cacheKey)
    }
}
```

**Análisis del Proceso:**

#### 1. Wrapping de Datos
```go
cachedResponse := CachedRoomResponse{
    Data: data,
}
```
- **Encapsulación**: Envuelve la sala en estructura de respuesta
- **Consistencia**: Mantiene formato uniforme en caché

#### 2. Serialización
```go
cacheData, err := json.Marshal(cachedResponse)
```
- **JSON encoding**: Convierte estructura a JSON
- **Error handling**: Solo procede si la serialización es exitosa

#### 3. Almacenamiento en Caché
```go
cache.Set(ctx, cacheKey, string(cacheData), 1*time.Hour)
```
- **TTL**: 1 hora de tiempo de vida
- **Storage**: Almacena en Redis con la clave especificada
- **Context**: Respeta cancelación del contexto

#### 4. Registro en Set de Miembros
```go
setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", roomId)
cache.SAdd(ctx, setKey, cacheKey)
```
- **Set key**: Clave del set que agrupa todas las versiones cacheadas de la sala
- **Membership**: Agrega la clave de caché al set de miembros
- **Propósito**: Facilita invalidación masiva de todas las versiones de la sala

## Actualización Atómica de Caché

### Función UpdateRoomCacheWithNewMessage

```go
func UpdateRoomCacheWithNewMessage(ctx context.Context, message *chatv1.MessageData) {
    if message == nil || message.RoomId == "" {
        return
    }
    
    roomLock := getRoomLock(message.RoomId)
    roomLock.Lock()
    defer roomLock.Unlock()
    
    setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", message.RoomId)
    cacheKeys, err := cache.SMembers(ctx, setKey)
    if err != nil {
        fmt.Println("error getting cache members for update", err)
        return
    }
    
    for _, key := range cacheKeys {
        cachedRoom, exists := GetCachedRoom(ctx, key)
        if exists {
            cachedRoom.LastMessage = message
            cachedRoom.LastMessageAt = message.CreatedAt
            // SetCachedRoom will re-add the key to the set, which is fine.
            SetCachedRoom(ctx, message.RoomId, key, cachedRoom)
        }
    }
}
```

**Análisis Detallado:**

#### 1. Validación de Entrada
```go
if message == nil || message.RoomId == "" {
    return
}
```
- **Null safety**: Verifica que el mensaje y roomID sean válidos
- **Early return**: Evita procesamiento innecesario

#### 2. Adquisición de Lock
```go
roomLock := getRoomLock(message.RoomId)
roomLock.Lock()
defer roomLock.Unlock()
```
- **Lock específico**: Obtiene lock específico para la sala
- **Atomicidad**: Garantiza que la actualización sea atómica
- **Defer unlock**: Asegura liberación del lock

#### 3. Obtención de Claves de Caché
```go
setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", message.RoomId)
cacheKeys, err := cache.SMembers(ctx, setKey)
```
- **Set key**: Construye clave del set de miembros
- **SMembers**: Obtiene todas las claves de caché para la sala
- **Error handling**: Maneja errores de Redis

#### 4. Actualización de Todas las Versiones
```go
for _, key := range cacheKeys {
    cachedRoom, exists := GetCachedRoom(ctx, key)
    if exists {
        cachedRoom.LastMessage = message
        cachedRoom.LastMessageAt = message.CreatedAt
        SetCachedRoom(ctx, message.RoomId, key, cachedRoom)
    }
}
```

**Proceso de Actualización:**
- **Iteración**: Procesa cada clave de caché encontrada
- **Recuperación**: Obtiene la sala cacheada
- **Modificación**: Actualiza último mensaje y timestamp
- **Persistencia**: Guarda la versión actualizada

**Ventajas del Approach:**
- **Consistencia**: Todas las versiones cacheadas se actualizan
- **Atomicidad**: Operación protegida por lock
- **Eficiencia**: Solo actualiza versiones que existen en caché

## Funciones de Caché de Mensajes

### Función GetCachedMessageSimple

```go
func GetCachedMessageSimple(ctx context.Context, cacheKey string) (*chatv1.MessageData, bool) {
    cacheValue, err := cache.Get(ctx, cacheKey)
    if err != nil || cacheValue == "" {
        return nil, false
    }
    
    var cachedResponse CachedMessageResponse
    if err := json.Unmarshal([]byte(cacheValue), &cachedResponse); err != nil {
        return nil, false
    }
    
    return cachedResponse.Data, true
}
```

**Análisis:**
- **Patrón similar**: Sigue el mismo patrón que `GetCachedRoom`
- **Simplicidad**: Versión simplificada para mensajes individuales
- **Uso**: Cache de mensajes frecuentemente accedidos

### Función SetCachedMessageSimple

```go
func SetCachedMessageSimple(ctx context.Context, cacheKey string, data *chatv1.MessageData) {
    cachedResponse := CachedMessageResponse{
        Data: data,
    }
    
    cacheData, err := json.Marshal(cachedResponse)
    if err == nil {
        cache.Set(ctx, cacheKey, string(cacheData), 1*time.Hour)
    }
}
```

**Diferencias con SetCachedRoom:**
- **Sin set membership**: No mantiene sets de miembros para mensajes
- **Más simple**: Solo almacena el mensaje individual
- **Mismo TTL**: 1 hora de tiempo de vida

## Funciones de Invalidación

### Función DeleteRoomCacheByRoomID

```go
func DeleteRoomCacheByRoomID(ctx context.Context, roomId string) {
    setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", roomId)
    keys, err := cache.SMembers(ctx, setKey)
    if err != nil {
        fmt.Println("error getting cache members", err)
        return
    }
    
    if len(keys) > 0 {
        err := cache.Del(ctx, keys...)
        if err != nil {
            fmt.Println("error deleting cache", err)
        }
    }
    
    err = cache.Del(ctx, setKey)
    if err != nil {
        fmt.Println("error deleting cache set", err)
    }
}
```

**Análisis del Proceso:**

#### 1. Obtención de Claves
```go
setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", roomId)
keys, err := cache.SMembers(ctx, setKey)
```
- **Set key**: Construye clave del set de miembros
- **Recuperación**: Obtiene todas las claves asociadas a la sala

#### 2. Eliminación en Lote
```go
if len(keys) > 0 {
    err := cache.Del(ctx, keys...)
    if err != nil {
        fmt.Println("error deleting cache", err)
    }
}
```
- **Batch delete**: Elimina múltiples claves en una operación
- **Eficiencia**: Reduce round-trips a Redis
- **Error handling**: Registra errores pero continúa

#### 3. Limpieza del Set
```go
err = cache.Del(ctx, setKey)
if err != nil {
    fmt.Println("error deleting cache set", err)
}
```
- **Set cleanup**: Elimina el set de miembros
- **Completitud**: Asegura limpieza completa

### Función DeleteCache

```go
func DeleteCache(ctx context.Context, key string) {
    err := cache.Del(ctx, key)
    if err != nil {
        fmt.Println("error deleting cache", err)
    }
}
```

**Análisis:**
- **Simplicidad**: Función utilitaria para eliminación simple
- **Error handling**: Registra errores pero no los propaga
- **Uso**: Eliminación de claves individuales

## Patrones de Diseño Implementados

### 1. Cache-Aside Pattern
```go
// Lectura
cachedRoom, exists := GetCachedRoom(ctx, cacheKey)
if !exists {
    room, err := repository.GetRoom(ctx, roomID)
    if err == nil {
        SetCachedRoom(ctx, roomID, cacheKey, room)
    }
    return room, err
}
return cachedRoom, nil
```

### 2. Write-Through Pattern (Implícito)
```go
// Actualización
room, err := repository.UpdateRoom(ctx, roomID, updates)
if err == nil {
    // Invalidar caché para forzar reload
    DeleteRoomCacheByRoomID(ctx, roomID)
}
```

### 3. Observer Pattern (Para Actualizaciones)
```go
// Cuando llega un nuevo mensaje
func onNewMessage(message *chatv1.MessageData) {
    // Actualizar todas las versiones cacheadas de la sala
    UpdateRoomCacheWithNewMessage(ctx, message)
}
```

## Consideraciones de Performance

### 1. TTL Optimization
```go
// TTL diferenciado por tipo de dato
const (
    RoomCacheTTL    = 1 * time.Hour
    MessageCacheTTL = 30 * time.Minute
    UserCacheTTL    = 2 * time.Hour
)
```

### 2. Batch Operations
```go
// Operaciones en lote para mejor performance
func SetMultipleCachedRooms(ctx context.Context, rooms map[string]*chatv1.Room) {
    pipeline := cache.Pipeline()
    for cacheKey, room := range rooms {
        // Agregar operaciones al pipeline
    }
    pipeline.Exec(ctx)
}
```

### 3. Compression
```go
// Compresión para datos grandes
func SetCachedRoomCompressed(ctx context.Context, cacheKey string, data *chatv1.Room) {
    jsonData, _ := json.Marshal(CachedRoomResponse{Data: data})
    compressed := compress(jsonData)
    cache.Set(ctx, cacheKey, compressed, 1*time.Hour)
}
```

## Testing

### Unit Tests

```go
func TestGetCachedRoom(t *testing.T) {
    // Mock cache
    mockCache := &MockCache{}
    
    room := &chatv1.Room{
        Id:   "room_123",
        Name: "Test Room",
    }
    
    cachedResponse := CachedRoomResponse{Data: room}
    jsonData, _ := json.Marshal(cachedResponse)
    
    mockCache.On("Get", mock.Anything, "test_key").Return(string(jsonData), nil)
    
    result, exists := GetCachedRoom(context.Background(), "test_key")
    
    assert.True(t, exists)
    assert.Equal(t, room.Id, result.Id)
    assert.Equal(t, room.Name, result.Name)
}

func TestUpdateRoomCacheWithNewMessage(t *testing.T) {
    // Test de actualización atómica
    message := &chatv1.MessageData{
        Id:        "msg_123",
        RoomId:    "room_456",
        Content:   "Test message",
        CreatedAt: "2024-01-01T00:00:00Z",
    }
    
    // Setup mock cache con sala existente
    // ...
    
    UpdateRoomCacheWithNewMessage(context.Background(), message)
    
    // Verificar que la sala cacheada fue actualizada
    // ...
}
```

## Mejores Prácticas Implementadas

1. **Atomic Updates**: Uso de locks para actualizaciones atómicas
2. **Batch Operations**: Eliminación en lote para eficiencia
3. **Error Handling**: Manejo robusto de errores de Redis
4. **TTL Management**: Tiempo de vida apropiado para diferentes tipos de datos
5. **Memory Management**: Estructuras optimizadas para serialización
6. **Consistency**: Invalidación coordinada de múltiples versiones
7. **Context Awareness**: Respeto a cancelación y timeouts

Este archivo implementa un sistema de caché sofisticado que mejora significativamente el rendimiento del sistema de chat, proporcionando acceso rápido a datos frecuentemente consultados mientras mantiene la consistencia en entornos concurrentes.