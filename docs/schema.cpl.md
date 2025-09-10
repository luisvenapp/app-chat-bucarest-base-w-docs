# 📄 Documentación: schema.cpl

## 🎯 Propósito
Schema de base de datos optimizado para ScyllaDB que implementa un modelo de datos desnormalizado para máximo rendimiento en operaciones de chat en tiempo real.

## 🏗️ Arquitectura de Datos

### 📊 Principios de Diseño
- **Query-First Design**: Tablas diseñadas por patrones de consulta
- **Desnormalización**: Duplicación estratégica para performance
- **Particionamiento**: Distribución eficiente de datos
- **Ordenamiento**: Clustering keys para orden temporal

## 📋 Tablas Principales

### 💬 `messages_by_room`
```cql
CREATE TABLE messages_by_room (
    room_id uuid,           -- Partition Key
    message_id timeuuid,    -- Clustering Key
    sender_id int,
    content text,
    content_decrypted text,
    type text,
    created_at timestamp,
    edited boolean,
    is_deleted boolean,
    reply_to_message_id timeuuid,
    forwarded_from_message_id timeuuid,
    file_url text,
    event text,
    sender_message_id text,
    PRIMARY KEY ((room_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
```

#### 🎯 Optimizada Para
- **GetMessagesFromRoom**: Historial de chat por sala
- **Ordenamiento temporal**: Mensajes más recientes primero
- **Paginación eficiente**: Con timeuuid como cursor

#### 🔑 Claves de Diseño
- **Partition Key**: `room_id` agrupa todos los mensajes de una sala
- **Clustering Key**: `timeuuid` garantiza orden cronológico único
- **Soft Delete**: Campo `is_deleted` para eliminación lógica

### 🏠 `rooms_by_user`
```cql
CREATE TABLE rooms_by_user (
    user_id int,            -- Partition Key
    is_pinned boolean,      -- Clustering Key 1
    last_message_at timestamp, -- Clustering Key 2
    room_id uuid,           -- Clustering Key 3
    -- Datos desnormalizados
    room_name text,
    room_image text,
    room_type text,
    is_muted boolean,
    role text,
    -- Último mensaje desnormalizado
    last_message_id timeuuid,
    last_message_preview text,
    last_message_type text,
    last_message_sender_id int,
    last_message_sender_name text,
    last_message_sender_phone text,
    last_message_status int,
    last_message_updated_at timestamp,
    PRIMARY KEY ((user_id), is_pinned, last_message_at, room_id)
) WITH CLUSTERING ORDER BY (is_pinned DESC, last_message_at DESC, room_id DESC);
```

#### 🎯 Optimizada Para
- **GetRooms**: Lista de chats del usuario
- **Ordenamiento**: Salas fijadas primero, luego por actividad
- **Datos completos**: Sin necesidad de JOINs adicionales

#### 🔑 Claves de Diseño
- **Desnormalización**: Último mensaje incluido para UI
- **Ordenamiento**: Pinned → Actividad → ID para consistencia
- **Fan-out**: Actualización manual en cada mensaje nuevo

### 📊 `room_counters_by_user`
```cql
CREATE TABLE room_counters_by_user (
    user_id int,            -- Partition Key
    room_id uuid,           -- Clustering Key
    unread_count counter,   -- Contador distribuido
    PRIMARY KEY ((user_id), room_id)
);
```

#### 🎯 Optimizada Para
- **Contadores de no leídos**: Operaciones atómicas
- **Performance**: Sin locks ni transacciones
- **Escalabilidad**: Contadores distribuidos

### 🏢 `room_details`
```cql
CREATE TABLE room_details (
    room_id uuid PRIMARY KEY,
    name text,
    description text,
    image text,
    type text,
    encryption_data text,
    created_at timestamp,
    updated_at timestamp,
    join_all_user boolean,
    send_message boolean,
    add_member boolean,
    edit_group boolean
);
```

#### 🎯 Optimizada Para
- **GetRoom**: Información estática de la sala
- **Configuración**: Permisos y metadatos
- **Encriptación**: Datos de claves por sala

### 👥 `participants_by_room`
```cql
CREATE TABLE participants_by_room (
    room_id uuid,           -- Partition Key
    user_id int,            -- Clustering Key
    role text,
    joined_at timestamp,
    is_muted boolean,
    is_partner_blocked boolean,
    PRIMARY KEY ((room_id), user_id)
);
```

#### 🎯 Optimizada Para
- **GetRoomParticipants**: Lista de miembros
- **Permisos**: Roles y estados por usuario
- **Moderación**: Mute y bloqueo

## 🔍 Tablas de Lookup

### 🔗 `p2p_room_by_users`
```cql
CREATE TABLE p2p_room_by_users (
    user1_id int,           -- Partition Key (menor ID)
    user2_id int,           -- Clustering Key (mayor ID)
    room_id uuid,
    PRIMARY KEY ((user1_id), user2_id)
);
```

