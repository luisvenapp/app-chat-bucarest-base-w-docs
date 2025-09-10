# Documentación Técnica: proto/services/chat/v1/types.proto

## Descripción General

Este archivo define todos los **tipos de datos** utilizados por el servicio de chat usando Protocol Buffers (proto3). Contiene las estructuras de datos fundamentales, enumeraciones, mensajes de request/response y eventos que forman el modelo de datos completo del sistema de chat. Es el archivo fuente desde el cual se generan automáticamente las estructuras de datos en Go y otros lenguajes.

## Estructura del Archivo

### Declaración de Sintaxis y Paquete

```proto
syntax = "proto3";

package services.chat.v1;
```

**Análisis:**
- **Sintaxis**: proto3 (versión moderna y recomendada)
- **Paquete**: `services.chat.v1` (versionado semántico)
- **Consistencia**: Mismo paquete que service.proto

### Comentarios de Validación

```proto
// For validations rules, check the https://github.com/bufbuild/protovalidate repo
// import "buf/validate/validate.proto";
```

**Análisis:**
- **Validación**: Referencia a protovalidate para validaciones futuras
- **Comentado**: No implementado actualmente
- **Extensibilidad**: Preparado para agregar validaciones

## Enumeraciones (Enums)

### MessageStatus

```proto
enum MessageStatus {
  MESSAGE_STATUS_UNSPECIFIED = 0;
  MESSAGE_STATUS_SENDING = 1;
  MESSAGE_STATUS_SENT = 2;
  MESSAGE_STATUS_DELIVERED = 3;
  MESSAGE_STATUS_READ = 4;
  MESSAGE_STATUS_ERROR = 5;
}
```

**Análisis Detallado:**

#### `MESSAGE_STATUS_UNSPECIFIED (0)`
- **Propósito**: Valor por defecto requerido en proto3
- **Uso**: Estado inicial o no definido
- **Buena práctica**: Siempre incluir valor UNSPECIFIED

#### `MESSAGE_STATUS_SENDING (1)`
- **Propósito**: Mensaje en proceso de envío
- **UI**: Mostrar indicador de "enviando..."
- **Duración**: Estado temporal hasta confirmación del servidor

#### `MESSAGE_STATUS_SENT (2)`
- **Propósito**: Mensaje enviado al servidor exitosamente
- **Confirmación**: Servidor ha recibido y procesado el mensaje
- **UI**: Mostrar marca de "enviado" (✓)

#### `MESSAGE_STATUS_DELIVERED (3)`
- **Propósito**: Mensaje entregado al dispositivo del destinatario
- **Confirmación**: Dispositivo del destinatario ha recibido
- **UI**: Mostrar marca de "entregado" (✓✓)

#### `MESSAGE_STATUS_READ (4)`
- **Propósito**: Mensaje leído por el destinatario
- **Confirmación**: Usuario ha visto el mensaje
- **UI**: Mostrar marca de "leído" (✓✓ azul)

#### `MESSAGE_STATUS_ERROR (5)`
- **Propósito**: Error en el envío del mensaje
- **Manejo**: Permitir reintento o mostrar error
- **UI**: Mostrar indicador de error (⚠️)

### SyncStrategy

```proto
enum SyncStrategy {
  SYNC_STRATEGY_UNSPECIFIED = 0;
  SYNC_STRATEGY_FULL = 1; // Historial completo
  SYNC_STRATEGY_RECENT = 2; // Solo mensajes recientes (30 días)
  SYNC_STRATEGY_MINIMAL = 3; // Solo último mensaje
  SYNC_STRATEGY_SMART = 4; // Inteligente según condiciones
}
```

**Análisis de Estrategias:**

#### `SYNC_STRATEGY_FULL (1)`
- **Propósito**: Sincronización completa del historial
- **Uso**: Primera instalación o reset completo
- **Impacto**: Alto uso de ancho de banda y almacenamiento
- **Tiempo**: Puede tomar varios minutos

#### `SYNC_STRATEGY_RECENT (2)`
- **Propósito**: Solo mensajes de los últimos 30 días
- **Uso**: Sincronización después de ausencia prolongada
- **Balance**: Equilibrio entre completitud y eficiencia
- **Configuración**: Período configurable

