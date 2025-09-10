# repository/rooms/room_scylladb_impl.go (ScyllaRoomRepository)

Visión general
- Implementación alternativa usando ScyllaDB/Cassandra (gocql). Modela tablas por acceso: room_details, participants_by_room, rooms_by_user, room_membership_lookup, messages_by_room, room_counters_by_user, message_status_by_user, etc.
- Integra un UserFetcher para enriquecer datos con nombres/avatares.

Crear sala
- P2P: evita duplicados consultando p2p_room_by_users con sortUserIDs.
- Genera UUID, encryption_data y usa batches para:
  - room_details, participants_by_room, rooms_by_user, room_membership_lookup.
  - p2p_room_by_users si aplica.
  - Inicializa contadores room_counters_by_user.
- Si es canal join_all_user, desata proceso asíncrono para añadir a todos los usuarios del sistema en lotes.

GetRoom / GetRoomList
- Lecturas denormalizadas desde rooms_by_user y room_membership_lookup para flags/último mensaje/unread_count.
- Enriquecimiento de participants para group (concurrente, máx 5 por sala) vía GetRoomParticipants.
- Búsqueda sin acentos mediante removeAccents para filtrar por nombre.

Participantes
- GetRoomParticipants: lista user_id/role y enriquece con GetUsersByID.
- AddParticipantToRoom: inserta en participants_by_room, rooms_by_user y membership_lookup en batch; inicializa counters.
- LeaveRoom: elimina rooms_by_user/membership_lookup/counters del usuario y registra en deleted_rooms_by_user; borra participants_by_room.
- UpdateParticipantRoom: actualiza role en participants_by_room y rooms_by_user.

Mensajes
- SaveMessage: genera timeuuid, inserta en messages_by_room + índices auxiliares (room_by_message, message_by_sender_message_id). Fan-out: reescribe fila en rooms_by_user por participante (patrón delete+insert) y actualiza membership_lookup.last_message_at. Counters/unread y message_status_by_user por usuario.
- GetMessage: busca room_id por message_id y recupera detalles. Enriquecimiento con UserFetcher y status por usuario.
- GetMessagesFromRoom (single room): filtros de paginación por message_id (antes/después), límites y enriquecimiento de usuario y status. All rooms: hace fan-out concurrente por sala con límite configurable y merge ordenado por fecha.
- ReactToMessage: inserta/elimina en reactions_by_message.
- Marcado de lectura: inserta en read_receipts_by_message y pone status READ en message_status_by_user; resetea counters a 0.

Borrado de sala
- Elimina por partición: participants_by_room, room_details, messages_by_room, y limpia vistas por usuario (rooms_by_user/membership_lookup/counters) con batches por usuario.

Consistencia y rendimiento
- Modelo de lectura-escritura optimizado: operaciones delete+insert para mantener claves de clustering ordenadas (room ordering por last_message_at, pin, etc.).
- Uso de batches por partición. Evita batches multi-partición costosos.

Cache
- Reutiliza helpers de cache para invalidar por sala cuando cambian datos.