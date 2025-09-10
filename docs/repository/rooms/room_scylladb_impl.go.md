# Documentación Técnica: repository/rooms/room_scylladb_impl.go

## Descripción General

Este archivo implementa la interfaz `RoomsRepository` utilizando **ScyllaDB** como backend de base de datos. ScyllaDB es una base de datos NoSQL distribuida compatible con Cassandra, optimizada para alta performance y baja latencia. Esta implementación está diseñada para manejar cargas de trabajo masivas con patrones de acceso específicos del dominio de chat.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    "errors"
    "fmt"
    "sort"
    "strings"
    "sync"
    "time"

    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
    "github.com/scylladb-solutions/gocql/v2"
    "google.golang.org/protobuf/proto"
)
```

**Análisis de Dependencias:**
- **`github.com/scylladb-solutions/gocql/v2`**: Driver oficial de ScyllaDB para Go
- **`sync`**: Para operaciones concurrentes y sincronización
- **`google.golang.org/protobuf/proto`**: Para manipulación de Protocol Buffers
- **`chatv1`**: Tipos generados desde Protocol Buffers

## Estructura Principal

### ScyllaRoomRepository

```go
type ScyllaRoomRepository struct {
    session     *gocql.Session
    userFetcher UserFetcher
}

func NewScyllaRoomRepository(session *gocql.Session, userFetcher UserFetcher) RoomsRepository {
    return &ScyllaRoomRepository{
        session:     session,
        userFetcher: userFetcher,
    }
}
```

**Características:**
- **Session**: Conexión a cluster ScyllaDB con pool de conexiones
- **UserFetcher**: Interfaz para obtener datos de usuarios desde servicios externos
- **Thread-safe**: gocql.Session es thread-safe

## Modelo de Datos ScyllaDB

### Principios de Diseño

ScyllaDB requiere un modelado de datos específico basado en patrones de consulta:

1. **Desnormalización**: Los datos se duplican para optimizar consultas
2. **Particionado**: Distribución eficiente de datos por partition key
3. **Clustering**: Ordenamiento automático dentro de particiones
4. **Eventual Consistency**: Consistencia eventual entre réplicas

### Tablas Principales

#### room_details
```cql
CREATE TABLE room_details (
    room_id UUID PRIMARY KEY,
    name TEXT,
    description TEXT,
    image TEXT,
    type TEXT,
    encryption_data TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    join_all_user BOOLEAN,
    send_message BOOLEAN,
    add_member BOOLEAN,
    edit_group BOOLEAN
);
```

#### rooms_by_user
```cql
CREATE TABLE rooms_by_user (
    user_id INT,
    is_pinned BOOLEAN,
    last_message_at TIMESTAMP,
    room_id UUID,
    room_name TEXT,
    room_image TEXT,
    room_type TEXT,
    role TEXT,
    is_muted BOOLEAN,
    last_message_id UUID,
    last_message_preview TEXT,
    last_message_type TEXT,
    last_message_sender_id INT,
    last_message_sender_name TEXT,
    last_message_sender_phone TEXT,
    last_message_status INT,
    last_message_updated_at TIMESTAMP,
    PRIMARY KEY (user_id, is_pinned, last_message_at, room_id)
) WITH CLUSTERING ORDER BY (is_pinned DESC, last_message_at DESC);
```

#### messages_by_room
```cql
CREATE TABLE messages_by_room (
    room_id UUID,
    message_id TIMEUUID,
    sender_id INT,
    content TEXT,
    content_decrypted TEXT,
    type TEXT,
    created_at TIMESTAMP,
    edited BOOLEAN,
    sender_message_id TEXT,
    PRIMARY KEY (room_id, message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
```

## Métodos Principales

### CreateRoom

```go
func (r *ScyllaRoomRepository) CreateRoom(ctx context.Context, userId int, req *chatv1.CreateRoomRequest) (*chatv1.Room, error)
```

**Proceso de Creación:**

#### 1. Verificación de Duplicados P2P
```go
if req.Type == "p2p" && len(req.Participants) > 0 {
    user1, user2 := sortUserIDs(userId, int(req.Participants[0]))
    var existingRoomID gocql.UUID
    err := r.session.Query(`SELECT room_id FROM p2p_room_by_users WHERE user1_id = ? AND user2_id = ?`, user1, user2).WithContext(ctx).Scan(&existingRoomID)
    if err == nil {
        return r.GetRoom(ctx, userId, existingRoomID.String(), true, false)
    }
}
```

**Función auxiliar:**
```go
func sortUserIDs(id1, id2 int) (int, int) {
    if id1 < id2 {
        return id1, id2
    }
    return id2, id1
}
```

#### 2. Generación de IDs y Datos
```go
roomID := gocql.MustRandomUUID()
now := time.Now()
participantsSet := make(map[int32]bool)

// Agregar participantes del request
for _, p := range req.Participants {
    participantsSet[p] = true
}
// El creador siempre es participante
participantsSet[int32(userId)] = true

encryptionData, err := utils.GenerateKeyEncript()
```

#### 3. Batch de Operaciones Principales
```go
batch := r.session.Batch(gocql.LoggedBatch)
batch.Query(`INSERT INTO room_details (room_id, name, description, image, type, encryption_data, created_at, updated_at, join_all_user, send_message, add_member, edit_group) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    roomID, req.Name, req.Description, req.PhotoUrl, req.Type, encryptionData, now, now, joinAllUser, sendMessage, addMember, editGroup)

for participantID := range participantsSet {
    role := "MEMBER"
    if int(participantID) == userId {
        role = "OWNER"
    }
    
    batch.Query(`INSERT INTO participants_by_room (room_id, user_id, role, joined_at, is_muted) VALUES (?, ?, ?, ?, ?)`,
        roomID, participantID, role, now, false)
    batch.Query(`INSERT INTO rooms_by_user (user_id, is_pinned, last_message_at, room_id, room_name, room_image, room_type, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
        participantID, false, now, roomID, req.Name, req.PhotoUrl, req.Type, role)
    batch.Query(`INSERT INTO room_membership_lookup (user_id, room_id, is_pinned, last_message_at) VALUES (?, ?, ?, ?)`,
        participantID, roomID, false, now)
}

if req.Type == "p2p" {
    user1, user2 := sortUserIDs(userId, int(req.Participants[0]))
    batch.Query(`INSERT INTO p2p_room_by_users (user1_id, user2_id, room_id) VALUES (?, ?, ?)`, user1, user2, roomID)
}

if err := r.session.ExecuteBatch(batch); err != nil {
    return nil, fmt.Errorf("error en batch de creación de sala: %w", err)
}
```

#### 4. Inicialización de Contadores
```go
// Los contadores no pueden ir en batches logged
for participantID := range participantsSet {
    err := r.session.Query(`UPDATE room_counters_by_user SET unread_count = unread_count + 0 WHERE user_id = ? AND room_id = ?`, participantID, roomID).WithContext(ctx).Exec()
    if err != nil {
        fmt.Printf("Advertencia: no se pudo inicializar el contador para el usuario %d en la sala %s: %v\n", participantID, roomID, err)
    }
}
```

#### 5. Adición Masiva para Canales
```go
if joinAllUser && req.Type == "channel" {
    go r.addAllSystemUsersToChannel(context.Background(), roomID)
}
```

**Función de adición masiva:**
```go
func (r *ScyllaRoomRepository) addAllSystemUsersToChannel(ctx context.Context, roomID gocql.UUID) {
    allUserIDs, err := r.userFetcher.GetAllUserIDs(ctx)
    if err != nil {
        fmt.Printf("ERROR CRÍTICO: no se pudieron obtener todos los usuarios para el canal %s: %v\n", roomID.String(), err)
        return
    }

    batchSize := 100
    for i := 0; i < len(allUserIDs); i += batchSize {
        end := min(i+batchSize, len(allUserIDs))
        batchIDs := allUserIDs[i:end]

        _, err := r.AddParticipantToRoom(ctx, 0, roomID.String(), batchIDs)
        if err != nil {
            fmt.Printf("Error al añadir lote de usuarios al canal %s: %v\n", roomID.String(), err)
        }
    }
}
```

### GetRoom

```go
func (r *ScyllaRoomRepository) GetRoom(ctx context.Context, userId int, roomId string, allData bool, useCache bool) (*chatv1.Room, error)
```

**Proceso de Obtención:**

#### 1. Verificación de Caché
```go
cacheKey := fmt.Sprintf("endpoint:chat:room:{%s}:shim:user:%d", roomId, userId)
if allData {
    cacheKey = fmt.Sprintf("endpoint:chat:room:{%s}:user:%d", roomId, userId)
}
if useCache {
    if dataCached, existsCached := GetCachedRoom(ctx, cacheKey); existsCached {
        return dataCached, nil
    }
}
```

#### 2. Obtención de Detalles de Sala
```go
roomUUID, err := gocql.ParseUUID(roomId)
if err != nil {
    return nil, fmt.Errorf("ID de sala inválido: %w", err)
}

room := &chatv1.Room{Id: roomId}
var createdAt, updatedAt time.Time
err = r.session.Query(`SELECT name, description, image, type, encryption_data, created_at, updated_at, join_all_user, send_message, add_member, edit_group FROM room_details WHERE room_id = ? LIMIT 1`, roomUUID).
    WithContext(ctx).Scan(&room.Name, &room.Description, &room.PhotoUrl, &room.Type, &room.EncryptionData, &createdAt, &updatedAt, &room.JoinAllUser, &room.SendMessage, &room.AddMember, &room.EditGroup)
```

#### 3. Obtención de Datos del Usuario
```go
var isPinned bool
var lastMessageAt time.Time
err = r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)

if err == nil {
    var lastMessage chatv1.MessageData
    var lastMessageID gocql.UUID
    var lastMessageUpdatedAt time.Time
    err = r.session.Query(`SELECT role, is_muted, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
        userId, isPinned, lastMessageAt, roomUUID).WithContext(ctx).Scan(&room.Role, &room.IsMuted, &lastMessageID, &lastMessage.Content, &lastMessage.Type, &lastMessage.SenderId, &lastMessage.SenderName, &lastMessage.SenderPhone, &lastMessage.Status, &lastMessageUpdatedAt)
}
```

#### 4. Contadores de No Leídos
```go
var unreadCount int64
err = r.session.Query(`SELECT unread_count FROM room_counters_by_user WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&unreadCount)
room.UnreadCount = int32(unreadCount)
```

### GetRoomList

```go
func (r *ScyllaRoomRepository) GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error)
```

**Proceso Optimizado:**

#### 1. Consulta Principal
```go
baseQuery := `SELECT room_id, room_name, room_image, room_type, last_message_at, is_muted, is_pinned, role, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at FROM rooms_by_user WHERE user_id = ?`
args := []any{userId}

iter := r.session.Query(baseQuery, args...).WithContext(ctx).Iter()
defer iter.Close()

var allRooms []*chatv1.Room
roomMap := make(map[string]*chatv1.Room)
var roomIDs []gocql.UUID
```

#### 2. Escaneo Eficiente
```go
scanner := iter.Scanner()
for scanner.Next() {
    var roomID gocql.UUID
    var roomName, roomImage, roomType, role string
    var lastMessageAt, lastMessageUpdatedAt time.Time
    var isMuted, isPinned bool
    var lastMessage chatv1.MessageData
    var lastMessageID gocql.UUID

    err := scanner.Scan(&roomID, &roomName, &roomImage, &roomType, &lastMessageAt, &isMuted, &isPinned, &role, &lastMessageID, &lastMessage.Content, &lastMessage.Type, &lastMessage.SenderId, &lastMessage.SenderName, &lastMessage.SenderPhone, &lastMessage.Status, &lastMessageUpdatedAt)
    
    room := &chatv1.Room{
        Id:            roomID.String(),
        Name:          roomName,
        PhotoUrl:      roomImage,
        Type:          roomType,
        LastMessageAt: lastMessageAt.Format(time.RFC3339),
        IsMuted:       isMuted,
        IsPinned:      isPinned,
        Role:          role,
    }
    
    allRooms = append(allRooms, room)
    roomIDs = append(roomIDs, roomID)
    roomMap[roomID.String()] = room
}
```

#### 3. Enriquecimiento con Contadores
```go
if len(roomIDs) > 0 {
    counterIter := r.session.Query(`SELECT room_id, unread_count FROM room_counters_by_user WHERE user_id = ? AND room_id IN ?`, userId, roomIDs).WithContext(ctx).Iter()
    var roomID gocql.UUID
    var unreadCount int64
    for counterIter.Scan(&roomID, &unreadCount) {
        if room, ok := roomMap[roomID.String()]; ok {
            room.UnreadCount = int32(unreadCount)
        }
    }
    counterIter.Close()
}
```

#### 4. Enriquecimiento Concurrente de Participantes
```go
var groupRoomIDs []string
for _, room := range allRooms {
    if room.Type == "group" {
        groupRoomIDs = append(groupRoomIDs, room.Id)
    }
}

if len(groupRoomIDs) > 0 {
    var wg sync.WaitGroup
    participantChan := make(chan struct {
        RoomID string
        Parts  []*chatv1.RoomParticipant
    }, len(groupRoomIDs))

    for _, roomID := range groupRoomIDs {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            parts, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: id, Limit: 5})
            if err == nil {
                participantChan <- struct {
                    RoomID string
                    Parts  []*chatv1.RoomParticipant
                }{RoomID: id, Parts: parts}
            }
        }(roomID)
    }
    wg.Wait()
    close(participantChan)

    for pData := range participantChan {
        if room, ok := roomMap[pData.RoomID]; ok {
            room.Participants = pData.Parts
        }
    }
}
```

#### 5. Filtrado en Aplicación
```go
// ScyllaDB no soporta LIKE, se filtra en aplicación
var filteredRooms []*chatv1.Room
if pagination != nil && pagination.Search != "" {
    searchTerm, err := removeAccents(strings.ToLower(pagination.Search))
    if err != nil {
        return nil, nil, fmt.Errorf("error al normalizar el término de búsqueda: %w", err)
    }

    for _, room := range allRooms {
        roomName, _ := removeAccents(strings.ToLower(room.Name))
        if strings.Contains(roomName, searchTerm) {
            filteredRooms = append(filteredRooms, room)
        }
    }
} else {
    filteredRooms = allRooms
}
```

### SaveMessage

```go
func (r *ScyllaRoomRepository) SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error)
```

**Proceso de Fan-out:**

#### 1. Inserción del Mensaje Principal
```go
roomUUID, err := gocql.ParseUUID(req.RoomId)
messageID := gocql.TimeUUID()
now := time.Now()
batch := r.session.Batch(gocql.LoggedBatch)

