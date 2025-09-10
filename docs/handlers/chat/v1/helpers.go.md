# handlers/chat/v1/helpers.go

Función clave
- publishChatEvent(generalParams, roomID, event)
  - Enriquecer: genera EventId (uuid), completa RoomId si falta y preserva Room.
  - Log estructurado con datos relevantes (roomID, tipo de evento, userID de quien lo dispara, clientID).
  - Despacha con dispatcher.Dispatch(ChatEvent{ roomID, userID=session.UserID, event }).

Contexto
- El dispatcher envía a NATS JetStream usando el subject que determina ChatEvent.Subject().
- Esto desacopla la lógica de negocio del detalle de publicación/serialización.