# Optimización de Funciones GetRoom y GetRoomList

## Resumen
Este documento describe las mejores prácticas para agregar el último mensaje y el conteo de mensajes no vistos a las funciones `GetRoom` y `GetRoomList` en el repositorio de salas.

## Análisis del Código Actual

### Estructura de las Funciones
- **`GetRoom`**: Obtiene una sala específica por ID
- **`GetRoomList`**: Obtiene una lista paginada de salas del usuario

### Tablas Involucradas
- **`room`**: Información principal de la sala
- **`room_member`**: Miembros de la sala
- **`room_message`**: Mensajes de la sala
- **`room_message_meta`**: Metadatos de mensajes (incluye estado de lectura)
- **`public.user`**: Información de usuarios

### Campos Disponibles
- `room.id`, `room.created_at`, `room.image`, `room.name`, `room.description`
- `room.type`, `room.encription_data`, `room.join_all_user`, `room."lastMessageAt"`
- `room.send_message`, `room.add_member`, `room.edit_group`
- `partner.id`, `partner.name`, `partner.phone`, `partner.avatar`
- `me.id`, `me.name`, `me.phone`, `mm.is_muted`, `mm."pinnedAt"`, `mm.is_partner_blocked`, `mm.role`

## 1. Agregar Último Mensaje de Cada Sala

### Opción A: LATERAL JOIN (Recomendada)

#### Ventajas
- Una sola consulta a la base de datos
- Consistencia de datos garantizada
- Mejor rendimiento para listas pequeñas (< 100 salas)

#### Implementación SQL
```sql
SELECT 
    room.id, room.created_at, room.image, room.name, room.description, 
    room.type, room.encription_data, room.join_all_user, room."lastMessageAt",
    room.send_message, room.add_member, room.edit_group,
    partner.id, partner.name, partner.phone, partner.avatar,
    me.id, me.name, me.phone, mm.is_muted, mm."pinnedAt", mm.is_partner_blocked, mm.role,
    -- Último mensaje
    last_msg.id AS last_message_id,
    last_msg.content AS last_message_content,
    last_msg.type AS last_message_type,
    last_msg.created_at AS last_message_created_at,
    last_sender.name AS last_message_sender_name,
    last_sender.phone AS last_message_sender_phone
FROM room_member AS mm
INNER JOIN room ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL AND mm.deleted_at IS NULL
INNER JOIN public."user" AS me ON mm.user_id = me.id
LEFT JOIN room_member AS pm ON room.id = pm.room_id AND pm.user_id <> ? AND room.type = 'p2p' AND pm.removed_at IS NULL AND pm.deleted_at IS NULL
LEFT JOIN public."user" AS partner ON pm.user_id = partner.id
-- LATERAL JOIN para obtener el último mensaje
LEFT JOIN LATERAL (
    SELECT msg.id, msg.content, msg.type, msg.created_at, msg.sender_id
    FROM room_message AS msg
    INNER JOIN room_message_meta AS meta ON msg.id = meta.message_id
    WHERE msg.room_id = room.id 
    AND msg.deleted_at IS NULL 
    AND meta."isDeleted" = false
    ORDER BY msg.created_at DESC
    LIMIT 1
) AS last_msg ON true
LEFT JOIN public."user" AS last_sender ON last_msg.sender_id = last_sender.id
WHERE room.deleted_at IS NULL
```

