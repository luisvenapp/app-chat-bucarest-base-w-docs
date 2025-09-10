# Documentación Técnica: handlers/chat/v1/init_stream.go

## Descripción General

El archivo `init_stream.go` define la configuración y inicialización de los streams de NATS JetStream para el sistema de chat. Establece la infraestructura de messaging persistente que permite la entrega garantizada de eventos de chat en tiempo real, configurando streams, subjects y políticas de retención.

## Estructura del Archivo

### Importaciones

```go
import (
    "strconv"
    "strings"
    "time"
    
    "github.com/nats-io/nats.go/jetstream"
)
```

**Análisis de Importaciones:**

- **`strconv`**: Conversión de enteros a strings para subjects
- **`strings`**: Manipulación de strings para construcción de subjects
- **`time`**: Definición de políticas de tiempo para retención
- **`jetstream`**: Cliente de NATS JetStream para streaming persistente

## Constantes de Configuración

```go
const (
    StreamChatEventsName                = "CHAT_EVENTS"
    StreamChatDirectEventsSubjectPrefix = "CHAT_DIRECT_EVENTS"
)
```

**Análisis de Constantes:**

### `StreamChatEventsName = "CHAT_EVENTS"`
- **Propósito**: Nombre del stream principal para eventos de chat
- **Scope**: Todos los eventos relacionados con salas de chat
- **Uso**: Identificador único del stream en NATS JetStream
- **Convención**: Nombre en mayúsculas siguiendo convenciones NATS

### `StreamChatDirectEventsSubjectPrefix = "CHAT_DIRECT_EVENTS"`
- **Propósito**: Prefijo para subjects de eventos directos a usuarios
- **Scope**: Eventos dirigidos a usuarios específicos
- **Uso**: Base para construir subjects personalizados por usuario
- **Patrón**: `CHAT_DIRECT_EVENTS.{userID}`

## Configuración de Streams

```go
var requiredStreams = []jetstream.StreamConfig{
    {
        Name: StreamChatEventsName,
        Subjects: []string{
            chatRoomEventSubject("*"),
            StreamChatDirectEventsSubjectPrefix + ".*",
        },
        Storage:           jetstream.FileStorage,
        Retention:         jetstream.LimitsPolicy,
        MaxMsgsPerSubject: 1000,
        Compression:       jetstream.S2Compression,
        MaxAge:            7 * 24 * time.Hour, // 7 días
    },
}
```

**Análisis Detallado de la Configuración:**

### Identificación del Stream
```go
Name: StreamChatEventsName,
```
- **Valor**: `"CHAT_EVENTS"`
- **Propósito**: Identificador único del stream
- **Uso**: Referencia para operaciones de stream

### Subjects Manejados
```go
Subjects: []string{
    chatRoomEventSubject("*"),
    StreamChatDirectEventsSubjectPrefix + ".*",
},
```

**Análisis de Subjects:**

#### `chatRoomEventSubject("*")`
- **Patrón**: `"CHAT_EVENTS.*"`
- **Cobertura**: Todos los eventos de salas de chat
- **Wildcard**: `*` captura cualquier roomID
- **Ejemplos**:
  - `CHAT_EVENTS.room_123`
  - `CHAT_EVENTS.room_456`
  - `CHAT_EVENTS.group_789`

#### `StreamChatDirectEventsSubjectPrefix + ".*"`
- **Patrón**: `"CHAT_DIRECT_EVENTS.*"`
- **Cobertura**: Todos los eventos directos a usuarios
- **Wildcard**: `*` captura cualquier userID
- **Ejemplos**:
  - `CHAT_DIRECT_EVENTS.123`
  - `CHAT_DIRECT_EVENTS.456`
  - `CHAT_DIRECT_EVENTS.789`

### Configuración de Almacenamiento
```go
Storage: jetstream.FileStorage,
```
- **Tipo**: Almacenamiento en archivo (persistente)
- **Ventajas**:
  - **Durabilidad**: Sobrevive a reinicios del servidor
  - **Capacidad**: Mayor capacidad que memoria
  - **Costo**: Menor costo por mensaje
- **Alternativa**: `jetstream.MemoryStorage` (más rápido pero volátil)

### Política de Retención
```go
Retention: jetstream.LimitsPolicy,
```
- **Política**: Retención basada en límites configurados
- **Comportamiento**: Elimina mensajes cuando se alcanzan límites
- **Límites aplicables**:
  - `MaxMsgsPerSubject`: Máximo de mensajes por subject
  - `MaxAge`: Edad máxima de mensajes
- **Alternativas**:
  - `jetstream.InterestPolicy`: Retiene solo mientras hay interés
  - `jetstream.WorkQueuePolicy`: Para colas de trabajo

