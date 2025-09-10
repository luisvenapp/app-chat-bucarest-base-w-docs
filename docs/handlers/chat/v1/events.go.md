# handlers/chat/v1/events.go

Propósito
- Adaptador de eventos para NATS/JetStream.

Puntos clave
- ChatEvent.Subject():
  - RoomJoin -> CHAT_DIRECT_EVENTS.<userId>
  - Otros -> CHAT_EVENTS.<roomId>
- JetStream(): true
- Payload():
  - Serializa protobuf (MessageEvent) y lo envuelve en JSON con user_id para trazabilidad.
- EventType(): "ChatEvent"