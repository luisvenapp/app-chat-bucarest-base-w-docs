# Documentación del Repositorio de Rooms

## Descripción General

Este directorio contiene la implementación del patrón Repository para la gestión de salas (rooms) en el sistema de chat. Proporciona una abstracción sobre las operaciones de persistencia de datos relacionadas con salas, mensajes y participantes, soportando múltiples backends de base de datos.

## Arquitectura del Repositorio

### Patrón Repository

El patrón Repository proporciona una interfaz uniforme para acceder a datos, independientemente del mecanismo de almacenamiento subyacente. Esto permite:

- **Abstracción de datos**: Separación entre lógica de negocio y persistencia
- **Testabilidad**: Fácil creación de mocks para testing
- **Flexibilidad**: Cambio de backend sin afectar la lógica de negocio
- **Mantenibilidad**: Código más limpio y organizado

### Estructura de Archivos

```
repository/rooms/
├── README.md                    # Este archivo
├── interfaces.go                # Definición de interfaces
├── room_postgres_impl.go        # Implementación para PostgreSQL
├── room_scylladb_impl.go       # Implementación para ScyllaDB
├── cache.go                     # Sistema de caché Redis
└── types.go                     # Tipos de datos compartidos
```

## Interfaces Principales

### RoomsRepository

Interfaz principal que define todas las operaciones disponibles para la gestión de salas:

```go
type RoomsRepository interface {
    // Gestión de Salas
    CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error)
    GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error)
    GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error)
    UpdateRoom(ctx context.Context, userId int, roomId string, room *chatv1.UpdateRoomRequest) error
    DeleteRoom(ctx context.Context, userId int, roomId string, partner *int) error
    
    // Gestión de Participantes
    GetRoomParticipants(ctx context.Context, pagination *chatv1.GetRoomParticipantsRequest) ([]*chatv1.RoomParticipant, *chatv1.PaginationMeta, error)
    AddParticipantToRoom(ctx context.Context, userId int, roomId string, participants []int) ([]User, error)
    LeaveRoom(ctx context.Context, userId int, roomId string, participants []int32, leaveAll bool) ([]User, error)
    UpdateParticipantRoom(ctx context.Context, userId int, req *chatv1.UpdateParticipantRoomRequest) error
    
    // Gestión de Mensajes
    SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error)
    GetMessage(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error)
    GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error)
    UpdateMessage(ctx context.Context, userId int, messageId string, content string) error
    DeleteMessage(ctx context.Context, userId int, messageIds []string) error
    
    // Funcionalidades Avanzadas
    PinRoom(ctx context.Context, userId int, roomId string, pin bool) error
    MuteRoom(ctx context.Context, userId int, roomId string, mute bool) error
    BlockUser(ctx context.Context, userId int, roomId string, block bool, partner *int) error
    ReactToMessage(ctx context.Context, userId int, messageId string, reaction string) error
    MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error)
}
```

### UserFetcher

Interfaz para obtener información de usuarios desde servicios externos:

```go
type UserFetcher interface {
    GetUserByID(ctx context.Context, id int) (*User, error)
    GetUsersByID(ctx context.Context, ids []int) ([]User, error)
    GetAllUserIDs(ctx context.Context) ([]int, error)
}
```

## Implementaciones

### PostgreSQL Implementation (room_postgres_impl.go)

**Características:**
- Base de datos relacional tradicional
- ACID compliance completo
- Consultas SQL complejas con JOINs
- Transacciones robustas
- Ideal para consistencia estricta

**Casos de uso recomendados:**
- Aplicaciones que requieren consistencia estricta
- Consultas complejas con múltiples relaciones
- Transacciones críticas
- Entornos con volúmenes moderados

**Ventajas:**
- Consistencia ACID
- Consultas SQL familiares
- Herramientas maduras
- Backup y recovery robustos

**Desventajas:**
- Escalabilidad horizontal limitada
- Performance en escrituras intensivas
- Complejidad en sharding

### ScyllaDB Implementation (room_scylladb_impl.go)

**Características:**
- Base de datos NoSQL distribuida
- Alto rendimiento y baja latencia
- Escalabilidad horizontal
- Modelo de datos desnormalizado
- Eventual consistency

**Casos de uso recomendados:**
- Aplicaciones de alta escala
- Requisitos de baja latencia
- Cargas de trabajo intensivas en escritura
- Sistemas distribuidos globalmente

**Ventajas:**
- Escalabilidad masiva
- Baja latencia
- Alta disponibilidad
- Performance predecible

**Desventajas:**
- Eventual consistency
- Modelo de datos más complejo
- Menos herramientas maduras
- Curva de aprendizaje

## Comparación de Implementaciones

| Aspecto | PostgreSQL | ScyllaDB |
|---------|------------|----------|
| **Consistencia** | ACID fuerte | Eventual |
| **Escalabilidad** | Vertical | Horizontal |
| **Latencia** | Moderada | Muy baja |
| **Throughput** | Moderado | Muy alto |
| **Complejidad** | Baja | Media-Alta |
| **Consultas** | SQL complejo | CQL simple |
| **Transacciones** | Completas | Limitadas |
| **Sharding** | Manual | Automático |

