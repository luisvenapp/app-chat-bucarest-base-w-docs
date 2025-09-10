# Documentación Técnica: repository/rooms/room.go

## Descripción General

El archivo `room.go` define las interfaces y estructuras fundamentales para el acceso a datos relacionados con salas de chat y usuarios. Implementa el patrón Repository, proporcionando una abstracción limpia entre la lógica de negocio y la capa de persistencia, permitiendo múltiples implementaciones (SQL, NoSQL, cache, etc.).

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
)
```

**Análisis de Importaciones:**

- **`context`**: Para manejo de contexto, cancelación y timeouts
- **`chatv1`**: Tipos generados de Protocol Buffers para el servicio de chat

## Estructura User

```go
type User struct {
    ID        int     `json:"id"`
    Name      string  `json:"name"`
    Phone     string  `json:"phone"`
    Email     *string `json:"email"`
    Avatar    *string `json:"avatar"`
    Dni       *string `json:"dni"`
    CreatedAt *string `json:"created_at"`
}
```

**Análisis Detallado:**

### Campos Obligatorios

#### `ID int`
- **Propósito**: Identificador único del usuario
- **Tipo**: Entero para eficiencia en índices de base de datos
- **Uso**: Clave primaria, referencias foráneas
- **JSON**: `"id"` para serialización

#### `Name string`
- **Propósito**: Nombre completo del usuario
- **Tipo**: String obligatorio
- **Uso**: Mostrar en interfaz de chat, notificaciones
- **JSON**: `"name"` para API responses

#### `Phone string`
- **Propósito**: Número de teléfono del usuario
- **Tipo**: String obligatorio
- **Uso**: Identificación única, contacto, verificación
- **JSON**: `"phone"` para serialización

### Campos Opcionales

#### `Email *string`
- **Propósito**: Dirección de correo electrónico
- **Tipo**: Puntero a string (nullable)
- **Uso**: Comunicación alternativa, recuperación de cuenta
- **JSON**: `"email"`, puede ser null

#### `Avatar *string`
- **Propósito**: URL de la imagen de perfil
- **Tipo**: Puntero a string (nullable)
- **Uso**: Mostrar foto de perfil en chat
- **JSON**: `"avatar"`, puede ser null

#### `Dni *string`
- **Propósito**: Documento de identidad
- **Tipo**: Puntero a string (nullable)
- **Uso**: Verificación de identidad, compliance
- **JSON**: `"dni"`, puede ser null

#### `CreatedAt *string`
- **Propósito**: Timestamp de creación de la cuenta
- **Tipo**: Puntero a string (nullable)
- **Formato**: ISO 8601 (ej: "2024-01-15T10:30:00Z")
- **JSON**: `"created_at"`, puede ser null

### Consideraciones de Diseño

#### Uso de Punteros para Campos Opcionales
```go
Email     *string `json:"email"`
Avatar    *string `json:"avatar"`
```

**Ventajas:**
- **Null Safety**: Distingue entre valor vacío y valor ausente
- **Database Mapping**: Mapea correctamente a campos NULL en SQL
- **JSON Serialization**: Serializa como null en lugar de string vacío
- **Memory Efficiency**: No almacena strings vacíos innecesarios

#### Tags JSON
```go
`json:"id"`
`json:"name"`
```

**Propósito:**
- **API Consistency**: Nombres consistentes en respuestas JSON
- **Serialization Control**: Control sobre cómo se serializa la estructura
- **Client Compatibility**: Compatibilidad con clientes que esperan nombres específicos

## Interface UserFetcher

```go
type UserFetcher interface {
    GetUserByID(ctx context.Context, id int) (*User, error)
    GetUsersByID(ctx context.Context, ids []int) ([]User, error)
    GetAllUserIDs(ctx context.Context) ([]int, error)
}
```

**Análisis de la Interface:**

### Propósito
- **Abstracción**: Define contrato para obtener información de usuarios
- **Composición**: Puede ser implementada independientemente
- **Reutilización**: Utilizable por múltiples repositories
- **Testing**: Facilita mocking para tests

### Métodos Definidos

#### `GetUserByID(ctx context.Context, id int) (*User, error)`

**Propósito**: Obtener un usuario específico por su ID

**Parámetros:**
- **`ctx context.Context`**: Contexto para cancelación y timeouts
- **`id int`**: Identificador único del usuario

**Retorno:**
- **`*User`**: Puntero a estructura User (nil si no existe)
- **`error`**: Error si ocurre problema en la consulta

**Casos de Uso:**
```go
// Obtener información del remitente de un mensaje
user, err := repo.GetUserByID(ctx, message.SenderID)
if err != nil {
    return fmt.Errorf("failed to get sender: %w", err)
}
if user == nil {
    return errors.New("sender not found")
}
```

#### `GetUsersByID(ctx context.Context, ids []int) ([]User, error)`

**Propósito**: Obtener múltiples usuarios en una sola consulta

**Parámetros:**
- **`ctx context.Context`**: Contexto para operación
- **`ids []int`**: Slice de IDs de usuarios a obtener

**Retorno:**
- **`[]User`**: Slice de usuarios encontrados
- **`error`**: Error si falla la consulta

**Ventajas:**
- **Performance**: Una consulta en lugar de N consultas
- **Atomicidad**: Operación atómica
- **Consistency**: Datos consistentes en un punto en el tiempo

**Casos de Uso:**
```go
// Obtener información de todos los participantes de una sala
participantIDs := []int{123, 456, 789}
participants, err := repo.GetUsersByID(ctx, participantIDs)
if err != nil {
    return fmt.Errorf("failed to get participants: %w", err)
}

