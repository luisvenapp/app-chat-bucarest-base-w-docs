package utils

import (
	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
)

func FormatRoom(room *chatv1.Room) *chatv1.Room {
	if room.Type == "p2p" && room.Partner != nil {
		room.PhotoUrl = room.Partner.Avatar
		room.Name = room.Partner.Name
	}
	if room.LastMessage != nil {
		room.LastMessageAt = room.LastMessage.CreatedAt
	}
	return room
}
