# Documentación Técnica: handlers/chat/v1/handler.go

## Descripción General

El archivo `handler.go` implementa el núcleo del servicio de chat, proporcionando todas las funcionalidades principales como gestión de salas, envío de mensajes, streaming en tiempo real y administración de participantes. Es el componente más crítico del sistema, manejando la lógica de negocio completa del chat.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "log/slog"
    "os"
    "slices"
    "strconv"
    "time"
    
    "connectrpc.com/connect"
    notificationsv1client "github.com/Venqis-NolaTech/campaing-app-notifications-api-go/proto/generated/services/notifications/v1/client"
    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
    "google.golang.org/protobuf/encoding/protojson"
    "google.golang.org/protobuf/proto"
    
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/database"
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1/chatv1connect"
    roomsrepository "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/repository/rooms"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
    natsmanager "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/broker/nats"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/events"
    notificationsv1 "github.com/Venqis-NolaTech/campaing-app-notifications-api-go/proto/generated/services/notifications/v1"
)
```

**Análisis de Importaciones:**

- **Estándar Go**: Context, JSON, errores, logging, tiempo
- **gRPC/Connect**: Framework para servicios gRPC
- **NATS**: Messaging system para eventos en tiempo real
- **Protobuf**: Serialización de mensajes
- **Servicios externos**: Cliente de notificaciones
- **Componentes internos**: Database, repository, utils
- **Core**: API, eventos, NATS manager

### Estructura del Handler

```go
type handlerImpl struct {
    logger          *slog.Logger
    nc              *nats.Conn                                 // Cliente de NATS
    js              jetstream.JetStream                        // Nuevo cliente de JetStream
    sm              *events.StreamManager[chatv1.MessageEvent] // Gestor de streams para la instancia actual
    dispatcher      *events.EventDispatcher
    roomsRepository roomsrepository.RoomsRepository
}
```

**Análisis de Campos:**

#### `logger *slog.Logger`
- **Propósito**: Logging estructurado para debugging y monitoreo
- **Uso**: Registra eventos importantes, errores y métricas
- **Configuración**: Nivel y formato configurables por entorno

#### `nc *nats.Conn`
- **Propósito**: Conexión principal a NATS para messaging
- **Uso**: Publicación y suscripción a eventos
- **Características**: Conexión persistente, auto-reconexión

#### `js jetstream.JetStream`
- **Propósito**: Cliente JetStream para streaming persistente
- **Uso**: Streams de mensajes con garantías de entrega
- **Ventajas**: Persistencia, replay, acknowledgments

#### `sm *events.StreamManager[chatv1.MessageEvent]`
- **Propósito**: Gestión de streams activos por cliente
- **Uso**: Mantiene conexiones WebSocket/gRPC activas
- **Type Safety**: Tipado específico para eventos de chat

#### `dispatcher *events.EventDispatcher`
- **Propósito**: Despachador de eventos asíncrono
- **Uso**: Procesa eventos en background sin bloquear requests
- **Características**: Pool de workers, retry logic

#### `roomsRepository roomsrepository.RoomsRepository`
- **Propósito**: Acceso a datos de salas y mensajes
- **Uso**: CRUD operations, queries complejas
- **Abstracción**: Interface que permite múltiples implementaciones

## Función de Inicialización

### NewHandler

```go
func NewHandler() chatv1connect.ChatServiceHandler {
    nm, err := natsmanager.Get()
    if err != nil {
        log.Fatal(err)
    }
    
    nc := nm.GetConn()
    if nc == nil {
        log.Fatal("No existe una conexión hacia NATS inicializada")
    }
    
    // Verificar que la conexión esté conectada
    if !nc.IsConnected() {
        log.Fatal("La conexión hacia NATS no está conectada")
    }
    
    js, err := jetstream.New(nc)
    if err != nil {
        log.Fatalf("Failed to create JetStream context: %v", err)
    }
    
    // Inicializar streams de NATS
    if err := natsmanager.EnsureStreams(context.Background(), js, requiredStreams...); err != nil {
        log.Printf("Error al inicializar streams de NATS: %v", err)
        // No fatal, pero registramos el error
    }
    
    logger := slog.Default()
    repo := roomsrepository.NewSQLRoomRepository(database.DB())
    if scylladb, _ := strconv.ParseBool(os.Getenv("USE_SCYLLADB")); scylladb {
        repo = roomsrepository.NewScyllaRoomRepository(database.CQLDB(), repo)
    }
    dispatcher, err := events.NewEventDispatcher(nc, logger, 5)
    if err != nil {
        log.Fatalf("Failed to create event dispatcher: %v", err)
    }
    
    return &handlerImpl{
        logger:          logger,
        sm:              events.NewStreamManager[chatv1.MessageEvent](logger),
        nc:              nc,
        js:              js,
        roomsRepository: repo,
        dispatcher:      dispatcher,
    }
}
```

**Análisis del Proceso de Inicialización:**

#### 1. Configuración de NATS
```go
nm, err := natsmanager.Get()
nc := nm.GetConn()
if !nc.IsConnected() {
    log.Fatal("La conexión hacia NATS no está conectada")
}
```
- **Manager**: Obtiene instancia del manager de NATS
- **Conexión**: Verifica que la conexión esté activa
- **Validación**: Falla rápido si NATS no está disponible

#### 2. Configuración de JetStream
```go
js, err := jetstream.New(nc)
if err := natsmanager.EnsureStreams(context.Background(), js, requiredStreams...); err != nil {
    log.Printf("Error al inicializar streams de NATS: %v", err)
}
```
- **JetStream**: Crea contexto para streaming persistente
- **Streams**: Asegura que los streams requeridos existan
- **Error handling**: No fatal para permitir degradación graceful

#### 3. Configuración de Repository
```go
repo := roomsrepository.NewSQLRoomRepository(database.DB())
if scylladb, _ := strconv.ParseBool(os.Getenv("USE_SCYLLADB")); scylladb {
    repo = roomsrepository.NewScyllaRoomRepository(database.CQLDB(), repo)
}
```
- **Base**: Repository SQL como base
- **Opcional**: ScyllaDB como capa adicional si está configurado
- **Patrón Decorator**: ScyllaDB envuelve SQL repository

#### 4. Configuración de Event Dispatcher
```go
dispatcher, err := events.NewEventDispatcher(nc, logger, 5)
```
- **Workers**: 5 workers para procesamiento asíncrono
- **NATS**: Usa la misma conexión para eventos
- **Logging**: Logger compartido para consistencia

## Funciones de Gestión de Salas

### CreateRoom

```go
func (h *handlerImpl) CreateRoom(ctx context.Context, req *connect.Request[chatv1.CreateRoomRequest]) (*connect.Response[chatv1.CreateRoomResponse], error) {
    //validate auth token
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        return nil, err
    }
    
    if len(req.Msg.Participants) < 1 {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
    }
    if len(req.Msg.Participants) > 1 && req.Msg.Type == "p2p" {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
    }
    if req.Msg.Type == "group" && req.Msg.Name == nil {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
    }
    
    room, err := h.roomsRepository.CreateRoom(ctx, userID, req.Msg)
    if err != nil {
        return nil, err
    }
    
    room = utils.FormatRoom(room)
    
    generalParams, _ := api.GeneralParamsFromConnectRequest(req)
    
    req.Msg.Participants = append(req.Msg.Participants, int32(userID))
    
    joinedAt := time.Now().UTC().Format(time.RFC3339)
    for _, id := range req.Msg.Participants {
        newRoomObject := proto.Clone(room).(*chatv1.Room)
        if id == int32(userID) {
            newRoomObject.Role = "OWNER"
        } else {
            newRoomObject.Role = "MEMBER"
        }
        event := &chatv1.MessageEvent{
            RoomId: room.Id,
            Event: &chatv1.MessageEvent_RoomJoin{RoomJoin: &chatv1.RoomJoinEvent{
                JoinedAt:    joinedAt,
                UserId:      id,
                OwnerUserId: int32(userID),
            }},
        }
        
        h.publishChatEvent(generalParams, room.GetId(), event)
    }
    room.Role = "OWNER"
    
    //subcribirse al topico del grupo
    if room.Type == "group" {
        //agregando al usuario principal
        req.Msg.Participants = append(req.Msg.Participants, int32(userID))
        
        if _, err := notificationsv1client.SubscribeToTopic(context.Background(), generalParams, &notificationsv1.SubscribeToTopicRequest{
            Event: &notificationsv1.SubscribeToTopicRequest_Data{
                Data: &notificationsv1.SubscribeToTopic{
                    Topic:   "room-" + room.Id,
                    UserIds: req.Msg.Participants,
                },
            },
        }); err != nil {
            h.logger.Error("Error enviando subscripcion al topico", "error", err)
        }
    }
    
    return connect.NewResponse(&chatv1.CreateRoomResponse{
        Success: true,
        Room:    room,
    }), nil
}
```

**Análisis del Proceso de Creación:**

#### 1. Autenticación y Validación
```go
userID, err := utils.ValidateAuthToken(req)
if len(req.Msg.Participants) < 1 {
    return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
}
```
- **Auth**: Valida token y obtiene userID
- **Participantes**: Requiere al menos un participante
- **Validaciones específicas**: Diferentes reglas por tipo de sala

#### 2. Validaciones por Tipo de Sala
```go
if len(req.Msg.Participants) > 1 && req.Msg.Type == "p2p" {
    return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
}
if req.Msg.Type == "group" && req.Msg.Name == nil {
    return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
}
```

**Reglas de Validación:**
- **P2P**: Máximo 1 participante (el creador se agrega automáticamente)
- **Group**: Requiere nombre de grupo

#### 3. Creación en Repository
```go
room, err := h.roomsRepository.CreateRoom(ctx, userID, req.Msg)
room = utils.FormatRoom(room)
```
- **Persistencia**: Crea sala en base de datos
- **Formateo**: Aplica formateo específico del dominio

#### 4. Eventos de Unión
```go
joinedAt := time.Now().UTC().Format(time.RFC3339)
for _, id := range req.Msg.Participants {
    newRoomObject := proto.Clone(room).(*chatv1.Room)
    if id == int32(userID) {
        newRoomObject.Role = "OWNER"
    } else {
        newRoomObject.Role = "MEMBER"
    }
    event := &chatv1.MessageEvent{
        RoomId: room.Id,
        Event: &chatv1.MessageEvent_RoomJoin{RoomJoin: &chatv1.RoomJoinEvent{
            JoinedAt:    joinedAt,
            UserId:      id,
            OwnerUserId: int32(userID),
        }},
    }
    
    h.publishChatEvent(generalParams, room.GetId(), event)
}
```

**Proceso de Eventos:**
- **Timestamp**: UTC para consistencia global
- **Clonación**: Copia independiente para cada participante
- **Roles**: Owner para creador, Member para otros
- **Publicación**: Evento para cada participante

#### 5. Suscripción a Notificaciones (Grupos)
```go
if room.Type == "group" {
    if _, err := notificationsv1client.SubscribeToTopic(context.Background(), generalParams, &notificationsv1.SubscribeToTopicRequest{
        Event: &notificationsv1.SubscribeToTopicRequest_Data{
            Data: &notificationsv1.SubscribeToTopic{
                Topic:   "room-" + room.Id,
                UserIds: req.Msg.Participants,
            },
        },
    }); err != nil {
        h.logger.Error("Error enviando subscripcion al topico", "error", err)
    }
}
```
- **Solo grupos**: P2P no requiere tópicos de notificación
- **Tópico**: Identificado por "room-" + roomID
- **Participantes**: Todos los miembros del grupo
- **Error handling**: Log pero no falla la operación

### GetRooms

```go
func (h *handlerImpl) GetRooms(ctx context.Context, req *connect.Request[chatv1.GetRoomsRequest]) (*connect.Response[chatv1.GetRoomsResponse], error) {
    //validate auth token
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        return nil, err
    }
    
    rooms, meta, err := h.roomsRepository.GetRoomList(ctx, userID, req.Msg)
    if err != nil {
        return nil, err
    }
    
    for i, room := range rooms {
        rooms[i] = utils.FormatRoom(room)
    }
    
    roomsResponse := &chatv1.GetRoomsResponse{
        Items: rooms,
        Meta:  meta,
    }
    
    return connect.NewResponse(roomsResponse), nil
}
```

**Características:**
- **Autenticación**: Valida usuario
- **Filtrado**: Solo salas del usuario autenticado
- **Formateo**: Aplica formateo a cada sala
- **Paginación**: Soporte para metadata de paginación

## Funciones de Gestión de Mensajes

### SendMessage

```go
func (h *handlerImpl) SendMessage(ctx context.Context, req *connect.Request[chatv1.SendMessageRequest]) (*connect.Response[chatv1.SendMessageResponse], error) {
    generalParams, err := api.GeneralParamsFromConnectRequest(req)
    if err != nil {
        return nil, err
    }
    
    userID, err := utils.ValidateAuthToken(req)
    if err != nil {
        return nil, err
    }
    
    if req.Msg.RoomId == "" {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
    }
    
    room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.RoomId, false, true)
    if err != nil {
        return nil, err
    }
    
    if room == nil {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
    }
    
    room = utils.FormatRoom(room)
    
    if room.Type == "group" && room.Role == "MEMBER" && !room.SendMessage {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
    }
    
    if len(req.Msg.Mentions) > 0 && room.Type == "p2p" {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
    }
    
    if room.Type == "p2p" && room.IsPartnerBlocked {
        return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
    }
    
    var contentDecrypted string
    if req.Msg.Content != "" {
        contentDecrypted, err = utils.DecryptMessage(req.Msg.Content, room.EncryptionData)
        if err != nil {
            h.logger.Error("Error al desencriptar el contenido", "error", err)
        }
    }
    
    req.Msg.Type = "user_message"
    
    msg, err := h.roomsRepository.SaveMessage(ctx, userID, req.Msg, room, &contentDecrypted)
    if err != nil {
        return nil, err
    }
    
    h.dispatcher.Dispatch(context.Background(), events.FanoutEvent{
        OnFanount: func(ctx context.Context, event events.FanoutEvent) {
            //TODO: Revisar si esto es necesario, porque si son muchos participantes, puede tardar.
            //la meta en realidad deberia ser solo para revisar mensajes vistos
            //sino hay meta se puede suponer que el mensaje no se ha leido...
            err := h.roomsRepository.CreateMessageMetaForParticipants(ctx, room.Id, msg.Id, int(msg.SenderId))
            if err != nil {
                // Consider adding a retry mechanism or pushing to a dead-letter queue for this as well.
                h.logger.Error("Failed to handle message fanout event", "error", err, "roomID", room.Id, "messageID", msg.Id)
            } else {
                h.logger.Info("Successfully fanned out message metadata", "roomID", msg.Id, "messageID", msg.SenderId)
            }
            
            senderEvent := &chatv1.MessageEvent{
                RoomId: room.Id,
                Event: &chatv1.MessageEvent_StatusUpdate{
                    StatusUpdate: &chatv1.MessageStatusUpdate{
                        MessageId: msg.GetId(),
                        Status:    msg.GetStatus(),
                        UpdatedAt: msg.GetUpdatedAt(),
                        UserId:    int32(userID),
                        SenderId:  int32(userID),
                    },
                },
            }
            remitentsEvent := &chatv1.MessageEvent{
                RoomId: room.Id,
                Event: &chatv1.MessageEvent_Message{
                    Message: msg,
                },
            }
            
            h.publishChatEvent(generalParams, msg.RoomId, senderEvent)
            h.publishChatEvent(generalParams, msg.RoomId, remitentsEvent)
            
            var participantsIds []int32
            if room.Type == "p2p" {
                participantsIds = append(participantsIds, int32(room.Partner.Id))
            } else {
                /*participants, _, err := h.roomsRepository.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{
                    Id: room.Id,
                })
                if err != nil {
                    return
                }
                for _, participant := range participants {
                    if int32(msg.SenderId) == participant.GetId() {
                        continue
                    }
                    participantsIds = append(participantsIds, participant.GetId())
                }*/
            }
            
            sendPushNotification := true
            
            //si el partner esta muteado, no se envía la notificación
            if room.Type == "p2p" {
                isPartnerMuted, err := h.roomsRepository.IsPartnerMuted(ctx, int(room.Partner.Id), room.Id)
                if err != nil {
                    fmt.Println("Error al obtener si el partner esta muteado", err)
                }
                if isPartnerMuted {
                    sendPushNotification = false
                }
            }
            
            if sendPushNotification {
                
                if _, err := notificationsv1client.SendPushNotificationEvent(context.Background(), generalParams, &notificationsv1.SendPushNotificationRequest{
                    Event: &notificationsv1.SendPushNotificationRequest_ChatMessage{
                        ChatMessage: &notificationsv1.ChatMessagePushEvent{
                            RecipientsUserId:  participantsIds,
                            SenderId:          int32(userID),
                            SenderDisplayName: msg.SenderName,
                            RoomName:          room.Name,
                            RoomId:            room.Id,
                            RoomType:          room.Type,
                            MessageContent:    contentDecrypted,
                        },
                    },
                }); err != nil {
                    h.logger.Error("Error enviando notificación push del mensaje", "error", err)
                }
            }
        },
    })
    
    response := &chatv1.SendMessageResponse{
        Success: true,
        Message: msg,
    }
    return connect.NewResponse(response), nil
}
```

**Análisis del Proceso de Envío:**

#### 1. Validaciones Iniciales
```go
userID, err := utils.ValidateAuthToken(req)
if req.Msg.RoomId == "" {
    return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
}
```
- **Autenticación**: Valida token de usuario
- **RoomID**: Requiere identificador de sala

#### 2. Validaciones de Permisos
```go
if room.Type == "group" && room.Role == "MEMBER" && !room.SendMessage {
    return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
}
if room.Type == "p2p" && room.IsPartnerBlocked {
    return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
}
```

**Reglas de Negocio:**
- **Grupos**: Miembros necesitan permiso explícito para enviar mensajes
- **P2P**: No se puede enviar si el partner está bloqueado
- **Menciones**: Solo permitidas en grupos

#### 3. Desencriptación del Contenido
```go
var contentDecrypted string
if req.Msg.Content != "" {
    contentDecrypted, err = utils.DecryptMessage(req.Msg.Content, room.EncryptionData)
    if err != nil {
        h.logger.Error("Error al desencriptar el contenido", "error", err)
    }
}
```
- **Encriptación**: Mensajes vienen encriptados del cliente
- **Desencriptación**: Para procesamiento interno y notificaciones
- **Error handling**: Log pero no falla la operación

#### 4. Persistencia del Mensaje
```go
req.Msg.Type = "user_message"
msg, err := h.roomsRepository.SaveMessage(ctx, userID, req.Msg, room, &contentDecrypted)
```
- **Tipo**: Marca como mensaje de usuario
- **Guardado**: Persiste en base de datos
- **Metadatos**: Incluye información de sala y contenido desencriptado

#### 5. Procesamiento Asíncrono
```go
h.dispatcher.Dispatch(context.Background(), events.FanoutEvent{
    OnFanount: func(ctx context.Context, event events.FanoutEvent) {
        // Crear metadata para participantes
        // Publicar eventos
        // Enviar notificaciones push
    },
})
```

**Operaciones Asíncronas:**
- **Metadata**: Crea registros de lectura para participantes
- **Eventos**: Publica eventos de mensaje y estado
- **Notificaciones**: Envía push notifications
- **Performance**: No bloquea la respuesta al cliente

## Función de Streaming

### StreamMessages

```go
func (h *handlerImpl) StreamMessages(ctx context.Context, req *connect.Request[chatv1.StreamMessagesRequest], stream *connect.ServerStream[chatv1.MessageEvent]) error {
    generalParams, err := api.GeneralParamsFromConnectRequest(req)
    if err != nil {
        return err
    }
    
    session, err := api.CheckSessionFromGeneralParams(generalParams)
    if err != nil {
        return err
    }
    
    clientID := generalParams.ClientId
    if clientID == "" {
        err := api.UpdateResponseInfoErrorMessage(errors.New("client_id_needed"), req.Header())
        return connect.NewError(connect.CodeInvalidArgument, err)
    }
    
    h.sm.Register(generalParams, stream)
    defer h.sm.Unregister(generalParams) // Garantiza la limpieza cuando el cliente se desconecta.
    
    specificRoomID := req.Msg.GetRoomId()
    
    allowedRooms, _, err := h.roomsRepository.GetRoomList(ctx, session.UserID, nil)
    if err != nil {
        return fmt.Errorf("no se pudieron obtener las salas del usuario: %w", err)
    }
    
    var allowedRoomsIds []string
    for _, room := range allowedRooms {
        allowedRoomsIds = append(allowedRoomsIds, room.GetId())
    }
    
    roomsConsumers := map[string]jetstream.ConsumeContext{}
    
    defer (func() {
        for _, cons := range roomsConsumers {
            cons.Stop()
        }
    })()
    
    directConsumerKey := fmt.Sprintf("client-%s-direct", clientID)
    directConsumer, err := h.subscribeAndConsume(
        ctx,
        StreamChatEventsName,
        chatDirectEventSubject(session.UserID),
        directConsumerKey,
        generalParams,
        stream,
        roomsConsumers,
    )
    if err != nil {
        return fmt.Errorf("failed to subscribe to direct events: %w", err)
    }
    roomsConsumers[directConsumerKey] = directConsumer
    
    if specificRoomID != "" && slices.Contains(allowedRoomsIds, specificRoomID) {
        h.logger.Info("Usuario suscribiéndose a una sala específica", "clientID", clientID, "roomID", specificRoomID)
        subject := chatRoomEventSubject(specificRoomID)
        
        roomConsumer, err := h.subscribeAndConsume(
            ctx,
            StreamChatEventsName,
            subject,
            fmt.Sprintf("client-%s-room-%s", clientID, specificRoomID),
            generalParams,
            stream,
            roomsConsumers,
        )
        if err != nil {
            return fmt.Errorf("failed to subscribe to room %s events: %w", specificRoomID, err)
        }
        roomsConsumers[specificRoomID] = roomConsumer
    } else {
        h.logger.Info("Usuario suscribiéndose a todas sus salas", "clientID", clientID)
        
        if len(allowedRooms) == 0 {
            h.logger.Warn("El usuario no pertenece a ninguna sala, esperando a que cree una o que ingrese en una", "clientID", clientID)
        }
        
        for _, roomID := range allowedRoomsIds {
            subject := chatRoomEventSubject(roomID)
            
            roomConsumer, err := h.subscribeAndConsume(
                ctx,
                StreamChatEventsName,
                subject,
                fmt.Sprintf("client-%s-room-%s", clientID, roomID),
                generalParams,
                stream,
                roomsConsumers,
            )
            if err != nil {
                return fmt.Errorf("failed to subscribe to room %s events: %w", roomID, err)
            }
            roomsConsumers[roomID] = roomConsumer
        }
    }
    
    h.logger.Info("Stream de usuario activo y escuchando eventos", "clientID", clientID)
    
    h.sm.Send(generalParams, &chatv1.MessageEvent{
        Event: &chatv1.MessageEvent_Connected{Connected: true},
    })
    
    ticker := time.NewTicker(15 * time.Second)
    done := make(chan bool)
    go func() {
        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                h.sm.Send(generalParams, &chatv1.MessageEvent{
                    Event: &chatv1.MessageEvent_Connected{Connected: true},
                })
            }
        }
    }()
    
    <-ctx.Done()
    done <- true
    
    h.logger.Info("Cliente desconectado, cerrando stream y desuscribiendo de NATS", "clientID", clientID)
    return nil
}
```

**Análisis del Streaming:**

#### 1. Validación y Registro
```go
session, err := api.CheckSessionFromGeneralParams(generalParams)
clientID := generalParams.ClientId
if clientID == "" {
    err := api.UpdateResponseInfoErrorMessage(errors.New("client_id_needed"), req.Header())
    return connect.NewError(connect.CodeInvalidArgument, err)
}