#### Implementación en Go
```go
// Agregar a la consulta existente
query := dbpq.QueryBuilder().
    Select("room.id", "room.created_at", "room.image", "room.name", "room.description", 
           "room.type", "room.encription_data", "room.join_all_user", "room.\"lastMessageAt\"",
           "room.send_message", "room.add_member", "room.edit_group",
           "partner.id", "partner.name", "partner.phone", "partner.avatar",
           "me.id", "me.name", "me.phone", "mm.is_muted", "mm.\"pinnedAt\"", "mm.is_partner_blocked", "mm.role",
           // Último mensaje
           "last_msg.id AS last_message_id",
           "last_msg.content AS last_message_content",
           "last_msg.type AS last_message_type",
           "last_msg.created_at AS last_message_created_at",
           "last_sender.name AS last_message_sender_name",
           "last_sender.phone AS last_message_sender_phone").
    From("room_member AS mm").
    InnerJoin("room ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL AND mm.deleted_at IS NULL", userId).
    InnerJoin("public.\"user\" AS me ON mm.user_id = me.id").
    LeftJoin("room_member AS pm ON room.id = pm.room_id AND pm.user_id <> ? AND room.type = 'p2p' AND pm.removed_at IS NULL AND pm.deleted_at IS NULL", userId).
    LeftJoin("public.\"user\" AS partner ON pm.user_id = partner.id").
    // LATERAL JOIN para obtener el último mensaje
    LeftJoin("LATERAL (SELECT msg.id, msg.content, msg.type, msg.created_at, msg.sender_id FROM room_message AS msg INNER JOIN room_message_meta AS meta ON msg.id = meta.message_id WHERE msg.room_id = room.id AND msg.deleted_at IS NULL AND meta.\"isDeleted\" = false ORDER BY msg.created_at DESC LIMIT 1) AS last_msg ON true").
    LeftJoin("public.\"user\" AS last_sender ON last_msg.sender_id = last_sender.id").
    Where(sq.Eq{"room.deleted_at": nil})
```

### Opción B: Consulta Separada

#### Ventajas
- Consulta principal más simple
- Mejor para listas grandes de salas
- Más fácil de mantener y debuggear

#### Implementación
```go
func (r *SQLRoomRepository) getLastMessagesForRooms(ctx context.Context, roomIds []string) (map[string]*chatv1.MessageData, error) {
    if len(roomIds) == 0 {
        return make(map[string]*chatv1.MessageData), nil
    }
    
    query := dbpq.QueryBuilder().
        Select("DISTINCT ON (msg.room_id) msg.room_id, msg.id, msg.content, msg.type, msg.created_at, msg.sender_id, sender.name, sender.phone").
        From("room_message AS msg").
        InnerJoin("room_message_meta AS meta ON msg.id = meta.message_id").
        InnerJoin("public.\"user\" AS sender ON msg.sender_id = sender.id").
        Where(sq.Eq{"msg.room_id": roomIds}).
        Where(sq.Eq{"msg.deleted_at": nil}).
        Where(sq.Eq{"meta.\"isDeleted\"": false}).
        OrderBy("msg.room_id, msg.created_at DESC")
    
    queryString, args, err := query.ToSql()
    if err != nil {
        return nil, err
    }
    
    rows, err := r.db.QueryContext(ctx, queryString, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    result := make(map[string]*chatv1.MessageData)
    for rows.Next() {
        var roomId string
        var message chatv1.MessageData
        
        err = rows.Scan(&roomId, &message.Id, &message.Content, &message.Type, 
                       &message.CreatedAt, &message.SenderId, &message.SenderName, &message.SenderPhone)
        if err != nil {
            return nil, err
        }
        
        result[roomId] = &message
    }
    
    return result, nil
}
```

## 2. Agregar Conteo de Mensajes No Vistos

### Opción A: Subconsulta con COUNT (Recomendada)

#### Ventajas
- Una sola consulta
- Consistencia de datos
- Mejor rendimiento

#### Implementación SQL
```sql
-- Agregar a la consulta principal
SELECT 
    -- ... campos existentes ...
    -- Conteo de mensajes no leídos
    (SELECT COUNT(*)
     FROM room_message AS unread_msg
     INNER JOIN room_message_meta AS unread_meta ON unread_msg.id = unread_meta.message_id
     WHERE unread_msg.room_id = room.id 
     AND unread_msg.deleted_at IS NULL 
     AND unread_meta."isDeleted" = false
     AND unread_meta.user_id = ?
     AND unread_meta.read_at IS NULL) AS unread_count
FROM room_member AS mm
-- ... resto de la consulta ...
```

#### Implementación en Go
```go
// Agregar a la consulta existente
query = query.Select("(SELECT COUNT(*) FROM room_message AS unread_msg INNER JOIN room_message_meta AS unread_meta ON unread_msg.id = unread_meta.message_id WHERE unread_msg.room_id = room.id AND unread_msg.deleted_at IS NULL AND unread_meta.\"isDeleted\" = false AND unread_meta.user_id = ? AND unread_meta.read_at IS NULL) AS unread_count", userId)
```

### Opción B: Función Separada para Conteos