### Límite de Mensajes por Subject
```go
MaxMsgsPerSubject: 1000,
```
- **Valor**: 1000 mensajes por subject
- **Aplicación**: Cada sala/usuario puede tener máximo 1000 mensajes
- **Comportamiento**: Elimina mensajes más antiguos cuando se alcanza el límite
- **Consideraciones**:
  - **Salas activas**: Pueden alcanzar el límite rápidamente
  - **Archivado**: Mensajes antiguos se pierden
  - **Performance**: Límite ayuda a mantener performance

### Compresión
```go
Compression: jetstream.S2Compression,
```
- **Algoritmo**: S2 (Snappy successor)
- **Ventajas**:
  - **Velocidad**: Compresión/descompresión rápida
  - **Ratio**: Buen ratio de compresión
  - **CPU**: Bajo uso de CPU
- **Alternativas**:
  - `jetstream.NoCompression`: Sin compresión
  - Otros algoritmos según disponibilidad

### Edad Máxima de Mensajes
```go
MaxAge: 7 * 24 * time.Hour, // 7 días
```
- **Valor**: 7 días (168 horas)
- **Propósito**: Limita la edad de mensajes en el stream
- **Comportamiento**: Elimina mensajes más antiguos de 7 días
- **Consideraciones**:
  - **Compliance**: Puede requerirse para regulaciones
  - **Storage**: Controla crecimiento del almacenamiento
  - **Performance**: Mantiene índices manejables

## Funciones de Construcción de Subjects

### Función chatRoomEventSubject

```go
func chatRoomEventSubject(roomId string) string {
    return strings.Join([]string{StreamChatEventsName, roomId}, ".")
}
```

**Análisis Detallado:**

#### Parámetros
- **`roomId string`**: Identificador único de la sala de chat

#### Construcción del Subject
```go
strings.Join([]string{StreamChatEventsName, roomId}, ".")
```
- **Componentes**: `["CHAT_EVENTS", roomId]`
- **Separador**: `.` (punto, estándar NATS)
- **Resultado**: `"CHAT_EVENTS.{roomId}"`

#### Ejemplos de Uso
```go
chatRoomEventSubject("room_123")     // "CHAT_EVENTS.room_123"
chatRoomEventSubject("group_456")    // "CHAT_EVENTS.group_456"
chatRoomEventSubject("*")            // "CHAT_EVENTS.*" (wildcard)
```

#### Casos de Uso
- **Publicación**: Publicar eventos específicos de una sala
- **Suscripción**: Suscribirse a eventos de una sala específica
- **Configuración**: Definir patterns para stream configuration

### Función chatDirectEventSubject

```go
func chatDirectEventSubject(userId int) string {
    return strings.Join([]string{StreamChatDirectEventsSubjectPrefix, strconv.Itoa(userId)}, ".")
}
```

**Análisis Detallado:**

#### Parámetros
- **`userId int`**: Identificador único del usuario

#### Construcción del Subject
```go
strings.Join([]string{StreamChatDirectEventsSubjectPrefix, strconv.Itoa(userId)}, ".")
```
- **Componentes**: `["CHAT_DIRECT_EVENTS", strconv.Itoa(userId)]`
- **Conversión**: `strconv.Itoa(userId)` convierte int a string
- **Separador**: `.` (punto, estándar NATS)
- **Resultado**: `"CHAT_DIRECT_EVENTS.{userId}"`

#### Ejemplos de Uso
```go
chatDirectEventSubject(123)    // "CHAT_DIRECT_EVENTS.123"
chatDirectEventSubject(456)    // "CHAT_DIRECT_EVENTS.456"
chatDirectEventSubject(789)    // "CHAT_DIRECT_EVENTS.789"
```

#### Casos de Uso
- **Notificaciones personales**: Eventos dirigidos a usuario específico
- **Unión a salas**: Notificar al usuario que se unió
- **Configuración de cuenta**: Cambios en configuración personal

## Arquitectura de Subjects

### Jerarquía de Subjects

```
CHAT_EVENTS
├── CHAT_EVENTS.room_123        (Eventos de sala específica)
├── CHAT_EVENTS.room_456        (Eventos de sala específica)
├── CHAT_EVENTS.group_789       (Eventos de grupo específico)
└── CHAT_EVENTS.*               (Wildcard para todas las salas)

CHAT_DIRECT_EVENTS
├── CHAT_DIRECT_EVENTS.123      (Eventos directos a usuario 123)
├── CHAT_DIRECT_EVENTS.456      (Eventos directos a usuario 456)
├── CHAT_DIRECT_EVENTS.789      (Eventos directos a usuario 789)
└── CHAT_DIRECT_EVENTS.*        (Wildcard para todos los usuarios)
```

### Patrones de Suscripción

#### Suscripción a Sala Específica
```go
subject := chatRoomEventSubject("room_123")
// Recibe solo eventos de room_123
```

#### Suscripción a Todas las Salas
```go
subject := chatRoomEventSubject("*")
// Recibe eventos de todas las salas
```

#### Suscripción a Usuario Específico
```go
subject := chatDirectEventSubject(123)
// Recibe solo eventos directos para usuario 123
```

