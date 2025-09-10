# Documentación Técnica: schema.cpl

## Descripción General

El archivo `schema.cpl` define el esquema de base de datos para Cassandra/ScyllaDB, optimizado específicamente para el sistema de chat de alto rendimiento. Implementa un diseño desnormalizado que prioriza la velocidad de lectura sobre la eficiencia de almacenamiento, utilizando patrones de modelado NoSQL para soportar millones de mensajes y usuarios concurrentes.

## Filosofía de Diseño

### Principios de Cassandra/ScyllaDB

1. **Query-First Design**: Cada tabla está optimizada para consultas específicas
2. **Desnormalización**: Datos duplicados para evitar JOINs costosos
3. **Partition Key Strategy**: Distribución uniforme de datos
4. **Clustering Keys**: Ordenamiento eficiente dentro de particiones
5. **Write-Heavy Optimization**: Optimizado para alta escritura de mensajes

## Tablas Principales (Core)

### Tabla messages_by_room

```sql
CREATE TABLE messages_by_room (
    room_id uuid,           -- Clave de Partición: Agrupa todos los mensajes de una sala juntos.
    message_id timeuuid,    -- Clave de Clúster: Ordena los mensajes cronológicamente dentro de la sala.
    sender_id int,
    content text,
    content_decrypted text, -- Para que coincida con la implementación de SQL
    type text,
    created_at timestamp,
    edited boolean,
    is_deleted boolean,     -- Usado para soft-delete.
    reply_to_message_id timeuuid,
    forwarded_from_message_id timeuuid,
    file_url text,
    event text,
    sender_message_id text, -- ID opcional del cliente para idempotencia.
    PRIMARY KEY ((room_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
```

**Análisis Detallado:**

#### Estrategia de Particionado
```sql
PRIMARY KEY ((room_id), message_id)
```

**Partition Key: `room_id`**
- **Propósito**: Agrupa todos los mensajes de una sala en la misma partición
- **Ventaja**: Consultas de historial de chat extremadamente rápidas
- **Distribución**: Cada sala es una partición independiente
- **Escalabilidad**: Distribución natural por número de salas

**Clustering Key: `message_id timeuuid`**
- **Tipo**: `timeuuid` combina timestamp + UUID
- **Ordenamiento**: `DESC` para mostrar mensajes más recientes primero
- **Ventaja**: Ordenamiento automático por tiempo de creación
- **Unicidad**: Garantiza unicidad incluso con alta concurrencia

#### Campos de Mensaje

##### Campos Básicos
```sql
sender_id int,              -- ID del usuario que envía
content text,               -- Contenido encriptado del mensaje
content_decrypted text,     -- Contenido desencriptado para búsqueda
type text,                  -- Tipo: "user_message", "system", "file", etc.
created_at timestamp,       -- Timestamp de creación
```

##### Campos de Estado
```sql
edited boolean,             -- Indica si el mensaje fue editado
is_deleted boolean,         -- Soft delete para mantener historial
```

##### Campos de Relación
```sql
reply_to_message_id timeuuid,        -- Respuesta a otro mensaje
forwarded_from_message_id timeuuid,  -- Mensaje reenviado
```

##### Campos Multimedia
```sql
file_url text,              -- URL de archivo adjunto
event text,                 -- Datos de evento (JSON)
sender_message_id text,     -- ID del cliente para idempotencia
```

#### Optimización de Consultas

**Query Principal: Historial de Chat**
```sql
SELECT * FROM messages_by_room 
WHERE room_id = ? 
ORDER BY message_id DESC 
LIMIT 50;
```

**Características:**
- **O(1) lookup**: Acceso directo por partition key
- **Ordenamiento automático**: Sin necesidad de ORDER BY costoso
- **Paginación eficiente**: LIMIT + token de continuación
- **Cache-friendly**: Datos contiguos en disco

### Tabla rooms_by_user

