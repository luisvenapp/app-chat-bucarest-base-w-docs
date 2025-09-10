# proto/services/chat/v1/types.proto

Descripción
- Define tipos del dominio de chat: Room, MessageData, RoomParticipant, eventos (RoomJoinEvent, RoomLeaveEvent, TypingEvent, MessageStatusUpdate, ErrorEvent) y enums (MessageStatus, SyncStrategy).

Puntos clave
- Room: información de la sala + último mensaje + banderas (muted, pinned, bloqueos, permisos del usuario).
- MessageData: contenido, metadata (respuestas, reenvíos, menciones, reacciones, adjuntos), estado y tiempos.
- MessageEvent (oneof): multiplexa eventos para el stream (mensaje, status_update, is_room_updated, join/leave, typing, error, update_message, delete_message, connected).
- SyncStrategy: estrategias de sincronización para InitialSync (full/recent/minimal/smart).

Uso
- Los handlers y repositorios producen/consumen estos tipos. El cliente (CLI/App) interpreta MessageEvent en tiempo real.