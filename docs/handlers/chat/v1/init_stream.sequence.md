sequenceDiagram
    participant Client
    participant Handler as Chat Handler
    participant NATS as NATS JetStream

    Client->>Handler: StreamMessages(room_id?, client_id)
    note over Handler: Register stream in StreamManager
    Handler->>NATS: Consume CHAT_DIRECT_EVENTS.<userId>
    alt room_id specific
        Handler->>NATS: Consume CHAT_EVENTS.<roomId>
    else all rooms
        Handler->>NATS: Consume CHAT_EVENTS.<allowed rooms>
    end
    loop Heartbeat 15s
        Handler-->>Client: MessageEvent{ connected: true }
    end

    note over NATS: ChatEvent published by publishChatEvent
    NATS-->>Handler: eventPayload{ user_id, payload(proto) }
    Handler->>Handler: handleJetStreamMessage()
    Handler-->>Client: MessageEvent (filtered/enriched)
