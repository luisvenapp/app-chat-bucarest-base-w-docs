# Documentación Técnica: proto/services/chat/v1/service.proto

## Descripción General

Este archivo define la **especificación del servicio de chat** usando Protocol Buffers (proto3). Es el archivo fuente principal que describe todos los métodos RPC del servicio, sus parámetros de entrada y salida, y las anotaciones HTTP que permiten el mapeo entre gRPC y REST. A partir de este archivo se generan automáticamente los clientes, servidores y documentación de la API.

## Estructura del Archivo

### Declaración de Sintaxis y Paquete

```proto
syntax = "proto3";

package services.chat.v1;
```

**Análisis:**
- **Sintaxis**: proto3 (versión moderna de Protocol Buffers)
- **Paquete**: `services.chat.v1` (versionado semántico)
- **Namespace**: Evita conflictos de nombres entre servicios

### Importaciones

```proto
import "google/api/annotations.proto";
import "services/chat/v1/types.proto";
```

**Análisis de Dependencias:**

#### `google/api/annotations.proto`
- **Propósito**: Anotaciones HTTP para mapeo gRPC-REST
- **Funcionalidad**: Permite definir rutas HTTP para métodos gRPC
- **Estándar**: Parte del ecosistema de Google APIs

#### `services/chat/v1/types.proto`
- **Propósito**: Tipos de datos específicos del servicio de chat
- **Contenido**: Mensajes, enums y estructuras de datos
- **Separación**: Mantiene separados servicios y tipos

## Definición del Servicio

```proto
service ChatService {
  // ... métodos RPC
}
```

**Características:**
- **Nombre**: ChatService (descriptivo y claro)
- **Métodos**: 23 métodos RPC que cubren toda la funcionalidad
- **Organización**: Agrupados por funcionalidad

## Métodos RPC del Servicio

### Gestión de Mensajes

