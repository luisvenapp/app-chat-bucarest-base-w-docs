# repository/rooms/room_cache.go

Objetivo
- Cachear representaciones de Room y MessageData para reducir carga de lecturas repetidas.

Claves
- Room cache (por usuario):
  - endpoint:chat:room:{<roomId>}:user:<userId> (vista completa)
  - endpoint:chat:room:{<roomId>}:shim:user:<userId> (vista ligera)
- Set de miembros por sala (para invalidación):
  - endpoint:chat:room:{<roomId>}:members

Funciones
- GetCachedRoom / SetCachedRoom: serializa a JSON (envoltura CachedRoomResponse) y TTL 1h. También mantiene set de miembros del cache por sala.
- UpdateRoomCacheWithNewMessage: actualiza atómicamente LastMessage/LastMessageAt en todas las entradas cacheadas de la sala (lock por roomId para concurrencia).
- Get/SetCachedMessageSimple: cachea MessageData parcial.
- DeleteRoomCacheByRoomID: borra todas las entradas de una sala y su set.
- DeleteCache: borra una clave arbitraria.

Notas
- El lock por sala (map[string]*sync.Mutex) evita condiciones de carrera al actualizar múltiples versiones cacheadas de una misma sala tras nuevos mensajes.