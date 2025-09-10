package chatv1handler

import (
	"encoding/json"

	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
	"google.golang.org/protobuf/proto"
)

type ChatEvent struct {
	roomID string
	userID int
	event  *chatv1.MessageEvent
}

func (e ChatEvent) Subject() string {
	switch detail := e.event.Event.(type) {
	case *chatv1.MessageEvent_RoomJoin:
		return chatDirectEventSubject(int(detail.RoomJoin.UserId))
	default:
		return chatRoomEventSubject(e.roomID)
	}
}

func (ChatEvent) JetStream() bool {
	return true
}

type eventPayload struct {
	UserId  int    `json:"user_id"`
	Payload []byte `json:"payload"`
}

func (e ChatEvent) Payload() ([]byte, error) {
	payload, err := proto.Marshal(e.event)
	if err != nil {
		return nil, err
	}
	return json.Marshal(eventPayload{
		UserId:  e.userID,
		Payload: payload,
	})
}

func (e ChatEvent) EventType() string {
	return "ChatEvent"
}