#### `SYNC_STRATEGY_MINIMAL (3)`
- **Propósito**: Solo el último mensaje por sala
- **Uso**: Vista rápida de salas activas
- **Eficiencia**: Mínimo uso de recursos
- **UX**: Carga rápida de la lista de chats

#### `SYNC_STRATEGY_SMART (4)`
- **Propósito**: Estrategia adaptativa según condiciones
- **Factores**: Conexión, almacenamiento, uso histórico
- **Algoritmo**: Decide automáticamente la mejor estrategia
- **Optimización**: Mejor experiencia de usuario

## Estructuras de Datos Principales

### Room

```proto
message Room {
  string id = 1;
  string name = 2;
  string description = 3;
  string photo_url = 4;
  string encryption_data = 5;
  string type = 6;
  int32 unread_count = 7;
  string role = 8;
  bool join_all_user = 10;
  string last_message_at = 11;
  bool send_message = 12;
  bool add_member = 13;
  bool edit_group = 14;
  string created_at = 15; // ISO 8601
  string updated_at = 16; // ISO 8601
  optional RoomParticipant partner = 17;
  repeated RoomParticipant participants = 18;
  bool is_partner_blocked = 19;
  bool is_muted = 20;
  bool is_pinned = 21;
  MessageData last_message = 22;
}
```

**Análisis Exhaustivo de Campos:**

#### Identificación y Metadatos
- **`id`**: Identificador único de la sala (UUID)
- **`name`**: Nombre de la sala (para grupos) o nombre del contacto (P2P)
- **`description`**: Descripción opcional de la sala
- **`photo_url`**: URL de la imagen/avatar de la sala

#### Seguridad y Tipo
- **`encryption_data`**: Datos de encriptación específicos de la sala
- **`type`**: Tipo de sala ("p2p" para persona a persona, "group" para grupo)

#### Estado del Usuario
- **`unread_count`**: Número de mensajes no leídos por el usuario
- **`role`**: Rol del usuario en la sala ("owner", "admin", "member")
- **`is_muted`**: Si las notificaciones están silenciadas para este usuario
- **`is_pinned`**: Si la sala está fijada en la lista del usuario

#### Permisos y Configuración
- **`join_all_user`**: Si cualquier usuario puede unirse automáticamente
- **`send_message`**: Si el usuario puede enviar mensajes
- **`add_member`**: Si el usuario puede agregar miembros
- **`edit_group`**: Si el usuario puede editar la configuración del grupo

#### Timestamps
- **`created_at`**: Fecha de creación (ISO 8601)
- **`updated_at`**: Fecha de última actualización (ISO 8601)
- **`last_message_at`**: Timestamp del último mensaje

#### Relaciones
- **`partner`**: Información del contacto en chats P2P (opcional)
- **`participants`**: Lista de participantes en grupos (repetido)
- **`last_message`**: Último mensaje de la sala

#### Estados Relacionales
- **`is_partner_blocked`**: Si el partner está bloqueado (solo P2P)

### RoomParticipant

```proto
message RoomParticipant {
  int32 id = 1;
  string phone = 2;
  string name = 3;
  string avatar = 4;
  string role = 5;
  bool is_partner_blocked = 6;
  bool is_partner_muted = 7;
}
```

**Análisis de Campos:**
- **`id`**: ID único del usuario
- **`phone`**: Número de teléfono del usuario
- **`name`**: Nombre del usuario
- **`avatar`**: URL del avatar del usuario
- **`role`**: Rol en la sala ("owner", "admin", "member")
- **`is_partner_blocked`**: Si este usuario está bloqueado por el usuario actual
- **`is_partner_muted`**: Si este usuario está silenciado por el usuario actual

### MessageData

