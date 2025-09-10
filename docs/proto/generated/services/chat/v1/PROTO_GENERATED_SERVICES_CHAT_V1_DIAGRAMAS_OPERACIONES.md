# Diagramas de flujo — proto/generated/services/chat/v1

```mermaid
flowchart TD
    A[Client ChatService] --> B[SendMessage/Edit/Delete/React]
    A --> C[GetRooms/Create/Get/Participants]
    A --> D[Pin/Mute/Leave/Block]
    A --> E[GetMessage(s)/Reactions/Read]
    A --> F[InitialSync/StreamMessages]
```
