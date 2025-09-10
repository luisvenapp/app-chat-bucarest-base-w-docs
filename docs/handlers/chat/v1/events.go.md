# Documentación Técnica: handlers/chat/v1/events.go

## Descripción General

El archivo `events.go` define la estructura y comportamiento de los eventos de chat que se publican a través del sistema de mensajería NATS. Implementa el patrón Event-Driven Architecture, proporcionando una abstracción para la distribución de eventos en tiempo real entre diferentes componentes del sistema.

## Estructura del Archivo

### Importaciones

```go
import (
    "encoding/json"
    
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
    "google.golang.org/protobuf/proto"
)
```

**Análisis de Importaciones:**

- **`encoding/json`**: Serialización JSON para el payload del evento
- **`chatv1`**: Tipos de Protocol Buffers para eventos de chat
- **`proto`**: Utilidades de Protocol Buffers para marshaling

## Estructura ChatEvent

```go
type ChatEvent struct {
    roomID string
    userID int
    event  *chatv1.MessageEvent
}
```

**Análisis de Campos:**

### `roomID string`
- **Propósito**: Identificador de la sala donde ocurre el evento
- **Uso**: Determina el subject NATS para routing
- **Formato**: String único por sala (UUID o similar)

### `userID int`
- **Propósito**: Identificador del usuario que dispara el evento
- **Uso**: Tracking, autorización y routing de eventos directos
- **Contexto**: Usuario que realiza la acción que genera el evento

### `event *chatv1.MessageEvent`
- **Propósito**: Datos específicos del evento de chat
- **Tipo**: Estructura de Protocol Buffers
- **Contenido**: Mensaje, estado, metadatos, etc.

## Métodos de ChatEvent

### Método Subject

```go
func (e ChatEvent) Subject() string {
    switch detail := e.event.Event.(type) {
    case *chatv1.MessageEvent_RoomJoin:
        return chatDirectEventSubject(int(detail.RoomJoin.UserId))
    default:
        return chatRoomEventSubject(e.roomID)
    }
}
```

**Análisis Detallado:**

#### Lógica de Routing
El método implementa una lógica de routing inteligente basada en el tipo de evento:

##### Eventos de Unión a Sala
```go
case *chatv1.MessageEvent_RoomJoin:
    return chatDirectEventSubject(int(detail.RoomJoin.UserId))
```

**Características:**
- **Subject directo**: Eventos van directamente al usuario que se une
- **Propósito**: Notificar al usuario específico sobre su unión
- **Formato**: `CHAT_DIRECT_EVENTS.{userID}`
- **Uso**: Actualizar UI del usuario que se unió

##### Eventos Generales de Sala
```go
default:
    return chatRoomEventSubject(e.roomID)
```

**Características:**
- **Subject de sala**: Eventos van a todos los miembros de la sala
- **Propósito**: Broadcast a todos los participantes
- **Formato**: `CHAT_EVENTS.{roomID}`
- **Uso**: Mensajes, actualizaciones de estado, etc.

#### Patrones de Subject

**Subject Directo:**
```
CHAT_DIRECT_EVENTS.123
```
- **Destinatario**: Usuario específico (ID: 123)
- **Uso**: Eventos personales, notificaciones directas

**Subject de Sala:**
```
CHAT_EVENTS.room_456
```
- **Destinatarios**: Todos los miembros de la sala (ID: room_456)
- **Uso**: Mensajes, eventos de sala

### Método JetStream

```go
func (ChatEvent) JetStream() bool {
    return true
}
```

**Análisis:**
- **Persistencia**: Indica que el evento debe ser persistido
- **Garantías**: Asegura entrega y posibilidad de replay
- **Durabilidad**: Eventos sobreviven a reinicios del sistema
- **Uso**: Crítico para eventos de chat que no pueden perderse

### Método EventType

```go
func (e ChatEvent) EventType() string {
    return "ChatEvent"
}
```

**Análisis:**
- **Identificación**: Tipo de evento para clasificación
- **Routing**: Usado por el sistema de eventos para routing
- **Monitoring**: Facilita métricas y observabilidad
- **Debugging**: Ayuda en logs y troubleshooting

## Estructura eventPayload

```go
type eventPayload struct {
    UserId  int    `json:"user_id"`
    Payload []byte `json:"payload"`
}
```

**Análisis de Campos:**

### `UserId int`
- **Propósito**: Identificador del usuario que origina el evento
- **Serialización**: Campo JSON para compatibilidad
- **Uso**: Tracking, autorización, filtering

