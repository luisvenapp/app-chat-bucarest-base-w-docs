package chatv1handler

import (
	"context"
	"fmt"

	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
	"github.com/google/uuid"
)

type MessageEvent struct {
	DispatcherUserID int
}

// publishChatEvent serializa y publica un evento en NATS.
func (h *handlerImpl) publishChatEvent(generalParams api.GeneralParams, roomID string, event *chatv1.MessageEvent) {
	h.logger.Info(
		"Dispatching chat event",
		"roomID", roomID,
		"eventType", fmt.Sprintf("%T", event.Event),
		"triggeredByUserID", generalParams.Session.UserID,
		"clientID", generalParams.ClientId,
	)

	eventVal := &chatv1.MessageEvent{
		EventId: uuid.NewString(),
		RoomId:  event.RoomId,
		Room:    event.Room,
		Event:   event.Event,
	}

	if eventVal.Room != nil && eventVal.RoomId == "" {
		eventVal.RoomId = event.Room.Id
	}

	if eventVal.RoomId == "" {
		eventVal.RoomId = roomID
	}

	// Luego, envío persistente a través del dispatcher
	h.dispatcher.Dispatch(context.Background(), ChatEvent{roomID: roomID, event: eventVal, userID: generalParams.Session.UserID})
}