```proto
message MessageData {
  string id = 1;
  string room_id = 2;
  int32 sender_id = 3;
  string sender_name = 4;
  string sender_avatar = 5;
  string sender_phone = 6;
  string content = 7;
  optional MessageData reply = 8;
  optional string forwarded_message_id = 9;
  optional int32 forwarded_message_sender_id = 10;
  optional string forwarded_message_sender_name = 11;
  optional string forwarded_message_sender_avatar = 12;
  optional string forwarded_message_sender_phone = 13;
  repeated Mention mentions = 14;
  MessageStatus status = 15;
  string created_at = 16; // ISO 8601
  string updated_at = 17; // ISO 8601
  string type = 18;
  bool edited = 19;
  bool is_deleted = 20;
  optional string audio_transcription = 21;
  optional string lifetime = 22;
  optional string location_name = 23;
  optional double location_latitude = 24;
  optional double location_longitude = 25;
  optional string origin = 26;
  optional string contact_id = 27;
  optional string contact_name = 28;
  optional string contact_phone = 29;
  optional string file = 30;
  repeated Reaction reactions = 31;
  optional string event = 32;
  optional string sender_message_id = 33;
}
```

**Análisis Exhaustivo:**

#### Identificación
- **`id`**: ID único del mensaje (TimeUUID)
- **`room_id`**: ID de la sala donde se envió
- **`sender_message_id`**: ID del cliente para idempotencia

#### Información del Remitente
- **`sender_id`**: ID del usuario que envió
- **`sender_name`**: Nombre del remitente
- **`sender_avatar`**: Avatar del remitente
- **`sender_phone`**: Teléfono del remitente

#### Contenido Principal
- **`content`**: Contenido del mensaje (puede estar encriptado)
- **`type`**: Tipo de mensaje ("user_message", "system", "file", etc.)

#### Estado y Metadatos
- **`status`**: Estado del mensaje (enum MessageStatus)
- **`created_at`**: Timestamp de creación (ISO 8601)
- **`updated_at`**: Timestamp de última actualización
- **`edited`**: Si el mensaje fue editado
- **`is_deleted`**: Si el mensaje fue eliminado (soft delete)

#### Funcionalidades de Respuesta
- **`reply`**: Mensaje al que responde (referencia recursiva opcional)

#### Funcionalidades de Reenvío
- **`forwarded_message_id`**: ID del mensaje original reenviado
- **`forwarded_message_sender_*`**: Información del remitente original

#### Interacciones Sociales
- **`mentions`**: Lista de menciones (@usuario)
- **`reactions`**: Lista de reacciones emoji

#### Contenido Multimedia
- **`file`**: URL de archivo adjunto
- **`audio_transcription`**: Transcripción de audio (opcional)

#### Ubicación
- **`location_name`**: Nombre de ubicación
- **`location_latitude`**: Latitud GPS
- **`location_longitude`**: Longitud GPS

#### Contactos Compartidos
- **`contact_id`**: ID del contacto compartido
- **`contact_name`**: Nombre del contacto
- **`contact_phone`**: Teléfono del contacto

#### Metadatos Adicionales
- **`lifetime`**: Tiempo de vida del mensaje (mensajes temporales)
- **`origin`**: Origen del mensaje (web, mobile, etc.)
- **`event`**: Datos de evento personalizado

### Mention

```proto
message Mention {
  string id = 1;
  string name = 2;
  string tag = 3;
  string phone = 4;
  string message_id = 5;
}
```

**Análisis:**
- **`id`**: ID del usuario mencionado
- **`name`**: Nombre del usuario mencionado
- **`tag`**: Tag usado en el mensaje (ej: "@juan")
- **`phone`**: Teléfono del usuario mencionado
- **`message_id`**: ID del mensaje que contiene la mención

### Reaction

```proto
message Reaction {
  string id = 1;
  string reaction = 2;
  string message_id = 3;
  string reacted_by_id = 4;
  string reacted_by_name = 5;
  string reacted_by_avatar = 6;
  string reacted_by_phone = 7;
}
```

**Análisis:**
- **`id`**: ID único de la reacción
- **`reaction`**: Emoji de la reacción
- **`message_id`**: ID del mensaje reaccionado
- **`reacted_by_*`**: Información completa del usuario que reaccionó

## Eventos en Tiempo Real

### MessageEvent

```proto
message MessageEvent {
  optional Room room = 1;
  string room_id = 2;
  string event_id = 100;

  oneof event {
    // Eventos de mensajes
    MessageData message = 3;
    MessageStatusUpdate status_update = 4;

    // Eventos de sala
    bool is_room_updated = 5;
    RoomJoinEvent room_join = 6;
    RoomLeaveEvent room_leave = 7;

    // Eventos de typing
    TypingEvent typing = 8;

    // Eventos de error
    ErrorEvent error = 9;

    // Evento de edición de mensaje
    MessageData update_message = 10;

    // Evento de eliminación de mensaje
    string delete_message = 11;

    // Evento de ping de conexión (para evitar que se muera)
    bool connected = 12;
  }
}
```