### `Payload []byte`
- **Propósito**: Datos serializados del evento en Protocol Buffers
- **Formato**: Bytes para eficiencia de transmisión
- **Contenido**: `chatv1.MessageEvent` serializado

### Método Payload

```go
func (e ChatEvent) Payload() ([]byte, error) {
    payload, err := proto.Marshal(e.event)
    if err != nil {
        return nil, err
    }
    return json.Marshal(eventPayload{
        UserId:  e.userID,
        Payload: payload,
    })
}
```

**Análisis del Proceso:**

#### 1. Serialización de Protocol Buffers
```go
payload, err := proto.Marshal(e.event)
```
- **Eficiencia**: Protocol Buffers es más eficiente que JSON
- **Compatibilidad**: Mantiene compatibilidad entre versiones
- **Tamaño**: Menor tamaño de payload para transmisión

#### 2. Encapsulación en JSON
```go
return json.Marshal(eventPayload{
    UserId:  e.userID,
    Payload: payload,
})
```
- **Metadatos**: Incluye información del usuario
- **Estructura**: JSON para compatibilidad con sistemas externos
- **Debugging**: JSON es más fácil de inspeccionar

#### Estructura Final del Payload
```json
{
    "user_id": 123,
    "payload": "CgQIARIAGgA..."  // Base64 encoded protobuf
}
```

## Tipos de Eventos Soportados

### Eventos de Mensajes

#### MessageEvent_Message
- **Propósito**: Nuevo mensaje en la sala
- **Subject**: `CHAT_EVENTS.{roomID}`
- **Destinatarios**: Todos los miembros de la sala
- **Contenido**: Mensaje completo con metadatos

#### MessageEvent_StatusUpdate
- **Propósito**: Actualización de estado del mensaje (enviado, entregado, leído)
- **Subject**: `CHAT_EVENTS.{roomID}`
- **Destinatarios**: Remitente y destinatarios
- **Contenido**: ID del mensaje y nuevo estado

### Eventos de Sala

#### MessageEvent_RoomJoin
- **Propósito**: Usuario se une a la sala
- **Subject**: `CHAT_DIRECT_EVENTS.{userID}` (para el usuario que se une)
- **Destinatario**: Usuario específico
- **Contenido**: Información de la sala y rol asignado

#### MessageEvent_RoomLeave
- **Propósito**: Usuario abandona la sala
- **Subject**: `CHAT_EVENTS.{roomID}`
- **Destinatarios**: Todos los miembros restantes
- **Contenido**: Lista de usuarios que abandonaron

#### MessageEvent_IsRoomUpdated
- **Propósito**: Metadatos de la sala fueron actualizados
- **Subject**: `CHAT_EVENTS.{roomID}`
- **Destinatarios**: Todos los miembros de la sala
- **Contenido**: Indicador de actualización (la sala se obtiene por separado)

### Eventos de Gestión

#### MessageEvent_UpdateMessage
- **Propósito**: Mensaje fue editado
- **Subject**: `CHAT_EVENTS.{roomID}`
- **Destinatarios**: Todos los miembros de la sala
- **Contenido**: Mensaje actualizado con flag de editado

#### MessageEvent_DeleteMessage
- **Propósito**: Mensaje fue eliminado
- **Subject**: `CHAT_EVENTS.{roomID}`
- **Destinatarios**: Todos los miembros de la sala
- **Contenido**: ID del mensaje eliminado

## Flujo de Eventos

### 1. Creación del Evento
```go
event := ChatEvent{
    roomID: "room_123",
    userID: 456,
    event: &chatv1.MessageEvent{
        EventId: uuid.NewString(),
        RoomId:  "room_123",
        Event: &chatv1.MessageEvent_Message{
            Message: messageData,
        },
    },
}
```

### 2. Determinación del Subject
```go
subject := event.Subject() // "CHAT_EVENTS.room_123"
```

### 3. Serialización del Payload
```go
payload, err := event.Payload()
// Resultado: JSON con userID y protobuf serializado
```

### 4. Publicación en NATS
```go
dispatcher.Dispatch(context.Background(), event)
```

### 5. Consumo por Clientes
```go
// Los clientes suscritos al subject reciben el evento
// Deserializan el payload y procesan el evento
```

## Patrones de Diseño Implementados

