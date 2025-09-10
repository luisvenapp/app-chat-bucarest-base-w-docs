package chatv1handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"time"

	"connectrpc.com/connect"
	notificationsv1client "github.com/Venqis-NolaTech/campaing-app-notifications-api-go/proto/generated/services/notifications/v1/client"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/database"
	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1/chatv1connect"
	roomsrepository "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/repository/rooms"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
	natsmanager "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/broker/nats"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/events"
	notificationsv1 "github.com/Venqis-NolaTech/campaing-app-notifications-api-go/proto/generated/services/notifications/v1"
)

type handlerImpl struct {
	logger          *slog.Logger
	nc              *nats.Conn                                 // Cliente de NATS
	js              jetstream.JetStream                        // Nuevo cliente de JetStream
	sm              *events.StreamManager[chatv1.MessageEvent] // Gestor de streams para la instancia actual
	dispatcher      *events.EventDispatcher
	roomsRepository roomsrepository.RoomsRepository
}

// NewHandler crea una nueva instancia del manejador del servicio de chat.
func NewHandler() chatv1connect.ChatServiceHandler {
	nm, err := natsmanager.Get()
	if err != nil {
		log.Fatal(err)
	}

	nc := nm.GetConn()
	if nc == nil {
		log.Fatal("No existe una conexión hacia NATS inicializada")
	}

	// Verificar que la conexión esté conectada
	if !nc.IsConnected() {
		log.Fatal("La conexión hacia NATS no está conectada")
	}

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	// Inicializar streams de NATS
	if err := natsmanager.EnsureStreams(context.Background(), js, requiredStreams...); err != nil {
		log.Printf("Error al inicializar streams de NATS: %v", err)
		// No fatal, pero registramos el error
	}

	logger := slog.Default()
	repo := roomsrepository.NewSQLRoomRepository(database.DB())
	if scylladb, _ := strconv.ParseBool(os.Getenv("USE_SCYLLADB")); scylladb {
		repo = roomsrepository.NewScyllaRoomRepository(database.CQLDB(), repo)
	}
	dispatcher, err := events.NewEventDispatcher(nc, logger, 5)
	if err != nil {
		log.Fatalf("Failed to create event dispatcher: %v", err)
	}

	return &handlerImpl{
		logger:          logger,
		sm:              events.NewStreamManager[chatv1.MessageEvent](logger),
		nc:              nc,
		js:              js,
		roomsRepository: repo,
		dispatcher:      dispatcher,
	}
}

// CreateRoom implements chatv1connect.ChatServiceHandler.
func (h *handlerImpl) CreateRoom(ctx context.Context, req *connect.Request[chatv1.CreateRoomRequest]) (*connect.Response[chatv1.CreateRoomResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	if len(req.Msg.Participants) < 1 {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}
	if len(req.Msg.Participants) > 1 && req.Msg.Type == "p2p" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}
	if req.Msg.Type == "group" && req.Msg.Name == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}

	room, err := h.roomsRepository.CreateRoom(ctx, userID, req.Msg)
	if err != nil {
		return nil, err
	}

	room = utils.FormatRoom(room)

	generalParams, _ := api.GeneralParamsFromConnectRequest(req)

	req.Msg.Participants = append(req.Msg.Participants, int32(userID))

	joinedAt := time.Now().UTC().Format(time.RFC3339)
	for _, id := range req.Msg.Participants {
		newRoomObject := proto.Clone(room).(*chatv1.Room)
		if id == int32(userID) {
			newRoomObject.Role = "OWNER"
		} else {
			newRoomObject.Role = "MEMBER"
		}
		event := &chatv1.MessageEvent{
			RoomId: room.Id,
			Event: &chatv1.MessageEvent_RoomJoin{RoomJoin: &chatv1.RoomJoinEvent{
				JoinedAt:    joinedAt,
				UserId:      id,
				OwnerUserId: int32(userID),
			}},
		}

		h.publishChatEvent(generalParams, room.GetId(), event)
	}
	room.Role = "OWNER"

	//subcribirse al topico del grupo
	if room.Type == "group" {
		//agregando al usuario principal
		req.Msg.Participants = append(req.Msg.Participants, int32(userID))

		if _, err := notificationsv1client.SubscribeToTopic(context.Background(), generalParams, &notificationsv1.SubscribeToTopicRequest{
			Event: &notificationsv1.SubscribeToTopicRequest_Data{
				Data: &notificationsv1.SubscribeToTopic{
					Topic:   "room-" + room.Id,
					UserIds: req.Msg.Participants,
				},
			},
		}); err != nil {
			h.logger.Error("Error enviando subscripcion al topico", "error", err)
		}
	}

	return connect.NewResponse(&chatv1.CreateRoomResponse{
		Success: true,
		Room:    room,
	}), nil
}

