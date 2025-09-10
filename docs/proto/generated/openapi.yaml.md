# Documentación Técnica: proto/generated/openapi.yaml

## Descripción General

Este archivo es **código generado automáticamente** por `protoc-gen-openapi` a partir de los archivos Protocol Buffers del servicio de chat. Proporciona una especificación OpenAPI 3.0.3 completa que documenta toda la API REST del servicio de chat, incluyendo endpoints, esquemas de datos, parámetros y respuestas. Es fundamental para la documentación automática, generación de clientes y testing de la API.

## Estructura del Archivo

### Información de Generación

```yaml
# Generated with protoc-gen-openapi
# https://github.com/google/gnostic/tree/master/cmd/protoc-gen-openapi
```

**Análisis:**
- **Herramienta**: `protoc-gen-openapi` de Google Gnostic
- **Fuente**: Generado desde archivos Protocol Buffers
- **Propósito**: Documentación automática de API REST

### Metadatos de la API

```yaml
openapi: 3.0.3
info:
    title: ChatMessages API
    version: "3"
```

**Características:**
- **Versión OpenAPI**: 3.0.3 (estándar actual)
- **Título**: ChatMessages API
- **Versión**: "3" (versión de la API)

## Endpoints de la API

### Gestión de Mensajes

#### POST /api/chat/v1/send
```yaml
/api/chat/v1/send:
    post:
        tags:
            - ChatService
        description: "Enviar mensaje\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_SendMessage
        requestBody:
            content:
                application/json:
                    schema:
                        $ref: '#/components/schemas/SendMessageRequest'
            required: true
        responses:
            "200":
                description: OK
                content:
                    application/json:
                        schema:
                            $ref: '#/components/schemas/SendMessageResponse'
```

**Análisis:**
- **Método**: POST (operación de escritura)
- **Autenticación**: Requerida (🔒)
- **Body**: JSON con esquema SendMessageRequest
- **Response**: JSON con esquema SendMessageResponse

#### POST /api/chat/v1/edit
```yaml
/api/chat/v1/edit:
    post:
        description: "Editar mensaje existente\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_EditMessage
```

**Propósito**: Editar contenido de mensajes existentes

#### POST /api/chat/v1/delete
```yaml
/api/chat/v1/delete:
    post:
        description: "Eliminar mensaje\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_DeleteMessage
```

**Propósito**: Eliminar mensajes (soft delete)

#### POST /api/chat/v1/react
```yaml
/api/chat/v1/react:
    post:
        description: "Reaccionar a un mensaje\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_ReactToMessage
```

**Propósito**: Agregar reacciones emoji a mensajes

### Gestión de Salas

#### GET /api/chat/v1/room/list
```yaml
/api/chat/v1/room/list:
    get:
        description: "Obtener lista de rooms del usuario\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetRooms
        parameters:
            - name: page
              in: query
              schema:
                type: integer
                format: uint32
            - name: limit
              in: query
              schema:
                type: integer
                format: uint32
            - name: search
              in: query
              schema:
                type: string
            - name: type
              in: query
              schema:
                type: string
            - name: since
              in: query
              schema:
                type: string
```

**Análisis de Parámetros:**
- **page**: Número de página para paginación
- **limit**: Límite de elementos por página
- **search**: Búsqueda por nombre de sala
- **type**: Filtro por tipo de sala (p2p, group)
- **since**: Sincronización incremental

#### POST /api/chat/v1/room/create
```yaml
/api/chat/v1/room/create:
    post:
        description: "Crear un nuevo room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_CreateRoom
```

**Propósito**: Crear nuevas salas de chat

#### GET /api/chat/v1/room/{id}
```yaml
/api/chat/v1/room/{id}:
    get:
        description: "Obtener un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetRoom
        parameters:
            - name: id
              in: path
              required: true
              schema:
                type: string
```

**Propósito**: Obtener detalles de una sala específica

