# Diagramas de flujo — handlers/chat

Este módulo contiene el servicio de chat, streaming de eventos y publicación en NATS/JetStream.

## Inicialización del Handler
```mermaid
flowchart TD
    A[NewHandler] --> B[natsmanager.Get()]
    B --> C{nc.IsConnected?}
    C -- No --> X[log.Fatal]
    C -- Sí --> D[jetstream.New(nc)]
    D --> E[EnsureStreams(...requiredStreams)]
    E --> F[Elegir repositorio Rooms: Postgres o Scylla]
    F --> G[events.NewEventDispatcher]
    G --> H[return handlerImpl]
```

## Crear sala (CreateRoom)
```mermaid
flowchart TD
    A[CreateRoom] --> B[ValidateAuthToken]
    B --> C{Validar request}
    C -- inválido --> X[api.InvalidRequestData]
    C -- válido --> D[roomsRepository.CreateRoom]
    D --> E[utils.FormatRoom]
    E --> F[Broadcast RoomJoin a participantes]
    F --> G{room.Type == group?}
    G -- Sí --> H[SubscribeToTopic room-<id>]
    G -- No --> I[continuar]
    H --> J[Respuesta con Room OWNER]
    I --> J[Respuesta con Room OWNER]
```

## Enviar mensaje (SendMessage)
```mermaid
flowchart TD
    A[SendMessage] --> B[GeneralParams + ValidateAuthToken]
    B --> C[GetRoom y validar permisos]
    C --> D[utils.DecryptMessage]
    D --> E[roomsRepository.SaveMessage]
    E --> F[dispatcher.Fanout:
        - CreateMessageMetaForParticipants
        - Enviar eventos: StatusUpdate (sender) y Message (receptores)
        - Notificación push condicional]
    F --> G[Response Success + Message]
```

## Stream de mensajes (StreamMessages)
```mermaid
flowchart TD
    A[StreamMessages] --> B[GeneralParams + CheckSession]
    B --> C{clientID?}
    C -- no --> X[InvalidArgument]
    C -- sí --> D[sm.Register]
    D --> E[Obtener allowedRooms]
    E --> F{req.room_id?}
    F -- sí y permitido --> G[subscribeAndConsume sala específica]
    F -- no --> H[subscribeAndConsume todas las salas]
    G --> I[Enviar Connected + Heartbeat]
    H --> I[Enviar Connected + Heartbeat]
    I --> J[ctx.Done -> Unregister y Stop consumers]
```

## Publicar evento (publishChatEvent)
```mermaid
flowchart TD
    A[publishChatEvent] --> B[Construir MessageEvent con EventId]
    B --> C[Determinar RoomId]
    C --> D[dispatcher.Dispatch(ChatEvent)]
```

## Manejo de mensajes JetStream
```mermaid
flowchart TD
    A[handleJetStreamMessage] --> B[Deserializar JSON + Proto]
    B --> C[Segun tipo de evento]
    C --> D[RoomJoin -> subscribe a room]
    C --> E[IsRoomUpdated -> fetch room y enviar]
    C --> F[StatusUpdate -> lógica por usuario]
    C --> G[RoomLeave -> unsubscribe de room]
    C --> H[Default -> enviar al stream]
```