h.sm.Register(generalParams, stream)
defer h.sm.Unregister(generalParams)
```
- **Sesión**: Valida autenticación
- **ClientID**: Requiere identificador único del cliente
- **Registro**: Registra stream en el manager
- **Cleanup**: Garantiza limpieza al desconectar

#### 2. Autorización de Salas
```go
allowedRooms, _, err := h.roomsRepository.GetRoomList(ctx, session.UserID, nil)
var allowedRoomsIds []string
for _, room := range allowedRooms {
    allowedRoomsIds = append(allowedRoomsIds, room.GetId())
}
```
- **Autorización**: Solo salas donde el usuario es miembro
- **Lista**: Obtiene todas las salas permitidas
- **IDs**: Extrae identificadores para validación

#### 3. Suscripción a Eventos Directos
```go
directConsumerKey := fmt.Sprintf("client-%s-direct", clientID)
directConsumer, err := h.subscribeAndConsume(
    ctx,
    StreamChatEventsName,
    chatDirectEventSubject(session.UserID),
    directConsumerKey,
    generalParams,
    stream,
    roomsConsumers,
)
```
- **Eventos directos**: Mensajes dirigidos específicamente al usuario
- **Consumer único**: Un consumer por cliente
- **Subject**: Específico del userID

#### 4. Suscripción a Salas
```go
if specificRoomID != "" && slices.Contains(allowedRoomsIds, specificRoomID) {
    // Suscribirse a sala específica
} else {
    // Suscribirse a todas las salas del usuario
}
```

**Estrategias de Suscripción:**
- **Sala específica**: Solo eventos de una sala
- **Todas las salas**: Eventos de todas las salas del usuario
- **Validación**: Solo salas autorizadas

#### 5. Keep-Alive y Cleanup
```go
ticker := time.NewTicker(15 * time.Second)
go func() {
    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            h.sm.Send(generalParams, &chatv1.MessageEvent{
                Event: &chatv1.MessageEvent_Connected{Connected: true},
            })
        }
    }
}()