#### PUT /api/chat/v1/room/update
```yaml
/api/chat/v1/room/update:
    put:
        description: "Actualizar un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_UpdateRoom
```

**Propósito**: Actualizar configuración de salas

### Gestión de Participantes

#### GET /api/chat/v1/room/{id}/participants
```yaml
/api/chat/v1/room/{id}/participants:
    get:
        description: "Obtener lista de participantes de un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetRoomParticipants
        parameters:
            - name: id
              in: path
              required: true
              schema:
                type: string
            - name: page
              in: query
              schema:
                type: integer
                format: uint32
            - name: limit
              in: query
              schema:
                type: integer
                format: uint32
            - name: search
              in: query
              schema:
                type: string
```

**Propósito**: Listar participantes con paginación y búsqueda

#### POST /api/chat/v1/room/participant/add
```yaml
/api/chat/v1/room/participant/add:
    post:
        description: "Agregar un participante a un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_AddParticipantToRoom
```

**Propósito**: Agregar nuevos participantes a salas

#### PUT /api/chat/v1/room/participant/update
```yaml
/api/chat/v1/room/participant/update:
    put:
        description: "Modificar role\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_UpdateParticipantRoom
```

**Propósito**: Cambiar roles de participantes

### Funcionalidades Avanzadas

#### POST /api/chat/v1/room/pin
```yaml
/api/chat/v1/room/pin:
    post:
        description: "Pinnear un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_PinRoom
```

**Propósito**: Fijar/desfijar salas en la lista del usuario

#### POST /api/chat/v1/room/mute
```yaml
/api/chat/v1/room/mute:
    post:
        description: "Mutear un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_MuteRoom
```

**Propósito**: Silenciar/activar notificaciones de salas

#### POST /api/chat/v1/room/leave
```yaml
/api/chat/v1/room/leave:
    post:
        description: "Salir o sacar a alguien de un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_LeaveRoom
```

**Propósito**: Salir de salas o remover participantes

#### POST /api/chat/v1/room/block
```yaml
/api/chat/v1/room/block:
    post:
        description: "Bloqueo de usuario\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_BlockUser
```

**Propósito**: Bloquear/desbloquear usuarios en chats P2P

### Consultas de Mensajes

#### GET /api/chat/v1/history/{id}
```yaml
/api/chat/v1/history/{id}:
    get:
        description: "Obtener historial de mensajes de un room\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetMessageHistory
        parameters:
            - name: id
              in: path
              required: true
              schema:
                type: string
            - name: page
              in: query
              schema:
                type: integer
                format: uint32
            - name: limit
              in: query
              schema:
                type: integer
                format: uint32
            - name: beforeMessageId
              in: query
              schema:
                type: string
            - name: beforeDate
              in: query
              schema:
                type: string
            - name: afterMessageId
              in: query
              schema:
                type: string
            - name: afterDate
              in: query
              schema:
                type: string
            - name: messagesPerRoom
              in: query
              schema:
                type: integer
                format: uint32
```

**Análisis de Parámetros Avanzados:**
- **beforeMessageId/afterMessageId**: Paginación basada en cursor
- **beforeDate/afterDate**: Filtrado por fechas (ISO 8601)
- **messagesPerRoom**: Límite específico por sala

#### GET /api/chat/v1/message/{id}
```yaml
/api/chat/v1/message/{id}:
    get:
        description: "Obtener mensaje\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetMessage
```

**Propósito**: Obtener mensaje específico por ID

#### GET /api/chat/v1/sender/message/{senderMessageId}
```yaml
/api/chat/v1/sender/message/{senderMessageId}:
    get:
        description: "Obtener mensaje por sender message\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetSenderMessage
```

**Propósito**: Buscar mensaje por ID del cliente (idempotencia)

#### GET /api/chat/v1/message/{id}/read
```yaml
/api/chat/v1/message/{id}/read:
    get:
        description: "Obtener lecturas de un mensaje por usuario\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetMessageRead
```