func (h *handlerImpl) GetRooms(ctx context.Context, req *connect.Request[chatv1.GetRoomsRequest]) (*connect.Response[chatv1.GetRoomsResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	rooms, meta, err := h.roomsRepository.GetRoomList(ctx, userID, req.Msg)
	if err != nil {
		return nil, err
	}

	for i, room := range rooms {
		rooms[i] = utils.FormatRoom(room)
	}

	roomsResponse := &chatv1.GetRoomsResponse{
		Items: rooms,
		Meta:  meta,
	}

	return connect.NewResponse(roomsResponse), nil
}

func (h *handlerImpl) GetRoom(ctx context.Context, req *connect.Request[chatv1.GetRoomRequest]) (*connect.Response[chatv1.GetRoomResponse], error) {
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, true, false)
	if err != nil {
		return nil, err
	}

	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	room = utils.FormatRoom(room)

	return connect.NewResponse(&chatv1.GetRoomResponse{
		Success: true,
		Room:    room,
	}), nil
}

func (h *handlerImpl) LeaveRoom(ctx context.Context, req *connect.Request[chatv1.LeaveRoomRequest]) (*connect.Response[chatv1.LeaveRoomResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	generalParams, _ := api.GeneralParamsFromConnectRequest(req)

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, true, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	switch room.Type {
	case "p2p":
		req.Msg.LeaveAll = true
	case "group":
		if room.Role == "MEMBER" && len(req.Msg.Participants) > 0 {
			return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
		}

		if req.Msg.LeaveAll && room.Role == "MEMBER" {
			return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
		}

		if len(req.Msg.Participants) == 0 {
			req.Msg.Participants = []int32{int32(userID)}
		}
	}

	if req.Msg.LeaveAll {
		participants, _, err := h.roomsRepository.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{
			Id: req.Msg.Id,
		})
		if err != nil {
			return nil, err
		}
		for _, user := range participants {
			req.Msg.Participants = append(req.Msg.Participants, user.Id)
		}
	}

	req.Msg.Participants = slices.Compact(req.Msg.Participants)

	users, err := h.roomsRepository.LeaveRoom(ctx, userID, req.Msg.Id, req.Msg.Participants, req.Msg.LeaveAll)
	if err != nil {
		return nil, err
	}

	if !req.Msg.LeaveAll {
		for _, user := range users {

			msg, err := h.roomsRepository.SaveMessage(ctx, userID, &chatv1.SendMessageRequest{
				RoomId:  room.Id,
				Content: user.Phone,
				Type:    "system_message",
				Event:   proto.String("remove_member"),
			}, nil, nil)
			if err != nil {
				return nil, err
			}

			event := &chatv1.MessageEvent{
				Event: &chatv1.MessageEvent_Message{
					Message: msg,
				},
			}

			h.publishChatEvent(generalParams, room.GetId(), event)
		}
	}

	//desuscribirse del topico
	if _, err := notificationsv1client.UnsubscribeFromTopic(context.Background(), generalParams, &notificationsv1.UnsubscribeFromTopicRequest{
		Event: &notificationsv1.UnsubscribeFromTopicRequest_Data{
			Data: &notificationsv1.UnsubscribeFromTopic{
				Topic:   "room-" + room.Id,
				UserIds: req.Msg.Participants,
			},
		},
	}); err != nil {
		h.logger.Error("Error enviando subscripcion al topico", "error", err)
	}

	if req.Msg.LeaveAll {
		err = h.roomsRepository.DeleteRoom(ctx, userID, req.Msg.Id, nil)
		if err != nil {
			return nil, err
		}
	}

	event := &chatv1.MessageEvent{
		Event: &chatv1.MessageEvent_RoomLeave{RoomLeave: &chatv1.RoomLeaveEvent{
			UsersId: req.Msg.Participants,
		}},
	}

	h.publishChatEvent(generalParams, room.GetId(), event)

	return connect.NewResponse(&chatv1.LeaveRoomResponse{
		Success: true,
	}), nil

}

func (h *handlerImpl) GetRoomParticipants(ctx context.Context, req *connect.Request[chatv1.GetRoomParticipantsRequest]) (*connect.Response[chatv1.GetRoomParticipantsResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	participants, meta, err := h.roomsRepository.GetRoomParticipants(ctx, req.Msg)
	if err != nil {
		return nil, err
	}

	participantsResponse := &chatv1.GetRoomParticipantsResponse{
		Participants: participants,
		Meta:         meta,
	}

	return connect.NewResponse(participantsResponse), nil
}

// PinRoom implements chatv1connect.ChatServiceHandler.
func (h *handlerImpl) PinRoom(ctx context.Context, req *connect.Request[chatv1.PinRoomRequest]) (*connect.Response[chatv1.PinRoomResponse], error) {

	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	err = h.roomsRepository.PinRoom(ctx, userID, req.Msg.Id, !room.IsPinned)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&chatv1.PinRoomResponse{Success: true}), nil
}

