# Documentación Técnica: utils/format.go

## Descripción General

El archivo `format.go` contiene funciones utilitarias para el formateo y transformación de datos específicos del dominio de chat. Su función principal es la normalización y enriquecimiento de objetos `Room` para presentación consistente en la API, aplicando lógica de negocio específica según el tipo de sala.

## Estructura del Archivo

### Importaciones

```go
import (
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
)
```

**Análisis de Importaciones:**

- **`chatv1`**: Tipos generados desde Protocol Buffers para el servicio de chat
- **Alias**: `chatv1` evita conflictos de nombres y mejora legibilidad
- **Dependencia única**: Función específica del dominio, sin dependencias externas

## Función FormatRoom

```go
func FormatRoom(room *chatv1.Room) *chatv1.Room {
    if room.Type == "p2p" && room.Partner != nil {
        room.PhotoUrl = room.Partner.Avatar
        room.Name = room.Partner.Name
    }
    if room.LastMessage != nil {
        room.LastMessageAt = room.LastMessage.CreatedAt
    }
    return room
}
```

### Análisis Detallado

#### Signature de la Función

```go
func FormatRoom(room *chatv1.Room) *chatv1.Room
```

**Características:**
- **Input**: Puntero a `chatv1.Room` (modificación in-place)
- **Output**: Mismo puntero (fluent interface pattern)
- **Mutabilidad**: Modifica el objeto original
- **Performance**: Evita copias innecesarias de estructuras grandes

#### Lógica de Formateo para Salas P2P

```go
if room.Type == "p2p" && room.Partner != nil {
    room.PhotoUrl = room.Partner.Avatar
    room.Name = room.Partner.Name
}
```

**Análisis del Bloque:**

##### Condición de Entrada
- **`room.Type == "p2p"`**: Verifica que sea una sala persona-a-persona
- **`room.Partner != nil`**: Asegura que existe información del partner
- **Lógica AND**: Ambas condiciones deben cumplirse

##### Transformaciones Aplicadas

**Asignación de Foto:**
```go
room.PhotoUrl = room.Partner.Avatar
```
- **Propósito**: En chats P2P, la foto de la sala es la foto del partner
- **Lógica de negocio**: La representación visual de la sala es el avatar del otro usuario
- **Fallback**: Si `Partner.Avatar` es nil/empty, `PhotoUrl` será nil/empty

**Asignación de Nombre:**
```go
room.Name = room.Partner.Name
```
- **Propósito**: En chats P2P, el nombre de la sala es el nombre del partner
- **Experiencia de usuario**: El usuario ve el nombre del contacto, no un nombre genérico
- **Consistencia**: Mantiene coherencia con aplicaciones de chat estándar

#### Lógica de Timestamp del Último Mensaje

```go
if room.LastMessage != nil {
    room.LastMessageAt = room.LastMessage.CreatedAt
}
```

**Análisis del Bloque:**

##### Condición de Entrada
- **`room.LastMessage != nil`**: Verifica que existe un último mensaje
- **Protección**: Evita null pointer exceptions

##### Transformación Aplicada
- **`room.LastMessageAt = room.LastMessage.CreatedAt`**: Copia timestamp del último mensaje
- **Propósito**: Facilita ordenamiento y filtrado de salas por actividad reciente
- **Desnormalización**: Evita joins complejos en queries de listado de salas

## Tipos de Salas y Comportamiento

### Salas Persona-a-Persona (P2P)

**Características:**
- **Tipo**: `room.Type == "p2p"`
- **Participantes**: Exactamente 2 usuarios
- **Nombre**: Dinámico basado en el partner
- **Foto**: Avatar del partner
- **Creación**: Automática al enviar primer mensaje

**Estructura típica antes del formateo:**
```go
room := &chatv1.Room{
    Id:       "room_123",
    Type:     "p2p",
    Name:     "",           // Vacío inicialmente
    PhotoUrl: "",           // Vacío inicialmente
    Partner: &chatv1.User{
        Id:     456,
        Name:   "Juan Pérez",
        Avatar: "https://example.com/avatar.jpg",
    },
    LastMessage: &chatv1.MessageData{
        Id:        "msg_789",
        Content:   "Hola!",
        CreatedAt: "2024-01-15T10:30:00Z",
    },
}
```