**Propósito**: Ver quién ha leído un mensaje

#### GET /api/chat/v1/message/{id}/reactions
```yaml
/api/chat/v1/message/{id}/reactions:
    get:
        description: "Obtener mentions de un mensaje por usuario\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_GetMessageReactions
```

**Propósito**: Obtener reacciones de un mensaje

### Operaciones de Estado

#### POST /api/chat/v1/mark_as_read
```yaml
/api/chat/v1/mark_as_read:
    post:
        description: "Marcar mensajes como leídos\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_MarkMessagesAsRead
```

**Propósito**: Marcar múltiples mensajes como leídos

#### POST /api/chat/v1/sync
```yaml
/api/chat/v1/sync:
    post:
        description: "Sincronización inicial completa\n 🔒 Need private token to access this endpoint"
        operationId: ChatService_InitialSync
```

**Propósito**: Sincronización inicial de datos del chat

## Esquemas de Datos (Components)

### Esquemas de Request

#### SendMessageRequest
```yaml
SendMessageRequest:
    type: object
    properties:
        roomId:
            type: string
        content:
            type: string
        replyId:
            type: string
        mentions:
            type: array
            items:
                $ref: '#/components/schemas/CreateMention'
        type:
            type: string
        lifetime:
            type: string
        locationName:
            type: string
        locationLatitude:
            type: number
            format: double
        locationLongitude:
            type: number
            format: double
        origin:
            type: string
        contactName:
            type: string
        contactPhone:
            type: string
        file:
            type: string
        forwardId:
            type: string
        event:
            type: string
        senderMessageId:
            type: string
```

**Análisis de Campos:**
- **Básicos**: roomId, content, type
- **Respuesta**: replyId para responder mensajes
- **Menciones**: Array de menciones a usuarios
- **Multimedia**: file, locationName/Latitude/Longitude
- **Contactos**: contactName, contactPhone
- **Reenvío**: forwardId para reenviar mensajes
- **Idempotencia**: senderMessageId para evitar duplicados

#### CreateRoomRequest
```yaml
CreateRoomRequest:
    type: object
    properties:
        type:
            type: string
        name:
            type: string
        description:
            type: string
        photoUrl:
            type: string
        sendMessage:
            type: boolean
        addMember:
            type: boolean
        editGroup:
            type: boolean
        participants:
            type: array
            items:
                type: integer
                format: int32
```

**Análisis de Campos:**
- **Metadatos**: type, name, description, photoUrl
- **Permisos**: sendMessage, addMember, editGroup
- **Participantes**: Array de IDs de usuarios

#### InitialSyncRequest
```yaml
InitialSyncRequest:
    type: object
    properties:
        lastSyncTimestamp:
            type: string
        syncStrategy:
            type: integer
            format: enum
        messagesPerRoom:
            type: integer
            format: int32
        includeArchivedRooms:
            type: boolean
    description: Request para sincronización inicial
```

**Análisis de Campos:**
- **Timestamp**: Última sincronización (ISO 8601)
- **Estrategia**: Enum de estrategias de sincronización
- **Límites**: Mensajes por sala
- **Archivados**: Incluir salas archivadas

### Esquemas de Response

#### SendMessageResponse
```yaml
SendMessageResponse:
    type: object
    properties:
        message:
            $ref: '#/components/schemas/MessageData'
        success:
            type: boolean
        errorMessage:
            type: string
```

**Patrón Estándar**: success + errorMessage + datos específicos

#### GetRoomsResponse
```yaml
GetRoomsResponse:
    type: object
    properties:
        items:
            type: array
            items:
                $ref: '#/components/schemas/Room'
        meta:
            $ref: '#/components/schemas/PaginationMeta'
    description: Response para obtener rooms
```

**Patrón de Paginación**: items + meta con información de paginación

### Esquemas de Datos Principales

