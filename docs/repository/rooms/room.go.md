# repository/rooms/room.go

Interfaz RoomsRepository
- Contrato que abstrae la persistencia/lectura de salas y mensajes. Implementaciones:
  - SQLRoomRepository (PostgreSQL)
  - ScyllaRoomRepository (ScyllaDB/Cassandra)

Responsabilidades
- Rooms: crear, obtener (con/ sin datos completos), listar (paginación, búsqueda), marcados (pin/mute), actualizar, bloquear usuario, borrar.
- Participantes: obtener, añadir, actualizar rol, salir (uno/varios/leaveAll), obtener IDs de usuarios.
- Mensajes: guardar (y fan-out de metadatos), obtener (simple/completo), editar, eliminar, reacciones, historial (multi filtro y sync), marcar como leídos, lecturas y reacciones.
- Utilidades: CreateMessageMetaForParticipants, IsPartnerMuted, GetMessageSender.

Tipos auxiliares
- User: estructura simple para enriquecer con datos de usuario desde servicio externo.
- UserFetcher: interfaz para obtener usuarios por IDs (inyectada en ScyllaRoomRepository).