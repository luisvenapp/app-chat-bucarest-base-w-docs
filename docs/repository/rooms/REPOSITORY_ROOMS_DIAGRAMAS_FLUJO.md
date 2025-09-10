# Diagramas de flujo — repository/rooms

## Crear sala (Postgres)
```mermaid
flowchart TD
    A[CreateRoom] --> B{type == p2p?}
    B -- Sí --> C[Buscar sala existente con partner]
    C -- Encontrada --> R[Retornar sala]
    C -- No encontrada --> D[Generar encryptionData]
    B -- No --> D
    D --> E[INSERT room]
    E --> F[INSERT room_member (owner y miembros)]
    F --> G[Commit]
    G --> H[Si p2p, cargar partner]
    H --> I[Si group, cargar participantes]
    I --> J[return Room]
```

## Obtener sala (GetRoom)
```mermaid
flowchart TD
    A[GetRoom(userId, roomId, allData)] --> B[Consultar room + mm + partner + último mensaje]
    B --> C[Calcular unread_count]
    C --> D[Si group y allData -> cargar participantes]
    D --> E[FormatRoom y cachear]
```

## Guardar mensaje
```mermaid
flowchart TD
    A[SaveMessage] --> B[Tx: INSERT room_message]
    B --> C[INSERT tags si hay]
    C --> D[INSERT meta para remitente]
    D --> E[Commit]
    E --> F[GetMessage]
    F --> G[UpdateRoomCacheWithNewMessage]
```

## GetMessagesFromRoom (paginación y filtros)
```mermaid
flowchart TD
    A[GetMessagesFromRoom] --> B[Construir query con filtros]
    B --> C[JOINs para reply/forward]
    C --> D[Aplicar row_number si MessagesPerRoom>0]
    D --> E[Traer tags y reacciones en lote]
    E --> F[Armar meta y retornar]
```

## ScyllaDB repositorio (altas luces)
```mermaid
flowchart TD
    A[CreateRoom] --> B[Anti-duplicado P2P en p2p_room_by_users]
    B --> C[Batch inserts en room_details/participants/...]
    C --> D[Inicializar contadores]
    D --> E[Si channel y join_all_user, agregar usuarios en background]

    F[SaveMessage] --> G[Batch insertar y fan-out rooms_by_user]
    G --> H[Actualizar counters y statuses]
```