batch.Query(`INSERT INTO messages_by_room (room_id, message_id, sender_id, content, content_decrypted, type, created_at, sender_message_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
    roomUUID, messageID, userId, req.Content, contentDecrypted, req.Type, now, req.SenderMessageId)

if req.SenderMessageId != nil && *req.SenderMessageId != "" {
    batch.Query(`INSERT INTO message_by_sender_message_id (sender_message_id, room_id, message_id) VALUES (?, ?, ?)`,
        *req.SenderMessageId, roomUUID, messageID)
}

batch.Query(`INSERT INTO room_by_message (message_id, room_id) VALUES (?, ?)`, messageID, roomUUID)

if err := r.session.ExecuteBatch(batch); err != nil {
    return nil, fmt.Errorf("error al ejecutar el batch de guardado de mensaje: %w", err)
}
```

#### 2. Fan-out a Participantes
```go
participants, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: req.RoomId})
sender, err := r.userFetcher.GetUserByID(ctx, userId)

// Fan-out para actualizar la lista de salas de cada participante
for _, p := range participants {
    r.updateRoomForUser(ctx, int(p.Id), roomUUID, now, messageID, req, sender)
}
```

#### 3. Actualización de Contadores y Estados
```go
for _, p := range participants {
    if p.Id != int32(userId) {
        r.session.Query(`UPDATE room_counters_by_user SET unread_count = unread_count + 1 WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID).WithContext(ctx).Exec()
        r.session.Query(`INSERT INTO message_status_by_user (user_id, room_id, message_id, status) VALUES (?, ?, ?, ?)`, p.Id, roomUUID, messageID, chatv1.MessageStatus_MESSAGE_STATUS_DELIVERED).WithContext(ctx).Exec()
    } else {
        r.session.Query(`INSERT INTO message_status_by_user (user_id, room_id, message_id, status) VALUES (?, ?, ?, ?)`, p.Id, roomUUID, messageID, chatv1.MessageStatus_MESSAGE_STATUS_SENT).WithContext(ctx).Exec()
    }
}
```

### updateRoomForUser (Función Helper)

```go
func (r *ScyllaRoomRepository) updateRoomForUser(ctx context.Context, userId int, roomUUID gocql.UUID, newTime time.Time, newMsgId gocql.UUID, req *chatv1.SendMessageRequest, sender *User)
```

**Proceso Complejo de Actualización:**

#### 1. Lectura de Estado Actual
```go
var isPinned bool
var lastMessageAt time.Time
err := r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)