**Análisis del OneOf:**

#### Eventos de Mensajes
- **`message`**: Nuevo mensaje recibido
- **`update_message`**: Mensaje editado
- **`delete_message`**: ID del mensaje eliminado
- **`status_update`**: Cambio de estado de mensaje

#### Eventos de Sala
- **`is_room_updated`**: Sala actualizada (boolean simple)
- **`room_join`**: Usuario se unió a la sala
- **`room_leave`**: Usuario salió de la sala

#### Eventos de Interacción
- **`typing`**: Usuario escribiendo
- **`error`**: Error en el stream
- **`connected`**: Ping de conexión (keepalive)

### Eventos Específicos

#### RoomJoinEvent
```proto
message RoomJoinEvent {
  int32 user_id = 1;
  string joined_at = 2; // ISO 8601
  optional string display_name = 3; // Nombre a mostrar en la sala
  int32 owner_user_id = 4;
}
```

#### RoomLeaveEvent
```proto
message RoomLeaveEvent {
  repeated int32 users_id = 1;
  string left_at = 2; // ISO 8601
  optional string reason = 3; // Razón opcional (ej: "user left", "kicked", etc.)
}
```

#### TypingEvent
```proto
message TypingEvent {
  int32 user_id = 1;
  bool is_typing = 2; // true = empezó a escribir, false = dejó de escribir
  string updated_at = 3; // ISO 8601
}
```

#### MessageStatusUpdate
```proto
message MessageStatusUpdate {
  string message_id = 1;
  MessageStatus status = 2;
  string updated_at = 3; // ISO 8601
  int32 user_id = 4;
  int32 sender_id = 5;
}
```

#### ErrorEvent
```proto
message ErrorEvent {
  string code = 1;
  string message = 2;
  string details = 3;
}
```

## Mensajes de Request y Response

### Mensajes de Envío

#### SendMessageRequest
```proto
message SendMessageRequest {
  string room_id = 1;
  string content = 2;
  optional string reply_id = 3;
  repeated CreateMention mentions = 4;
  string type = 5;
  optional string lifetime = 6;
  optional string location_name = 7;
  optional double location_latitude = 8;
  optional double location_longitude = 9;
  optional string origin = 10;
  optional string contact_name = 11;
  optional string contact_phone = 12;
  optional string file = 13;
  optional string forward_id = 14;
  optional string event = 15;
  optional string sender_message_id = 16;
}
```

**Análisis de Campos:**
- **Básicos**: room_id, content, type
- **Respuesta**: reply_id para responder mensajes
- **Menciones**: Array de menciones a crear
- **Multimedia**: file, location_*, contact_*
- **Funcionalidades**: forward_id, lifetime, event
- **Idempotencia**: sender_message_id

#### CreateMention
```proto
message CreateMention {
  string tag = 1;
  string user = 2;
}
```

### Mensajes de Gestión de Salas

#### CreateRoomRequest
```proto
message CreateRoomRequest {
  string type = 1;
  optional string name = 2;
  optional string description = 3;
  optional string photo_url = 4;
  optional bool send_message = 6;
  optional bool add_member = 7;
  optional bool edit_group = 8;
  repeated int32 participants = 10;
}
```

#### GetRoomsRequest
```proto
message GetRoomsRequest {
  uint32 page = 1;
  uint32 limit = 2; // Máximo 50 rooms por request
  string search = 3;
  string type = 4;
  string since = 5;
}
```

### Mensajes de Sincronización

#### InitialSyncRequest
```proto
message InitialSyncRequest {
  string last_sync_timestamp = 1; // ISO 8601, opcional
  SyncStrategy sync_strategy = 2;
  int32 messages_per_room = 3; // Máximo 100 mensajes por room
  bool include_archived_rooms = 4;
}
```

#### InitialSyncResponse
```proto
message InitialSyncResponse {
  repeated Room rooms = 1;
  repeated string rooms_deleted = 2;
  repeated MessageData messages = 3;
  string sync_timestamp = 4; // ISO 8601
  SyncSummary summary = 5;
}
```