func (h *handlerImpl) MuteRoom(ctx context.Context, req *connect.Request[chatv1.MuteRoomRequest]) (*connect.Response[chatv1.MuteRoomResponse], error) {
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	room.IsMuted = !room.IsMuted

	err = h.roomsRepository.MuteRoom(ctx, userID, req.Msg.Id, room.IsMuted)
	if err != nil {
		return nil, err
	}

	generalParams, _ := api.GeneralParamsFromConnectRequest(req)

	if room.Type == "group" {
		if room.IsMuted {
			//desuscribirse del topico
			if _, err := notificationsv1client.UnsubscribeFromTopic(context.Background(), generalParams, &notificationsv1.UnsubscribeFromTopicRequest{
				Event: &notificationsv1.UnsubscribeFromTopicRequest_Data{
					Data: &notificationsv1.UnsubscribeFromTopic{
						Topic:   "room-" + room.Id,
						UserIds: []int32{int32(userID)},
					},
				},
			}); err != nil {
				h.logger.Error("Error enviando subscripcion al topico", "error", err)
			}
		} else {
			//suscribirse al topico
			if _, err := notificationsv1client.SubscribeToTopic(context.Background(), generalParams, &notificationsv1.SubscribeToTopicRequest{
				Event: &notificationsv1.SubscribeToTopicRequest_Data{
					Data: &notificationsv1.SubscribeToTopic{
						Topic:   "room-" + room.Id,
						UserIds: []int32{int32(userID)},
					},
				},
			}); err != nil {
				h.logger.Error("Error enviando subscripcion al topico", "error", err)
			}
		}
	}

	event := &chatv1.MessageEvent{
		RoomId: room.Id,
		Event:  &chatv1.MessageEvent_IsRoomUpdated{IsRoomUpdated: true},
	}

	h.publishChatEvent(generalParams, room.GetId(), event)

	return connect.NewResponse(&chatv1.MuteRoomResponse{Success: true}), nil
}

func (h *handlerImpl) UpdateRoom(ctx context.Context, req *connect.Request[chatv1.UpdateRoomRequest]) (*connect.Response[chatv1.UpdateRoomResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}
	if room.Type != "group" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}
	if room.Role == "MEMBER" && !room.EditGroup {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	err = h.roomsRepository.UpdateRoom(ctx, userID, room.Id, req.Msg)
	if err != nil {
		return nil, err
	}

	roomNew, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, false)
	if err != nil {
		return nil, err
	}

	generalParams, _ := api.GeneralParamsFromConnectRequest(req)

	event := &chatv1.MessageEvent{
		RoomId: roomNew.Id,
		Event:  &chatv1.MessageEvent_IsRoomUpdated{IsRoomUpdated: true},
	}

	h.publishChatEvent(generalParams, room.GetId(), event)

	// mensaje de sistema para notificar cambio de nombre
	if req.Msg.Name != nil && *req.Msg.Name != room.Name {
		msg, err := h.roomsRepository.SaveMessage(ctx, userID, &chatv1.SendMessageRequest{
			RoomId:  roomNew.Id,
			Content: *req.Msg.Name,
			Type:    "system_message",
			Event:   proto.String("new_name"),
		}, nil, nil)
		if err != nil {
			return nil, err
		}

		event = &chatv1.MessageEvent{
			RoomId: roomNew.Id,
			Event: &chatv1.MessageEvent_Message{
				Message: msg,
			},
		}

		h.publishChatEvent(generalParams, room.GetId(), event)
	}

	// mensaje de sistema para notificar cambio de foto
	if req.Msg.PhotoUrl != nil && *req.Msg.PhotoUrl != room.PhotoUrl {
		msg, err := h.roomsRepository.SaveMessage(ctx, userID, &chatv1.SendMessageRequest{
			RoomId:  roomNew.Id,
			Content: *req.Msg.PhotoUrl,
			Type:    "system_message",
			Event:   proto.String("new_photo"),
		}, nil, nil)
		if err != nil {
			return nil, err
		}

		event = &chatv1.MessageEvent{
			RoomId: roomNew.Id,
			Event: &chatv1.MessageEvent_Message{
				Message: msg,
			},
		}

		h.publishChatEvent(generalParams, room.GetId(), event)
	}

	return connect.NewResponse(&chatv1.UpdateRoomResponse{Success: true}), nil
}

func (h *handlerImpl) AddParticipantToRoom(ctx context.Context, req *connect.Request[chatv1.AddParticipantToRoomRequest]) (*connect.Response[chatv1.AddParticipantToRoomResponse], error) {

	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	if room.Type != "group" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}
	if room.Role == "MEMBER" && !room.AddMember {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	var participants []int
	for _, id := range req.Msg.Participants {
		participants = append(participants, int(id))
	}
	participantsData, err := h.roomsRepository.AddParticipantToRoom(ctx, userID, req.Msg.Id, participants)
	if err != nil {
		return nil, err
	}

	generalParams, _ := api.GeneralParamsFromConnectRequest(req)

	joinedAt := time.Now().UTC().Format(time.RFC3339)
	for _, participant := range participantsData {
		//crear mensaje de notificacion

		msg, err := h.roomsRepository.SaveMessage(ctx, userID, &chatv1.SendMessageRequest{
			RoomId:  room.Id,
			Content: participant.Phone,
			Type:    "system_message",
			Event:   proto.String("new_member"),
		}, nil, nil)
		if err != nil {
			return nil, err
		}

		event := &chatv1.MessageEvent{
			RoomId: room.Id,
			Event: &chatv1.MessageEvent_Message{
				Message: msg,
			},
		}

		h.publishChatEvent(generalParams, msg.RoomId, event)

		room.Role = "MEMBER"
		event = &chatv1.MessageEvent{
			RoomId: room.Id,
			Event: &chatv1.MessageEvent_RoomJoin{RoomJoin: &chatv1.RoomJoinEvent{
				JoinedAt: joinedAt,
				UserId:   int32(participant.ID),
			}},
		}

		h.publishChatEvent(generalParams, room.GetId(), event)
	}

	//suscribirse al topico
	if _, err := notificationsv1client.SubscribeToTopic(context.Background(), generalParams, &notificationsv1.SubscribeToTopicRequest{
		Event: &notificationsv1.SubscribeToTopicRequest_Data{
			Data: &notificationsv1.SubscribeToTopic{
				Topic:   "room-" + room.Id,
				UserIds: req.Msg.Participants,
			},
		},
	}); err != nil {
		h.logger.Error("Error enviando subscripcion al topico", "error", err)
	}

	return connect.NewResponse(&chatv1.AddParticipantToRoomResponse{Success: true}), nil
}