var roomName, roomImage, roomType, role string
var isMuted bool
err = r.session.Query(`SELECT room_name, room_image, room_type, is_muted, role FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
    userId, isPinned, lastMessageAt, roomUUID).WithContext(ctx).Scan(&roomName, &roomImage, &roomType, &isMuted, &role)
```

#### 2. Patrón Delete-Insert
```go
// ScyllaDB requiere delete-insert para cambiar clustering keys
batch := r.session.Batch(gocql.LoggedBatch)
batch.Query(`DELETE FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
    userId, isPinned, lastMessageAt, roomUUID)
batch.Query(`INSERT INTO rooms_by_user (user_id, is_pinned, last_message_at, room_id, room_name, room_image, room_type, is_muted, role, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    userId, isPinned, newTime, roomUUID, roomName, roomImage, roomType, isMuted, role, newMsgId, req.Content, req.Type, sender.ID, sender.Name, sender.Phone, int(chatv1.MessageStatus_MESSAGE_STATUS_DELIVERED), newTime)
batch.Query(`UPDATE room_membership_lookup SET last_message_at = ? WHERE user_id = ? AND room_id = ?`, newTime, userId, roomUUID)

if err := r.session.ExecuteBatch(batch); err != nil {
    fmt.Printf("Error en batch de fan-out para usuario %d: %v\n", userId, err)
}
```

### GetMessagesFromRoom

```go
func (r *ScyllaRoomRepository) GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error)
```

**Estrategias de Consulta:**

#### 1. Mensajes de Una Sala
```go
func (r *ScyllaRoomRepository) getMessagesFromSingleRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error) {
    roomUUID, err := gocql.ParseUUID(req.Id)
    
    baseQuery := `SELECT message_id, sender_id, content, type, created_at, edited FROM messages_by_room WHERE room_id = ?`
    args := []any{roomUUID}

    // Paginación por TimeUUID
    if req.BeforeMessageId != nil && *req.BeforeMessageId != "" {
        beforeUUID, err := gocql.ParseUUID(*req.BeforeMessageId)
        baseQuery += " AND message_id < ?"
        args = append(args, beforeUUID)
    }
    
    if req.AfterMessageId != nil && *req.AfterMessageId != "" {
        afterUUID, err := gocql.ParseUUID(*req.AfterMessageId)
        baseQuery += " AND message_id > ?"
        args = append(args, afterUUID)
    }

    if req.Limit > 0 {
        baseQuery += " LIMIT ?"
        args = append(args, int(req.Limit))
    }
}
```

#### 2. Mensajes de Todas las Salas (Concurrente)
```go
func (r *ScyllaRoomRepository) getMessagesFromAllRooms(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error) {
    userRooms, _, err := r.GetRoomList(ctx, userId, &chatv1.GetRoomsRequest{})
    
    var wg sync.WaitGroup
    msgChan := make(chan []*chatv1.MessageData, len(userRooms))
    errChan := make(chan error, len(userRooms))

    for _, room := range userRooms {
        wg.Add(1)
        go func(roomId string) {
            defer wg.Done()

            limit := uint32(10)
            if req != nil && req.MessagesPerRoom > 0 {
                limit = uint32(req.MessagesPerRoom)
            }

            singleRoomReq := &chatv1.GetMessageHistoryRequest{
                Id:    roomId,
                Limit: limit,
            }

            messages, _, err := r.getMessagesFromSingleRoom(ctx, userId, singleRoomReq)
            if err != nil {
                errChan <- fmt.Errorf("error obteniendo mensajes de la sala %s: %w", roomId, err)
                return
            }
            msgChan <- messages
        }(room.Id)
    }

    wg.Wait()
    close(msgChan)
    close(errChan)

    // Combinar y ordenar resultados
    var allMessages []*chatv1.MessageData
    for messages := range msgChan {
        allMessages = append(allMessages, messages...)
    }

    sort.Slice(allMessages, func(i, j int) bool {
        return allMessages[i].CreatedAt > allMessages[j].CreatedAt
    })
}
```

### MarkMessagesAsRead

```go
func (r *ScyllaRoomRepository) MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error)
```

**Proceso Optimizado:**

#### 1. Expansión por Fecha con TimeUUID
```go
if since != "" {
    sinceTime, err := time.Parse(time.RFC3339Nano, since)
    if err != nil {
        return 0, fmt.Errorf("formato de fecha 'since' inválido: %w", err)
    }
    // Generar un timeuuid a partir del timestamp para la comparación
    sinceUUID := gocql.MaxTimeUUID(sinceTime)

    // Obtener todos los IDs de mensajes en la sala antes de la fecha `since`
    iter := r.session.Query(`SELECT message_id FROM messages_by_room WHERE room_id = ? AND message_id < ?`, roomUUID, sinceUUID).WithContext(ctx).Iter()
    var msgID gocql.UUID
    for iter.Scan(&msgID) {
        messageIds = append(messageIds, msgID.String())
    }
}
```

#### 2. Operaciones Batch
```go
if len(finalMessageIds) > 0 {
    batch := r.session.Batch(gocql.LoggedBatch)
    now := time.Now()
    for _, msgIdStr := range finalMessageIds {
        msgUUID, err := gocql.ParseUUID(msgIdStr)
        if err != nil {
            continue
        }
        batch.Query(`INSERT INTO read_receipts_by_message (message_id, user_id, read_at) VALUES (?, ?, ?)`, msgUUID, userId, now)
        batch.Query(`INSERT INTO message_status_by_user (user_id, room_id, message_id, status) VALUES (?, ?, ?, ?)`, userId, roomUUID, msgUUID, chatv1.MessageStatus_MESSAGE_STATUS_READ)
    }
    if err := r.session.ExecuteBatch(batch); err != nil {
        return 0, fmt.Errorf("error al marcar mensajes como leídos: %w", err)
    }
}

// Resetear contador
err = r.session.Query(`UPDATE room_counters_by_user SET unread_count = 0 WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Exec()
```

## Operaciones Complejas

### PinRoom

```go
func (r *ScyllaRoomRepository) PinRoom(ctx context.Context, userId int, roomId string, pin bool) error
```

**Proceso Delete-Insert:**

```go
// 1. Leer la clave de clúster actual
var isPinnedOld bool
var lastMessageAt time.Time
err = r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&isPinnedOld, &lastMessageAt)