#### Room
```yaml
Room:
    type: object
    properties:
        id:
            type: string
        name:
            type: string
        description:
            type: string
        photoUrl:
            type: string
        encryptionData:
            type: string
        type:
            type: string
        unreadCount:
            type: integer
            format: int32
        role:
            type: string
        joinAllUser:
            type: boolean
        lastMessageAt:
            type: string
        sendMessage:
            type: boolean
        addMember:
            type: boolean
        editGroup:
            type: boolean
        createdAt:
            type: string
        updatedAt:
            type: string
        partner:
            $ref: '#/components/schemas/RoomParticipant'
        participants:
            type: array
            items:
                $ref: '#/components/schemas/RoomParticipant'
        isPartnerBlocked:
            type: boolean
        isMuted:
            type: boolean
        isPinned:
            type: boolean
        lastMessage:
            $ref: '#/components/schemas/MessageData'
    description: Estructuras de datos principales
```

**Análisis Completo:**
- **Identificación**: id, name, description, photoUrl
- **Seguridad**: encryptionData
- **Estado**: unreadCount, lastMessageAt, createdAt, updatedAt
- **Permisos**: role, sendMessage, addMember, editGroup
- **Configuración**: joinAllUser, isMuted, isPinned
- **Relaciones**: partner, participants, lastMessage

#### MessageData
```yaml
MessageData:
    type: object
    properties:
        id:
            type: string
        roomId:
            type: string
        senderId:
            type: integer
            format: int32
        senderName:
            type: string
        senderAvatar:
            type: string
        senderPhone:
            type: string
        content:
            type: string
        reply:
            $ref: '#/components/schemas/MessageData'
        forwardedMessageId:
            type: string
        forwardedMessageSenderId:
            type: integer
            format: int32
        forwardedMessageSenderName:
            type: string
        forwardedMessageSenderAvatar:
            type: string
        forwardedMessageSenderPhone:
            type: string
        mentions:
            type: array
            items:
                $ref: '#/components/schemas/Mention'
        status:
            type: integer
            format: enum
        createdAt:
            type: string
        updatedAt:
            type: string
        type:
            type: string
        edited:
            type: boolean
        isDeleted:
            type: boolean
        audioTranscription:
            type: string
        lifetime:
            type: string
        locationName:
            type: string
        locationLatitude:
            type: number
            format: double
        locationLongitude:
            type: number
            format: double
        origin:
            type: string
        contactId:
            type: string
        contactName:
            type: string
        contactPhone:
            type: string
        file:
            type: string
        reactions:
            type: array
            items:
                $ref: '#/components/schemas/Reaction'
        event:
            type: string
        senderMessageId:
            type: string
```

**Análisis Exhaustivo:**
- **Identificación**: id, roomId, senderMessageId
- **Remitente**: senderId, senderName, senderAvatar, senderPhone
- **Contenido**: content, type, file
- **Estado**: status (enum), edited, isDeleted
- **Timestamps**: createdAt, updatedAt
- **Respuesta**: reply (referencia recursiva)
- **Reenvío**: forwarded* fields
- **Interacciones**: mentions, reactions
- **Multimedia**: audioTranscription, locationName/Latitude/Longitude
- **Contactos**: contactId, contactName, contactPhone
- **Eventos**: event, origin

### Esquemas de Utilidad

#### PaginationMeta
```yaml
PaginationMeta:
    type: object
    properties:
        totalItems:
            type: integer
            format: uint32
        itemCount:
            type: integer
            format: uint32
        itemsPerPage:
            type: integer
            format: uint32
        totalPages:
            type: integer
            format: uint32
        currentPage:
            type: integer
            format: uint32
```

**Información Completa de Paginación:**
- **totalItems**: Total de elementos disponibles
- **itemCount**: Elementos en la página actual
- **itemsPerPage**: Límite por página
- **totalPages**: Total de páginas
- **currentPage**: Página actual

#### Reaction
```yaml
Reaction:
    type: object
    properties:
        id:
            type: string
        reaction:
            type: string
        messageId:
            type: string
        reactedById:
            type: string
        reactedByName:
            type: string
        reactedByAvatar:
            type: string
        reactedByPhone:
            type: string
```

