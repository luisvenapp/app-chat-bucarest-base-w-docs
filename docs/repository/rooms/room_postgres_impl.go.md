# Documentación Técnica: repository/rooms/room_postgres_impl.go

## Descripción General

Este archivo implementa la interfaz `RoomsRepository` utilizando **PostgreSQL** como backend de base de datos. Proporciona una implementación completa y robusta para la gestión de salas, mensajes y participantes en el sistema de chat, aprovechando las características ACID de PostgreSQL para garantizar consistencia de datos.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    "database/sql"
    "fmt"
    "math"
    "slices"
    "time"

    sq "github.com/Masterminds/squirrel"
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
    dbpq "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/postgres"
)
```

**Análisis de Dependencias:**
- **`database/sql`**: Driver estándar de Go para bases de datos SQL
- **`github.com/Masterminds/squirrel`**: Query builder para construir consultas SQL de forma programática
- **`chatv1`**: Tipos generados desde Protocol Buffers
- **`utils`**: Utilidades del proyecto (encriptación, formateo)
- **`dbpq`**: Wrapper personalizado para PostgreSQL

## Estructura Principal

### SQLRoomRepository

```go
type SQLRoomRepository struct {
    db *sql.DB
}

func NewSQLRoomRepository(db *sql.DB) RoomsRepository {
    return &SQLRoomRepository{
        db: db,
    }
}
```

**Características:**
- **Simplicidad**: Solo requiere una conexión a la base de datos
- **Thread-safe**: `sql.DB` es thread-safe por diseño
- **Pool de conexiones**: Maneja automáticamente el pool de conexiones

## Métodos Principales

### CreateRoom

```go
func (r *SQLRoomRepository) CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error)
```

**Funcionalidad:**
1. **Verificación de duplicados P2P**: Para salas persona a persona, verifica si ya existe una sala entre los usuarios
2. **Generación de encriptación**: Crea claves de encriptación únicas para cada sala
3. **Transacción atómica**: Usa transacciones para garantizar consistencia
4. **Configuración por defecto**: Establece permisos según el tipo de sala

**Proceso detallado:**

#### 1. Verificación de Salas P2P Existentes
```go
if room.Type == "p2p" {
    query := dbpq.QueryBuilder().
        Select("room.id", "room.created_at", /* ... */).
        From("room").
        InnerJoin("room_member AS pm ON room.id = pm.room_id AND pm.user_id = ? AND pm.removed_at IS NULL", room.Participants[0]).
        InnerJoin("room_member AS mm ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL", userId).
        Where(sq.Eq{"room.type": "p2p"}).
        Where(sq.Eq{"room.deleted_at": nil})
}
```

**Propósito**: Evitar duplicación de salas P2P entre los mismos usuarios

#### 2. Generación de Datos de Encriptación
```go
encryptionData, err := utils.GenerateKeyEncript()
if err != nil {
    return nil, err
}
```

**Seguridad**: Cada sala tiene sus propias claves de encriptación

#### 3. Transacción para Creación
```go
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return nil, err
}
defer tx.Rollback()

// Insertar sala
query := dbpq.QueryBuilder().
    Insert("public.room").
    SetMap(sq.Eq{
        "name":            room.Name,
        "image":           room.PhotoUrl,
        "description":     room.Description,
        "encription_data": encryptionData,
        "created_at":      sq.Expr("NOW()"),
        "type":            room.Type,
    }).
    Suffix("RETURNING id")

// Insertar creador como OWNER
query = dbpq.QueryBuilder().
    Insert("public.room_member").
    SetMap(sq.Eq{
        "room_id": newRoom.Id,
        "user_id": userId,
        "role":    "OWNER",
    })

// Insertar participantes como MEMBER
for _, participant := range room.Participants {
    if participant != int32(userId) {
        queryParticipants = queryParticipants.Values(newRoom.Id, participant, "MEMBER")
    }
}

