# Diagramas de flujo — repository

```mermaid
flowchart TD
    A[rooms] --> B[SQLRoomRepository]
    A --> C[ScyllaRoomRepository]
    D[tokens] --> E[SQLTokensRepository]
```