func (h *handlerImpl) UpdateParticipantRoom(ctx context.Context, req *connect.Request[chatv1.UpdateParticipantRoomRequest]) (*connect.Response[chatv1.UpdateParticipantRoomResponse], error) {
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}

	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	if room.Role == "MEMBER" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	err = h.roomsRepository.UpdateParticipantRoom(ctx, userID, req.Msg)
	if err != nil {
		return nil, err
	}

	generalParams, _ := api.GeneralParamsFromConnectRequest(req)

	event := &chatv1.MessageEvent{
		RoomId: room.Id,
		Event:  &chatv1.MessageEvent_IsRoomUpdated{IsRoomUpdated: true},
	}

	h.publishChatEvent(generalParams, room.GetId(), event)

	return connect.NewResponse(&chatv1.UpdateParticipantRoomResponse{Success: true}), nil
}

func (h *handlerImpl) BlockUser(ctx context.Context, req *connect.Request[chatv1.BlockUserRequest]) (*connect.Response[chatv1.BlockUserResponse], error) {

	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}

	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	if room.Type != "p2p" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	partnerID := int(room.Partner.Id)
	err = h.roomsRepository.BlockUser(ctx, userID, req.Msg.Id, !room.IsPartnerBlocked, &partnerID)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&chatv1.BlockUserResponse{Success: true}), nil
}

// SendMessage implementa la lógica para enviar un mensaje.
func (h *handlerImpl) SendMessage(ctx context.Context, req *connect.Request[chatv1.SendMessageRequest]) (*connect.Response[chatv1.SendMessageResponse], error) {
	generalParams, err := api.GeneralParamsFromConnectRequest(req)
	if err != nil {
		return nil, err
	}

	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	if req.Msg.RoomId == "" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.RoomId, false, true)
	if err != nil {
		return nil, err
	}

	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	room = utils.FormatRoom(room)

	if room.Type == "group" && room.Role == "MEMBER" && !room.SendMessage {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}

	if len(req.Msg.Mentions) > 0 && room.Type == "p2p" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}

	if room.Type == "p2p" && room.IsPartnerBlocked {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}

	var contentDecrypted string
	if req.Msg.Content != "" {
		contentDecrypted, err = utils.DecryptMessage(req.Msg.Content, room.EncryptionData)
		if err != nil {
			h.logger.Error("Error al desencriptar el contenido", "error", err)
		}
	}

	req.Msg.Type = "user_message"

	msg, err := h.roomsRepository.SaveMessage(ctx, userID, req.Msg, room, &contentDecrypted)
	if err != nil {
		return nil, err
	}

	h.dispatcher.Dispatch(context.Background(), events.FanoutEvent{
		OnFanount: func(ctx context.Context, event events.FanoutEvent) {
			//TODO: Revisar si esto es necesario, porque si son muchos participantes, puede tardar.
			//la meta en realidad deberia ser solo para revisar mensajes vistos
			//sino hay meta se puede suponer que el mensaje no se ha leido...
			err := h.roomsRepository.CreateMessageMetaForParticipants(ctx, room.Id, msg.Id, int(msg.SenderId))
			if err != nil {
				// Consider adding a retry mechanism or pushing to a dead-letter queue for this as well.
				h.logger.Error("Failed to handle message fanout event", "error", err, "roomID", room.Id, "messageID", msg.Id)
			} else {
				h.logger.Info("Successfully fanned out message metadata", "roomID", msg.Id, "messageID", msg.SenderId)
			}

			senderEvent := &chatv1.MessageEvent{
				RoomId: room.Id,
				Event: &chatv1.MessageEvent_StatusUpdate{
					StatusUpdate: &chatv1.MessageStatusUpdate{
						MessageId: msg.GetId(),
						Status:    msg.GetStatus(),
						UpdatedAt: msg.GetUpdatedAt(),
						UserId:    int32(userID),
						SenderId:  int32(userID),
					},
				},
			}
			remitentsEvent := &chatv1.MessageEvent{
				RoomId: room.Id,
				Event: &chatv1.MessageEvent_Message{
					Message: msg,
				},
			}

			h.publishChatEvent(generalParams, msg.RoomId, senderEvent)
			h.publishChatEvent(generalParams, msg.RoomId, remitentsEvent)

			var participantsIds []int32
			if room.Type == "p2p" {
				participantsIds = append(participantsIds, int32(room.Partner.Id))
			} else {
				/*participants, _, err := h.roomsRepository.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{
					Id: room.Id,
				})
				if err != nil {
					return
				}
				for _, participant := range participants {
					if int32(msg.SenderId) == participant.GetId() {
						continue
					}
					participantsIds = append(participantsIds, participant.GetId())
				}*/
			}

			sendPushNotification := true

			//si el partner esta muteado, no se envía la notificación
			if room.Type == "p2p" {
				isPartnerMuted, err := h.roomsRepository.IsPartnerMuted(ctx, int(room.Partner.Id), room.Id)
				if err != nil {
					fmt.Println("Error al obtener si el partner esta muteado", err)
				}
				if isPartnerMuted {
					sendPushNotification = false
				}
			}

			if sendPushNotification {

				if _, err := notificationsv1client.SendPushNotificationEvent(context.Background(), generalParams, &notificationsv1.SendPushNotificationRequest{
					Event: &notificationsv1.SendPushNotificationRequest_ChatMessage{
						ChatMessage: &notificationsv1.ChatMessagePushEvent{
							RecipientsUserId:  participantsIds,
							SenderId:          int32(userID),
							SenderDisplayName: msg.SenderName,
							RoomName:          room.Name,
							RoomId:            room.Id,
							RoomType:          room.Type,
							MessageContent:    contentDecrypted,
						},
					},
				}); err != nil {
					h.logger.Error("Error enviando notificación push del mensaje", "error", err)
				}
			}
		},
	})

	response := &chatv1.SendMessageResponse{
		Success: true,
		Message: msg,
	}
	return connect.NewResponse(response), nil
}