err = tx.Commit()
```

### GetRoom

```go
func (r *SQLRoomRepository) GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error)
```

**Funcionalidad:**
- **Caché inteligente**: Verifica caché antes de consultar la base de datos
- **Consulta optimizada**: Usa LATERAL JOIN para obtener el último mensaje eficientemente
- **Conteo de no leídos**: Calcula mensajes no leídos en tiempo real
- **Datos completos**: Opcionalmente incluye participantes

**Consulta principal:**
```sql
SELECT 
    room.id, room.created_at, room.updated_at, room.image, room.name, room.description, 
    room.type, room.encription_data, room.join_all_user, room."lastMessageAt", 
    room.send_message, room.add_member, room.edit_group,
    partner.id, partner.name, partner.phone, partner.avatar,
    -- Último mensaje usando LATERAL JOIN
    last_msg.id AS last_message_id,
    last_msg.content AS last_message_content,
    last_msg.type AS last_message_type,
    last_msg.created_at AS last_message_created_at,
    last_sender.name AS last_message_sender_name,
    -- Conteo de mensajes no leídos
    (SELECT COUNT(*) FROM room_message AS unread_msg 
     LEFT JOIN room_message_meta AS unread_meta ON unread_msg.id = unread_meta.message_id 
     WHERE unread_msg.room_id = room.id AND unread_meta.read_at IS NULL) AS unread_count
FROM room_member AS mm
INNER JOIN room ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL
LEFT JOIN LATERAL (
    SELECT msg.id, msg.content, msg.type, msg.created_at, msg.sender_id, msg.status, msg.updated_at 
    FROM room_message AS msg 
    LEFT JOIN room_message_meta AS meta ON msg.id = meta.message_id AND meta.user_id = me.id
    WHERE msg.room_id = room.id AND msg.deleted_at IS NULL AND meta."isDeleted" = false 
    ORDER BY msg.created_at DESC LIMIT 1
) AS last_msg ON true
```

**Optimizaciones:**
- **LATERAL JOIN**: Obtiene el último mensaje de forma eficiente
- **Subconsulta para conteo**: Calcula mensajes no leídos sin afectar performance
- **Índices implícitos**: Aprovecha índices en foreign keys y timestamps

### GetRoomList

```go
func (r *SQLRoomRepository) GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error)
```

**Funcionalidades:**
- **Paginación completa**: Soporte para page/limit con metadatos
- **Búsqueda**: Filtrado por nombre con soporte para acentos
- **Filtros**: Por tipo de sala y sincronización incremental
- **Ordenamiento**: Salas fijadas primero, luego por actividad reciente

**Características avanzadas:**

#### 1. Búsqueda sin Acentos
```go
if pagination.Search != "" {
    query = query.Where("unaccent(room.name) ILIKE unaccent(?) OR unaccent(partner.name) ILIKE unaccent(?)", 
        "%"+pagination.Search+"%", "%"+pagination.Search+"%")
}
```

#### 2. Sincronización Incremental
```go
if pagination.Since != "" {
    query = query.Where("room.updated_at > ? OR mm.updated_at > ?", pagination.Since, pagination.Since)
}
```

#### 3. Ordenamiento Inteligente
```go
query = query.OrderBy("mm.\"is_pinned\" DESC, room.\"lastMessageAt\" DESC, room.created_at DESC")
```

#### 4. Optimización de Participantes
```go
// Obtener participantes para salas de grupo en una sola consulta
queryParticipants := dbpq.QueryBuilder().
    Select("room_member.user_id", "room_member.role", "uu.name", "uu.phone", "uu.avatar", "room_member.room_id", 
           "ROW_NUMBER() OVER (PARTITION BY room_member.room_id ORDER BY room_member.created_at DESC) as rn").
    From("room_member").
    InnerJoin("public.\"user\" AS uu ON room_member.user_id = uu.id").
    Where(sq.Eq{"room_member.room_id": allRoomIds}).
    OrderBy("room_member.created_at DESC")

// Limitar a 5 participantes por sala
queryParticipantsPerRoom := dbpq.QueryBuilder().
    Select("*").
    From("(" + queryString + ") ranked_participants").
    Where("rn <= 5")
```

### SaveMessage

```go
func (r *SQLRoomRepository) SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error)
```

**Proceso completo:**

#### 1. Resolución de Referencias
```go
// Buscar contacto si es un mensaje de contacto
var contactId sql.NullString
if req.Type == "contact" && req.ContactPhone != nil {
    err := tx.QueryRowContext(ctx, "SELECT id FROM public.\"user\" WHERE phone = $1", req.ContactPhone).Scan(&contactId)
}

