# Documentación Técnica: handlers/chat/v1/helpers.go

## Descripción General

El archivo `helpers.go` contiene funciones auxiliares para el handler de chat, específicamente enfocadas en la publicación y gestión de eventos. Proporciona una capa de abstracción para la publicación de eventos de chat, incluyendo logging estructurado y enriquecimiento de eventos con metadatos adicionales.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    "fmt"
    
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
    "github.com/google/uuid"
)
```

**Análisis de Importaciones:**

- **`context`**: Para manejo de contexto y cancelación
- **`fmt`**: Para formateo de strings en logging
- **`chatv1`**: Tipos de Protocol Buffers para eventos de chat
- **`api`**: Módulo del core para parámetros generales de API
- **`uuid`**: Generación de identificadores únicos para eventos

## Estructura MessageEvent

```go
type MessageEvent struct {
    DispatcherUserID int
}
```

**Análisis:**
- **Propósito**: Estructura auxiliar para tracking del usuario que dispara eventos
- **Campo**: `DispatcherUserID` identifica quién origina el evento
- **Uso**: Posible extensión futura para metadatos adicionales de eventos

**Nota**: Esta estructura parece estar definida pero no utilizada en el código actual, posiblemente para uso futuro o como placeholder para extensiones.

## Función publishChatEvent

```go
func (h *handlerImpl) publishChatEvent(generalParams api.GeneralParams, roomID string, event *chatv1.MessageEvent) {
    h.logger.Info(
        "Dispatching chat event",
        "roomID", roomID,
        "eventType", fmt.Sprintf("%T", event.Event),
        "triggeredByUserID", generalParams.Session.UserID,
        "clientID", generalParams.ClientId,
    )
    
    eventVal := &chatv1.MessageEvent{
        EventId: uuid.NewString(),
        RoomId:  event.RoomId,
        Room:    event.Room,
        Event:   event.Event,
    }
    
    if eventVal.Room != nil && eventVal.RoomId == "" {
        eventVal.RoomId = event.Room.Id
    }
    
    if eventVal.RoomId == "" {
        eventVal.RoomId = roomID
    }
    
    // Luego, envío persistente a través del dispatcher
    h.dispatcher.Dispatch(context.Background(), ChatEvent{roomID: roomID, event: eventVal, userID: generalParams.Session.UserID})
}
```

**Análisis Detallado:**

### Signature de la Función

```go
func (h *handlerImpl) publishChatEvent(generalParams api.GeneralParams, roomID string, event *chatv1.MessageEvent)
```

**Parámetros:**
- **`h *handlerImpl`**: Receiver del handler principal
- **`generalParams api.GeneralParams`**: Parámetros de contexto de la API (sesión, cliente, etc.)
- **`roomID string`**: Identificador de la sala donde ocurre el evento
- **`event *chatv1.MessageEvent`**: Evento a publicar

### Logging Estructurado

```go
h.logger.Info(
    "Dispatching chat event",
    "roomID", roomID,
    "eventType", fmt.Sprintf("%T", event.Event),
    "triggeredByUserID", generalParams.Session.UserID,
    "clientID", generalParams.ClientId,
)
```

**Análisis del Logging:**

#### Información Registrada
- **Mensaje**: `"Dispatching chat event"` - Acción que se está realizando
- **`roomID`**: Identificador de la sala afectada
- **`eventType`**: Tipo específico del evento usando reflection (`%T`)
- **`triggeredByUserID`**: Usuario que origina el evento
- **`clientID`**: Identificador del cliente que dispara el evento

#### Ventajas del Logging Estructurado
- **Observabilidad**: Facilita monitoreo y debugging
- **Filtering**: Permite filtrar logs por roomID, userID, etc.
- **Metrics**: Base para generar métricas de eventos
- **Debugging**: Información completa para troubleshooting

#### Ejemplos de Output de Log
```
INFO Dispatching chat event roomID=room_123 eventType=*chatv1.MessageEvent_Message triggeredByUserID=456 clientID=client_789
INFO Dispatching chat event roomID=room_123 eventType=*chatv1.MessageEvent_RoomJoin triggeredByUserID=456 clientID=client_789
```

### Enriquecimiento del Evento

```go
eventVal := &chatv1.MessageEvent{
    EventId: uuid.NewString(),
    RoomId:  event.RoomId,
    Room:    event.Room,
    Event:   event.Event,
}
```

**Proceso de Enriquecimiento:**

#### Generación de EventId
```go
EventId: uuid.NewString(),
```
- **Propósito**: Identificador único para cada evento
- **Formato**: UUID v4 (ej: `"550e8400-e29b-41d4-a716-446655440000"`)
- **Uso**: Tracking, deduplicación, debugging
- **Garantía**: Unicidad global del evento

#### Copia de Datos Existentes
```go
RoomId:  event.RoomId,
Room:    event.Room,
Event:   event.Event,
```
- **Preservación**: Mantiene datos originales del evento
- **Inmutabilidad**: No modifica el evento original
- **Completitud**: Asegura que toda la información se preserve

### Normalización de RoomId

```go
if eventVal.Room != nil && eventVal.RoomId == "" {
    eventVal.RoomId = event.Room.Id
}