// 2. Leer el resto de los datos
err = r.session.Query(`SELECT room_name, room_image, room_type, is_muted, role, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
    userId, isPinnedOld, lastMessageAt, roomUUID).WithContext(ctx).Scan(&roomName, &roomImage, &roomType, &isMuted, &role, &lastMessageID, &lastMessage.Content, &lastMessage.Type, &lastMessage.SenderId, &lastMessage.SenderName, &lastMessage.SenderPhone, &lastMessage.Status, &lastMessageUpdatedAt)

// 3. Ejecutar delete-insert en batch
batch := r.session.Batch(gocql.LoggedBatch)
batch.Query(`DELETE FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`, userId, isPinnedOld, lastMessageAt, roomUUID)
batch.Query(`INSERT INTO rooms_by_user (user_id, is_pinned, last_message_at, room_id, room_name, room_image, room_type, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at, is_muted, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    userId, pin, lastMessageAt, roomUUID, roomName, roomImage, roomType, lastMessageID, lastMessage.Content, lastMessage.Type, lastMessage.SenderId, lastMessage.SenderName, lastMessage.SenderPhone, lastMessage.Status, lastMessageUpdatedAt, isMuted, role)
batch.Query(`UPDATE room_membership_lookup SET is_pinned = ? WHERE user_id = ? AND room_id = ?`, pin, userId, roomUUID)

if err := r.session.ExecuteBatch(batch); err != nil {
    return fmt.Errorf("error en el batch de pin/unpin: %w", err)
}
```

### DeleteRoom

```go
func (r *ScyllaRoomRepository) DeleteRoom(ctx context.Context, userId int, roomId string, partner *int) error
```

**Proceso de Eliminación Completa:**

```go
participants, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: roomId})