// Buscar usuario original si es un mensaje reenviado
var forwardUserId sql.NullString
if req.ForwardId != nil {
    err := tx.QueryRowContext(ctx, "SELECT sender_id FROM room_message WHERE id = $1", req.ForwardId).Scan(&forwardUserId)
}
```

#### 2. Inserción del Mensaje Principal
```go
insertMessageQuery := dbpq.QueryBuilder().
    Insert("public.room_message").
    SetMap(sq.Eq{
        "room_id":                           req.RoomId,
        "sender_id":                         userId,
        "content":                           req.Content,
        "content_decrypted":                 contentDecrypted,
        "status":                            int(chatv1.MessageStatus_MESSAGE_STATUS_SENT),
        "created_at":                        sq.Expr("NOW()"),
        "type":                              req.Type,
        "replied_message_id":                req.ReplyId,
        "forwarded_message_id":              req.ForwardId,
        "forwarded_message_original_sender": forwardUserId,
        "sender_message_id":                 req.SenderMessageId,
    }).
    Suffix("RETURNING id")
```

#### 3. Inserción de Menciones
```go
if len(req.Mentions) > 0 {
    mentionQuery := dbpq.QueryBuilder().
        Insert("public.room_message_tag").
        Columns("message_id", "user_id", "tag")

    for _, mention := range req.Mentions {
        mentionQuery = mentionQuery.Values(messageId, mention.User, mention.Tag)
    }
}
```

#### 4. Metadatos del Remitente
```go
// Solo crear metadatos para el remitente inicialmente
_, err = dbpq.QueryBuilder().
    Insert("public.room_message_meta").
    Columns("message_id", "user_id", "read_at", "\"isDeleted\"", "\"isSenderBlocked\"").
    Values(messageId, userId, sq.Expr("NOW()"), false, false).
    RunWith(tx).
    ExecContext(ctx)
```

### GetMessagesFromRoom

```go
func (r *SQLRoomRepository) GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error)
```

**Características avanzadas:**

#### 1. Paginación por Cursor
```go
// Obtener timestamp del mensaje de referencia
if req.BeforeMessageId != nil {
    query := dbpq.QueryBuilder().Select("created_at").From("room_message").Where(sq.Eq{"id": *req.BeforeMessageId})
    rows.Scan(&beforeCreatedAt)
}

// Usar timestamp para paginación
if beforeCreatedAt != "" {
    query = query.Where(sq.Lt{"msg.created_at": beforeCreatedAt})
}
```

#### 2. Filtrado por Mensajes por Sala
```go
rowNumber := "1"
if req.MessagesPerRoom > 0 {
    rowNumber = "ROW_NUMBER() OVER (PARTITION BY msg.room_id ORDER BY msg.created_at DESC) as rn"
}

// Aplicar límite por sala
if req.MessagesPerRoom > 0 {
    queryMessagesPerRoom := dbpq.QueryBuilder().
        Select("*").
        From("(" + queryString + ") ranked_messages").
        Where("rn <= " + fmt.Sprintf("%d", req.MessagesPerRoom))
}
```

#### 3. Optimización de Menciones y Reacciones
```go
// Obtener todas las menciones en una consulta
queryTags := dbpq.QueryBuilder().
    Select("room_message_tag.user_id", "public.\"user\".name", "public.\"user\".phone", "room_message_tag.tag", "room_message_tag.message_id").
    From("room_message_tag").
    InnerJoin("public.\"user\" ON room_message_tag.user_id = public.\"user\".id").
    Where(sq.Eq{"room_message_tag.message_id": allMessageIds})

// Obtener todas las reacciones en una consulta
queryReactions := dbpq.QueryBuilder().
    Select("room_message_reaction.\"reactedById\"", "room_message_reaction.reaction", "room_message_reaction.\"messageId\"").
    From("room_message_reaction").
    Where(sq.Eq{"room_message_reaction.\"messageId\"": allMessageIds})

