# repository/rooms/room_postgres_impl.go (SQLRoomRepository)

Visión general
- Implementación sobre PostgreSQL usando Squirrel para query building y el módulo dbpq del core. Incluye caching agresivo para lecturas de Room/Message.

Crear sala (CreateRoom)
- Para p2p intenta reutilizar sala existente entre usuario y participante (join múltiple para armar Room + Partner + flags). Si no existe:
  - Genera encryption_data (utils.GenerateKeyEncript).
  - Inserta en room, crea room_member OWNER (creador) y MEMBERS (participantes distintos del creador).
  - Si p2p, enriquece Partner consultando usuario.
  - Si group, precarga Participants (máx 5).

Obtener sala (GetRoom)
- Soporta vistas shim (allData=false) y completas (allData=true).
- Usa LEFT JOIN LATERAL para el último mensaje + datos del remitente.
- Calcula unread_count con subconsulta sobre room_message + room_message_meta.
- Añade Partner (p2p) y, si group/allData, carga Participants.
- Normaliza con utils.FormatRoom y cachea por usuario.

Listar salas (GetRoomList)
- Similar a GetRoom pero para múltiples salas:
  - Join con partner condicional para p2p.
  - LEFT JOIN LATERAL last_msg y conteo unread por sala.
  - Filtros: search (ILIKE unaccent), type, since.
  - Orden: is_pinned DESC, lastMessageAt DESC, created_at DESC.
- Carga hasta 5 participantes por sala (subconsulta window rn <= 5).
- Devuelve meta (TotalItems, etc.).

Eliminadas (GetRoomListDeleted)
- Devuelve IDs de salas eliminadas o de las que el usuario fue removido (since opcional).

Participantes
- GetRoomParticipants: lista paginable con búsqueda por nombre (unaccent), orden alfabético. Devuelve meta total.
- AddParticipantToRoom: obtiene usuarios válidos, re-activa miembros soft-deleted, inserta faltantes.
- UpdateParticipantRoom: cambia rol (actualiza room_member).
- LeaveRoom: marca removed_at a los participantes, limpia caches y devuelve datos de usuarios afectados.

Flags/acciones
- PinRoom/MuteRoom: actualiza flags para el usuario, y limpia caches.
- BlockUser: actualiza is_partner_blocked en room_member (del usuario actual).
- DeleteRoom: soft delete de room (p2p) y marca removed_at para todos los miembros. Limpia caches.

Mensajería
- SaveMessage:
  - Transaccional: inserta en room_message con content cifrado + content_decrypted (si se pasó descifrado), status SENT, metadata inicial del remitente (read_at=NOW(), isDeleted=false).
  - Inserta menciones (room_message_tag) si vienen.
  - Devuelve MessageData completo (GetMessage) y actualiza caches de room (LastMessage*).
- GetMessageSimple: lectura mínima (usada para MarkMessagesAsRead) con cache.
- GetMessage: lectura completa (joins a user del sender, reply y forwarded), menciones y reacciones.
- UpdateMessage/DeleteMessage: edita contenido/flags o marca isDeleted con timestamps.
- GetMessagesFromRoom:
  - Filtros por before/after (id o fecha), paginación, y modo messages_per_room (para InitialSync multi-sala via window function rn).
  - Calcula status READ/SENT para mensajes del partner en base a meta.read_at.
  - Enriquecimiento por lotes de menciones y reacciones.
- ReactToMessage: upsert/elimina reacción del usuario sobre un mensaje.

Utilidades
- MarkMessagesAsRead: marca como leídos por IDs o desde "since" (resuelve timestamps), devuelve count y publica eventos desde el handler.
- IsPartnerMuted: consulta si el partner está muteado (usado para decidir envío push en p2p).

Consideraciones de rendimiento
- Uso intensivo de índices y LATERAL para último mensaje.
- Cache por usuario/sala para GetRoom/GetMessageSimple.
- Limpieza de caches al cambiar flags/membresías/actualizar datos.