## Sistema de Caché

### Estrategia de Caché

El sistema implementa un caché Redis para optimizar el rendimiento:

```go
// Patrones de clave de caché
func GetCachedRoom(ctx context.Context, cacheKey string) (*chatv1.Room, bool)
func SetCachedRoom(ctx context.Context, roomId, cacheKey string, room *chatv1.Room)
func DeleteRoomCacheByRoomID(ctx context.Context, roomId string)
```

**Tipos de caché:**
- **Room Cache**: Datos completos de salas
- **Message Cache**: Mensajes individuales
- **Participant Cache**: Listas de participantes

**Estrategias de invalidación:**
- **Write-through**: Actualización inmediata
- **TTL**: Expiración automática (1 hora)
- **Manual**: Invalidación explícita en cambios

## Tipos de Datos

### User

```go
type User struct {
    ID        int     `json:"id"`
    Name      string  `json:"name"`
    Phone     string  `json:"phone"`
    Email     *string `json:"email,omitempty"`
    Avatar    *string `json:"avatar,omitempty"`
    CreatedAt *string `json:"created_at,omitempty"`
    Dni       *string `json:"dni,omitempty"`
}
```

### Estructuras de Protocol Buffers

El repositorio utiliza extensivamente las estructuras generadas desde Protocol Buffers:

- `chatv1.Room`: Información completa de salas
- `chatv1.MessageData`: Datos de mensajes
- `chatv1.RoomParticipant`: Información de participantes
- `chatv1.PaginationMeta`: Metadatos de paginación

## Patrones de Uso

### Inicialización

```go
// PostgreSQL
postgresRepo := NewSQLRoomRepository(db)

// ScyllaDB
scyllaRepo := NewScyllaRoomRepository(session, userFetcher)
```

### Gestión de Salas

```go
// Crear sala
room, err := repo.CreateRoom(ctx, userId, &chatv1.CreateRoomRequest{
    Type: "group",
    Name: proto.String("Mi Grupo"),
    Participants: []int32{123, 456},
})

// Obtener sala con caché
room, err := repo.GetRoom(ctx, userId, roomId, true, true)

// Listar salas con paginación
rooms, meta, err := repo.GetRoomList(ctx, userId, &chatv1.GetRoomsRequest{
    Page:   1,
    Limit:  20,
    Search: "trabajo",
})
```

### Gestión de Mensajes

```go
// Enviar mensaje
message, err := repo.SaveMessage(ctx, userId, &chatv1.SendMessageRequest{
    RoomId:  roomId,
    Content: "Hola mundo!",
    Type:    "text",
}, room, &decryptedContent)

// Obtener historial
messages, meta, err := repo.GetMessagesFromRoom(ctx, userId, &chatv1.GetMessageHistoryRequest{
    Id:    roomId,
    Page:  1,
    Limit: 50,
})

// Marcar como leído
count, err := repo.MarkMessagesAsRead(ctx, userId, roomId, messageIds, "")
```

### Gestión de Participantes

```go
// Agregar participantes
users, err := repo.AddParticipantToRoom(ctx, userId, roomId, []int{789, 101})

// Obtener participantes
participants, meta, err := repo.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{
    Id:    roomId,
    Page:  1,
    Limit: 100,
})

// Salir de sala
users, err := repo.LeaveRoom(ctx, userId, roomId, []int32{userId}, false)
```

## Consideraciones de Performance

### PostgreSQL

**Optimizaciones implementadas:**
- Índices en columnas frecuentemente consultadas
- LATERAL JOINs para consultas eficientes
- Paginación con OFFSET/LIMIT
- Subconsultas optimizadas para conteos
- Transacciones mínimas

**Consultas críticas:**
```sql
-- Último mensaje con LATERAL JOIN
LEFT JOIN LATERAL (
    SELECT msg.id, msg.content, msg.type, msg.created_at 
    FROM room_message AS msg 
    WHERE msg.room_id = room.id 
    ORDER BY msg.created_at DESC 
    LIMIT 1
) AS last_msg ON true
```

### ScyllaDB

**Optimizaciones implementadas:**
- Modelado de datos por patrones de consulta
- Desnormalización estratégica
- Batches para operaciones atómicas
- Fan-out para actualizaciones
- Particionado eficiente

**Patrones de tabla:**
```cql
-- Mensajes por sala (particionado por room_id)
CREATE TABLE messages_by_room (
    room_id UUID,
    message_id TIMEUUID,
    sender_id INT,
    content TEXT,
    PRIMARY KEY (room_id, message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);

-- Salas por usuario (particionado por user_id)
CREATE TABLE rooms_by_user (
    user_id INT,
    is_pinned BOOLEAN,
    last_message_at TIMESTAMP,
    room_id UUID,
    PRIMARY KEY (user_id, is_pinned, last_message_at, room_id)
) WITH CLUSTERING ORDER BY (is_pinned DESC, last_message_at DESC);
```