// Asociar menciones y reacciones a mensajes
for i, message := range data {
    for _, tag := range allTags {
        if tag.MessageId == message.Id {
            data[i].Mentions = append(data[i].Mentions, tag)
        }
    }
    for _, reaction := range allReactions {
        if reaction.MessageId == message.Id {
            data[i].Reactions = append(data[i].Reactions, reaction)
        }
    }
}
```

### MarkMessagesAsRead

```go
func (r *SQLRoomRepository) MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error)
```

**Proceso optimizado:**

#### 1. Expansión por Fecha
```go
if since != "" {
    query := dbpq.QueryBuilder().
        Select("room_message.id").
        From("room_message").
        LeftJoin("room_message_meta ON room_message.id = room_message_meta.message_id AND room_message_meta.user_id = ?").
        Where(sq.Lt{"room_message.created_at": since}).
        Where(sq.Eq{"room_message.room_id": roomId}).
        Where(sq.Expr("room_message_meta.read_at IS NULL"))
    
    // Agregar IDs encontrados a la lista
    for rows.Next() {
        messageIds = append(messageIds, messageId)
    }
}
```

#### 2. Verificación de Metadatos Existentes
```go
query := dbpq.QueryBuilder().
    Select("message_id", "read_at").
    From("public.room_message_meta").
    Where(sq.Eq{"user_id": userId}).
    Where(sq.Eq{"message_id": messageIds})

// Clasificar mensajes
existingMessages := make(map[string]bool)
messagesToUpdate := make([]string, 0)
messagesToCreate := make([]string, 0)

for rows.Next() {
    var messageId string
    var readAt sql.NullString
    
    existingMessages[messageId] = true
    
    if !readAt.Valid {
        messagesToUpdate = append(messagesToUpdate, messageId)
    }
}

// Identificar mensajes que necesitan nuevos registros
for _, messageId := range messageIds {
    if !existingMessages[messageId] {
        messagesToCreate = append(messagesToCreate, messageId)
    }
}
```

#### 3. Operaciones Batch
```go
// Actualizar registros existentes
if len(messagesToUpdate) > 0 {
    updateQuery := dbpq.QueryBuilder().
        Update("public.room_message_meta").
        Set("read_at", sq.Expr("NOW()")).
        Where(sq.Eq{"user_id": userId}).
        Where(sq.Eq{"message_id": messagesToUpdate}).
        Where(sq.Eq{"read_at": nil})
}

// Crear nuevos registros
if len(messagesToCreate) > 0 {
    insertQuery := dbpq.QueryBuilder().
        Insert("public.room_message_meta").
        Columns("message_id", "user_id", "read_at", "\"isDeleted\"", "\"isSenderBlocked\"")

    for _, messageId := range messagesToCreate {
        insertQuery = insertQuery.Values(messageId, userId, sq.Expr("NOW()"), false, false)
    }
}

// Actualizar estado de mensajes
queryUpdateMessages := dbpq.QueryBuilder().
    Update("room_message").
    Set("status", chatv1.MessageStatus_MESSAGE_STATUS_READ).
    Where(sq.Eq{"id": messageIds})
```

### ReactToMessage

```go
func (r *SQLRoomRepository) ReactToMessage(ctx context.Context, userId int, messageId string, reaction string) error
```

**Lógica de reacciones:**

#### 1. Verificación de Reacción Existente
```go
query := dbpq.QueryBuilder().
    Select("id", "reaction").
    From("room_message_reaction").
    Where(sq.Eq{"\"messageId\"": messageId}).
    Where(sq.Eq{"\"reactedById\"": userId}).
    Where(sq.Eq{"deleted_at": nil}).
    Limit(1)

var existingReactionId sql.NullString
var existingReaction sql.NullString
var reactionExists bool

if rows.Next() {
    err = rows.Scan(&existingReactionId, &existingReaction)
    reactionExists = true
}
```

#### 2. Lógica de Actualización
```go
if reactionExists {
    if reaction == "" {
        // Eliminar reacción (soft delete)
        queryDelete := dbpq.QueryBuilder().
            Update("room_message_reaction").
            Set("deleted_at", sq.Expr("NOW()")).
            Where(sq.Eq{"id": existingReactionId.String})
    } else {
        // Actualizar reacción existente
        queryUpdate := dbpq.QueryBuilder().
            Update("room_message_reaction").
            Set("reaction", reaction).
            Set("updated_at", sq.Expr("NOW()")).
            Where(sq.Eq{"id": existingReactionId.String})
    }
} else {
    // Crear nueva reacción
    queryInsert := dbpq.QueryBuilder().
        Insert("room_message_reaction").
        SetMap(sq.Eq{
            "messageId":   messageId,
            "reactedById": userId,
            "reaction":    reaction,
            "created_at":  sq.Expr("NOW()"),
        })
}

// Actualizar timestamp del mensaje
queryUpdate := dbpq.QueryBuilder().
    Update("room_message").
    Set("updated_at", sq.Expr("NOW()")).
    Where(sq.Eq{"id": messageId})