// EditMessage implementa la lógica para editar un mensaje.
func (h *handlerImpl) EditMessage(ctx context.Context, req *connect.Request[chatv1.EditMessageRequest]) (*connect.Response[chatv1.EditMessageResponse], error) {
	generalParams, err := api.GeneralParamsFromConnectRequest(req)
	if err != nil {
		return nil, err
	}

	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.UnauthorizedCode, req.Header())
	}

	message, err := h.roomsRepository.GetMessage(ctx, userID, req.Msg.MessageId)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}
	if message == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	if message.SenderId != int32(userID) {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.UnauthorizedCode, req.Header())
	}

	err = h.roomsRepository.UpdateMessage(ctx, userID, req.Msg.MessageId, req.Msg.NewContent)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	message.Content = req.Msg.NewContent
	message.Edited = true

	event := &chatv1.MessageEvent{
		RoomId: message.RoomId,
		Event:  &chatv1.MessageEvent_UpdateMessage{UpdateMessage: message},
	}

	h.publishChatEvent(generalParams, message.RoomId, event)

	response := &chatv1.EditMessageResponse{
		Success: true,
		Message: message,
	}
	return connect.NewResponse(response), nil
}

// DeleteMessage implementa la lógica para eliminar un mensaje.
func (h *handlerImpl) DeleteMessage(ctx context.Context, req *connect.Request[chatv1.DeleteMessageRequest]) (*connect.Response[chatv1.DeleteMessageResponse], error) {
	generalParams, err := api.GeneralParamsFromConnectRequest(req)
	if err != nil {
		return nil, err
	}

	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.UnauthorizedCode, req.Header())
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.RoomId, false, true)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	err = h.roomsRepository.DeleteMessage(ctx, userID, req.Msg.MessageIds)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("no se pudo eliminar el mensaje: %w", err))
	}

	for i := 0; i < len(req.Msg.MessageIds); i++ {

		event := &chatv1.MessageEvent{
			Event: &chatv1.MessageEvent_DeleteMessage{DeleteMessage: req.Msg.MessageIds[i]},
		}

		h.publishChatEvent(generalParams, req.Msg.RoomId, event)

	}

	response := &chatv1.DeleteMessageResponse{Success: true}
	return connect.NewResponse(response), nil
}

func (h *handlerImpl) GetMessageHistory(ctx context.Context, req *connect.Request[chatv1.GetMessageHistoryRequest]) (*connect.Response[chatv1.GetMessageHistoryResponse], error) {

	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.UnauthorizedCode, req.Header())
	}

	if req.Msg.Id == "" {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InvalidRequestDataCode, req.Header())
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.Id, false, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	//if room.Role == "MEMBER" {
	//	return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	//}

	messages, meta, err := h.roomsRepository.GetMessagesFromRoom(ctx, userID, req.Msg)
	if err != nil {
		return nil, err
	}

	response := &chatv1.GetMessageHistoryResponse{
		Items: messages,
		Meta:  meta,
	}

	return connect.NewResponse(response), nil
}