#### Implementación
```go
func (r *SQLRoomRepository) getUnreadCountsForRooms(ctx context.Context, userId string, roomIds []string) (map[string]int32, error) {
    if len(roomIds) == 0 {
        return make(map[string]int32), nil
    }
    
    query := dbpq.QueryBuilder().
        Select("room_id", "COUNT(*) as unread_count").
        From("room_message_meta").
        Where(sq.Eq{"user_id": userId}).
        Where(sq.Eq{"message_id": sq.Select("id").From("room_message").Where(sq.Eq{"room_id": roomIds}).Where(sq.Eq{"deleted_at": nil})}).
        Where(sq.Eq{"read_at": nil}).
        Where(sq.Eq{"\"isDeleted\"": false}).
        GroupBy("room_id")
    
    queryString, args, err := query.ToSql()
    if err != nil {
        return nil, err
    }
    
    rows, err := r.db.QueryContext(ctx, queryString, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    result := make(map[string]int32)
    for rows.Next() {
        var roomId string
        var unreadCount int32
        
        err = rows.Scan(&roomId, &unreadCount)
        if err != nil {
            return nil, err
        }
        
        result[roomId] = unreadCount
    }
    
    return result, nil
}
```

## 3. Recomendaciones por Función

### Para `GetRoom`
- **Último mensaje**: Usar **Opción A** (LATERAL JOIN)
- **Conteo no leídos**: Usar **Opción A** (Subconsulta COUNT)
- **Justificación**: Solo se obtiene una sala, el impacto en rendimiento es mínimo

### Para `GetRoomList`
- **Último mensaje**: 
  - Si < 100 salas: Usar **Opción A** (LATERAL JOIN)
  - Si > 100 salas: Usar **Opción B** (Consulta separada)
- **Conteo no leídos**: Usar **Opción A** (Subconsulta COUNT)
- **Justificación**: Balance entre rendimiento y complejidad de consulta

## 4. Estructura de Datos Propuesta

### Campos a Agregar al Objeto Room
```go
type Room struct {
    // ... campos existentes ...
    
    // Último mensaje
    LastMessage *MessageData `json:"last_message,omitempty"`
    
    // Conteo de mensajes no leídos
    UnreadCount int32 `json:"unread_count"`
}

type MessageData struct {
    Id           string `json:"id"`
    Content      string `json:"content"`
    Type         string `json:"type"`
    CreatedAt    string `json:"created_at"`
    SenderId     string `json:"sender_id"`
    SenderName   string `json:"sender_name"`
    SenderPhone  string `json:"sender_phone"`
}
```

## 5. Consideraciones de Rendimiento

### Índices Recomendados
```sql
-- Para optimizar la consulta del último mensaje
CREATE INDEX idx_room_message_room_created ON room_message(room_id, created_at DESC);

-- Para optimizar el conteo de mensajes no leídos
CREATE INDEX idx_room_message_meta_user_read ON room_message_meta(user_id, read_at) WHERE read_at IS NULL;

-- Para optimizar la consulta principal
CREATE INDEX idx_room_member_user_room ON room_member(user_id, room_id) WHERE removed_at IS NULL AND deleted_at IS NULL;
```

### Monitoreo de Rendimiento
- Usar `EXPLAIN ANALYZE` para verificar el plan de ejecución
- Monitorear tiempos de respuesta de las consultas
- Considerar cachear resultados para salas con muchos mensajes

## 6. Implementación Gradual

### Fase 1: Implementar en GetRoom
- Agregar campos de último mensaje y conteo no leído
- Usar LATERAL JOIN para obtener el último mensaje
- Usar subconsulta para el conteo

### Fase 2: Implementar en GetRoomList
- Evaluar rendimiento con LATERAL JOIN
- Si es aceptable, mantener LATERAL JOIN
- Si no es aceptable, migrar a consultas separadas

### Fase 3: Optimización
- Agregar índices necesarios
- Implementar cache si es necesario
- Monitorear métricas de rendimiento

## 7. Testing

### Casos de Prueba
- Sala sin mensajes
- Sala con un solo mensaje
- Sala con muchos mensajes
- Sala con mensajes eliminados
- Usuario con múltiples salas
- Paginación con diferentes límites

### Métricas a Medir
- Tiempo de respuesta de las consultas
- Uso de memoria
- Número de consultas a la base de datos
- Tiempo de parsing de resultados

---

**Nota**: Este documento debe ser actualizado conforme se implementen las optimizaciones y se obtengan métricas de rendimiento reales. 