**Estructura después del formateo:**
```go
room := &chatv1.Room{
    Id:            "room_123",
    Type:          "p2p",
    Name:          "Juan Pérez",                    // ← Asignado desde Partner.Name
    PhotoUrl:      "https://example.com/avatar.jpg", // ← Asignado desde Partner.Avatar
    LastMessageAt: "2024-01-15T10:30:00Z",          // ← Asignado desde LastMessage.CreatedAt
    Partner: &chatv1.User{
        Id:     456,
        Name:   "Juan Pérez",
        Avatar: "https://example.com/avatar.jpg",
    },
    LastMessage: &chatv1.MessageData{
        Id:        "msg_789",
        Content:   "Hola!",
        CreatedAt: "2024-01-15T10:30:00Z",
    },
}
```

### Salas Grupales

**Características:**
- **Tipo**: `room.Type == "group"`
- **Participantes**: 3 o más usuarios
- **Nombre**: Estático, definido al crear el grupo
- **Foto**: Imagen del grupo o por defecto
- **Administración**: Roles de admin/miembro

**Comportamiento en FormatRoom:**
- **Nombre**: No se modifica (mantiene el nombre del grupo)
- **Foto**: No se modifica (mantiene la foto del grupo)
- **LastMessageAt**: Se actualiza igual que en P2P

## Casos de Uso en la Aplicación

### En Handlers de Salas

```go
func (h *handlerImpl) GetRooms(ctx context.Context, req *connect.Request[chatv1.GetRoomsRequest]) (*connect.Response[chatv1.GetRoomsResponse], error) {
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        return nil, err
    }
    
    rooms, meta, err := h.roomsRepository.GetRoomList(ctx, userID, req.Msg)
    if err != nil {
        return nil, err
    }
    
    // Formatear cada sala antes de retornar
    for i, room := range rooms {
        rooms[i] = utils.FormatRoom(room)
    }
    
    return connect.NewResponse(&chatv1.GetRoomsResponse{
        Items: rooms,
        Meta:  meta,
    }), nil
}
```

### En Handlers de Sala Individual

```go
func (h *handlerImpl) GetRoom(ctx context.Context, req *connect.Request[chatv1.GetRoomRequest]) (*connect.Response[chatv1.GetRoomResponse], error) {
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        return nil, err
    }
    
    room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, true, false)
    if err != nil {
        return nil, err
    }
    
    if room == nil {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
    }
    
    // Formatear sala antes de retornar
    room = utils.FormatRoom(room)
    
    return connect.NewResponse(&chatv1.GetRoomResponse{
        Success: true,
        Room:    room,
    }), nil
}
```

### En Creación de Salas

```go
func (h *handlerImpl) CreateRoom(ctx context.Context, req *connect.Request[chatv1.CreateRoomRequest]) (*connect.Response[chatv1.CreateRoomResponse], error) {
    // ... lógica de validación y creación
    
    room, err := h.roomsRepository.CreateRoom(ctx, userID, req.Msg)
    if err != nil {
        return nil, err
    }
    
    // Formatear sala recién creada
    room = utils.FormatRoom(room)
    
    return connect.NewResponse(&chatv1.CreateRoomResponse{
        Success: true,
        Room:    room,
    }), nil
}
```

## Patrones de Diseño Implementados

### 1. Transformer Pattern

**Implementación:**
```go
func FormatRoom(room *chatv1.Room) *chatv1.Room {
    // Transformaciones específicas del dominio
    return room
}
```

**Características:**
- **Input/Output del mismo tipo**: Transformación in-place
- **Lógica de dominio**: Reglas específicas del negocio
- **Reutilizable**: Aplicable en múltiples contextos

### 2. Fluent Interface

**Uso:**
```go
room = utils.FormatRoom(room)
// O encadenado (si se extendiera):
// room = utils.FormatRoom(room).ValidateRoom().EnrichRoom()
```

### 3. Null Object Pattern (Implícito)

**Protección contra nulos:**
```go
if room.Partner != nil {
    // Solo procesar si existe
}
if room.LastMessage != nil {
    // Solo procesar si existe
}
```

## Consideraciones de Performance

### 1. Modificación In-Place

**Ventajas:**
- **Memoria**: No crea copias adicionales
- **Performance**: Operaciones O(1)
- **Cache**: Mejor localidad de datos

