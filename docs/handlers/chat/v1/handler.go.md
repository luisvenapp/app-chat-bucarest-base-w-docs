# handlers/chat/v1/handler.go

Resumen
- Implementación completa del ChatService. Orquesta repositorio de salas/mensajes (SQL/Scylla), NATS JetStream (publicación/consumo), envío de notificaciones push y validaciones de acceso.

Inicialización (NewHandler)
- Obtiene conexión NATS desde natsmanager.Get(). Verifica IsConnected().
- Crea contexto JetStream (jetstream.New) y asegura streams requeridos (EnsureStreams).
- Instancia repositorio de rooms:
  - SQLRoomRepository(database.DB()) por defecto.
  - Si USE_SCYLLADB=true, envuelve con ScyllaRoomRepository(database.CQLDB(), SQLRepo) para uso mixto.
- Crea dispatcher de eventos (events.NewEventDispatcher) y un StreamManager[MessageEvent] para manejar streams por cliente.

Validaciones comunes
- utils.ValidateAuthToken(req): requiere sesión válida. En errores, responde con códigos de api.UpdateResponseInfoErrorMessageFromCode / connect.NewError.
- utils.FormatRoom(room): normaliza representación (p2p: nombre/avatar del partner, lastMessageAt).

Operaciones sobre rooms
- CreateRoom
  - Valida tipo/participantes.
  - Crea sala en repositorio (genera encryption_data por sala).
  - Emite evento RoomJoin para cada participante (OWNER/MEMBER), publica vía publishChatEvent.
  - Si es group: suscribe a topic de notificaciones push (room-<roomId>) a los participantes.
- GetRooms / GetRoom
  - Lista/obtiene sala con metadatos (último mensaje, unread_count). Aplica FormatRoom.
- LeaveRoom
  - Reglas distintas según p2p/group. Soporta leaveAll.
  - Actualiza repositorio, emite system messages (remove_member) si aplica y evento RoomLeave.
  - Desuscribe de topic push y, si leaveAll, elimina la sala (p2p) para todos.
- GetRoomParticipants
  - Paginado y búsqueda (ver repositorio). Devuelve meta.
- PinRoom / MuteRoom
  - Alterna flags a nivel de usuario en room_member (SQL) o tablas equivalentes (Scylla).
  - Si group y mute/unmute cambia, desuscribe/suscribe al topic push.
  - Publica evento IsRoomUpdated.
- UpdateRoom
  - Requiere role con permiso (OWNER o MEMBER con EditGroup=true).
  - Actualiza campos (name/description/photo/sendMessage/addMember/editGroup).
  - Publica IsRoomUpdated y, si name/photo cambian, guarda system_message (new_name/new_photo) y publica evento Message.
- AddParticipantToRoom
  - Valida permisos (solo group, y rol con AddMember).
  - Inserta participantes faltantes; para cada uno, guarda system_message (new_member) y publica evento Message + RoomJoin.
  - Suscribe nuevos usuarios al topic push de la sala.
- UpdateParticipantRoom
  - Cambia rol de un usuario en la sala. Publica IsRoomUpdated.
- BlockUser (p2p)
  - Toggles is_partner_blocked a nivel de usuario.

Mensajería
- SendMessage
  - Valida sala, permisos (no enviar si MEMBER sin permisos en group; no permitir mentions en p2p; no enviar si partner bloqueado).
  - Descifra content si viene cifrado para construir notificación; fuerza req.Msg.Type="user_message".
  - Guarda mensaje y metadata del remitente; crea metadata para participantes de forma asíncrona (CreateMessageMetaForParticipants) y LOG.
  - Publica dos eventos:
    - StatusUpdate (SENT) solo para el remitente (discriminado en handleJetStreamMessage).
    - Message para todos los destinatarios de la sala.
  - Decide envío de push (omite si partner está muteado en p2p); llama notificationsv1client.SendPushNotificationEvent.
- EditMessage
  - Solo el remitente puede editar. Actualiza contenido/edited. Publica UpdateMessage con la entidad completa.
- DeleteMessage
  - Soft delete de mensajes por IDs. Publica DeleteMessage por cada ID.
- GetMessageHistory
  - Paginación por fecha/ID, soporta messages_per_room para sync multi-sala. Aplica cálculo de status (READ/SENT) por user y filtros de bloqueos.
- GetMessage / GetSenderMessage
  - Búsqueda directa por ID o por sender_message_id (Scylla); valida acceso del usuario a la sala.
- ReactToMessage
  - Inserta/actualiza/elimina la reacción del usuario sobre el mensaje.
- MarkMessagesAsRead
  - Marca como leídos por IDs (o por "since"). Publica StatusUpdate(READ) por cada mensaje con readAt.
- GetMessageRead / GetMessageReactions
  - Consultas paginadas con enriquecimiento de usuario.

Streaming en tiempo real
- StreamMessages(ctx, req, stream)
  - Requiere generalParams.ClientId.
  - Registra el stream en h.sm (alta/baja automática con defer).
  - Determina salas permitidas del usuario.
  - Siempre consume eventos directos CHAT_DIRECT_EVENTS.<userId> con durable "client-<clientID>-direct".
  - Si se pasa room_id y es permitido, consume además CHAT_EVENTS.<roomId>; de lo contrario, se suscribe a todas las salas del usuario.
  - Heartbeat: envía MessageEvent{connected=true} cada 15s.
  - handleJetStreamMessage: decodifica JSON + protobuf, enriquece algunos eventos con Room, gestiona suscripción dinámica:
    - RoomJoin: si el usuario entra (o él mismo la creó), se auto-suscribe a la sala.
    - IsRoomUpdated/RoomLeave/Message/StatusUpdate: reenvía al cliente; en StatusUpdate(SENT) sólo se reenvía al remitente.
    - RoomLeave: si el usuario sale, se desuscribe de esa sala.

Publicación de eventos
- publishChatEvent (helpers.go): crea EventId, completa RoomId y despacha un ChatEvent via dispatcher (JetStream). Subject depende de tipo (directo o por sala).

Errores y respuestas
- Se usan helpers del core para normalizar errores HTTP/Connect (api.UpdateResponseInfoErrorMessageFromCode).
- Logs estructurados con slog para trazabilidad (roomID, clientID, event, payload).