#### SendMessage
```proto
// Enviar mensaje
// 🔒 Need private token to access this endpoint
rpc SendMessage(SendMessageRequest) returns (SendMessageResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/send"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Enviar nuevo mensaje a una sala
- **Autenticación**: Requerida (🔒)
- **HTTP**: POST /api/chat/v1/send
- **Body**: Todo el request como JSON
- **Uso**: Funcionalidad principal del chat

#### EditMessage
```proto
// Editar mensaje existente
// 🔒 Need private token to access this endpoint
rpc EditMessage(EditMessageRequest) returns (EditMessageResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/edit"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Modificar contenido de mensajes existentes
- **Restricciones**: Solo el autor puede editar
- **Historial**: Mantiene registro de ediciones
- **HTTP**: POST (operación de modificación)

#### DeleteMessage
```proto
// Eliminar mensaje
// 🔒 Need private token to access this endpoint
rpc DeleteMessage(DeleteMessageRequest) returns (DeleteMessageResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/delete"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Eliminar mensajes
- **Comportamiento**: Soft delete (preserva historial)
- **Batch**: Permite eliminar múltiples mensajes
- **Permisos**: Autor o administrador de sala

#### ReactToMessage
```proto
// Reaccionar a un mensaje
// 🔒 Need private token to access this endpoint
rpc ReactToMessage(ReactToMessageRequest) returns (ReactToMessageResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/react"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Agregar reacciones emoji a mensajes
- **Limitación**: Una reacción por usuario por mensaje
- **Actualización**: Reemplaza reacción existente
- **UI**: Mejora la experiencia de usuario

### Gestión de Salas

#### GetRooms
```proto
// Obtener lista de rooms del usuario
// 🔒 Need private token to access this endpoint
rpc GetRooms(GetRoomsRequest) returns (GetRoomsResponse) {
  option (google.api.http) = {get: "/api/chat/v1/room/list"};
}
```

**Análisis:**
- **Propósito**: Listar salas del usuario
- **HTTP**: GET (operación de lectura)
- **Paginación**: Soporta paginación completa
- **Filtros**: Búsqueda, tipo, sincronización
- **Ordenamiento**: Por actividad reciente

#### CreateRoom
```proto
// Crear un nuevo room
// 🔒 Need private token to access this endpoint
rpc CreateRoom(CreateRoomRequest) returns (CreateRoomResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/room/create"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Crear nuevas salas de chat
- **Tipos**: P2P (persona a persona) o grupo
- **Permisos**: Creador se convierte en owner
- **Configuración**: Permisos y participantes iniciales

#### GetRoom
```proto
// Obtener un room
// 🔒 Need private token to access this endpoint
rpc GetRoom(GetRoomRequest) returns (GetRoomResponse) {
  option (google.api.http) = {get: "/api/chat/v1/room/{id}"};
}
```

**Análisis:**
- **Propósito**: Obtener detalles de sala específica
- **HTTP**: GET con parámetro de ruta {id}
- **Autorización**: Solo miembros pueden acceder
- **Información**: Metadatos completos de la sala

#### UpdateRoom
```proto
// Actualizar un room
// 🔒 Need private token to access this endpoint
rpc UpdateRoom(UpdateRoomRequest) returns (UpdateRoomResponse) {
  option (google.api.http) = {
    put: "/api/chat/v1/room/update"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Actualizar configuración de sala
- **HTTP**: PUT (actualización completa)
- **Permisos**: Solo owners y admins
- **Campos**: Nombre, descripción, imagen, permisos

### Gestión de Participantes

#### GetRoomParticipants
```proto
// Obtener lista de participantes de un room
// 🔒 Need private token to access this endpoint
rpc GetRoomParticipants(GetRoomParticipantsRequest) returns (GetRoomParticipantsResponse) {
  option (google.api.http) = {get: "/api/chat/v1/room/{id}/participants"};
}
```

**Análisis:**
- **Propósito**: Listar participantes de una sala
- **HTTP**: GET con ruta anidada
- **Paginación**: Soporte para salas con muchos miembros
- **Búsqueda**: Filtrar participantes por nombre

#### AddParticipantToRoom
```proto
// Agregar un participante a un room
// 🔒 Need private token to access this endpoint
rpc AddParticipantToRoom(AddParticipantToRoomRequest) returns (AddParticipantToRoomResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/room/participant/add"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Agregar nuevos participantes
- **Permisos**: Según configuración de la sala
- **Batch**: Permite agregar múltiples usuarios
- **Notificaciones**: Genera eventos de unión

#### UpdateParticipantRoom
```proto
// Modificar role
// 🔒 Need private token to access this endpoint
rpc UpdateParticipantRoom(UpdateParticipantRoomRequest) returns (UpdateParticipantRoomResponse) {
  option (google.api.http) = {
    put: "/api/chat/v1/room/participant/update"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Modificar rol de participante
- **HTTP**: PUT (actualización de estado)
- **Roles**: Owner, Admin, Member
- **Restricciones**: Solo owners pueden cambiar roles

#### LeaveRoom
```proto
// Salir o sacar a alguien de un room
// 🔒 Need private token to access this endpoint
rpc LeaveRoom(LeaveRoomRequest) returns (LeaveRoomResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/room/leave"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Salir de sala o remover participantes
- **Flexibilidad**: Auto-salida o remoción de otros
- **Cleanup**: Limpieza automática de datos relacionados
- **Eventos**: Genera eventos de salida

### Funcionalidades Avanzadas

#### PinRoom
```proto
// Pinnear un room
// 🔒 Need private token to access this endpoint
rpc PinRoom(PinRoomRequest) returns (PinRoomResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/room/pin"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Fijar/desfijar sala en lista del usuario
- **UI**: Salas fijadas aparecen primero
- **Personal**: Configuración personal por usuario
- **Toggle**: Mismo endpoint para fijar/desfijar

#### MuteRoom
```proto
// Mutear un room
// 🔒 Need private token to access this endpoint
rpc MuteRoom(MuteRoomRequest) returns (MuteRoomResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/room/mute"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Silenciar/activar notificaciones
- **Granularidad**: Por sala individual
- **Persistencia**: Configuración persistente
- **UX**: Mejora experiencia de usuario

#### BlockUser
```proto
// Bloqueo de usuario
// 🔒 Need private token to access this endpoint
rpc BlockUser(BlockUserRequest) returns (BlockUserResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/room/block"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Bloquear/desbloquear usuario en chat P2P
- **Efecto**: Previene envío de mensajes
- **Bidireccional**: Afecta comunicación en ambas direcciones
- **Moderación**: Herramienta de moderación personal

### Consultas y Historial

#### GetMessageHistory
```proto
// Obtener historial de mensajes de un room
// 🔒 Need private token to access this endpoint
rpc GetMessageHistory(GetMessageHistoryRequest) returns (GetMessageHistoryResponse) {
  option (google.api.http) = {get: "/api/chat/v1/history/{id}"};
}
```

**Análisis:**
- **Propósito**: Obtener historial de mensajes
- **HTTP**: GET con ID de sala en ruta
- **Paginación**: Soporte para historial largo
- **Filtrado**: Por fecha, tipo, usuario
- **Performance**: Optimizado para consultas grandes

#### GetMessage
```proto
// Obtener mensaje
// 🔒 Need private token to access this endpoint
rpc GetMessage(GetMessageRequest) returns (MessageData) {
  option (google.api.http) = {get: "/api/chat/v1/message/{id}"};
}
```

**Análisis:**
- **Propósito**: Obtener mensaje específico
- **HTTP**: GET con ID de mensaje
- **Autorización**: Solo miembros de la sala
- **Retorno**: MessageData directamente (no wrapper)

#### GetSenderMessage
```proto
// Obtener mensaje por sender message
// 🔒 Need private token to access this endpoint
rpc GetSenderMessage(GetSenderMessageRequest) returns (GetSenderMessageResponse) {
  option (google.api.http) = {get: "/api/chat/v1/sender/message/{sender_message_id}"};
}
```

**Análisis:**
- **Propósito**: Obtener mensaje por ID del cliente
- **Idempotencia**: Para evitar duplicados
- **Lookup**: Búsqueda por sender_message_id
- **Uso**: Verificar estado de envío

#### GetMessageRead
```proto
// Obtener lecturas de un mensaje por usuario
// 🔒 Need private token to access this endpoint
rpc GetMessageRead(GetMessageReadRequest) returns (GetMessageReadResponse) {
  option (google.api.http) = {get: "/api/chat/v1/message/{id}/read"};
}
```

**Análisis:**
- **Propósito**: Obtener información de lectura
- **Uso**: Mostrar "visto por" en grupos
- **Privacidad**: Solo en salas donde el usuario es miembro
- **Paginación**: Lista paginada de lectores

#### GetMessageReactions
```proto
// Obtener mentions de un mensaje por usuario
// 🔒 Need private token to access this endpoint
rpc GetMessageReactions(GetMessageReactionsRequest) returns (GetMessageReactionsResponse) {
  option (google.api.http) = {get: "/api/chat/v1/message/{id}/reactions"};
}
```

**Análisis:**
- **Propósito**: Obtener reacciones de un mensaje
- **Agrupación**: Por tipo de reacción
- **Usuarios**: Lista de usuarios que reaccionaron
- **UI**: Para mostrar contadores y listas

#### MarkMessagesAsRead
```proto
// Marcar mensajes como leídos
// 🔒 Need private token to access this endpoint
rpc MarkMessagesAsRead(MarkMessagesAsReadRequest) returns (MarkMessagesAsReadResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/mark_as_read"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Marcar mensajes como leídos
- **Batch**: Múltiples mensajes en una operación
- **Estado**: Actualiza estado de lectura
- **Performance**: Operación optimizada

### Sincronización

#### InitialSync
```proto
// Sincronización inicial completa
// 🔒 Need private token to access this endpoint
rpc InitialSync(InitialSyncRequest) returns (InitialSyncResponse) {
  option (google.api.http) = {
    post: "/api/chat/v1/sync"
    body: "*"
  };
}
```

**Análisis:**
- **Propósito**: Sincronización inicial completa
- **Uso**: Primera carga de la aplicación
- **Estrategias**: Múltiples estrategias de sincronización
- **Optimización**: Datos agregados en una respuesta

#### StreamMessages
```proto
// Stream unidireccional para mensajes en tiempo real
// 🔒 Need private token to access this endpoint
rpc StreamMessages(StreamMessagesRequest) returns (stream MessageEvent) {
  option idempotency_level = IDEMPOTENT;
}
```

**Análisis:**
- **Propósito**: Streaming en tiempo real
- **Tipo**: Server streaming (unidireccional)
- **Eventos**: MessageEvent con múltiples tipos
- **Idempotencia**: Marcado como idempotente
- **Sin HTTP**: Solo gRPC (streaming no mapea bien a HTTP)

## Patrones de Diseño

### 1. **Consistencia en Autenticación**
```proto
// 🔒 Need private token to access this endpoint
```
- **Universal**: Todos los métodos requieren autenticación
- **Documentación**: Comentario estándar en todos los métodos
- **Seguridad**: Enfoque security-first

### 2. **Patrones de Nomenclatura**
```proto
// Patrón: {Verbo}{Recurso}
SendMessage
GetRooms
CreateRoom
UpdateRoom
DeleteMessage
```

### 3. **Patrones HTTP**
```proto
// GET para consultas
option (google.api.http) = {get: "/api/chat/v1/room/list"};

// POST para creación y operaciones complejas
option (google.api.http) = {
  post: "/api/chat/v1/send"
  body: "*"
};

// PUT para actualizaciones
option (google.api.http) = {
  put: "/api/chat/v1/room/update"
  body: "*"
};
```

### 4. **Patrones de URL**
```proto
// Recursos simples
/api/chat/v1/{action}

// Recursos con ID
/api/chat/v1/{resource}/{id}

// Recursos anidados
/api/chat/v1/{resource}/{id}/{subresource}

// Acciones específicas
/api/chat/v1/{resource}/{action}
```

### 5. **Patrones de Request/Response**
```proto
// Patrón estándar
rpc MethodName(MethodNameRequest) returns (MethodNameResponse)

// Excepción: GetMessage retorna directamente
rpc GetMessage(GetMessageRequest) returns (MessageData)

// Streaming
rpc StreamMessages(StreamMessagesRequest) returns (stream MessageEvent)
```

## Anotaciones HTTP Detalladas

### Métodos GET
```proto
// Consulta simple
{get: "/api/chat/v1/room/list"}

// Con parámetro de ruta
{get: "/api/chat/v1/room/{id}"}

// Con múltiples parámetros
{get: "/api/chat/v1/message/{id}/read"}
```

### Métodos POST
```proto
// Body completo
{
  post: "/api/chat/v1/send"
  body: "*"
}
```

### Métodos PUT
```proto
// Actualización completa
{
  put: "/api/chat/v1/room/update"
  body: "*"
}
```

## Versionado y Evolución

### Estrategia de Versionado
- **Paquete**: `services.chat.v1` (versión en el paquete)
- **URL**: `/api/chat/v1/` (versión en la URL)
- **Compatibilidad**: Mantener compatibilidad hacia atrás

### Evolución de la API
```proto
// Agregar nuevos métodos (compatible)
rpc NewMethod(NewMethodRequest) returns (NewMethodResponse);

// Agregar campos opcionales (compatible)
message ExistingRequest {
  string existing_field = 1;
  optional string new_field = 2; // Compatible
}

// Cambios incompatibles requieren v2
```

## Generación de Código

### Archivos Generados
```bash
# Desde este archivo se generan:
- service.pb.go          # Definiciones de servicio
- service.connect.go     # Cliente/servidor ConnectRPC
- service_grpc.pb.go     # Cliente/servidor gRPC (si se usa)
- openapi.yaml          # Especificación OpenAPI
```

### Comandos de Generación
```bash
# Protocol Buffers
protoc --go_out=. --go_opt=paths=source_relative service.proto

# ConnectRPC
protoc --connect-go_out=. --connect-go_opt=paths=source_relative service.proto

# OpenAPI
protoc --openapi_out=. service.proto
```

## Mejores Prácticas Aplicadas

### 1. **Documentación Clara**
- Comentarios descriptivos en español
- Indicadores de autenticación (🔒)
- Propósito claro de cada método

### 2. **Organización Lógica**
- Métodos agrupados por funcionalidad
- Orden lógico de operaciones
- Separación clara de responsabilidades

### 3. **Consistencia**
- Patrones de nomenclatura uniformes
- Estructura de URL consistente
- Manejo de errores estándar

### 4. **Seguridad**
- Autenticación requerida en todos los métodos
- Autorización implícita en la documentación
- Principio de menor privilegio

### 5. **Performance**
- Operaciones batch donde es apropiado
- Paginación en consultas grandes
- Streaming para tiempo real

### 6. **Usabilidad**
- Métodos intuitivos y bien nombrados
- Funcionalidades completas para casos de uso
- Flexibilidad en parámetros

## Casos de Uso Cubiertos

### Chat Básico
- Enviar, editar, eliminar mensajes
- Reacciones y menciones
- Historial de mensajes

### Gestión de Salas
- Crear, actualizar, obtener salas
- Gestión de participantes
- Permisos y roles

### Funcionalidades Avanzadas
- Fijar y silenciar salas
- Bloqueo de usuarios
- Sincronización de datos

### Tiempo Real
- Streaming de eventos
- Estados de lectura
- Notificaciones de typing

Este archivo de servicio proporciona una API completa y bien diseñada para un sistema de chat moderno, siguiendo las mejores prácticas de Protocol Buffers y diseño de APIs.