<-ctx.Done()
done <- true
```
- **Keep-alive**: Ping cada 15 segundos
- **Heartbeat**: Mantiene conexión activa
- **Cleanup**: Limpieza automática al desconectar

## Consideraciones de Performance

### 1. Procesamiento Asíncrono
- **Event Dispatcher**: Operaciones pesadas en background
- **Fanout Events**: Distribución de mensajes sin bloquear
- **Worker Pool**: Múltiples workers para paralelismo

### 2. Streaming Eficiente
- **JetStream**: Persistencia y garantías de entrega
- **Consumer Groups**: Distribución de carga
- **Selective Subscription**: Solo salas relevantes

### 3. Caché y Optimización
- **Repository Layer**: Caché en ScyllaDB
- **Connection Pooling**: Reutilización de conexiones
- **Batch Operations**: Operaciones en lote cuando es posible

## Seguridad

### 1. Autenticación y Autorización
- **Token Validation**: En cada request
- **Room Authorization**: Solo salas del usuario
- **Permission Checks**: Permisos específicos por operación

### 2. Encriptación
- **End-to-End**: Mensajes encriptados en tránsito
- **Key Management**: Claves por sala
- **Secure Storage**: Claves protegidas en base de datos

### 3. Rate Limiting y Abuse Prevention
- **Message Limits**: Límites por usuario/sala
- **Connection Limits**: Máximo de conexiones por usuario
- **Validation**: Validación exhaustiva de inputs

## Mejores Prácticas Implementadas

1. **Error Handling**: Manejo robusto de errores con logging
2. **Resource Management**: Cleanup automático de recursos
3. **Separation of Concerns**: Lógica separada por responsabilidad
4. **Async Processing**: Operaciones no críticas en background
5. **Type Safety**: Uso de tipos generados desde protobuf
6. **Observability**: Logging estructurado para monitoreo
7. **Scalability**: Diseño preparado para escalamiento horizontal

Este handler representa el núcleo del sistema de chat, implementando todas las funcionalidades críticas con un enfoque en performance, seguridad y escalabilidad.