now := time.Now()
// Procesar cada usuario secuencialmente para evitar batches multi-partición
for _, p := range participants {
    var isPinned bool
    var lastMessageAt time.Time
    err := r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)

    userBatch := r.session.Batch(gocql.LoggedBatch)
    if err == nil {
        userBatch.Query(`DELETE FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`, p.Id, isPinned, lastMessageAt, roomUUID)
    }
    userBatch.Query(`DELETE FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID)
    userBatch.Query(`INSERT INTO deleted_rooms_by_user (user_id, deleted_at, room_id, reason) VALUES (?, ?, ?, ?)`, p.Id, now, roomUUID, "deleted")
    
    if err := r.session.ExecuteBatch(userBatch); err != nil {
        fmt.Printf("Error al procesar la eliminación para el usuario %d: %v\n", p.Id, err)
    }
    r.session.Query(`DELETE FROM room_counters_by_user WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID).WithContext(ctx).Exec()
}

// Eliminar datos de la sala (partición única)
batch := r.session.Batch(gocql.LoggedBatch)
batch.Query(`DELETE FROM participants_by_room WHERE room_id = ?`, roomUUID)
batch.Query(`DELETE FROM room_details WHERE room_id = ?`, roomUUID)
batch.Query(`DELETE FROM messages_by_room WHERE room_id = ?`, roomUUID)