```sql
CREATE TABLE rooms_by_user (
    user_id int,            -- Clave de Partición: Agrupa todas las salas de un usuario.
    is_pinned boolean,      -- Clave de Clúster 1: Pone las salas pineadas primero.
    last_message_at timestamp, -- Clave de Clúster 2: Ordena por la actividad más reciente.
    room_id uuid,           -- Clave de Clúster 3: Asegura la unicidad de la fila.
    -- Datos desnormalizados para evitar lecturas adicionales
    room_name text,
    room_image text,
    room_type text,
    is_muted boolean,
    role text,
    -- Campos desnormalizados del último mensaje para construir el objeto completo
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

**Análisis de Diseño:**

#### Estrategia de Clustering Compleja
```sql
PRIMARY KEY ((user_id), is_pinned, last_message_at, room_id)
WITH CLUSTERING ORDER BY (is_pinned DESC, last_message_at DESC, room_id DESC)
```

**Clustering Key Compuesta:**
1. **`is_pinned DESC`**: Salas fijadas aparecen primero
2. **`last_message_at DESC`**: Ordenamiento por actividad reciente
3. **`room_id DESC`**: Desempate para unicidad

**Resultado del Ordenamiento:**
```
1. Salas pinned, ordenadas por último mensaje
2. Salas no pinned, ordenadas por último mensaje
```

#### Desnormalización Agresiva

**Datos de Sala:**
```sql
room_name text,             -- Nombre de la sala
room_image text,            -- Avatar/imagen de la sala
room_type text,             -- "p2p" o "group"
is_muted boolean,           -- Estado de notificaciones
role text,                  -- Rol del usuario en la sala
```

**Datos del Último Mensaje:**
```sql
last_message_id timeuuid,           -- ID del último mensaje
last_message_preview text,          -- Preview del contenido
last_message_type text,             -- Tipo de mensaje
last_message_sender_id int,         -- ID del remitente
last_message_sender_name text,      -- Nombre del remitente
last_message_sender_phone text,     -- Teléfono del remitente
last_message_status int,            -- Estado del mensaje
last_message_updated_at timestamp,  -- Última actualización
```

**Ventajas de la Desnormalización:**
- **Una sola consulta**: Lista completa de chats en una query
- **Sin JOINs**: Evita consultas adicionales costosas
- **UI responsiva**: Datos completos para renderizar inmediatamente
- **Menos latencia**: Reducción significativa de round-trips

#### Query Optimizada

**Consulta Principal: Lista de Chats**
```sql
SELECT * FROM rooms_by_user 
WHERE user_id = ? 
ORDER BY is_pinned DESC, last_message_at DESC 
LIMIT 20;
```

### Tabla room_counters_by_user

```sql
CREATE TABLE room_counters_by_user (
    user_id int,            -- Clave de Partición
    room_id uuid,           -- Clave de Clúster
    unread_count counter,   -- El contador de mensajes no leídos
    PRIMARY KEY ((user_id), room_id)
);
```

**Análisis de Contadores:**

#### Tipo Counter
```sql
unread_count counter,
```

**Características de Counters en Cassandra:**
- **Operaciones atómicas**: Increment/decrement thread-safe
- **Distributed**: Funciona en clusters distribuidos
- **Eventually consistent**: Consistencia eventual
- **Performance**: Operaciones muy rápidas

#### Operaciones Típicas

**Incrementar contador:**
```sql
UPDATE room_counters_by_user 
SET unread_count = unread_count + 1 
WHERE user_id = ? AND room_id = ?;
```

**Resetear contador:**
```sql
UPDATE room_counters_by_user 
SET unread_count = 0 
WHERE user_id = ? AND room_id = ?;
```

**Consultar contadores:**
```sql
SELECT room_id, unread_count 
FROM room_counters_by_user 
WHERE user_id = ?;
```

### Tabla room_details

```sql
CREATE TABLE room_details (
    room_id uuid PRIMARY KEY, -- Clave de Partición: Búsqueda directa por ID de sala.
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

**Análisis:**

#### Partition Key Simple
```sql
room_id uuid PRIMARY KEY
```
- **Acceso directo**: Lookup O(1) por ID de sala
- **Distribución**: UUIDs distribuyen uniformemente
- **Uso**: Consultas de detalles específicos de sala

#### Campos de Configuración
```sql
join_all_user boolean,      -- Cualquiera puede unirse
send_message boolean,       -- Permisos de envío
add_member boolean,         -- Permisos de agregar miembros
edit_group boolean,         -- Permisos de edición
```

#### Datos de Encriptación
```sql
encryption_data text,       -- Claves de encriptación de la sala
```
- **Seguridad**: Claves específicas por sala
- **Formato**: JSON encriptado con clave maestra
- **Acceso**: Solo miembros autorizados

### Tabla participants_by_room

```sql
CREATE TABLE participants_by_room (
    room_id uuid,           -- Clave de Partición: Agrupa a todos los miembros de una sala.
    user_id int,            -- Clave de Clúster: Identifica unívocamente al miembro dentro de la sala.
    role text,
    joined_at timestamp,
    is_muted boolean,       -- Para la lógica de mute del usuario en la sala
    is_partner_blocked boolean, -- Para la lógica de bloqueo en chats p2p.
    PRIMARY KEY ((room_id), user_id)
);
```

**Análisis:**

#### Estrategia de Particionado
- **Partition Key**: `room_id` agrupa todos los miembros
- **Clustering Key**: `user_id` para acceso directo a miembro específico

#### Campos de Estado
```sql
role text,                      -- "OWNER", "ADMIN", "MEMBER"
joined_at timestamp,            -- Timestamp de unión
is_muted boolean,               -- Estado de notificaciones
is_partner_blocked boolean,     -- Bloqueo en chats P2P
```

#### Consultas Optimizadas

**Listar miembros de sala:**
```sql
SELECT * FROM participants_by_room 
WHERE room_id = ?;
```

**Verificar membresía:**
```sql
SELECT role FROM participants_by_room 
WHERE room_id = ? AND user_id = ?;
```

## Tablas de Metadatos y Búsqueda Inversa

### Tabla p2p_room_by_users

```sql
CREATE TABLE p2p_room_by_users (
    user1_id int, -- Clave de partición (siempre el ID menor)
    user2_id int, -- Clave de clúster (siempre el ID mayor)
    room_id uuid,
    PRIMARY KEY ((user1_id), user2_id)
);
```

**Análisis:**

#### Prevención de Duplicados
- **Ordenamiento**: `user1_id` siempre menor que `user2_id`
- **Unicidad**: Garantiza una sola sala P2P por par de usuarios
- **Búsqueda**: Lookup eficiente de sala existente

#### Uso en Aplicación
```go
func findP2PRoom(userA, userB int) uuid.UUID {
    user1, user2 := sortUserIDs(userA, userB)
    // SELECT room_id FROM p2p_room_by_users 
    // WHERE user1_id = ? AND user2_id = ?
}
```

### Tabla reactions_by_message

```sql
CREATE TABLE reactions_by_message (
    message_id timeuuid,    -- Clave de Partición: Agrupa todas las reacciones de un mensaje.
    user_id int,            -- Clave de Clúster: Identifica qué usuario reaccionó.
    reaction text,
    created_at timestamp,
    PRIMARY KEY ((message_id), user_id)
);
```

**Análisis:**

#### Agrupación por Mensaje
- **Partition Key**: `message_id` agrupa todas las reacciones
- **Clustering Key**: `user_id` para una reacción por usuario
- **Limitación**: Un usuario solo puede tener una reacción por mensaje

#### Consultas Típicas

**Obtener reacciones de mensaje:**
```sql
SELECT user_id, reaction, created_at 
FROM reactions_by_message 
WHERE message_id = ?;
```

**Agregar/actualizar reacción:**
```sql
INSERT INTO reactions_by_message 
(message_id, user_id, reaction, created_at) 
VALUES (?, ?, ?, ?);
```

### Tabla read_receipts_by_message

```sql
CREATE TABLE read_receipts_by_message (
    message_id timeuuid,    -- Clave de Partición: Agrupa todos los lectores de un mensaje.
    user_id int,            -- Clave de Clúster: Identifica qué usuario leyó el mensaje.
    read_at timestamp,
    PRIMARY KEY ((message_id), user_id)
);
```

**Análisis:**

#### Tracking de Lectura
- **Granularidad**: Por mensaje individual
- **Usuarios**: Todos los que leyeron el mensaje
- **Timestamp**: Momento exacto de lectura

#### Consultas de "Visto por"

**Obtener lectores:**
```sql
SELECT user_id, read_at 
FROM read_receipts_by_message 
WHERE message_id = ?;
```

**Marcar como leído:**
```sql
INSERT INTO read_receipts_by_message 
(message_id, user_id, read_at) 
VALUES (?, ?, ?);
```

## Tablas de Búsqueda Inversa

### Tabla message_by_sender_message_id

```sql
CREATE TABLE message_by_sender_message_id (
    sender_message_id text PRIMARY KEY, -- Clave de Partición: Búsqueda directa por el ID del cliente.
    room_id uuid,
    message_id timeuuid
);
```

**Propósito:**
- **Idempotencia**: Evitar mensajes duplicados del cliente
- **Lookup**: Encontrar mensaje por ID del cliente
- **Deduplicación**: Prevenir reenvíos accidentales

### Tabla room_by_message

```sql
CREATE TABLE room_by_message (
    message_id timeuuid PRIMARY KEY, -- Clave de Partición: Búsqueda directa por el ID del mensaje.
    room_id uuid
);
```

**Propósito:**
- **Lookup inverso**: Encontrar sala de un mensaje
- **Operaciones**: Editar, eliminar, reaccionar a mensaje
- **Autorización**: Verificar permisos de sala

### Tabla room_membership_lookup

```sql
CREATE TABLE room_membership_lookup (
    user_id int,
    room_id uuid,
    is_pinned boolean,
    last_message_at timestamp,
    PRIMARY KEY ((user_id), room_id)
);
```

**Propósito:**
- **Evitar ALLOW FILTERING**: Operaciones eficientes en `rooms_by_user`
- **Metadata**: Información para actualizar clustering keys
- **Performance**: Evita scans costosos

## Tablas de Gestión

### Tabla deleted_rooms_by_user

```sql
CREATE TABLE deleted_rooms_by_user (
    user_id int,            -- Clave de Partición: Agrupa todas las eliminaciones para un usuario.
    deleted_at timestamp,    -- Clave de Clúster 1: Ordena las eliminaciones cronológicamente.
    room_id uuid,           -- Clave de Clúster 2: Asegura la unicidad.
    reason text,            -- 'deleted' (sala eliminada) o 'removed' (usuario eliminado de la sala)
    PRIMARY KEY ((user_id), deleted_at, room_id)
) WITH CLUSTERING ORDER BY (deleted_at DESC);
```

**Análisis:**

#### Sincronización de Eliminaciones
- **Propósito**: Tracking de salas eliminadas para sincronización
- **Ordenamiento**: Por timestamp descendente
- **Razones**: Diferencia entre eliminación y remoción

#### Consultas de Sincronización

**Obtener eliminaciones desde timestamp:**
```sql
SELECT room_id, reason, deleted_at 
FROM deleted_rooms_by_user 
WHERE user_id = ? AND deleted_at > ?;
```

### Tabla message_status_by_user

```sql
CREATE TABLE message_status_by_user (
    user_id int,
    room_id uuid,
    message_id timeuuid,
    status int, -- 0:UNSPECIFIED, 1:SENDING, 2:SENT, 3:DELIVERED, 4:READ, 5:ERROR
    PRIMARY KEY ((user_id, room_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
```

**Análisis:**

#### Partition Key Compuesta
```sql
PRIMARY KEY ((user_id, room_id), message_id)
```
- **Distribución**: Por usuario y sala
- **Agrupación**: Estados de mensajes por contexto
- **Escalabilidad**: Distribución uniforme

#### Estados de Mensaje
```
0: UNSPECIFIED - Estado inicial
1: SENDING     - Enviando
2: SENT        - Enviado al servidor
3: DELIVERED   - Entregado al destinatario
4: READ        - Leído por el destinatario
5: ERROR       - Error en envío
```

## Patrones de Consulta Optimizados

### 1. Historial de Chat
```sql
-- Obtener últimos 50 mensajes de una sala
SELECT * FROM messages_by_room 
WHERE room_id = ? 
ORDER BY message_id DESC 
LIMIT 50;

-- Paginación con token
SELECT * FROM messages_by_room 
WHERE room_id = ? AND message_id < ?
ORDER BY message_id DESC 
LIMIT 50;
```

### 2. Lista de Chats del Usuario
```sql
-- Obtener lista de chats ordenada
SELECT * FROM rooms_by_user 
WHERE user_id = ? 
ORDER BY is_pinned DESC, last_message_at DESC 
LIMIT 20;
```

### 3. Contadores de No Leídos
```sql
-- Obtener todos los contadores
SELECT room_id, unread_count 
FROM room_counters_by_user 
WHERE user_id = ?;

-- Incrementar contador
UPDATE room_counters_by_user 
SET unread_count = unread_count + 1 
WHERE user_id = ? AND room_id = ?;
```

### 4. Verificación de Membresía
```sql
-- Verificar si usuario es miembro
SELECT role FROM participants_by_room 
WHERE room_id = ? AND user_id = ?;
```

## Consideraciones de Performance

### 1. **Distribución de Datos**
- **Partition Keys**: Distribuyen uniformemente la carga
- **Hot Partitions**: Evitadas mediante UUIDs y distribución por usuario
- **Replication Factor**: Configurado para alta disponibilidad

### 2. **Compactación**
- **Size-Tiered**: Para tablas con muchas escrituras
- **Leveled**: Para tablas con muchas lecturas
- **TTL**: Para datos temporales

### 3. **Consistency Levels**
- **Writes**: QUORUM para consistencia
- **Reads**: LOCAL_QUORUM para performance
- **Counters**: QUORUM para precisión

## Mejores Prácticas Implementadas

1. **Query-First Design**: Cada tabla optimizada para consultas específicas
2. **Desnormalización**: Datos duplicados para evitar JOINs
3. **Partition Strategy**: Distribución uniforme de datos
4. **Clustering Optimization**: Ordenamiento eficiente
5. **Counter Usage**: Para métricas que requieren atomicidad
6. **Lookup Tables**: Para búsquedas inversas eficientes
7. **Soft Deletes**: Preservación de historial
8. **Idempotency**: Prevención de duplicados

Este esquema representa un diseño maduro y optimizado para un sistema de chat de alta escala, aprovechando las fortalezas de Cassandra/ScyllaDB para proporcionar latencia baja y alta disponibilidad.