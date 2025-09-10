# repository/rooms/helpers.go

Funciones auxiliares
- sortUserIDs(id1, id2) -> ordena IDs para claves p2p consistentes (user1 < user2).
- removeAccents(s) -> normaliza cadenas removiendo acentos (NFD + eliminación de Mn + NFC). Útil para búsquedas case/diacritic-insensitive.

Uso
- Scylla: detección de duplicados P2P (tabla p2p_room_by_users) usa sortUserIDs.
- Búsqueda: normalizaciones para comparaciones sin acentos.