if err := r.session.ExecuteBatch(batch); err != nil {
    return fmt.Errorf("error en el batch de eliminación de datos de la sala: %w", err)
}
```

## Funciones Auxiliares

### scanMessagesAndCollectUserIDs

```go
func (r *ScyllaRoomRepository) scanMessagesAndCollectUserIDs(iter *gocql.Iter) ([]*chatv1.MessageData, []int, error) {
    var messages []*chatv1.MessageData
    userIDsMap := make(map[int]bool)
    var userIDs []int

    scanner := iter.Scanner()
    for scanner.Next() {
        msg := &chatv1.MessageData{}
        var msgID gocql.UUID
        var createdAt time.Time
        err := scanner.Scan(&msgID, &msg.SenderId, &msg.Content, &msg.Type, &createdAt, &msg.Edited)
        if err != nil {
            return nil, nil, err
        }
        msg.Id = msgID.String()
        msg.CreatedAt = createdAt.Format(time.RFC3339)
        messages = append(messages, msg)

        if !userIDsMap[int(msg.SenderId)] {
            userIDs = append(userIDs, int(msg.SenderId))
            userIDsMap[int(msg.SenderId)] = true
        }
    }
    return messages, userIDs, scanner.Err()
}
```

### enrichMessagesWithUserDetails

```go
func (r *ScyllaRoomRepository) enrichMessagesWithUserDetails(ctx context.Context, messages []*chatv1.MessageData, userIDs []int) error {
    if len(userIDs) == 0 {
        return nil
    }
    users, err := r.userFetcher.GetUsersByID(ctx, userIDs)
    if err != nil {
        return err
    }
    userMap := make(map[int32]User)
    for _, u := range users {
        userMap[int32(u.ID)] = u
    }

    for _, msg := range messages {
        if user, ok := userMap[msg.SenderId]; ok {
            msg.SenderName = user.Name
            if user.Avatar != nil {
                msg.SenderAvatar = *user.Avatar
            }
        }
    }
    return nil
}
```

### removeAccents

```go
func removeAccents(s string) (string, error) {
    // Implementación para remover acentos en búsquedas
    // Necesario porque ScyllaDB no tiene funciones como unaccent de PostgreSQL
    replacements := map[string]string{
        "á": "a", "à": "a", "ä": "a", "â": "a",
        "é": "e", "è": "e", "ë": "e", "ê": "e",
        "í": "i", "ì": "i", "ï": "i", "î": "i",
        "ó": "o", "ò": "o", "ö": "o", "ô": "o",
        "ú": "u", "ù": "u", "ü": "u", "û": "u",
        "ñ": "n",
    }
    
    result := s
    for accented, plain := range replacements {
        result = strings.ReplaceAll(result, accented, plain)
        result = strings.ReplaceAll(result, strings.ToUpper(accented), strings.ToUpper(plain))
    }
    return result, nil
}
```

## Consideraciones de Performance

### Modelado de Datos

**Principios aplicados:**
1. **Query-driven design**: Tablas diseñadas para patrones específicos de consulta
2. **Desnormalización**: Datos duplicados para evitar JOINs
3. **Particionado eficiente**: Distribución uniforme de datos
4. **Clustering apropiado**: Ordenamiento automático de datos

### Batches y Consistencia

```go
// Logged Batch para operaciones en la misma partición
batch := r.session.Batch(gocql.LoggedBatch)