### 1. Event-Driven Architecture
- **Desacoplamiento**: Productores y consumidores independientes
- **Escalabilidad**: Múltiples consumidores por evento
- **Flexibilidad**: Fácil agregar nuevos tipos de eventos

### 2. Strategy Pattern (Implícito)
- **Subject Selection**: Diferentes estrategias según tipo de evento
- **Routing**: Lógica de routing encapsulada en el método Subject

### 3. Serialization Strategy
- **Protocol Buffers**: Para eficiencia y compatibilidad
- **JSON Wrapper**: Para metadatos y debugging
- **Hybrid Approach**: Combina ventajas de ambos formatos

## Consideraciones de Performance

### 1. Serialización Eficiente
- **Protocol Buffers**: Menor overhead que JSON puro
- **Lazy Serialization**: Solo se serializa cuando se necesita
- **Caching**: Posibilidad de cachear payloads serializados

### 2. Routing Inteligente
- **Subject Específicos**: Evita broadcast innecesario
- **Filtering**: Eventos van solo a destinatarios relevantes
- **Load Distribution**: Distribución de carga entre consumers

### 3. Memory Management
- **Struct Pequeño**: ChatEvent es liviano
- **Reference Sharing**: Comparte referencias cuando es posible
- **Garbage Collection**: Estructuras diseñadas para GC eficiente

## Testing

### Unit Tests

```go
func TestChatEventSubject(t *testing.T) {
    tests := []struct {
        name     string
        event    ChatEvent
        expected string
    }{
        {
            name: "room join event",
            event: ChatEvent{
                roomID: "room_123",
                userID: 456,
                event: &chatv1.MessageEvent{
                    Event: &chatv1.MessageEvent_RoomJoin{
                        RoomJoin: &chatv1.RoomJoinEvent{
                            UserId: 789,
                        },
                    },
                },
            },
            expected: "CHAT_DIRECT_EVENTS.789",
        },
        {
            name: "message event",
            event: ChatEvent{
                roomID: "room_123",
                userID: 456,
                event: &chatv1.MessageEvent{
                    Event: &chatv1.MessageEvent_Message{
                        Message: &chatv1.MessageData{},
                    },
                },
            },
            expected: "CHAT_EVENTS.room_123",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.event.Subject()
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestChatEventPayload(t *testing.T) {
    event := ChatEvent{
        roomID: "room_123",
        userID: 456,
        event: &chatv1.MessageEvent{
            EventId: "event_123",
            RoomId:  "room_123",
        },
    }
    
    payload, err := event.Payload()
    require.NoError(t, err)
    
    var eventPayload eventPayload
    err = json.Unmarshal(payload, &eventPayload)
    require.NoError(t, err)
    
    assert.Equal(t, 456, eventPayload.UserId)
    assert.NotEmpty(t, eventPayload.Payload)
    
    // Verificar que se puede deserializar el protobuf
    var messageEvent chatv1.MessageEvent
    err = proto.Unmarshal(eventPayload.Payload, &messageEvent)
    require.NoError(t, err)
    assert.Equal(t, "event_123", messageEvent.EventId)
}
```

## Monitoreo y Observabilidad

### Métricas Recomendadas

```go
// Contador de eventos por tipo
eventCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "chat_events_total",
        Help: "Total number of chat events by type",
    },
    []string{"event_type", "room_type"},
)

// Tamaño de payload
payloadSizeHistogram := prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "chat_event_payload_size_bytes",
        Help: "Size of chat event payloads in bytes",
    },
    []string{"event_type"},
)
```

### Logging Estructurado

```go
func (e ChatEvent) LogFields() []slog.Attr {
    return []slog.Attr{
        slog.String("event_type", e.EventType()),
        slog.String("room_id", e.roomID),
        slog.Int("user_id", e.userID),
        slog.String("subject", e.Subject()),
    }
}
```

## Mejores Prácticas Implementadas

1. **Type Safety**: Uso de tipos específicos para eventos
2. **Immutability**: Estructuras inmutables después de creación
3. **Error Handling**: Manejo explícito de errores de serialización
4. **Performance**: Serialización eficiente con Protocol Buffers
5. **Observability**: Métodos que facilitan logging y métricas
6. **Extensibility**: Fácil agregar nuevos tipos de eventos
7. **Separation of Concerns**: Lógica de routing separada de datos

Este archivo es fundamental para la arquitectura event-driven del sistema de chat, proporcionando una abstracción limpia y eficiente para la distribución de eventos en tiempo real.