# handlers/chat/v1/init_stream.go

Constantes y configuración de JetStream
- StreamChatEventsName = "CHAT_EVENTS"
- StreamChatDirectEventsSubjectPrefix = "CHAT_DIRECT_EVENTS"
- requiredStreams: define el stream CHAT_EVENTS con:
  - Subjects:
    - CHAT_EVENTS.* (eventos por sala: CHAT_EVENTS.<roomId>)
    - CHAT_DIRECT_EVENTS.* (eventos dirigidos por usuario)
  - Storage: File, Retention: Limits, MaxMsgsPerSubject: 1000, Compression S2, MaxAge 7 días.

Helpers de subjects
- chatRoomEventSubject(roomId) => CHAT_EVENTS.<roomId>
- chatDirectEventSubject(userId) => CHAT_DIRECT_EVENTS.<userId>

Notas
- Esta configuración permite reentrega y consumo duradero por cliente (durable consumers), útil para reconexiones de apps.