// Unlogged Batch para operaciones en diferentes particiones (más rápido)
batch := r.session.Batch(gocql.UnloggedBatch)

// Counter Batch para operaciones de contador
batch := r.session.Batch(gocql.CounterBatch)
```

### Concurrencia

```go
// Operaciones concurrentes para enriquecimiento de datos
var wg sync.WaitGroup
resultChan := make(chan Result, len(items))

for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        result := processItem(item)
        resultChan <- result
    }(item)
}

wg.Wait()
close(resultChan)
```

## Limitaciones y Workarounds

### Limitaciones de ScyllaDB

1. **No JOINs**: Requiere desnormalización
2. **No LIKE**: Filtrado en aplicación
3. **Clustering key updates**: Requiere delete-insert
4. **No transacciones ACID**: Solo batches con garantías limitadas

### Workarounds Implementados

#### 1. Búsqueda de Texto
```go
// En lugar de LIKE, filtrar en aplicación
var filteredRooms []*chatv1.Room
for _, room := range allRooms {
    if strings.Contains(strings.ToLower(room.Name), strings.ToLower(searchTerm)) {
        filteredRooms = append(filteredRooms, room)
    }
}
```

#### 2. Actualización de Clustering Keys
```go
// Delete-Insert pattern para cambiar clustering keys
batch.Query(`DELETE FROM table WHERE pk = ? AND ck1 = ? AND ck2 = ?`, pk, oldCk1, oldCk2)
batch.Query(`INSERT INTO table (pk, ck1, ck2, data) VALUES (?, ?, ?, ?)`, pk, newCk1, newCk2, data)
```

#### 3. Contadores Separados
```go
// Los contadores no pueden ir en logged batches
for _, participant := range participants {
    r.session.Query(`UPDATE room_counters_by_user SET unread_count = unread_count + 1 WHERE user_id = ? AND room_id = ?`, participant, roomID).Exec()
}
```

## Testing

### Mocks para ScyllaDB

```go
type MockScyllaRoomRepository struct {
    mock.Mock
}