func (h *handlerImpl) GetMessage(ctx context.Context, req *connect.Request[chatv1.GetMessageRequest]) (*connect.Response[chatv1.MessageData], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.UnauthorizedCode, req.Header())
	}

	message, err := h.roomsRepository.GetMessage(ctx, userID, req.Msg.Id)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if message == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, message.RoomId, false, true)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	return connect.NewResponse(message), nil
}

func (h *handlerImpl) ReactToMessage(ctx context.Context, req *connect.Request[chatv1.ReactToMessageRequest]) (*connect.Response[chatv1.ReactToMessageResponse], error) {

	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.UnauthorizedCode, req.Header())
	}

	message, err := h.roomsRepository.GetMessage(ctx, userID, req.Msg.MessageId)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if message == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, message.RoomId, false, true)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	err = h.roomsRepository.ReactToMessage(ctx, userID, req.Msg.MessageId, req.Msg.Reaction)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	return connect.NewResponse(&chatv1.ReactToMessageResponse{Success: true}), nil
}

func (h *handlerImpl) InitialSync(ctx context.Context, req *connect.Request[chatv1.InitialSyncRequest]) (*connect.Response[chatv1.InitialSyncResponse], error) {

	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.UnauthorizedCode, req.Header())
	}

	//get current timestamp
	now := time.Now()

	rooms, _, err := h.roomsRepository.GetRoomList(ctx, userID, &chatv1.GetRoomsRequest{
		Since: req.Msg.LastSyncTimestamp,
	})
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	roomsDeleted, err := h.roomsRepository.GetRoomListDeleted(ctx, userID, req.Msg.LastSyncTimestamp)
	if err != nil {
		fmt.Println("Error al obtener las salas eliminadas", err)
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	messages, _, err := h.roomsRepository.GetMessagesFromRoom(ctx, userID, &chatv1.GetMessageHistoryRequest{
		AfterDate:       &req.Msg.LastSyncTimestamp,
		MessagesPerRoom: uint32(req.Msg.MessagesPerRoom),
	})
	if err != nil {
		fmt.Println("Error al sync messages from room", err)
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	finish := time.Now()
	duration := finish.Sub(now).Milliseconds()
	durationString := strconv.FormatInt(duration, 10)

	nowString := now.UTC().Format("2006-01-02T15:04:05-07:00")

	return connect.NewResponse(&chatv1.InitialSyncResponse{
		Rooms:         rooms,
		RoomsDeleted:  roomsDeleted,
		Messages:      messages,
		SyncTimestamp: nowString,
		Summary: &chatv1.SyncSummary{
			RoomsSynced:    int32(len(rooms)),
			RoomsDeleted:   int32(len(roomsDeleted)),
			MessagesSynced: int32(len(messages)),
			SyncDurationMs: durationString,
		},
	}), nil
}

func (h *handlerImpl) MarkMessagesAsRead(ctx context.Context, req *connect.Request[chatv1.MarkMessagesAsReadRequest]) (*connect.Response[chatv1.MarkMessagesAsReadResponse], error) {
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	room, err := h.roomsRepository.GetRoom(ctx, userID, req.Msg.RoomId, false, true)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	var since string
	if len(req.Msg.MessageIds) > 0 {
		message, err := h.roomsRepository.GetMessageSimple(ctx, userID, req.Msg.MessageIds[0])
		if err != nil {
			return nil, err
		}
		if message == nil {
			return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
		}
		since = message.CreatedAt
	}

	markedCount, err := h.roomsRepository.MarkMessagesAsRead(ctx, userID, req.Msg.RoomId, req.Msg.MessageIds, since)
	if err != nil {
		return nil, err
	}

	generalParams, _ := api.GeneralParamsFromConnectRequest(req)

	readAt := time.Now().UTC().Format(time.RFC3339)
	for _, msgId := range req.Msg.MessageIds {
		var senderId int32
		msg, err := h.roomsRepository.GetMessageSimple(ctx, userID, msgId)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
		}
		senderId = msg.SenderId
		statusUpdate := &chatv1.MessageStatusUpdate{
			MessageId: msgId,
			UpdatedAt: readAt,
			UserId:    int32(userID),
			Status:    chatv1.MessageStatus_MESSAGE_STATUS_READ,
			SenderId:  senderId,
		}
		event := &chatv1.MessageEvent{
			Event: &chatv1.MessageEvent_StatusUpdate{
				StatusUpdate: statusUpdate,
			},
		}

		h.publishChatEvent(generalParams, room.GetId(), event)
	}

	return connect.NewResponse(&chatv1.MarkMessagesAsReadResponse{
		Success:     true,
		MarkedCount: markedCount,
	}), nil
}

