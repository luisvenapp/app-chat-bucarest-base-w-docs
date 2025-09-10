# proto/services/chat/v1/service.proto

Descripción
- Define la interfaz gRPC/Connect del ChatService y su mapeo HTTP JSON (via google.api.http).

RPCs principales
- SendMessage, EditMessage, DeleteMessage, ReactToMessage.
- GetRooms, CreateRoom, GetRoom, GetMessageHistory, GetRoomParticipants.
- PinRoom, MuteRoom, LeaveRoom.
- AddParticipantToRoom, UpdateRoom, UpdateParticipantRoom, BlockUser.
- GetSenderMessage, GetMessage, GetMessageRead, GetMessageReactions.
- MarkMessagesAsRead, InitialSync.
- StreamMessages (unidireccional server-stream de MessageEvent). Idempotency: IDEMPOTENT.

HTTP/REST
- Se proporcionan rutas REST amigables (e.g., POST /api/chat/v1/send, GET /api/chat/v1/room/{id}, ...).

Seguridad
- Comentarios marcan que todos los métodos requieren token privado (validación se hace en handlers vía utils.ValidateAuthToken).

Relación con tipos
- Importa services/chat/v1/types.proto donde están los mensajes (Room, MessageData, eventos, etc.).