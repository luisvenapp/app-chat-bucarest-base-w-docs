package roomsrepository

import (
	"context"

	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
)

type User struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Phone     string  `json:"phone"`
	Email     *string `json:"email"`
	Avatar    *string `json:"avatar"`
	Dni       *string `json:"dni"`
	CreatedAt *string `json:"created_at"`
}

type RoomsRepository interface {
	UserFetcher
	CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error)
	GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error)
	GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error)
	GetRoomListDeleted(ctx context.Context, userId int, since string) ([]string, error)
	LeaveRoom(ctx context.Context, userId int, roomId string, participants []int32, leaveAll bool) ([]User, error)
	DeleteRoom(ctx context.Context, userId int, roomId string, partner *int) error
	GetRoomParticipants(ctx context.Context, pagination *chatv1.GetRoomParticipantsRequest) ([]*chatv1.RoomParticipant, *chatv1.PaginationMeta, error)
	PinRoom(ctx context.Context, userId int, roomId string, pin bool) error
	MuteRoom(ctx context.Context, userId int, roomId string, mute bool) error
	BlockUser(ctx context.Context, userId int, roomId string, block bool, partner *int) error
	UpdateRoom(ctx context.Context, userId int, roomId string, room *chatv1.UpdateRoomRequest) error
	AddParticipantToRoom(ctx context.Context, userId int, roomId string, participants []int) ([]User, error)
	UpdateParticipantRoom(ctx context.Context, userId int, req *chatv1.UpdateParticipantRoomRequest) error
	SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error)
	GetMessage(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error)
	GetMessageSimple(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error)
	UpdateMessage(ctx context.Context, userId int, messageId string, content string) error
	DeleteMessage(ctx context.Context, userId int, messageId []string) error
	ReactToMessage(ctx context.Context, userId int, messageId string, reaction string) error
	GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error)
	MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error)
	GetMessageRead(ctx context.Context, req *chatv1.GetMessageReadRequest) ([]*chatv1.MessageUserRead, *chatv1.PaginationMeta, error)
	GetMessageReactions(ctx context.Context, req *chatv1.GetMessageReactionsRequest) ([]*chatv1.Reaction, *chatv1.PaginationMeta, error)
	GetUserByID(ctx context.Context, id int) (*User, error)
	GetAllUserIDs(ctx context.Context) ([]int, error)
	GetMessageSender(ctx context.Context, userId int, senderMessageId string) (*chatv1.MessageData, error)
	CreateMessageMetaForParticipants(ctx context.Context, roomID string, messageID string, senderID int) error
	IsPartnerMuted(ctx context.Context, userId int, roomId string) (bool, error)
}

type UserFetcher interface {
	GetUserByID(ctx context.Context, id int) (*User, error)
	GetUsersByID(ctx context.Context, ids []int) ([]User, error)
	GetAllUserIDs(ctx context.Context) ([]int, error)
}