```

## Optimizaciones de Performance

### Índices Recomendados

```sql
-- Índices para room_member
CREATE INDEX idx_room_member_user_room ON room_member(user_id, room_id) WHERE removed_at IS NULL;
CREATE INDEX idx_room_member_room_user ON room_member(room_id, user_id) WHERE removed_at IS NULL;

-- Índices para room_message
CREATE INDEX idx_room_message_room_created ON room_message(room_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_room_message_sender_message ON room_message(sender_message_id) WHERE sender_message_id IS NOT NULL;

-- Índices para room_message_meta
CREATE INDEX idx_room_message_meta_user_message ON room_message_meta(user_id, message_id);
CREATE INDEX idx_room_message_meta_message_user ON room_message_meta(message_id, user_id);

-- Índices para búsqueda
CREATE INDEX idx_room_name_search ON room USING gin(to_tsvector('spanish', name)) WHERE deleted_at IS NULL;
```

### Query Builder Optimizations

```go
// Reutilización de query builder
baseQuery := dbpq.QueryBuilder().
    Select("room.id", "room.name", /* ... */).
    From("room_member AS mm").
    InnerJoin("room ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL", userId)

// Aplicar filtros condicionalmente
if pagination.Search != "" {
    baseQuery = baseQuery.Where("unaccent(room.name) ILIKE unaccent(?)", "%"+pagination.Search+"%")
}

if pagination.Type != "" {
    baseQuery = baseQuery.Where(sq.Eq{"room.type": pagination.Type})
}
```

### Manejo de Conexiones

```go
// Usar contexto para timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Cerrar rows explícitamente
rows, err := r.db.QueryContext(ctx, queryString, args...)
if err != nil {
    return nil, err
}
defer rows.Close()

// Usar transacciones cortas
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

## Manejo de Errores

### Patrones de Error

```go
// Error wrapping
if err != nil {
    return nil, fmt.Errorf("failed to create room: %w", err)
}

// Verificación de tipos de error
if err == sql.ErrNoRows {
    return nil, nil // No encontrado
}

// Logging de errores
if err != nil {
    log.Printf("Error in SaveMessage: %v", err)
    return nil, err
}
```

### Rollback Automático

```go
// Patrón estándar de transacción
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return nil, err
}
defer func() {
    if err != nil {
        tx.Rollback()
    }
}()

// Operaciones...

if err = tx.Commit(); err != nil {
    return nil, err
}
```

## Integración con Caché

### Patrones de Caché

```go
// Verificar caché antes de consulta
cacheKey := fmt.Sprintf("endpoint:chat:room:{%s}:user:%d", roomId, userId)
if dataCached, existsCached := GetCachedRoom(ctx, cacheKey); existsCached {
    return dataCached, nil
}

// Cachear después de obtener datos
SetCachedRoom(ctx, roomId, cacheKey, item)

// Invalidar caché en modificaciones
DeleteRoomCacheByRoomID(ctx, roomId)
```

### Estrategias de Invalidación

```go
// Invalidación específica por sala
func (r *SQLRoomRepository) UpdateRoom(ctx context.Context, userId int, roomId string, room *chatv1.UpdateRoomRequest) error {
    // Actualizar en base de datos
    _, err = r.db.ExecContext(ctx, queryString, args...)
    if err != nil {
        return err
    }
    
    // Invalidar caché
    DeleteRoomCacheByRoomID(ctx, roomId)
    return nil
}

// Actualización de caché con nuevo mensaje
func (r *SQLRoomRepository) SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error) {
    // Guardar mensaje...
    
    // Actualizar caché con nuevo mensaje
    UpdateRoomCacheWithNewMessage(context.Background(), message)
    
    return message, nil
}
```

## Testing

### Mocks para Testing

```go
type MockSQLRoomRepository struct {
    mock.Mock
}

func (m *MockSQLRoomRepository) CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error) {
    args := m.Called(ctx, userId, room)
    return args.Get(0).(*chatv1.Room), args.Error(1)
}

func (m *MockSQLRoomRepository) GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error) {
    args := m.Called(ctx, userId, roomId, allData, cache)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*chatv1.Room), args.Error(1)
}
```

### Tests de Integración

```go
func TestSQLRoomRepository_Integration(t *testing.T) {
    // Setup database
    db := setupTestDatabase(t)
    repo := NewSQLRoomRepository(db)
    
    ctx := context.Background()
    
    // Test crear sala
    room, err := repo.CreateRoom(ctx, 123, &chatv1.CreateRoomRequest{
        Type: "group",
        Name: proto.String("Test Room"),
        Participants: []int32{456, 789},
    })
    
    require.NoError(t, err)
    assert.NotEmpty(t, room.Id)
    assert.Equal(t, "group", room.Type)
    assert.Equal(t, "Test Room", room.Name)
    
    // Test obtener sala
    retrieved, err := repo.GetRoom(ctx, 123, room.Id, true, false)
    require.NoError(t, err)
    assert.Equal(t, room.Id, retrieved.Id)
    assert.Equal(t, "OWNER", retrieved.Role)
    
    // Test enviar mensaje
    message, err := repo.SaveMessage(ctx, 123, &chatv1.SendMessageRequest{
        RoomId:  room.Id,
        Content: "Test message",
        Type:    "text",
    }, room, nil)
    
    require.NoError(t, err)
    assert.NotEmpty(t, message.Id)
    assert.Equal(t, "Test message", message.Content)
    
    // Test obtener mensajes
    messages, meta, err := repo.GetMessagesFromRoom(ctx, 123, &chatv1.GetMessageHistoryRequest{
        Id:    room.Id,
        Limit: 10,
    })
    
    require.NoError(t, err)
    assert.Len(t, messages, 1)
    assert.Equal(t, message.Id, messages[0].Id)
    assert.Equal(t, uint32(1), meta.ItemCount)
}
```

## Mejores Prácticas

### Manejo de Transacciones

```go
// Siempre usar defer para rollback
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return nil, err
}
defer tx.Rollback()

// Operaciones dentro de la transacción...

// Commit explícito al final
if err = tx.Commit(); err != nil {
    return nil, err
}
```

### Uso de Query Builder

```go
// Construir queries de forma segura
query := dbpq.QueryBuilder().
    Select("id", "name", "created_at").
    From("room").
    Where(sq.Eq{"user_id": userId}).
    Where(sq.Eq{"deleted_at": nil}).
    OrderBy("created_at DESC").
    Limit(uint64(limit))

queryString, args, err := query.ToSql()
if err != nil {
    return nil, err
}

rows, err := r.db.QueryContext(ctx, queryString, args...)
```

### Manejo de Null Values

```go
// Usar sql.NullString para campos opcionales
var name sql.NullString
var description sql.NullString
var photoURL sql.NullString

err = rows.Scan(&item.Id, &name, &description, &photoURL)
if err != nil {
    return nil, err
}

// Asignar valores de forma segura
item.Name = name.String
item.Description = description.String
item.PhotoUrl = photoURL.String
```

## Consideraciones de Seguridad

### Prevención de SQL Injection

```go
// CORRECTO: Usar parámetros
query := dbpq.QueryBuilder().
    Select("*").
    From("room").
    Where(sq.Eq{"id": roomId})

// INCORRECTO: Concatenación de strings
// query := "SELECT * FROM room WHERE id = '" + roomId + "'"
```

### Validación de Entrada

```go
func (r *SQLRoomRepository) GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error) {
    if roomId == "" {
        return nil, fmt.Errorf("room ID cannot be empty")
    }
    
    if userId <= 0 {
        return nil, fmt.Errorf("invalid user ID")
    }
    
    // Continuar con la lógica...
}
```

### Manejo de Permisos

```go
// Verificar que el usuario es miembro de la sala
query := dbpq.QueryBuilder().
    Select("room.id").
    From("room").
    InnerJoin("room_member ON room.id = room_member.room_id").
    Where(sq.Eq{"room.id": roomId}).
    Where(sq.Eq{"room_member.user_id": userId}).
    Where(sq.Eq{"room_member.removed_at": nil})

// Solo proceder si el usuario tiene acceso
```

## Conclusión

La implementación PostgreSQL del repositorio de rooms proporciona una solución robusta y completa para la gestión de datos de chat. Aprovecha las fortalezas de PostgreSQL como ACID compliance, consultas complejas y transacciones robustas, mientras implementa optimizaciones específicas para casos de uso de chat como paginación eficiente, búsqueda de texto y manejo de relaciones complejas.

La implementación está diseñada para ser mantenible, testeable y performante, siguiendo las mejores prácticas de Go y PostgreSQL.