**Desventajas:**
- **Mutabilidad**: Modifica el objeto original
- **Side effects**: Puede afectar otros usuarios del objeto

### 2. Optimización para Listas

```go
// Procesamiento eficiente de múltiples salas
for i, room := range rooms {
    rooms[i] = utils.FormatRoom(room) // Modifica in-place
}
```

## Testing

### Unit Tests

```go
func TestFormatRoom(t *testing.T) {
    tests := []struct {
        name     string
        input    *chatv1.Room
        expected *chatv1.Room
    }{
        {
            name: "P2P room with partner",
            input: &chatv1.Room{
                Type: "p2p",
                Partner: &chatv1.User{
                    Name:   "John Doe",
                    Avatar: "avatar.jpg",
                },
                LastMessage: &chatv1.MessageData{
                    CreatedAt: "2024-01-01T00:00:00Z",
                },
            },
            expected: &chatv1.Room{
                Type:          "p2p",
                Name:          "John Doe",
                PhotoUrl:      "avatar.jpg",
                LastMessageAt: "2024-01-01T00:00:00Z",
                Partner: &chatv1.User{
                    Name:   "John Doe",
                    Avatar: "avatar.jpg",
                },
                LastMessage: &chatv1.MessageData{
                    CreatedAt: "2024-01-01T00:00:00Z",
                },
            },
        },
        {
            name: "Group room unchanged",
            input: &chatv1.Room{
                Type:     "group",
                Name:     "My Group",
                PhotoUrl: "group.jpg",
            },
            expected: &chatv1.Room{
                Type:     "group",
                Name:     "My Group",
                PhotoUrl: "group.jpg",
            },
        },
        {
            name: "P2P room without partner",
            input: &chatv1.Room{
                Type:    "p2p",
                Partner: nil,
            },
            expected: &chatv1.Room{
                Type:    "p2p",
                Partner: nil,
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FormatRoom(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

```go
func TestFormatRoomIntegration(t *testing.T) {
    // Test con datos reales de la base de datos
    room := &chatv1.Room{
        Id:   "real_room_id",
        Type: "p2p",
        Partner: &chatv1.User{
            Id:     123,
            Name:   "Real User",
            Avatar: "real_avatar.jpg",
        },
    }
    
    formatted := utils.FormatRoom(room)
    
    assert.Equal(t, "Real User", formatted.Name)
    assert.Equal(t, "real_avatar.jpg", formatted.PhotoUrl)
}
```

## Extensibilidad

### Agregar Nuevos Tipos de Sala

```go
func FormatRoom(room *chatv1.Room) *chatv1.Room {
    switch room.Type {
    case "p2p":
        if room.Partner != nil {
            room.PhotoUrl = room.Partner.Avatar
            room.Name = room.Partner.Name
        }
    case "group":
        // Lógica específica para grupos
        formatGroupRoom(room)
    case "channel":
        // Lógica específica para canales
        formatChannelRoom(room)
    case "broadcast":
        // Lógica específica para broadcasts
        formatBroadcastRoom(room)
    }
    
    if room.LastMessage != nil {
        room.LastMessageAt = room.LastMessage.CreatedAt
    }
    
    return room
}
```

### Formateo Condicional

```go
func FormatRoom(room *chatv1.Room, options FormatOptions) *chatv1.Room {
    if options.IncludePartnerInfo && room.Type == "p2p" && room.Partner != nil {
        room.PhotoUrl = room.Partner.Avatar
        room.Name = room.Partner.Name
    }
    
    if options.IncludeTimestamps && room.LastMessage != nil {
        room.LastMessageAt = room.LastMessage.CreatedAt
    }
    
    return room
}
```

## Mejores Prácticas Implementadas

1. **Single Responsibility**: Función específica para formateo de salas
2. **Null Safety**: Verificaciones antes de acceder a punteros
3. **Immutable Input**: No modifica la estructura de datos, solo los valores
4. **Performance**: Modificación in-place para eficiencia
5. **Type Safety**: Uso de tipos generados desde protobuf
6. **Domain Logic**: Encapsula reglas de negocio específicas del chat

Este archivo, aunque simple, es crucial para la presentación consistente de datos en la API, asegurando que las salas se muestren correctamente según su tipo y contexto.