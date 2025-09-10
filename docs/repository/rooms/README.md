# Repositorios de Salas

- SQLRoomRepository: implementación PostgreSQL, consultas complejas con JOIN LATERAL para último mensaje y contadores.
- ScyllaRoomRepository: implementación ScyllaDB/Cassandra, modelo denormalizado optimizado para lecturas por usuario.

Conmutación
- Controlada por USE_SCYLLADB (env). Si está activo, se crea un repositorio Scylla que delega en SQL para algunas funciones vía UserFetcher.

Caché
- Se cachean representaciones de Room/Message simple por usuario/sala y se invalidan en operaciones de cambio.