func (h *handlerImpl) GetMessageRead(ctx context.Context, req *connect.Request[chatv1.GetMessageReadRequest]) (*connect.Response[chatv1.GetMessageReadResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	message, err := h.roomsRepository.GetMessageSimple(ctx, userID, req.Msg.Id)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if message == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	if message.SenderId != int32(userID) {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	items, meta, err := h.roomsRepository.GetMessageRead(ctx, req.Msg)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	return connect.NewResponse(&chatv1.GetMessageReadResponse{
		Items: items,
		Meta:  meta,
	}), nil

}

func (h *handlerImpl) GetMessageReactions(ctx context.Context, req *connect.Request[chatv1.GetMessageReactionsRequest]) (*connect.Response[chatv1.GetMessageReactionsResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	message, err := h.roomsRepository.GetMessageSimple(ctx, userID, req.Msg.Id)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if message == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	if message.SenderId != int32(userID) {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	items, meta, err := h.roomsRepository.GetMessageReactions(ctx, req.Msg)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	return connect.NewResponse(&chatv1.GetMessageReactionsResponse{
		Items: items,
		Meta:  meta,
	}), nil
}

func (h *handlerImpl) GetSenderMessage(ctx context.Context, req *connect.Request[chatv1.GetSenderMessageRequest]) (*connect.Response[chatv1.GetSenderMessageResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	message, err := h.roomsRepository.GetMessageSender(ctx, userID, req.Msg.SenderMessageId)
	if err != nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.InternalServerErrorCode, req.Header())
	}

	if message == nil {
		return nil, api.UpdateResponseInfoErrorMessageFromCode(api.NotFoundCode, req.Header())
	}

	return connect.NewResponse(&chatv1.GetSenderMessageResponse{
		Status: message.Status,
	}), nil
}

// StreamMessages gestiona una conexión de streaming para eventos en tiempo real.
// Si se proporciona un roomID, se suscribe solo a esa sala.
// Si no se proporciona roomID, se suscribe a todas las salas del usuario.
func (h *handlerImpl) StreamMessages(ctx context.Context, req *connect.Request[chatv1.StreamMessagesRequest], stream *connect.ServerStream[chatv1.MessageEvent]) error {
	generalParams, err := api.GeneralParamsFromConnectRequest(req)
	if err != nil {
		return err
	}

	session, err := api.CheckSessionFromGeneralParams(generalParams)
	if err != nil {
		return err
	}

	clientID := generalParams.ClientId
	if clientID == "" {
		err := api.UpdateResponseInfoErrorMessage(errors.New("client_id_needed"), req.Header())
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	h.sm.Register(generalParams, stream)
	defer h.sm.Unregister(generalParams) // Garantiza la limpieza cuando el cliente se desconecta.

	specificRoomID := req.Msg.GetRoomId()

	allowedRooms, _, err := h.roomsRepository.GetRoomList(ctx, session.UserID, nil)
	if err != nil {
		return fmt.Errorf("no se pudieron obtener las salas del usuario: %w", err)
	}

	var allowedRoomsIds []string
	for _, room := range allowedRooms {
		allowedRoomsIds = append(allowedRoomsIds, room.GetId())
	}

	roomsConsumers := map[string]jetstream.ConsumeContext{}

	defer (func() {
		for _, cons := range roomsConsumers {
			cons.Stop()
		}
	})()

	directConsumerKey := fmt.Sprintf("client-%s-direct", clientID)
	directConsumer, err := h.subscribeAndConsume(
		ctx,
		StreamChatEventsName,
		chatDirectEventSubject(session.UserID),
		directConsumerKey,
		generalParams,
		stream,
		roomsConsumers,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to direct events: %w", err)
	}
	roomsConsumers[directConsumerKey] = directConsumer

	if specificRoomID != "" && slices.Contains(allowedRoomsIds, specificRoomID) {
		h.logger.Info("Usuario suscribiéndose a una sala específica", "clientID", clientID, "roomID", specificRoomID)
		subject := chatRoomEventSubject(specificRoomID)

		roomConsumer, err := h.subscribeAndConsume(
			ctx,
			StreamChatEventsName,
			subject,
			fmt.Sprintf("client-%s-room-%s", clientID, specificRoomID),
			generalParams,
			stream,
			roomsConsumers,
		)
		if err != nil {
			return fmt.Errorf("failed to subscribe to room %s events: %w", specificRoomID, err)
		}
		roomsConsumers[specificRoomID] = roomConsumer
	} else {
		h.logger.Info("Usuario suscribiéndose a todas sus salas", "clientID", clientID)

		if len(allowedRooms) == 0 {
			h.logger.Warn("El usuario no pertenece a ninguna sala, esperando a que cree una o que ingrese en una", "clientID", clientID)
		}

		for _, roomID := range allowedRoomsIds {
			subject := chatRoomEventSubject(roomID)

			roomConsumer, err := h.subscribeAndConsume(
				ctx,
				StreamChatEventsName,
				subject,
				fmt.Sprintf("client-%s-room-%s", clientID, roomID),
				generalParams,
				stream,
				roomsConsumers,
			)
			if err != nil {
				return fmt.Errorf("failed to subscribe to room %s events: %w", roomID, err)
			}
			roomsConsumers[roomID] = roomConsumer
		}
	}

	h.logger.Info("Stream de usuario activo y escuchando eventos", "clientID", clientID)

	h.sm.Send(generalParams, &chatv1.MessageEvent{
		Event: &chatv1.MessageEvent_Connected{Connected: true},
	})

	ticker := time.NewTicker(15 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				h.sm.Send(generalParams, &chatv1.MessageEvent{
					Event: &chatv1.MessageEvent_Connected{Connected: true},
				})
			}
		}
	}()

	<-ctx.Done()
	done <- true

	h.logger.Info("Cliente desconectado, cerrando stream y desuscribiendo de NATS", "clientID", clientID)
	return nil
}

func (h handlerImpl) handleJetStreamMessage(
	ctx context.Context,
	generalParams api.GeneralParams,
	msg jetstream.Msg,
	stream *connect.ServerStream[chatv1.MessageEvent],
	roomsConsumers map[string]jetstream.ConsumeContext,
) {
	clientID := generalParams.ClientId
	session, _ := api.CheckSessionFromGeneralParams(generalParams)

	sendEvent := func(eventSubject string, event *chatv1.MessageEvent) {
		payload, _ := protojson.Marshal(event)
		h.logger.Info("Enviando evento al stream del usuario", "subject", eventSubject, "clientID", clientID, "roomID", event.RoomId, "payload", string(payload))
		h.sm.Send(generalParams, event)
	}

	var data eventPayload
	if err := json.Unmarshal(msg.Data(), &data); err != nil {
		h.logger.Error("Error al decodificar evento de NATS (JSON)", "error", err, "subject", msg.Subject())
		return
	}
	event := &chatv1.MessageEvent{}
	if err := proto.Unmarshal(data.Payload, event); err != nil {
		h.logger.Error("Error al decodificar evento de NATS (Proto)", "error", err, "subject", msg.Subject())
		return
	}
	dispatchUserId := data.UserId

	roomID := event.RoomId
	natsSubject := chatRoomEventSubject(roomID)

	if roomID == "" {
		return
	}

	switch detail := event.Event.(type) {
	case *chatv1.MessageEvent_RoomJoin:
		room, err := h.roomsRepository.GetRoom(context.Background(), session.UserID, roomID, true, true)
		if err != nil {
			h.logger.Error("Error fetching room", "error", err)
			return
		}
		event.Room = room
		if detail.RoomJoin.GetUserId() == int32(session.UserID) || data.UserId == session.UserID {
			durableName := fmt.Sprintf("client-%s-room-%s", clientID, roomID)

			if _, ok := roomsConsumers[roomID]; ok {
				h.logger.Info("Already subscribed to room, skipping", "roomID", roomID)
			} else {
				roomConsumer, err := h.subscribeAndConsume(
					ctx,
					StreamChatEventsName,
					natsSubject,
					durableName,
					generalParams,
					stream,
					roomsConsumers,
				)
				if err != nil {
					h.logger.Error("Failed to subscribe to new room on RoomJoin event", "error", err, "roomID", roomID)
				} else {
					roomsConsumers[roomID] = roomConsumer // Store it for cleanup
					h.logger.Info("Successfully subscribed to new room on RoomJoin event", "roomID", roomID)
				}
			}
		}
		sendEvent(msg.Subject(), event)

	case *chatv1.MessageEvent_IsRoomUpdated:
		room, err := h.roomsRepository.GetRoom(context.Background(), session.UserID, roomID, true, true)
		if err != nil {
			h.logger.Error("Error fetching room", "error", err)
			return
		}
		event.Room = room
		sendEvent(natsSubject, event)

	case *chatv1.MessageEvent_StatusUpdate:
		if detail.StatusUpdate.Status == chatv1.MessageStatus_MESSAGE_STATUS_SENT {
			if dispatchUserId == session.UserID {
				sendEvent(natsSubject, event)
			}
		} else {
			sendEvent(natsSubject, event)
		}

	case *chatv1.MessageEvent_RoomLeave:
		sendEvent(natsSubject, event)

		if slices.Contains(detail.RoomLeave.GetUsersId(), int32(session.UserID)) {
			if consumer, ok := roomsConsumers[roomID]; ok {
				consumer.Stop()
				delete(roomsConsumers, roomID)
				h.logger.Info("Successfully unsubscribed from room on RoomLeave event", "roomID", roomID)
			} else {
				h.logger.Warn("Attempted to unsubscribe from a room not found in consumers map", "roomID", roomID)
			}
		}

	default:
		sendEvent(natsSubject, event)
	}
}

func (h *handlerImpl) subscribeAndConsume(
	ctx context.Context,
	streamName string,
	filterSubject string,
	durableName string,
	generalParams api.GeneralParams,
	stream *connect.ServerStream[chatv1.MessageEvent],
	roomsConsumers map[string]jetstream.ConsumeContext,
) (jetstream.ConsumeContext, error) {
	consumerConfig := jetstream.ConsumerConfig{
		Durable:       durableName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		FilterSubject: filterSubject,
	}

	// Create or update the consumer
	cons, err := h.js.CreateOrUpdateConsumer(ctx, streamName, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update consumer for subject %s: %w", filterSubject, err)
	}

	consumeCtx, err := cons.Consume(func(msg jetstream.Msg) {
		h.handleJetStreamMessage(ctx, generalParams, msg, stream, roomsConsumers)
		msg.Ack()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming messages for subject %s: %w", filterSubject, err)
	}

	return consumeCtx, nil
}