// Procesar cada participante
for _, participant := range participants {
    fmt.Printf("Participant: %s (%s)\n", participant.Name, participant.Phone)
}
```

#### `GetAllUserIDs(ctx context.Context) ([]int, error)`

**Propósito**: Obtener lista de todos los IDs de usuarios

**Parámetros:**
- **`ctx context.Context`**: Contexto para operación

**Retorno:**
- **`[]int`**: Slice con todos los IDs de usuarios
- **`error`**: Error si falla la consulta

**Casos de Uso:**
```go
// Para operaciones de broadcast o migración
allUserIDs, err := repo.GetAllUserIDs(ctx)
if err != nil {
    return fmt.Errorf("failed to get all user IDs: %w", err)
}

// Enviar notificación a todos los usuarios
for _, userID := range allUserIDs {
    go sendNotification(userID, message)
}
```

**Consideraciones de Performance:**
- **Large Datasets**: Puede retornar muchos IDs
- **Memory Usage**: Considerar paginación para datasets grandes
- **Caching**: Candidato para caching si se usa frecuentemente

## Interface RoomsRepository

```go
type RoomsRepository interface {
    UserFetcher
    CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error)
    GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error)
    GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error)
    GetRoomListDeleted(ctx context.Context, userId int, since string) ([]string, error)
    LeaveRoom(ctx context.Context, userId int, roomId string, participants []int32, leaveAll bool) ([]User, error)
    DeleteRoom(ctx context.Context, userId int, roomId string, partner *int) error
    GetRoomParticipants(ctx context.Context, pagination *chatv1.GetRoomParticipantsRequest) ([]*chatv1.RoomParticipant, *chatv1.PaginationMeta, error)
    PinRoom(ctx context.Context, userId int, roomId string, pin bool) error
    MuteRoom(ctx context.Context, userId int, roomId string, mute bool) error
    BlockUser(ctx context.Context, userId int, roomId string, block bool, partner *int) error
    UpdateRoom(ctx context.Context, userId int, roomId string, room *chatv1.UpdateRoomRequest) error
    AddParticipantToRoom(ctx context.Context, userId int, roomId string, participants []int) ([]User, error)
    UpdateParticipantRoom(ctx context.Context, userId int, req *chatv1.UpdateParticipantRoomRequest) error
    SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error)
    GetMessage(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error)
    GetMessageSimple(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error)
    UpdateMessage(ctx context.Context, userId int, messageId string, content string) error
    DeleteMessage(ctx context.Context, userId int, messageId []string) error
    ReactToMessage(ctx context.Context, userId int, messageId string, reaction string) error
    GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error)
    MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error)
    GetMessageRead(ctx context.Context, req *chatv1.GetMessageReadRequest) ([]*chatv1.MessageUserRead, *chatv1.PaginationMeta, error)
    GetMessageReactions(ctx context.Context, req *chatv1.GetMessageReactionsRequest) ([]*chatv1.Reaction, *chatv1.PaginationMeta, error)
    GetUserByID(ctx context.Context, id int) (*User, error)
    GetAllUserIDs(ctx context.Context) ([]int, error)
    GetMessageSender(ctx context.Context, userId int, senderMessageId string) (*chatv1.MessageData, error)
    CreateMessageMetaForParticipants(ctx context.Context, roomID string, messageID string, senderID int) error
    IsPartnerMuted(ctx context.Context, userId int, roomId string) (bool, error)
}
```

**Análisis de la Interface:**

### Composición de Interfaces
```go
type RoomsRepository interface {
    UserFetcher
    // ... otros métodos
}
```

**Ventajas:**
- **Reutilización**: Reutiliza funcionalidad de UserFetcher
- **Cohesión**: Agrupa funcionalidades relacionadas
- **Flexibilidad**: Permite implementaciones parciales

### Categorización de Métodos

#### Gestión de Salas

##### `CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error)`
- **Propósito**: Crear nueva sala de chat
- **Autorización**: userId debe tener permisos para crear
- **Retorno**: Sala creada con metadatos completos

##### `GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error)`
- **Propósito**: Obtener información de una sala específica
- **Parámetros especiales**:
  - **`allData bool`**: Si incluir todos los datos o solo básicos
  - **`cache bool`**: Si usar cache o ir directo a DB
- **Autorización**: userId debe ser miembro de la sala

##### `GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error)`
- **Propósito**: Listar salas del usuario con paginación
- **Filtrado**: Solo salas donde userId es miembro
- **Paginación**: Soporte completo para paginación

##### `DeleteRoom(ctx context.Context, userId int, roomId string, partner *int) error`
- **Propósito**: Eliminar sala completamente
- **Autorización**: Solo owner puede eliminar
- **Partner**: Para salas P2P, especifica el partner

#### Gestión de Participantes

##### `GetRoomParticipants(ctx context.Context, pagination *chatv1.GetRoomParticipantsRequest) ([]*chatv1.RoomParticipant, *chatv1.PaginationMeta, error)`
- **Propósito**: Listar participantes de una sala
- **Paginación**: Soporte para salas con muchos miembros

##### `AddParticipantToRoom(ctx context.Context, userId int, roomId string, participants []int) ([]User, error)`
- **Propósito**: Agregar nuevos participantes a la sala
- **Batch**: Permite agregar múltiples usuarios
- **Retorno**: Información de usuarios agregados

##### `LeaveRoom(ctx context.Context, userId int, roomId string, participants []int32, leaveAll bool) ([]User, error)`
- **Propósito**: Remover participantes de la sala
- **Flexible**: Puede remover uno o todos los participantes
- **Retorno**: Información de usuarios removidos

#### Gestión de Mensajes

##### `SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error)`
- **Propósito**: Guardar nuevo mensaje en la sala
- **Encriptación**: Recibe contenido desencriptado para indexing
- **Contexto**: Incluye información completa de la sala

##### `GetMessage(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error)`
- **Propósito**: Obtener mensaje específico con todos los datos
- **Autorización**: Usuario debe tener acceso al mensaje

##### `GetMessageSimple(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error)`
- **Propósito**: Obtener mensaje con datos básicos (optimizado)
- **Performance**: Versión optimizada para casos simples

##### `GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error)`
- **Propósito**: Obtener historial de mensajes de una sala
- **Paginación**: Soporte para historial largo
- **Filtrado**: Múltiples opciones de filtrado

#### Gestión de Estados

##### `MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error)`
- **Propósito**: Marcar mensajes como leídos
- **Batch**: Permite marcar múltiples mensajes
- **Timestamp**: Opción de marcar desde cierto tiempo

##### `GetMessageRead(ctx context.Context, req *chatv1.GetMessageReadRequest) ([]*chatv1.MessageUserRead, *chatv1.PaginationMeta, error)`
- **Propósito**: Obtener información de lectura de mensajes
- **Uso**: Para mostrar "visto por" en grupos

#### Funcionalidades Adicionales

##### `PinRoom(ctx context.Context, userId int, roomId string, pin bool) error`
- **Propósito**: Fijar/desfijar sala en la lista del usuario

##### `MuteRoom(ctx context.Context, userId int, roomId string, mute bool) error`
- **Propósito**: Silenciar/activar notificaciones de la sala

##### `BlockUser(ctx context.Context, userId int, roomId string, block bool, partner *int) error`
- **Propósito**: Bloquear/desbloquear usuario en chat P2P

##### `ReactToMessage(ctx context.Context, userId int, messageId string, reaction string) error`
- **Propósito**: Agregar reacción emoji a un mensaje

## Patrones de Diseño Implementados

### 1. Repository Pattern
- **Abstracción**: Separa lógica de negocio de persistencia
- **Flexibilidad**: Permite múltiples implementaciones
- **Testing**: Facilita mocking y testing unitario

### 2. Interface Segregation
- **UserFetcher**: Interface específica para operaciones de usuario
- **RoomsRepository**: Interface completa que compone UserFetcher
- **Principio**: Clientes no dependen de métodos que no usan

### 3. Dependency Injection
- **Context**: Inyección de contexto para control de lifecycle
- **Parameters**: Inyección de parámetros específicos por operación

### 4. Command Query Separation
- **Commands**: Métodos que modifican estado (Create, Update, Delete)
- **Queries**: Métodos que solo leen datos (Get, List)
- **Claridad**: Separación clara de responsabilidades

## Consideraciones de Performance

### 1. Batch Operations
```go
GetUsersByID(ctx context.Context, ids []int) ([]User, error)
AddParticipantToRoom(ctx context.Context, userId int, roomId string, participants []int) ([]User, error)
```
- **Eficiencia**: Operaciones en lote para reducir round-trips
- **Atomicidad**: Operaciones atómicas cuando es posible

### 2. Caching Support
```go
GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error)
```
- **Flexibilidad**: Opción de usar cache o ir directo a DB
- **Performance**: Mejora significativa para datos frecuentemente accedidos

### 3. Pagination
```go
GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error)
```
- **Escalabilidad**: Manejo de datasets grandes
- **Memory**: Evita cargar todos los datos en memoria

## Testing

### Mock Implementation

```go
type MockRoomsRepository struct {
    mock.Mock
}