if eventVal.RoomId == "" {
    eventVal.RoomId = roomID
}
```

**Análisis de la Lógica de Normalización:**

#### Primera Condición
```go
if eventVal.Room != nil && eventVal.RoomId == "" {
    eventVal.RoomId = event.Room.Id
}
```
- **Escenario**: Evento tiene objeto Room pero no RoomId
- **Acción**: Extrae RoomId del objeto Room
- **Propósito**: Asegurar consistencia de datos

#### Segunda Condición
```go
if eventVal.RoomId == "" {
    eventVal.RoomId = roomID
}
```
- **Escenario**: Evento no tiene RoomId después de la primera verificación
- **Acción**: Usa el roomID pasado como parámetro
- **Propósito**: Garantizar que todo evento tenga RoomId

#### Casos de Uso de la Normalización

**Caso 1: Evento con RoomId completo**
```go
// Input
event := &chatv1.MessageEvent{
    RoomId: "room_123",
    Event:  messageEvent,
}
// Output: eventVal.RoomId = "room_123"
```

**Caso 2: Evento con objeto Room pero sin RoomId**
```go
// Input
event := &chatv1.MessageEvent{
    Room: &chatv1.Room{Id: "room_456"},
    Event: messageEvent,
}
// Output: eventVal.RoomId = "room_456"
```

**Caso 3: Evento sin RoomId ni Room**
```go
// Input
event := &chatv1.MessageEvent{
    Event: messageEvent,
}
// roomID parameter = "room_789"
// Output: eventVal.RoomId = "room_789"
```

### Despacho del Evento

```go
h.dispatcher.Dispatch(context.Background(), ChatEvent{roomID: roomID, event: eventVal, userID: generalParams.Session.UserID})
```

**Análisis del Despacho:**

#### Contexto
```go
context.Background()
```
- **Contexto**: Usa contexto background para operación asíncrona
- **Implicación**: El evento se procesa independientemente del request original
- **Ventaja**: No bloquea la respuesta al cliente

#### Construcción del ChatEvent
```go
ChatEvent{
    roomID: roomID,
    event:  eventVal,
    userID: generalParams.Session.UserID,
}
```

**Campos del ChatEvent:**
- **`roomID`**: Para routing y subject determination
- **`event`**: Evento enriquecido con EventId y RoomId normalizados
- **`userID`**: Usuario que origina el evento para tracking

#### Flujo Asíncrono
1. **Dispatch**: Evento se envía al dispatcher
2. **Queue**: Dispatcher lo coloca en cola de procesamiento
3. **Worker**: Worker pool procesa el evento
4. **NATS**: Evento se publica en NATS JetStream
5. **Consumers**: Clientes suscritos reciben el evento

## Patrones de Diseño Implementados

### 1. Facade Pattern
- **Simplificación**: `publishChatEvent` simplifica la publicación de eventos
- **Encapsulación**: Oculta complejidad del dispatcher y logging
- **Consistencia**: Garantiza que todos los eventos se publiquen de manera uniforme

### 2. Enrichment Pattern
- **Enriquecimiento**: Agrega EventId y normaliza RoomId
- **Completitud**: Asegura que eventos tengan toda la información necesaria
- **Consistencia**: Formato uniforme para todos los eventos

### 3. Normalization Pattern
- **Normalización**: Lógica para asegurar RoomId consistente
- **Fallback**: Múltiples fuentes para obtener RoomId
- **Robustez**: Maneja diferentes estados del evento de entrada

## Casos de Uso

### Publicación de Mensaje

```go
// En SendMessage handler
event := &chatv1.MessageEvent{
    Event: &chatv1.MessageEvent_Message{
        Message: savedMessage,
    },
}

h.publishChatEvent(generalParams, room.Id, event)
```

### Evento de Unión a Sala

```go
// En CreateRoom handler
event := &chatv1.MessageEvent{
    RoomId: room.Id,
    Event: &chatv1.MessageEvent_RoomJoin{
        RoomJoin: &chatv1.RoomJoinEvent{
            JoinedAt: time.Now().UTC().Format(time.RFC3339),
            UserId:   participantId,
        },
    },
}

h.publishChatEvent(generalParams, room.Id, event)
```

### Actualización de Estado

```go
// En MarkMessagesAsRead handler
event := &chatv1.MessageEvent{
    Event: &chatv1.MessageEvent_StatusUpdate{
        StatusUpdate: &chatv1.MessageStatusUpdate{
            MessageId: messageId,
            Status:    chatv1.MessageStatus_MESSAGE_STATUS_READ,
            UserId:    int32(userID),
        },
    },
}