### Utilidades

#### PaginationMeta
```proto
message PaginationMeta {
  uint32 total_items = 1;
  uint32 item_count = 2;
  uint32 items_per_page = 3;
  uint32 total_pages = 4;
  uint32 current_page = 5;
}
```

#### SyncSummary
```proto
message SyncSummary {
  int32 rooms_synced = 1;
  int32 rooms_deleted = 2;
  int32 messages_synced = 3;
  string sync_duration_ms = 4;
}
```

## Patrones de Diseño

### 1. **Uso de Optional**
```proto
optional MessageData reply = 8;
optional string forwarded_message_id = 9;
```
- **Proto3**: Uso de optional para campos que pueden no estar presentes
- **Diferenciación**: Entre "no establecido" y "valor por defecto"

### 2. **Campos Repeated**
```proto
repeated Mention mentions = 14;
repeated Reaction reactions = 31;
repeated RoomParticipant participants = 18;
```
- **Listas**: Para colecciones de elementos
- **Flexibilidad**: Permite 0 o más elementos

### 3. **Referencias Recursivas**
```proto
optional MessageData reply = 8;
```
- **Anidamiento**: Mensajes que responden a otros mensajes
- **Profundidad**: Limitada por la implementación

### 4. **OneOf para Eventos**
```proto
oneof event {
  MessageData message = 3;
  MessageStatusUpdate status_update = 4;
  // ...
}
```
- **Polimorfismo**: Un evento puede ser de múltiples tipos
- **Eficiencia**: Solo se serializa el tipo activo

### 5. **Timestamps ISO 8601**
```proto
string created_at = 16; // ISO 8601
string updated_at = 17; // ISO 8601
```
- **Estándar**: Formato internacional estándar
- **Zona horaria**: Incluye información de zona horaria

### 6. **IDs como Strings**
```proto
string id = 1;
string room_id = 2;
string message_id = 3;
```
- **Flexibilidad**: Permite UUIDs, TimeUUIDs, etc.
- **Futuro**: Fácil cambio de esquema de IDs

## Consideraciones de Performance

### 1. **Campos Opcionales**
- **Memoria**: Solo se almacenan si están presentes
- **Serialización**: Solo se serializan si tienen valor
- **Bandwidth**: Reduce el tamaño de los mensajes

### 2. **Repeated Fields**
- **Eficiencia**: Optimizados para listas
- **Paginación**: Combinados con PaginationMeta

### 3. **Referencias vs Embedding**
```proto
// Referencia (eficiente)
string forwarded_message_id = 9;

// Embedding (completo pero más grande)
optional MessageData reply = 8;
```

## Validaciones Futuras

### Preparación para Protovalidate
```proto
// Ejemplo de validaciones futuras
message SendMessageRequest {
  string room_id = 1; // [(validate.rules).string.min_len = 1];
  string content = 2; // [(validate.rules).string.max_len = 4096];
  // ...
}
```

## Evolución y Versionado

### Compatibilidad hacia Atrás
- **Campos nuevos**: Siempre opcionales
- **Campos eliminados**: Marcar como deprecated
- **Cambios de tipo**: Requieren nueva versión

### Estrategias de Migración
```proto
// Deprecar campo
string old_field = 1 [deprecated = true];
string new_field = 2;

// Agregar campo opcional
optional string new_feature = 100;
```

## Mejores Prácticas Aplicadas

1. **Documentación**: Comentarios claros en español
2. **Consistencia**: Patrones uniformes de nomenclatura
3. **Extensibilidad**: Campos opcionales para futuras características
4. **Performance**: Diseño eficiente para serialización
5. **Usabilidad**: Estructuras intuitivas y bien organizadas
6. **Seguridad**: Campos para encriptación y autenticación
7. **Internacionalización**: Soporte para múltiples idiomas
8. **Tiempo real**: Eventos optimizados para streaming

Este archivo de tipos proporciona un modelo de datos completo y bien diseñado para un sistema de chat moderno, cubriendo todos los casos de uso desde mensajería básica hasta funcionalidades avanzadas como ubicación, contactos compartidos y eventos en tiempo real.