## Testing

### Mocks

```go
// Mock para testing
type MockRoomsRepository struct {
    mock.Mock
}

func (m *MockRoomsRepository) CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error) {
    args := m.Called(ctx, userId, room)
    return args.Get(0).(*chatv1.Room), args.Error(1)
}
```

### Tests de Integración

```go
func TestRoomRepository_Integration(t *testing.T) {
    // Setup
    repo := setupTestRepository(t)
    
    // Test crear sala
    room, err := repo.CreateRoom(ctx, 123, &chatv1.CreateRoomRequest{
        Type: "group",
        Name: proto.String("Test Room"),
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, room.Id)
    
    // Test obtener sala
    retrieved, err := repo.GetRoom(ctx, 123, room.Id, true, false)
    assert.NoError(t, err)
    assert.Equal(t, room.Id, retrieved.Id)
}
```

## Mejores Prácticas

### Manejo de Errores

```go
// Siempre propagar contexto y errores
func (r *SQLRoomRepository) GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error) {
    if roomId == "" {
        return nil, fmt.Errorf("room ID cannot be empty")
    }
    
    // Usar contexto en todas las operaciones
    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query room: %w", err)
    }
    
    return room, nil
}
```

### Transacciones

```go
// PostgreSQL - Transacciones explícitas
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return nil, err
}
defer tx.Rollback()

// Operaciones...

if err = tx.Commit(); err != nil {
    return nil, err
}
```

```go
// ScyllaDB - Batches para atomicidad
batch := r.session.Batch(gocql.LoggedBatch)
batch.Query(`INSERT INTO table1 ...`, args1...)
batch.Query(`INSERT INTO table2 ...`, args2...)

if err := r.session.ExecuteBatch(batch); err != nil {
    return fmt.Errorf("batch failed: %w", err)
}
```

### Caché

```go
// Siempre verificar caché primero
if useCache {
    if cached, exists := GetCachedRoom(ctx, cacheKey); exists {
        return cached, nil
    }
}

// Después de obtener datos, cachear
if useCache {
    SetCachedRoom(ctx, roomId, cacheKey, room)
}

// Invalidar caché en modificaciones
DeleteRoomCacheByRoomID(ctx, roomId)
```

## Configuración

### Variables de Entorno

```bash
# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=chat_db
POSTGRES_USER=chat_user
POSTGRES_PASSWORD=secret

# ScyllaDB
SCYLLA_HOSTS=localhost:9042
SCYLLA_KEYSPACE=chat_keyspace
SCYLLA_CONSISTENCY=QUORUM

# Redis Cache
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
CACHE_TTL=3600
```

### Inicialización

```go
// Configuración de repositorio
func NewRoomRepository(config *Config) RoomsRepository {
    switch config.DatabaseType {
    case "postgres":
        db := setupPostgreSQL(config)
        return NewSQLRoomRepository(db)
    case "scylla":
        session := setupScyllaDB(config)
        userFetcher := NewUserFetcher(config)
        return NewScyllaRoomRepository(session, userFetcher)
    default:
        panic("unsupported database type")
    }
}
```

## Migración entre Implementaciones

### Estrategia de Migración

1. **Dual Write**: Escribir en ambos sistemas
2. **Data Migration**: Migrar datos históricos
3. **Read Switch**: Cambiar lecturas gradualmente
4. **Cleanup**: Remover sistema antiguo

```go
// Ejemplo de dual write
type DualWriteRepository struct {
    primary   RoomsRepository
    secondary RoomsRepository
}

func (d *DualWriteRepository) CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error) {
    // Escribir en primario
    result, err := d.primary.CreateRoom(ctx, userId, room)
    if err != nil {
        return nil, err
    }
    
    // Escribir en secundario (async)
    go func() {
        d.secondary.CreateRoom(context.Background(), userId, room)
    }()
    
    return result, nil
}
```

## Monitoreo y Métricas

### Métricas Recomendadas

- **Latencia**: P50, P95, P99 por operación
- **Throughput**: Operaciones por segundo
- **Error Rate**: Porcentaje de errores
- **Cache Hit Rate**: Efectividad del caché
- **Database Connections**: Pool de conexiones

### Logging

```go
// Logging estructurado
log.WithFields(log.Fields{
    "operation": "CreateRoom",
    "user_id":   userId,
    "room_type": room.Type,
    "duration":  time.Since(start),
}).Info("Room created successfully")
```

## Conclusión

Este repositorio proporciona una abstracción robusta y flexible para la gestión de datos de chat, soportando múltiples backends con diferentes características de rendimiento y consistencia. La elección entre PostgreSQL y ScyllaDB debe basarse en los requisitos específicos de la aplicación en términos de escala, consistencia y latencia.

La implementación del patrón Repository facilita el testing, mantenimiento y evolución del sistema, mientras que el sistema de caché optimiza el rendimiento para casos de uso comunes.