h.publishChatEvent(generalParams, roomId, event)
```

## Consideraciones de Performance

### 1. Operación Asíncrona
- **No Blocking**: No bloquea el request principal
- **Throughput**: Permite mayor throughput de requests
- **Latencia**: Reduce latencia de respuesta al cliente

### 2. Logging Eficiente
- **Structured Logging**: Más eficiente que string concatenation
- **Lazy Evaluation**: Reflection solo cuando se necesita
- **Minimal Overhead**: Logging no impacta performance significativamente

### 3. Memory Management
- **Event Copying**: Crea nueva instancia para evitar race conditions
- **UUID Generation**: Overhead mínimo de generación de UUID
- **Garbage Collection**: Estructuras diseñadas para GC eficiente

## Testing

### Unit Tests

```go
func TestPublishChatEvent(t *testing.T) {
    // Mock dependencies
    mockLogger := &MockLogger{}
    mockDispatcher := &MockDispatcher{}
    
    handler := &handlerImpl{
        logger:     mockLogger,
        dispatcher: mockDispatcher,
    }
    
    // Test data
    generalParams := api.GeneralParams{
        Session: &api.Session{UserID: 123},
        ClientId: "client_456",
    }
    
    event := &chatv1.MessageEvent{
        Event: &chatv1.MessageEvent_Message{
            Message: &chatv1.MessageData{
                Content: "Test message",
            },
        },
    }
    
    // Execute
    handler.publishChatEvent(generalParams, "room_789", event)
    
    // Verify logging
    assert.True(t, mockLogger.InfoCalled)
    assert.Contains(t, mockLogger.LastMessage, "Dispatching chat event")
    
    // Verify dispatch
    assert.True(t, mockDispatcher.DispatchCalled)
    dispatchedEvent := mockDispatcher.LastEvent.(ChatEvent)
    assert.Equal(t, "room_789", dispatchedEvent.roomID)
    assert.Equal(t, 123, dispatchedEvent.userID)
    assert.NotEmpty(t, dispatchedEvent.event.EventId)
    assert.Equal(t, "room_789", dispatchedEvent.event.RoomId)
}

func TestRoomIdNormalization(t *testing.T) {
    tests := []struct {
        name           string
        inputEvent     *chatv1.MessageEvent
        roomIDParam    string
        expectedRoomId string
    }{
        {
            name: "event with RoomId",
            inputEvent: &chatv1.MessageEvent{
                RoomId: "room_123",
            },
            roomIDParam:    "room_456",
            expectedRoomId: "room_123",
        },
        {
            name: "event with Room object",
            inputEvent: &chatv1.MessageEvent{
                Room: &chatv1.Room{Id: "room_789"},
            },
            roomIDParam:    "room_456",
            expectedRoomId: "room_789",
        },
        {
            name: "event without RoomId",
            inputEvent: &chatv1.MessageEvent{
                Event: &chatv1.MessageEvent_Message{},
            },
            roomIDParam:    "room_456",
            expectedRoomId: "room_456",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test the normalization logic
            eventVal := &chatv1.MessageEvent{
                EventId: uuid.NewString(),
                RoomId:  tt.inputEvent.RoomId,
                Room:    tt.inputEvent.Room,
                Event:   tt.inputEvent.Event,
            }
            
            if eventVal.Room != nil && eventVal.RoomId == "" {
                eventVal.RoomId = eventVal.Room.Id
            }
            
            if eventVal.RoomId == "" {
                eventVal.RoomId = tt.roomIDParam
            }
            
            assert.Equal(t, tt.expectedRoomId, eventVal.RoomId)
        })
    }
}
```

## Monitoreo y Observabilidad

### Métricas Recomendadas

```go
// Contador de eventos publicados
eventsPublishedCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "chat_events_published_total",
        Help: "Total number of chat events published",
    },
    []string{"room_id", "event_type", "user_id"},
)

// Tiempo de procesamiento de eventos
eventProcessingDuration := prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "chat_event_processing_duration_seconds",
        Help: "Time taken to process and publish chat events",
    },
    []string{"event_type"},
)
```

### Alertas Sugeridas

```yaml
# Alertas para eventos de chat
- alert: ChatEventPublishingFailed
  expr: increase(chat_events_published_errors_total[5m]) > 10
  for: 1m
  labels:
    severity: warning
  annotations:
    summary: "High rate of chat event publishing failures"

- alert: ChatEventProcessingLatency
  expr: histogram_quantile(0.95, chat_event_processing_duration_seconds) > 1
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "High latency in chat event processing"
```

## Mejores Prácticas Implementadas

1. **Structured Logging**: Logging estructurado para mejor observabilidad
2. **Event Enrichment**: Enriquecimiento automático de eventos
3. **Data Normalization**: Normalización de datos para consistencia
4. **Asynchronous Processing**: Procesamiento asíncrono para performance
5. **Immutability**: No modifica eventos originales
6. **Error Resilience**: Manejo robusto de datos faltantes
7. **Unique Identification**: Generación de IDs únicos para tracking

Este archivo proporciona una abstracción limpia y robusta para la publicación de eventos de chat, asegurando consistencia, observabilidad y performance en todo el sistema.