#### Suscripción a Todos los Usuarios
```go
subject := StreamChatDirectEventsSubjectPrefix + ".*"
// Recibe eventos directos para todos los usuarios
```

## Configuración de Consumers

### Consumer para Sala Específica

```go
consumerConfig := jetstream.ConsumerConfig{
    Durable:       fmt.Sprintf("client-%s-room-%s", clientID, roomID),
    AckPolicy:     jetstream.AckExplicitPolicy,
    DeliverPolicy: jetstream.DeliverNewPolicy,
    FilterSubject: chatRoomEventSubject(roomID),
}
```

### Consumer para Eventos Directos

```go
consumerConfig := jetstream.ConsumerConfig{
    Durable:       fmt.Sprintf("client-%s-direct", clientID),
    AckPolicy:     jetstream.AckExplicitPolicy,
    DeliverPolicy: jetstream.DeliverNewPolicy,
    FilterSubject: chatDirectEventSubject(userID),
}
```

## Consideraciones de Performance

### 1. Límites de Mensajes
- **MaxMsgsPerSubject**: 1000 mensajes por subject
- **Impacto**: Salas muy activas pueden perder historial
- **Solución**: Aumentar límite o implementar archivado

### 2. Compresión S2
- **Ventajas**: Reduce uso de disco y ancho de banda
- **Overhead**: Mínimo overhead de CPU
- **Ratio**: Típicamente 2-4x reducción de tamaño

### 3. Retención por Tiempo
- **7 días**: Balance entre utilidad y recursos
- **Cleanup**: Automático, no requiere intervención manual
- **Configuración**: Ajustable según necesidades

## Monitoreo y Métricas

### Métricas de Stream

```go
// Métricas recomendadas para monitoreo
streamMetrics := []string{
    "jetstream_stream_messages",      // Total de mensajes
    "jetstream_stream_bytes",         // Total de bytes
    "jetstream_stream_consumers",     // Número de consumers
    "jetstream_stream_subjects",      // Número de subjects
}
```

### Alertas Sugeridas

```yaml
# Alertas para streams de chat
- alert: ChatStreamHighMessageCount
  expr: jetstream_stream_messages{stream="CHAT_EVENTS"} > 900000
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Chat stream approaching message limit"

- alert: ChatStreamOldMessages
  expr: jetstream_stream_first_seq_age_seconds{stream="CHAT_EVENTS"} > 604800
  for: 1m
  labels:
    severity: info
  annotations:
    summary: "Chat stream has messages older than 7 days"
```

## Testing

### Unit Tests

```go
func TestChatRoomEventSubject(t *testing.T) {
    tests := []struct {
        name     string
        roomID   string
        expected string
    }{
        {
            name:     "regular room",
            roomID:   "room_123",
            expected: "CHAT_EVENTS.room_123",
        },
        {
            name:     "group room",
            roomID:   "group_456",
            expected: "CHAT_EVENTS.group_456",
        },
        {
            name:     "wildcard",
            roomID:   "*",
            expected: "CHAT_EVENTS.*",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := chatRoomEventSubject(tt.roomID)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestChatDirectEventSubject(t *testing.T) {
    tests := []struct {
        name     string
        userID   int
        expected string
    }{
        {
            name:     "user 123",
            userID:   123,
            expected: "CHAT_DIRECT_EVENTS.123",
        },
        {
            name:     "user 456",
            userID:   456,
            expected: "CHAT_DIRECT_EVENTS.456",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := chatDirectEventSubject(tt.userID)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

```go
func TestStreamConfiguration(t *testing.T) {
    // Test que la configuración del stream es válida
    config := requiredStreams[0]
    
    assert.Equal(t, "CHAT_EVENTS", config.Name)
    assert.Equal(t, jetstream.FileStorage, config.Storage)
    assert.Equal(t, jetstream.LimitsPolicy, config.Retention)
    assert.Equal(t, int64(1000), config.MaxMsgsPerSubject)
    assert.Equal(t, jetstream.S2Compression, config.Compression)
    assert.Equal(t, 7*24*time.Hour, config.MaxAge)
    
    // Verificar subjects
    expectedSubjects := []string{
        "CHAT_EVENTS.*",
        "CHAT_DIRECT_EVENTS.*",
    }
    assert.Equal(t, expectedSubjects, config.Subjects)
}
```

## Mejores Prácticas Implementadas

1. **Naming Conventions**: Nombres consistentes y descriptivos
2. **Wildcard Support**: Soporte para patterns flexibles
3. **Separation of Concerns**: Subjects separados por tipo de evento
4. **Resource Management**: Límites para controlar recursos
5. **Durability**: Almacenamiento persistente para garantías
6. **Compression**: Optimización de almacenamiento
7. **Time-based Cleanup**: Limpieza automática por edad

Este archivo establece la base de la infraestructura de messaging para el sistema de chat, proporcionando una configuración robusta y escalable para la entrega de eventos en tiempo real.