func (m *MockRoomsRepository) GetUserByID(ctx context.Context, id int) (*User, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockRoomsRepository) CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error) {
    args := m.Called(ctx, userId, room)
    return args.Get(0).(*chatv1.Room), args.Error(1)
}

// ... implementar todos los métodos
```

### Test Examples

```go
func TestGetUserByID(t *testing.T) {
    mockRepo := &MockRoomsRepository{}
    expectedUser := &User{
        ID:    123,
        Name:  "John Doe",
        Phone: "+1234567890",
    }
    
    mockRepo.On("GetUserByID", mock.Anything, 123).Return(expectedUser, nil)
    
    user, err := mockRepo.GetUserByID(context.Background(), 123)
    
    assert.NoError(t, err)
    assert.Equal(t, expectedUser, user)
    mockRepo.AssertExpectations(t)
}
```

## Mejores Prácticas Implementadas

1. **Context Propagation**: Todos los métodos reciben context
2. **Error Handling**: Retorno explícito de errores
3. **Type Safety**: Uso de tipos generados desde protobuf
4. **Nullable Fields**: Uso correcto de punteros para campos opcionales
5. **Batch Operations**: Soporte para operaciones en lote
6. **Pagination**: Soporte consistente para paginación
7. **Authorization**: UserID en todos los métodos para autorización

Este archivo define la base arquitectónica para el acceso a datos del sistema de chat, proporcionando una abstracción limpia y completa que facilita el mantenimiento, testing y evolución del sistema.