#### 🎯 Propósito
- **Anti-duplicados**: Evita múltiples salas P2P entre mismos usuarios
- **Búsqueda eficiente**: Lookup directo por par de usuarios

### 🔄 `room_membership_lookup`
```cql
CREATE TABLE room_membership_lookup (
    user_id int,
    room_id uuid,
    is_pinned boolean,
    last_message_at timestamp,
    PRIMARY KEY ((user_id), room_id)
);
```

#### 🎯 Propósito
- **Evitar ALLOW FILTERING**: Lookup de claves de clustering
- **Operaciones de actualización**: Pin/unpin, mute/unmute

### 📨 `room_by_message`
```cql
CREATE TABLE room_by_message (
    message_id timeuuid PRIMARY KEY,
    room_id uuid
);
```

#### 🎯 Propósito
- **Búsqueda inversa**: Encontrar sala por mensaje
- **Operaciones**: Edit, delete, react por message_id

## 📈 Tablas de Metadatos

### 🎭 `reactions_by_message`
```cql
CREATE TABLE reactions_by_message (
    message_id timeuuid,    -- Partition Key
    user_id int,            -- Clustering Key
    reaction text,
    created_at timestamp,
    PRIMARY KEY ((message_id), user_id)
);
```

### 👁️ `read_receipts_by_message`
```cql
CREATE TABLE read_receipts_by_message (
    message_id timeuuid,    -- Partition Key
    user_id int,            -- Clustering Key
    read_at timestamp,
    PRIMARY KEY ((message_id), user_id)
);
```

### 🔍 `message_by_sender_message_id`
```cql
CREATE TABLE message_by_sender_message_id (
    sender_message_id text PRIMARY KEY,
    room_id uuid,
    message_id timeuuid
);
```

### 📊 `message_status_by_user`
```cql
CREATE TABLE message_status_by_user (
    user_id int,
    room_id uuid,
    message_id timeuuid,
    status int, -- 0:UNSPECIFIED, 1:SENDING, 2:SENT, 3:DELIVERED, 4:READ, 5:ERROR
    PRIMARY KEY ((user_id, room_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
```

### 🗑️ `deleted_rooms_by_user`
```cql
CREATE TABLE deleted_rooms_by_user (
    user_id int,            -- Partition Key
    deleted_at timestamp,   -- Clustering Key 1
    room_id uuid,           -- Clustering Key 2
    reason text,            -- 'deleted' o 'removed'
    PRIMARY KEY ((user_id), deleted_at, room_id)
) WITH CLUSTERING ORDER BY (deleted_at DESC);
```

## 🎯 Patrones de Consulta Optimizados

### 📱 Pantalla Principal de Chat
```cql
-- Una sola query para lista completa
SELECT * FROM rooms_by_user WHERE user_id = ?;
```

### 💬 Historial de Mensajes
```cql
-- Paginación eficiente con timeuuid
SELECT * FROM messages_by_room 
WHERE room_id = ? AND message_id < ? 
LIMIT 50;
```

### 🔢 Contadores No Leídos
```cql
-- Operación atómica
UPDATE room_counters_by_user 
SET unread_count = unread_count + 1 
WHERE user_id = ? AND room_id = ?;
```

## 🚀 Optimizaciones de Performance

### Particionamiento
- **Hot Partitions**: Evitadas con distribución por user_id
- **Partition Size**: Controlado con TTL y archivado
- **Load Balancing**: Distribución automática

### Clustering
- **Ordenamiento**: Temporal descendente para recientes primero
- **Rango de Consultas**: Eficientes con clustering keys
- **Paginación**: Sin OFFSET, usando tokens

### Desnormalización
- **Fan-out Writes**: Actualización de múltiples vistas
- **Read Performance**: Datos completos en una query
- **Consistency**: Eventual con reconciliación

## 📊 Consideraciones de Escalabilidad

### Crecimiento de Datos
- **Retention**: TTL automático para datos antiguos
- **Archivado**: Migración a cold storage
- **Compactación**: Estrategias optimizadas

### Throughput
- **Write Heavy**: Optimizado para alta escritura
- **Read Patterns**: Cacheable y predecible
- **Batch Operations**: Para operaciones relacionadas

## 🔧 Configuraciones Recomendadas

### Consistency Levels
- **Writes**: LOCAL_QUORUM para durabilidad
- **Reads**: LOCAL_QUORUM para consistencia
- **Counters**: LOCAL_QUORUM siempre

### Compaction
- **Strategy**: Size-tiered para write-heavy
- **Tombstone**: GC agresivo para deletes
- **Bloom Filters**: Optimizados para reads

## 💡 Mejores Prácticas Implementadas

### ✅ Diseño
- Query-first modeling
- Evitar ALLOW FILTERING
- Particiones balanceadas
- Clustering eficiente

### ✅ Performance
- Desnormalización estratégica
- Contadores distribuidos
- Batches optimizados
- TTL para limpieza

### ✅ Mantenibilidad
- Naming conventions claras
- Comentarios descriptivos
- Versionado de schema
- Migración planificada