**Información de Reacciones:**
- **Identificación**: id, messageId
- **Reacción**: reaction (emoji)
- **Usuario**: reactedById, reactedByName, reactedByAvatar, reactedByPhone

## Tags y Organización

```yaml
tags:
    - name: ChatService
```

**Organización**: Todos los endpoints están agrupados bajo el tag "ChatService"

## Patrones de Diseño de la API

### 1. **Consistencia en Autenticación**
- Todos los endpoints requieren token privado (🔒)
- Patrón uniforme de autenticación

### 2. **Patrones de URL**
```
/api/chat/v1/{resource}/{action}
/api/chat/v1/{resource}/{id}
/api/chat/v1/{resource}/{id}/{subresource}
```

### 3. **Métodos HTTP Semánticos**
- **GET**: Consultas y obtención de datos
- **POST**: Creación y operaciones complejas
- **PUT**: Actualizaciones completas

### 4. **Respuestas Consistentes**
```yaml
# Patrón estándar
{
  "success": boolean,
  "errorMessage": string (opcional),
  "data": object (específico)
}

# Patrón de lista
{
  "items": array,
  "meta": PaginationMeta
}
```

### 5. **Paginación Estándar**
- **Query params**: page, limit
- **Metadata**: PaginationMeta en responses
- **Búsqueda**: search parameter

### 6. **Filtrado Avanzado**
- **Temporal**: beforeDate, afterDate
- **Cursor**: beforeMessageId, afterMessageId
- **Tipo**: type parameter
- **Búsqueda**: search parameter

## Uso de la Especificación OpenAPI

### 1. **Generación de Documentación**
```bash
# Swagger UI
swagger-ui-serve openapi.yaml

# Redoc
redoc-cli serve openapi.yaml
```

### 2. **Generación de Clientes**
```bash
# Cliente JavaScript
openapi-generator generate -i openapi.yaml -g javascript -o ./client-js

# Cliente Python
openapi-generator generate -i openapi.yaml -g python -o ./client-python

# Cliente Java
openapi-generator generate -i openapi.yaml -g java -o ./client-java
```

### 3. **Validación de API**
```bash
# Validar especificación
swagger-codegen validate -i openapi.yaml

# Testing con Postman
newman run postman-collection.json --environment env.json
```

### 4. **Mock Server**
```bash
# Prism mock server
prism mock openapi.yaml

# Wiremock
wiremock --port 8080 --root-dir ./mocks
```

## Ventajas de la Especificación

### 1. **Documentación Automática**
- **Siempre actualizada**: Generada desde código fuente
- **Completa**: Incluye todos los endpoints y esquemas
- **Interactiva**: Swagger UI permite testing directo

### 2. **Generación de Clientes**
- **Múltiples lenguajes**: JavaScript, Python, Java, etc.
- **Type safety**: Clientes tipados automáticamente
- **Consistencia**: Misma API en todos los clientes

### 3. **Testing Automatizado**
- **Contract testing**: Validar que la API cumple el contrato
- **Mock testing**: Testing sin servidor real
- **Integration testing**: Validar integración completa

### 4. **Desarrollo Frontend**
- **Prototipado rápido**: Mock server para desarrollo
- **Documentación clara**: Frontend sabe exactamente qué esperar
- **Validación**: Validar requests/responses automáticamente

## Mejores Prácticas

1. **Mantener sincronizada**: Regenerar cuando cambien los .proto
2. **Validar regularmente**: Usar herramientas de validación
3. **Documentar cambios**: Versionar la especificación
4. **Testing**: Usar para contract testing
5. **Clientes**: Generar clientes automáticamente
6. **Mock**: Usar para desarrollo y testing

Esta especificación OpenAPI proporciona una documentación completa y precisa de la API de chat, facilitando el desarrollo, testing e integración de clientes.