func (m *MockScyllaRoomRepository) CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error) {
    args := m.Called(ctx, userId, room)
    return args.Get(0).(*chatv1.Room), args.Error(1)
}
```

### Tests de Integración

```go
func TestScyllaRoomRepository_Integration(t *testing.T) {
    // Setup ScyllaDB test cluster
    cluster := gocql.NewCluster("127.0.0.1")
    cluster.Keyspace = "test_chat"
    session, err := cluster.CreateSession()
    require.NoError(t, err)
    defer session.Close()
    
    userFetcher := &MockUserFetcher{}
    repo := NewScyllaRoomRepository(session, userFetcher)
    
    ctx := context.Background()
    
    // Test crear sala
    room, err := repo.CreateRoom(ctx, 123, &chatv1.CreateRoomRequest{
        Type: "group",
        Name: proto.String("Test Room"),
        Participants: []int32{456, 789},
    })
    
    require.NoError(t, err)
    assert.NotEmpty(t, room.Id)
    
    // Verificar que se puede obtener
    retrieved, err := repo.GetRoom(ctx, 123, room.Id, true, false)
    require.NoError(t, err)
    assert.Equal(t, room.Id, retrieved.Id)
}
```

## Mejores Prácticas

### Manejo de Errores

```go
// Verificar tipos específicos de error de ScyllaDB
if err != nil {
    if err == gocql.ErrNotFound {
        return nil, nil // No encontrado
    }
    return nil, fmt.Errorf("error de ScyllaDB: %w", err)
}
```

### Uso de Context

```go
// Siempre usar contexto con timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := r.session.Query(`SELECT * FROM table WHERE id = ?`, id).WithContext(ctx).Scan(&result)
```

### Preparación de Statements

```go
// Para consultas frecuentes, preparar statements
type PreparedStatements struct {
    getRoomStmt     *gocql.Query
    saveMessageStmt *gocql.Query
}

func (r *ScyllaRoomRepository) initPreparedStatements() {
    r.prepared.getRoomStmt = r.session.Query(`SELECT * FROM room_details WHERE room_id = ?`)
    r.prepared.saveMessageStmt = r.session.Query(`INSERT INTO messages_by_room (room_id, message_id, sender_id, content) VALUES (?, ?, ?, ?)`)
}
```

## Conclusión

La implementación ScyllaDB del repositorio de rooms está optimizada para alta escala y baja latencia. Utiliza patrones específicos de NoSQL como desnormalización, fan-out writes y modelado query-driven para proporcionar performance excepcional en cargas de trabajo de chat.

Las principales ventajas incluyen escalabilidad horizontal masiva, latencia predecible y alta disponibilidad, mientras que las limitaciones se mitigan mediante workarounds específicos y patrones de diseño apropiados para bases de datos distribuidas.