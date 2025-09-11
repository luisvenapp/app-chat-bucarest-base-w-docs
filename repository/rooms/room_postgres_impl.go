package roomsrepository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"slices"
	"time"

	sq "github.com/Masterminds/squirrel"
	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
	dbpq "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/postgres"
)

type SQLRoomRepository struct {
	db *sql.DB
}

func NewSQLRoomRepository(db *sql.DB) RoomsRepository {
	return &SQLRoomRepository{
		db: db,
	}
}

func (r *SQLRoomRepository) CreateRoom(ctx context.Context, userId int, room *chatv1.CreateRoomRequest) (*chatv1.Room, error) {

	if room.Type == "p2p" {
		query := dbpq.QueryBuilder().
			Select("room.id", "room.created_at", "room.updated_at", "room.image", "room.name", "room.description", "room.type", "room.encription_data", "room.join_all_user", "room.\"lastMessageAt\"", "room.send_message", "room.add_member", "room.	edit_group", "partner.id", "partner.name", "partner.phone", "partner.avatar", "me.id", "me.name", "me.phone", "mm.is_muted", "mm.\"is_pinned\"", "mm.is_partner_blocked", "mm.role").
			From("room").
			InnerJoin("room_member AS pm ON room.id = pm.room_id AND pm.user_id = ? AND pm.removed_at IS NULL AND pm.deleted_at IS NULL", room.Participants[0]).
			InnerJoin(`public."user" AS partner ON pm.user_id = partner.id`).
			InnerJoin("room_member AS mm ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL AND mm.deleted_at IS NULL", userId).
			InnerJoin(`public."user" AS me ON mm.user_id = me.id`).
			Where(sq.Eq{"room.type": "p2p"}).
			Where(sq.Eq{"room.deleted_at": nil}).
			Limit(1)

		queryString, args, err := query.ToSql()
		if err != nil {
			return nil, err
		}

		rows, err := r.db.QueryContext(ctx, queryString, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		if rows.Next() {
			item := &chatv1.Room{}
			var photoURL sql.NullString
			var name sql.NullString
			var description sql.NullString
			var joinAllUser sql.NullBool
			var encryptionData sql.NullString
			var lastMessageAt sql.NullString
			var partnerID sql.NullInt32
			var partnerName sql.NullString
			var partnerPhone sql.NullString
			var partnerAvatar sql.NullString
			var meID sql.NullString
			var meName sql.NullString
			var mePhone sql.NullString
			var isMuted sql.NullBool
			var isPinned sql.NullBool
			var isPartnerBlocked sql.NullBool
			var role sql.NullString

			var err = rows.Scan(&item.Id, &item.CreatedAt, &item.UpdatedAt, &photoURL, &name, &description, &item.Type, &encryptionData, &joinAllUser, &lastMessageAt, &item.SendMessage, &item.AddMember, &item.EditGroup, &partnerID, &partnerName, &partnerPhone, &partnerAvatar, &meID, &meName, &mePhone, &isMuted, &isPinned, &isPartnerBlocked, &role)
			if err != nil {
				return nil, err
			}
			item.PhotoUrl = photoURL.String
			item.Name = name.String
			item.Description = description.String
			item.EncryptionData = encryptionData.String
			item.JoinAllUser = joinAllUser.Bool
			item.LastMessageAt = lastMessageAt.String
			item.Role = role.String
			if partnerID.Valid {
				item.Partner = &chatv1.RoomParticipant{
					Id:     partnerID.Int32,
					Name:   partnerName.String,
					Phone:  partnerPhone.String,
					Avatar: partnerAvatar.String,
				}
			}
			item = utils.FormatRoom(item)

			return item, nil
		}
	}

	//generate key and iv
	encryptionData, err := utils.GenerateKeyEncript()
	if err != nil {
		fmt.Println("error", err)
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if room.Type == "p2p" {
		room.SendMessage = &[]bool{true}[0]
		room.AddMember = &[]bool{false}[0]
		room.EditGroup = &[]bool{false}[0]
	}

	if room.SendMessage == nil {
		room.SendMessage = &[]bool{true}[0]
	}
	if room.AddMember == nil {
		room.AddMember = &[]bool{false}[0]
	}
	if room.EditGroup == nil {
		room.EditGroup = &[]bool{false}[0]
	}
	if room.Name == nil {
		room.Name = &[]string{""}[0]
	}
	if room.PhotoUrl == nil {
		room.PhotoUrl = &[]string{""}[0]
	}
	if room.Description == nil {
		room.Description = &[]string{""}[0]
	}

	query := dbpq.QueryBuilder().
		Insert("public.room").
		SetMap(sq.Eq{
			"name":            room.Name,
			"image":           room.PhotoUrl,
			"description":     room.Description,
			"join_all_user":   false,
			"send_message":    room.SendMessage,
			"add_member":      room.AddMember,
			"edit_group":      room.EditGroup,
			"encription_data": encryptionData,
			"created_at":      sq.Expr("NOW()"),
			"type":            room.Type,
		}).
		Suffix("RETURNING id")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	var newRoom chatv1.Room
	if rows.Next() {
		err = rows.Scan(&newRoom.Id)
		if err != nil {
			return nil, err
		}
		newRoom.Name = *room.Name
		newRoom.PhotoUrl = *room.PhotoUrl
		newRoom.Description = *room.Description
		newRoom.JoinAllUser = false
		newRoom.SendMessage = *room.SendMessage
		newRoom.AddMember = *room.AddMember
		newRoom.EditGroup = *room.EditGroup
		newRoom.EncryptionData = encryptionData
		newRoom.CreatedAt = time.Now().Format("2006-01-02T15:04:05.000000-07:00")
		newRoom.UpdatedAt = time.Now().Format("2006-01-02T15:04:05.000000-07:00")
		newRoom.Type = room.Type
		newRoom.IsMuted = false
		newRoom.IsPartnerBlocked = false
		newRoom.IsPinned = false
		newRoom.LastMessageAt = ""
		newRoom.Partner = nil
		newRoom.Role = "OWNER"
	}

	rows.Close()

	query = dbpq.QueryBuilder().
		Insert("public.room_member").
		SetMap(sq.Eq{
			"room_id": newRoom.Id,
			"user_id": userId,
			"role":    "OWNER",
		})

	queryString, args, err = query.ToSql()
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, queryString, args...)
	if err != nil {
		fmt.Println("error", err)
		return nil, err
	}

	queryParticipants := dbpq.QueryBuilder().
		Insert("public.room_member").
		Columns("room_id", "user_id", "role")

	for _, participant := range room.Participants {
		if participant != int32(userId) {
			queryParticipants = queryParticipants.Values(newRoom.Id, participant, "MEMBER")
		}
	}

	queryString, args, err = queryParticipants.ToSql()
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	if newRoom.Type == "p2p" {
		query := dbpq.QueryBuilder().
			Select("partner.id", "partner.name", "partner.phone", "partner.avatar").
			From("public.\"user\" AS partner").
			Where(sq.Eq{"partner.id": room.Participants[0]}).
			Limit(1)

		queryString, args, err := query.ToSql()
		if err != nil {
			return nil, err
		}

		rows, err := r.db.QueryContext(ctx, queryString, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		if rows.Next() {
			var partnerID sql.NullInt32
			var partnerName sql.NullString
			var partnerPhone sql.NullString
			var partnerAvatar sql.NullString

			var err = rows.Scan(&partnerID, &partnerName, &partnerPhone, &partnerAvatar)
			if err != nil {
				return nil, err
			}

			newRoom.Partner = &chatv1.RoomParticipant{
				Id:     partnerID.Int32,
				Name:   partnerName.String,
				Phone:  partnerPhone.String,
				Avatar: partnerAvatar.String,
			}
		}
	}
	if newRoom.Type == "group" {
		newRoom.Participants, _, err = r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: newRoom.Id, Page: 1, Limit: 5})
		if err != nil {
			return nil, err
		}
	}

	return &newRoom, nil
}

func (r *SQLRoomRepository) GetRoom(ctx context.Context, userId int, roomId string, allData bool, cache bool) (*chatv1.Room, error) {

	cacheKey := fmt.Sprintf("endpoint:chat:room:{%s}:shim:user:%d", roomId, userId)
	if allData {
		cacheKey = fmt.Sprintf("endpoint:chat:room:{%s}:user:%d", roomId, userId)
	}
	dataCached, existsCached := GetCachedRoom(ctx, cacheKey)
	if existsCached {
		return dataCached, nil
	}

	query := dbpq.QueryBuilder().
		Select("room.id", "room.created_at", "room.updated_at", "room.image", "room.name", "room.description", "room.type", "room.encription_data", "room.join_all_user", "room.\"lastMessageAt\"", "room.send_message", "room.add_member", "room.edit_group", "partner.id", "partner.name", "partner.phone", "partner.avatar", "pm.is_partner_blocked", "pm.is_muted", "me.id", "me.name", "me.phone", "mm.is_muted", "mm.\"is_pinned\"", "mm.is_partner_blocked", "mm.role",
			// Último mensaje
			"last_msg.id AS last_message_id",
			"last_msg.content AS last_message_content",
			"last_msg.type AS last_message_type",
			"last_msg.created_at AS last_message_created_at",
			"last_sender.name AS last_message_sender_name",
			"last_sender.phone AS last_message_sender_phone",
			"last_msg.status AS last_message_status",
			"last_msg.updated_at AS last_message_updated_at",
			// Conteo de mensajes no leídos
			"(SELECT COUNT(*) FROM room_message AS unread_msg LEFT JOIN room_message_meta AS unread_meta ON unread_msg.id = unread_meta.message_id AND unread_meta.user_id = ? AND (unread_meta.\"isDeleted\" = false OR unread_meta.\"isDeleted\" IS NULL) WHERE unread_msg.room_id = room.id AND unread_msg.deleted_at IS NULL AND unread_meta.read_at IS NULL) AS unread_count").
		From("room_member AS mm").
		InnerJoin("room ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL AND mm.deleted_at IS NULL", userId).
		InnerJoin("public.\"user\" AS me ON mm.user_id = me.id").
		LeftJoin("room_member AS pm ON room.id = pm.room_id AND pm.user_id <> ? AND room.type = 'p2p' AND pm.removed_at IS NULL AND pm.deleted_at IS NULL", userId).
		LeftJoin("public.\"user\" AS partner ON pm.user_id = partner.id").
		// LATERAL JOIN para obtener el último mensaje
		LeftJoin(`LATERAL (
			SELECT 
				msg.id, msg.content, msg.type, msg.created_at, msg.sender_id, msg.status, msg.updated_at 
			FROM room_message AS msg 	
			LEFT JOIN room_message_meta AS meta ON msg.id = meta.message_id AND meta.user_id = me.id AND (meta."isSenderBlocked" = false OR meta."isSenderBlocked" IS NULL)
			WHERE msg.room_id = room.id AND msg.deleted_at IS NULL AND meta."isDeleted" = false ORDER BY msg.created_at DESC LIMIT 1) 
			AS last_msg ON true`).
		LeftJoin("public.\"user\" AS last_sender ON last_msg.sender_id = last_sender.id").
		Where(sq.Eq{"room.deleted_at": nil}).
		Where(sq.Eq{"room.id": roomId}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	args = append([]any{userId}, args...)

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		item := &chatv1.Room{}
		var updatedAt sql.NullString
		var photoURL sql.NullString
		var name sql.NullString
		var description sql.NullString
		var joinAllUser sql.NullBool
		var encryptionData sql.NullString
		var lastMessageAt sql.NullString
		var partnerID sql.NullInt32
		var partnerName sql.NullString
		var partnerPhone sql.NullString
		var partnerAvatar sql.NullString
		var partnerBlocked sql.NullBool
		var partnerMuted sql.NullBool
		var meID sql.NullString
		var meName sql.NullString
		var mePhone sql.NullString
		var isMuted sql.NullBool
		var isPinned sql.NullBool
		var isPartnerBlocked sql.NullBool
		var role sql.NullString
		// Campos del último mensaje
		var lastMessageId sql.NullString
		var lastMessageContent sql.NullString
		var lastMessageType sql.NullString
		var lastMessageCreatedAt sql.NullString
		var lastMessageSenderName sql.NullString
		var lastMessageSenderPhone sql.NullString
		var lastMessageStatus sql.NullInt32
		var lastMessageUpdatedAt sql.NullString
		// Conteo de mensajes no leídos
		var unreadCount sql.NullInt32

		var err = rows.Scan(&item.Id, &item.CreatedAt, &updatedAt, &photoURL, &name, &description, &item.Type, &encryptionData, &joinAllUser, &lastMessageAt, &item.SendMessage, &item.AddMember, &item.EditGroup, &partnerID, &partnerName, &partnerPhone, &partnerAvatar, &partnerBlocked, &partnerMuted, &meID, &meName, &mePhone, &isMuted, &isPinned, &isPartnerBlocked, &role,
			&lastMessageId, &lastMessageContent, &lastMessageType, &lastMessageCreatedAt, &lastMessageSenderName, &lastMessageSenderPhone, &lastMessageStatus, &lastMessageUpdatedAt, &unreadCount)
		if err != nil {
			return nil, err
		}
		item.UpdatedAt = updatedAt.String
		item.PhotoUrl = photoURL.String
		item.Name = name.String
		item.Description = description.String
		item.EncryptionData = encryptionData.String
		item.JoinAllUser = joinAllUser.Bool
		item.LastMessageAt = lastMessageAt.String
		item.Role = role.String
		item.IsPinned = isPinned.Bool
		item.IsMuted = isMuted.Bool
		item.IsPartnerBlocked = isPartnerBlocked.Bool

		// Agregar el último mensaje si existe
		if lastMessageId.Valid {
			item.LastMessage = &chatv1.MessageData{
				Id:          lastMessageId.String,
				Content:     lastMessageContent.String,
				Type:        lastMessageType.String,
				CreatedAt:   lastMessageCreatedAt.String,
				SenderName:  lastMessageSenderName.String,
				SenderPhone: lastMessageSenderPhone.String,
				Status:      chatv1.MessageStatus(lastMessageStatus.Int32),
				UpdatedAt:   lastMessageUpdatedAt.String,
			}
		}

		// Agregar el conteo de mensajes no leídos
		if unreadCount.Valid {
			item.UnreadCount = unreadCount.Int32
		} else {
			item.UnreadCount = 0
		}

		if partnerID.Valid {
			item.Partner = &chatv1.RoomParticipant{
				Id:               partnerID.Int32,
				Name:             partnerName.String,
				Phone:            partnerPhone.String,
				Avatar:           partnerAvatar.String,
				IsPartnerBlocked: partnerBlocked.Bool,
				IsPartnerMuted:   partnerMuted.Bool,
			}
		}

		if item.Type == "group" && allData {
			item.Participants, _, err = r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: item.Id, Page: 1, Limit: 5})
			if err != nil {
				return nil, err
			}
		}

		item = utils.FormatRoom(item)

		SetCachedRoom(ctx, roomId, cacheKey, item)

		return item, nil
	}

	return nil, nil
}

func (r *SQLRoomRepository) GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error) {

	query := dbpq.QueryBuilder().
		Select("room.id", "room.created_at", "room.updated_at", "room.image", "room.name", "room.description", "room.type", "room.encription_data", "room.join_all_user", "room.\"lastMessageAt\"", "room.send_message", "room.add_member", "room.edit_group", "partner.id", "partner.name", "partner.phone", "partner.avatar", "pm.is_partner_blocked", "me.id", "me.name", "me.phone", "mm.is_muted", "mm.\"is_pinned\"", "mm.is_partner_blocked", "mm.role",
			// Último mensaje
			"last_msg.id AS last_message_id",
			"last_msg.content AS last_message_content",
			"last_msg.type AS last_message_type",
			"last_msg.created_at AS last_message_created_at",
			"last_sender.name AS last_message_sender_name",
			"last_sender.phone AS last_message_sender_phone",
			"last_msg.status AS last_message_status",
			"last_msg.updated_at AS last_message_updated_at",
			// Conteo de mensajes no leídos
			"(SELECT COUNT(*) FROM room_message AS unread_msg LEFT JOIN room_message_meta AS unread_meta ON unread_msg.id = unread_meta.message_id AND unread_meta.user_id = ? AND (unread_meta.\"isDeleted\" = false OR unread_meta.\"isDeleted\" IS NULL) WHERE unread_msg.room_id = room.id AND unread_msg.deleted_at IS NULL AND unread_meta.read_at IS NULL) AS unread_count").
		From("room_member AS mm").
		InnerJoin("room ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL AND mm.deleted_at IS NULL", userId).
		InnerJoin("public.\"user\" AS me ON mm.user_id = me.id").
		LeftJoin("room_member AS pm ON room.id = pm.room_id AND pm.user_id <> ? AND room.type = 'p2p' AND pm.removed_at IS NULL AND pm.deleted_at IS NULL", userId).
		LeftJoin("public.\"user\" AS partner ON pm.user_id = partner.id").
		// LATERAL JOIN para obtener el último mensaje
		LeftJoin(`LATERAL (
			SELECT msg.id, msg.content, msg.type, msg.created_at, msg.sender_id, msg.status, msg.updated_at 
			FROM room_message AS msg 
			LEFT JOIN room_message_meta AS meta ON msg.id = meta.message_id AND meta.user_id = me.id AND (meta."isSenderBlocked" = false OR meta."isSenderBlocked" IS NULL)
			WHERE msg.room_id = room.id AND msg.deleted_at IS NULL AND meta."isDeleted" = false ORDER BY msg.created_at DESC LIMIT 1) 
			AS last_msg ON true`).
		LeftJoin("public.\"user\" AS last_sender ON last_msg.sender_id = last_sender.id").
		Where(sq.Eq{"room.deleted_at": nil})

	if pagination != nil {
		if pagination.Search != "" {
			query = query.Where("unaccent(room.name) ILIKE unaccent(?) OR unaccent(partner.name) ILIKE unaccent(?)", "%"+pagination.Search+"%", "%"+pagination.Search+"%")
		}

		if pagination.Page > 0 && pagination.Limit > 0 {
			query = query.Offset(uint64((pagination.Page - 1) * pagination.Limit)).Limit(uint64(pagination.Limit))
		}

		if pagination.Type != "" {
			query = query.Where(sq.Eq{"room.type": pagination.Type})
		}

		if pagination.Since != "" {
			query = query.Where("room.updated_at > ? OR mm.updated_at > ?", pagination.Since, pagination.Since)
		}
	}

	query = query.OrderBy("mm.\"is_pinned\" DESC, room.\"lastMessageAt\" DESC, room.created_at DESC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, nil, err
	}

	// Agregar el userId para la subconsulta de conteo de no leídos
	args = append([]any{userId}, args...)

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, nil, err
	}

	data := []*chatv1.Room{}

	for rows.Next() {
		item := &chatv1.Room{}
		var updatedAt sql.NullString
		var photoURL sql.NullString
		var name sql.NullString
		var description sql.NullString
		var joinAllUser sql.NullBool
		var encryptionData sql.NullString
		var lastMessageAt sql.NullString
		var partnerID sql.NullInt32
		var partnerName sql.NullString
		var partnerPhone sql.NullString
		var partnerAvatar sql.NullString
		var partnerBlocked sql.NullBool
		var meID sql.NullString
		var meName sql.NullString
		var mePhone sql.NullString
		var isMuted sql.NullBool
		var isPinned sql.NullBool
		var isPartnerBlocked sql.NullBool
		var role sql.NullString
		// Campos del último mensaje
		var lastMessageId sql.NullString
		var lastMessageContent sql.NullString
		var lastMessageType sql.NullString
		var lastMessageCreatedAt sql.NullString
		var lastMessageSenderName sql.NullString
		var lastMessageSenderPhone sql.NullString
		var lastMessageStatus sql.NullInt32
		var lastMessageUpdatedAt sql.NullString
		// Conteo de mensajes no leídos
		var unreadCount sql.NullInt32

		var err = rows.Scan(&item.Id, &item.CreatedAt, &updatedAt, &photoURL, &name, &description, &item.Type, &encryptionData, &joinAllUser, &lastMessageAt, &item.SendMessage, &item.AddMember, &item.EditGroup, &partnerID, &partnerName, &partnerPhone, &partnerAvatar, &partnerBlocked, &meID, &meName, &mePhone, &isMuted, &isPinned, &isPartnerBlocked, &role,
			&lastMessageId, &lastMessageContent, &lastMessageType, &lastMessageCreatedAt, &lastMessageSenderName, &lastMessageSenderPhone, &lastMessageStatus, &lastMessageUpdatedAt, &unreadCount)
		if err != nil {
			return nil, nil, err
		}
		item.UpdatedAt = updatedAt.String
		item.PhotoUrl = photoURL.String
		item.Name = name.String
		item.Description = description.String
		item.EncryptionData = encryptionData.String
		item.JoinAllUser = joinAllUser.Bool
		item.LastMessageAt = lastMessageAt.String
		item.Role = role.String
		item.IsPinned = isPinned.Bool
		item.IsMuted = isMuted.Bool
		item.IsPartnerBlocked = isPartnerBlocked.Bool

		// Agregar el último mensaje si existe
		if lastMessageId.Valid {
			item.LastMessage = &chatv1.MessageData{
				Id:          lastMessageId.String,
				Content:     lastMessageContent.String,
				Type:        lastMessageType.String,
				CreatedAt:   lastMessageCreatedAt.String,
				SenderName:  lastMessageSenderName.String,
				SenderPhone: lastMessageSenderPhone.String,
				Status:      chatv1.MessageStatus(lastMessageStatus.Int32),
				UpdatedAt:   lastMessageUpdatedAt.String,
			}
		}

		// Agregar el conteo de mensajes no leídos
		if unreadCount.Valid {
			item.UnreadCount = unreadCount.Int32
		} else {
			item.UnreadCount = 0
		}

		if partnerID.Valid {
			item.Partner = &chatv1.RoomParticipant{
				Id:               partnerID.Int32,
				Name:             partnerName.String,
				Phone:            partnerPhone.String,
				Avatar:           partnerAvatar.String,
				IsPartnerBlocked: partnerBlocked.Bool,
			}
		}

		item = utils.FormatRoom(item)

		data = append(data, item)
	}

	//get room participants (max 5) only if type is group
	allRoomIds := []string{}
	for _, room := range data {
		if room.Type == "group" {
			allRoomIds = append(allRoomIds, room.Id)
		}
	}

	queryParticipants := dbpq.QueryBuilder().
		Select("room_member.user_id", "room_member.role", "uu.name", "uu.phone", "uu.avatar", "room_member.room_id", "ROW_NUMBER() OVER (PARTITION BY room_member.room_id ORDER BY room_member.created_at DESC) as rn").
		From("room_member").
		InnerJoin("public.\"user\" AS uu ON room_member.user_id = uu.id").
		Where(sq.Eq{"room_member.room_id": allRoomIds}).
		Where(sq.Eq{"room_member.removed_at": nil}).
		Where(sq.Eq{"uu.removed_at": nil}).
		OrderBy("room_member.created_at DESC")

	queryParticipantsString, argsParticipants, err := queryParticipants.ToSql()
	if err != nil {
		return nil, nil, err
	}

	queryParticipantsPerRoom := dbpq.QueryBuilder().
		Select("*").
		From("(" + queryParticipantsString + ") ranked_participants").
		Where("rn <= " + fmt.Sprintf("%d", 5))

	queryParticipantsPerRoomString, _, err := queryParticipantsPerRoom.ToSql()
	if err != nil {
		return nil, nil, err
	}

	rowsParticipantsByRoom, err := r.db.QueryContext(ctx, queryParticipantsPerRoomString, argsParticipants...)
	if err != nil {
		return nil, nil, err
	}

	defer rowsParticipantsByRoom.Close()

	for rowsParticipantsByRoom.Next() {
		item := &chatv1.RoomParticipant{}
		var roomId sql.NullString
		var rn sql.NullInt32
		var err = rowsParticipantsByRoom.Scan(&item.Id, &item.Role, &item.Name, &item.Phone, &item.Avatar, &roomId, &rn)
		if err != nil {
			return nil, nil, err
		}

		for _, room := range data {
			if room.Id == roomId.String {
				room.Participants = append(room.Participants, item)
			}
		}
	}

	///////////////////////////////////////////////////

	queryTotal := dbpq.QueryBuilder().
		Select("COUNT(*)").
		From("room_member AS mm").
		InnerJoin("room ON room.id = mm.room_id AND mm.user_id = ? AND mm.removed_at IS NULL AND mm.deleted_at IS NULL", userId).
		LeftJoin("room_member AS pm ON room.id = pm.room_id AND pm.user_id <> ? AND room.type = 'p2p' AND pm.removed_at IS NULL AND pm.deleted_at IS NULL", userId).
		LeftJoin("public.\"user\" AS partner ON pm.user_id = partner.id").
		Where(sq.Eq{"room.deleted_at": nil})

	if pagination != nil {
		if pagination.Type != "" {
			queryTotal = queryTotal.Where(sq.Eq{"room.type": pagination.Type})
		}

		if pagination.Search != "" {
			queryTotal = queryTotal.Where("unaccent(room.name) ILIKE unaccent(?) OR unaccent(partner.name) ILIKE unaccent(?)", "%"+pagination.Search+"%", "%"+pagination.Search+"%")
		}

		if pagination.Since != "" {
			queryTotal = queryTotal.Where("room.updated_at > ? OR mm.updated_at > ?", pagination.Since, pagination.Since)
		}
	}

	queryTotalString, argsTotal, err := queryTotal.ToSql()
	if err != nil {
		return nil, nil, err
	}

	totalItems, err := r.db.QueryContext(ctx, queryTotalString, argsTotal...)
	if err != nil {
		return nil, nil, err
	}

	defer totalItems.Close()

	var totalItemsCount int64

	if totalItems.Next() {
		err = totalItems.Scan(&totalItemsCount)
		if err != nil {
			return nil, nil, err
		}
	}

	var limit uint32
	var page uint32
	if pagination != nil {
		page = pagination.GetPage()
		limit = pagination.GetLimit()
	}

	meta := chatv1.PaginationMeta{
		TotalItems:   uint32(totalItemsCount),
		ItemCount:    uint32(len(data)),
		ItemsPerPage: limit,
		TotalPages:   uint32(math.Ceil(float64(totalItemsCount) / float64(limit))),
		CurrentPage:  page,
	}

	return data, &meta, nil
}

func (r *SQLRoomRepository) GetRoomListDeleted(ctx context.Context, userId int, since string) ([]string, error) {

	query := dbpq.QueryBuilder().
		Select("room.id").
		From("room_member AS mm").
		InnerJoin("room ON room.id = mm.room_id AND mm.user_id = ? AND (room.deleted_at IS NOT NULL OR mm.removed_at IS NOT NULL)", userId)

	if since != "" {
		query = query.Where("(room.deleted_at > ? OR mm.removed_at > ?)", since, since)
	}

	query = query.OrderBy("mm.\"is_pinned\" DESC, room.\"lastMessageAt\" DESC, room.created_at DESC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	var data []string

	for rows.Next() {
		var id string

		var err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}

		data = append(data, id)
	}

	return data, nil
}

func (r *SQLRoomRepository) LeaveRoom(ctx context.Context, userId int, roomId string, participants []int32, leaveAll bool) ([]User, error) {

	if leaveAll {
		query := dbpq.QueryBuilder().
			Select("user_id").
			From("room_member").
			Where(sq.Eq{"room_id": roomId}).
			Where(sq.Eq{"removed_at": nil}).
			OrderBy("user_id ASC")

		queryString, args, err := query.ToSql()
		if err != nil {
			return nil, err
		}

		rows, err := r.db.QueryContext(ctx, queryString, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var userId int32
			err = rows.Scan(&userId)
			if err != nil {
				return nil, err
			}
			participants = append(participants, userId)
		}
	}

	participants = slices.Compact(participants)

	query := dbpq.QueryBuilder().
		Update("room_member").
		Set("removed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"room_id": roomId}).
		Where(sq.Eq{"user_id": participants}).
		Where(sq.Eq{"removed_at": nil})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	// Convert []int32 a []int para que sea compatible con GetUsersByID
	ids := make([]int, len(participants))
	for i, v := range participants {
		ids[i] = int(v)
	}

	users, err := r.GetUsersByID(ctx, ids)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *SQLRoomRepository) DeleteRoom(ctx context.Context, userId int, roomId string, partner *int) error {

	query := dbpq.QueryBuilder().
		Update("room").
		Set("updated_at", sq.Expr("NOW()")).
		Set("deleted_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": roomId}).
		Where(sq.Eq{"type": "p2p"}).
		Where(sq.Eq{"deleted_at": nil})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	query = dbpq.QueryBuilder().
		Update("room_member").
		Set("updated_at", sq.Expr("NOW()")).
		Set("removed_at", sq.Expr("NOW()")).
		Where(sq.Eq{"room_id": roomId}).
		Where(sq.Eq{"removed_at": nil})

	queryString, args, err = query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	// if partner != nil {
	// 	cacheKey = "endpoint:chat:room:room=" + roomId + ":user=" + strconv.Itoa(*partner)
	// 	DeleteCache(ctx, cacheKey)
	// 	cacheKey = "endpoint:chat:room:room_shim=" + roomId + ":user=" + strconv.Itoa(*partner)
	// 	DeleteCache(ctx, cacheKey)
	// }

	return nil
}

func (r *SQLRoomRepository) GetRoomParticipants(ctx context.Context, pagination *chatv1.GetRoomParticipantsRequest) ([]*chatv1.RoomParticipant, *chatv1.PaginationMeta, error) {

	query := dbpq.QueryBuilder().
		Select("room_member.user_id", "room_member.role", "uu.name", "uu.phone", "uu.avatar").
		From("room_member").
		InnerJoin("public.\"user\" AS uu ON room_member.user_id = uu.id").
		Where(sq.Eq{"room_member.room_id": pagination.Id}).
		Where(sq.Eq{"room_member.removed_at": nil}).
		Where(sq.Eq{"uu.removed_at": nil})

	if pagination.Search != "" {
		query = query.Where("unaccent(uu.name) ILIKE unaccent(?)", "%"+pagination.Search+"%")
	}

	if pagination.Page > 0 && pagination.Limit > 0 {
		query = query.Offset(uint64((pagination.Page - 1) * pagination.Limit)).Limit(uint64(pagination.Limit))
	}

	query = query.OrderBy("uu.name ASC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, nil, err
	}

	data := []*chatv1.RoomParticipant{}

	for rows.Next() {
		item := &chatv1.RoomParticipant{}
		var name sql.NullString
		var phone sql.NullString
		var avatar sql.NullString

		var err = rows.Scan(&item.Id, &item.Role, &name, &phone, &avatar)
		if err != nil {
			return nil, nil, err
		}
		item.Name = name.String
		item.Phone = phone.String
		item.Avatar = avatar.String

		data = append(data, item)
	}

	queryTotal := dbpq.QueryBuilder().
		Select("COUNT(*)").
		From("room_member").
		InnerJoin("public.\"user\" AS uu ON room_member.user_id = uu.id").
		Where(sq.Eq{"room_member.room_id": pagination.Id}).
		Where(sq.Eq{"room_member.removed_at": nil}).
		Where(sq.Eq{"uu.removed_at": nil})

	if pagination.Search != "" {
		queryTotal = queryTotal.Where("unaccent(uu.name) ILIKE unaccent(?)", "%"+pagination.Search+"%")
	}

	queryTotalString, argsTotal, err := queryTotal.ToSql()
	if err != nil {
		return nil, nil, err
	}

	totalItems, err := r.db.QueryContext(ctx, queryTotalString, argsTotal...)
	if err != nil {
		return nil, nil, err
	}

	defer totalItems.Close()

	var totalItemsCount int64

	if totalItems.Next() {
		err = totalItems.Scan(&totalItemsCount)
		if err != nil {
			return nil, nil, err
		}
	}

	var limit uint32
	var page uint32
	if pagination != nil {
		page = pagination.GetPage()
		limit = pagination.GetLimit()
	}

	meta := chatv1.PaginationMeta{
		TotalItems:   uint32(totalItemsCount),
		ItemCount:    uint32(len(data)),
		ItemsPerPage: limit,
		TotalPages:   uint32(math.Ceil(float64(totalItemsCount) / float64(limit))),
		CurrentPage:  page,
	}

	return data, &meta, nil
}

func (r *SQLRoomRepository) PinRoom(ctx context.Context, userId int, roomId string, pin bool) error {

	query := dbpq.QueryBuilder().
		Update("room_member").
		Set("\"is_pinned\"", pin).
		Where(sq.Eq{"room_id": roomId}).
		Where(sq.Eq{"user_id": userId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	return nil
}

func (r *SQLRoomRepository) MuteRoom(ctx context.Context, userId int, roomId string, mute bool) error {

	query := dbpq.QueryBuilder().
		Update("room_member").
		Set("\"is_muted\"", mute).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"room_id": roomId}).
		Where(sq.Eq{"user_id": userId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	return nil
}

func (r *SQLRoomRepository) UpdateRoom(ctx context.Context, userId int, roomId string, room *chatv1.UpdateRoomRequest) error {

	query := dbpq.QueryBuilder().
		Update("room").
		Set("updated_at", sq.Expr("NOW()"))

	if room.Name != nil {
		query = query.Set("name", room.Name)
	}
	if room.Description != nil {
		query = query.Set("description", room.Description)
	}
	if room.PhotoUrl != nil {
		query = query.Set("image", room.PhotoUrl)
	}
	if room.SendMessage != nil {
		query = query.Set("send_message", room.SendMessage)
	}
	if room.AddMember != nil {
		query = query.Set("add_member", room.AddMember)
	}
	if room.EditGroup != nil {
		query = query.Set("edit_group", room.EditGroup)
	}

	query = query.Where(sq.Eq{"id": roomId})
	query = query.Where(sq.Eq{"deleted_at": nil})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	return nil
}

func (r *SQLRoomRepository) AddParticipantToRoom(ctx context.Context, userId int, roomId string, participants []int) ([]User, error) {

	//transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	//buscar primero en users si existe
	queryUsers := dbpq.QueryBuilder().
		Select("id", "name", "phone", "email", "avatar", "created_at", "dni").
		From("public.\"user\"").
		Where(sq.Eq{"id": participants}).
		Where(sq.Eq{"removed_at": nil}).
		Where(sq.Eq{"deleted_at": nil})

	queryUsersString, argsUsers, err := queryUsers.ToSql()
	if err != nil {
		return nil, err
	}

	rowsUsers, err := tx.QueryContext(ctx, queryUsersString, argsUsers...)
	if err != nil {
		return nil, err
	}

	var newParticipantsData []User

	//filtrar los que no existen
	for rowsUsers.Next() {
		var id int
		var name string
		var phone string
		var email *string
		var avatar *string
		var createdAt *string
		var dni *string

		var err = rowsUsers.Scan(&id, &name, &phone, &email, &avatar, &createdAt, &dni)
		if err != nil {
			return nil, err
		}
		newParticipantsData = append(newParticipantsData, User{
			ID:        id,
			Name:      name,
			Phone:     phone,
			Email:     email,
			Avatar:    avatar,
			CreatedAt: createdAt,
			Dni:       dni,
		})
	}

	rowsUsers.Close()

	query := dbpq.QueryBuilder().
		Select("room_member.id", "room_member.room_id", "room_member.user_id", "room_member.role", "room_member.removed_at", "room_member.deleted_at").
		From("room_member").
		Where(sq.Eq{"room_member.room_id": roomId}).
		Where(sq.Eq{"room_member.user_id": participants})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Mapa para rastrear participantes que ya existen
	existingParticipants := make(map[int]bool)

	var isNeedToUpdateRoomMember = false

	for rows.Next() {
		var id sql.NullString
		var roomId sql.NullString
		var userId sql.NullInt32
		var role sql.NullString
		var removedAt sql.NullString
		var deletedAt sql.NullString

		var err = rows.Scan(&id, &roomId, &userId, &role, &removedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		if removedAt.Valid || deletedAt.Valid {
			// update room_member to null
			if removedAt.String != "" || deletedAt.String != "" {
				isNeedToUpdateRoomMember = true
			}
		}
		if !removedAt.Valid && !deletedAt.Valid {
			// remover del array de newParticipantsData
			for i, participant := range newParticipantsData {
				if participant.ID == int(userId.Int32) {
					newParticipantsData = append(newParticipantsData[:i], newParticipantsData[i+1:]...)
					break
				}
			}
		}

		// Marcar este participante como ya existente
		existingParticipants[int(userId.Int32)] = true
	}

	rows.Close()

	if isNeedToUpdateRoomMember {

		query := dbpq.QueryBuilder().
			Update("room_member").
			Set("removed_at", nil).
			Set("deleted_at", nil).
			Where(sq.Eq{"room_id": roomId}).
			Where(sq.Eq{"user_id": participants})

		queryString, args, err := query.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.ExecContext(ctx, queryString, args...)
		if err != nil {
			return nil, err
		}

	}

	// Filtrar participantes que no existen aún
	var newParticipants []int
	for _, participant := range participants {
		if !existingParticipants[participant] {
			newParticipants = append(newParticipants, participant)
		}
	}

	// Insertar nuevos participantes
	if len(newParticipants) > 0 {

		queryParticipants := dbpq.QueryBuilder().
			Insert("public.room_member").
			Columns("room_id", "user_id", "role")

		for _, participant := range newParticipants {
			queryParticipants = queryParticipants.Values(roomId, participant, "MEMBER")
		}

		queryString, args, err = queryParticipants.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.ExecContext(ctx, queryString, args...)
		if err != nil {
			return nil, err
		}
	}

	// Commit de la transacción
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	return newParticipantsData, nil
}

func (r *SQLRoomRepository) UpdateParticipantRoom(ctx context.Context, userId int, req *chatv1.UpdateParticipantRoomRequest) error {
	query := dbpq.QueryBuilder().
		Update("room_member").
		Set("role", req.Role).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"room_id": req.Id}).
		Where(sq.Eq{"user_id": req.Participant})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, req.GetId())

	return nil
}

func (r *SQLRoomRepository) BlockUser(ctx context.Context, userId int, roomId string, block bool, partner *int) error {

	query := dbpq.QueryBuilder().
		Update("room_member").
		Set("is_partner_blocked", block).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"room_id": roomId}).
		Where(sq.Eq{"user_id": userId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	// if partner != nil {
	// 	cacheKey = "endpoint:chat:room:room=" + roomId + ":user=" + strconv.Itoa(*partner)
	// 	DeleteCache(ctx, cacheKey)
	// 	cacheKey = "endpoint:chat:room:room_shim=" + roomId + ":user=" + strconv.Itoa(*partner)
	// 	DeleteCache(ctx, cacheKey)
	// }

	return nil
}

// SaveMessage guarda el mensaje principal y los metadatos solo para el remitente.
// La creación de metadatos para otros participantes se maneja de forma asíncrona.
func (r *SQLRoomRepository) SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var contactId sql.NullInt32
	if req.Type == "contact" && req.ContactPhone != nil {
		// This query can be run inside the transaction
		err := tx.QueryRowContext(ctx, "SELECT id FROM public.\"user\" WHERE phone = $1 AND removed_at IS NULL AND deleted_at IS NULL LIMIT 1", req.ContactPhone).Scan(&contactId)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	var forwardUserId sql.NullInt32
	if req.ForwardId != nil {
		err := tx.QueryRowContext(ctx, "SELECT sender_id FROM room_message WHERE id = $1 AND deleted_at IS NULL LIMIT 1", req.ForwardId).Scan(&forwardUserId)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	if req.Lifetime == nil {
		req.Lifetime = &[]string{"normal"}[0]
	}
	if req.Origin == nil {
		req.Origin = &[]string{"app"}[0]
	}
	if req.Type == "" {
		req.Type = "message"
	}
	if contentDecrypted == nil {
		contentDecrypted = &[]string{""}[0]
	}

	// 1. Insertar el mensaje principal
	var messageId string
	insertMessageQuery := dbpq.QueryBuilder().
		Insert("public.room_message").
		SetMap(sq.Eq{
			"room_id":                           req.RoomId,
			"sender_id":                         userId,
			"content":                           req.Content,
			"content_decrypted":                 contentDecrypted,
			"status":                            int(chatv1.MessageStatus_MESSAGE_STATUS_SENT),
			"created_at":                        sq.Expr("NOW()"),
			"updated_at":                        sq.Expr("NOW()"),
			"type":                              req.Type,
			"lifetime":                          req.Lifetime,
			"location_name":                     req.LocationName,
			"location_latitude":                 req.LocationLatitude,
			"location_longitude":                req.LocationLongitude,
			"origin":                            req.Origin,
			"contact_id":                        contactId,
			"contact_name":                      req.ContactName,
			"contact_phone":                     req.ContactPhone,
			"file":                              req.File,
			"edited":                            false,
			"\"isDeleted\"":                     false,
			"replied_message_id":                req.ReplyId,
			"forwarded_message_id":              req.ForwardId,
			"forwarded_message_original_sender": forwardUserId,
			"event":                             req.Event,
			"sender_message_id":                 req.SenderMessageId,
		}).
		Suffix("RETURNING id").
		RunWith(tx)

	err = insertMessageQuery.QueryRowContext(ctx).Scan(&messageId)
	if err != nil {
		return nil, fmt.Errorf("failed to insert message: %w", err)
	}

	// 2. Insertar menciones si existen
	if len(req.Mentions) > 0 {

		mentionQuery := dbpq.QueryBuilder().
			Insert("public.room_message_tag").
			Columns("message_id", "user_id", "tag")

		for _, mention := range req.Mentions {

			mentionQuery = mentionQuery.Values(messageId, mention.User, mention.Tag)
		}

		_, err = mentionQuery.RunWith(tx).ExecContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to insert mentions: %w", err)
		}
	}

	// 3. Insertar metadatos solo para el remitente
	_, err = dbpq.QueryBuilder().
		Insert("public.room_message_meta").
		Columns("message_id", "user_id", "read_at", "\"isDeleted\"", "\"isSenderBlocked\"").
		Values(messageId, userId, sq.Expr("NOW()"), false, false).
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to insert sender message meta: %w", err)
	}

	// 4. Confirmar la transacción
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// 5. Obtener y devolver el mensaje completo (fuera de la transacción)
	message, err := r.GetMessage(ctx, userId, messageId)
	if err != nil {
		// The message was saved, but we couldn't fetch it.
		// Log the error but consider the operation successful at a basic level.
		return nil, fmt.Errorf("failed to get message after saving: %w", err)
	}

	UpdateRoomCacheWithNewMessage(context.Background(), message)

	return message, nil
}

func (r *SQLRoomRepository) GetMessageSimple(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error) {

	cacheKey := fmt.Sprintf("endpoint:chat:messagesimple:messageId:{%s}", messageId)
	dataCached, existsCached := GetCachedMessageSimple(ctx, cacheKey)
	if existsCached {
		return dataCached, nil
	}

	query := dbpq.QueryBuilder().
		Select(
			"room_message.id",
			"room_message.created_at",
			"room_message.room_id",
			"room_message.sender_id",
			"room_message.content",
			"room_message.audio_transcription",
			"room_message.file",
			"room_message.type",
		).
		From("room_message").
		Where(sq.Eq{"room_message.id": messageId}).
		Where(sq.Eq{"room_message.deleted_at": nil}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		var message chatv1.MessageData

		err = rows.Scan(
			&message.Id,
			&message.CreatedAt,
			&message.RoomId,
			&message.SenderId,
			&message.Content,
			&message.AudioTranscription,
			&message.File,
			&message.Type,
		)

		if err != nil {
			fmt.Println("error scanning message", err)
			return nil, err
		}

		SetCachedMessageSimple(ctx, cacheKey, &message)

		return &message, nil
	}

	rows.Close()

	return nil, nil
}

func (r *SQLRoomRepository) GetMessage(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error) {

	query := dbpq.QueryBuilder().
		Select(
			"room_message.id",
			"room_message.room_id",
			"room_message.sender_id",
			"public.\"user\".name",
			"public.\"user\".phone",
			"public.\"user\".avatar",
			"room_message.content",
			"room_message.status",
			"room_message.created_at",
			"room_message.updated_at",
			"room_message.type",
			"room_message.lifetime",
			"room_message.location_name",
			"room_message.location_latitude",
			"room_message.location_longitude",
			"room_message.origin",
			"room_message.contact_id",
			"room_message.contact_name",
			"room_message.contact_phone",
			"room_message.file",
			"room_message.edited",
			"room_message.\"isDeleted\"",
			"room_message.event",
			"room_message.sender_message_id",

			"room_message.forwarded_message_id",
			"forwarded_user.id AS forwarded_user_id",
			"forwarded_user.name AS forwarded_user_name",
			"forwarded_user.phone AS forwarded_user_phone",
			"forwarded_user.avatar AS forwarded_user_avatar",

			"room_message.replied_message_id",
			"reply_message.sender_id AS reply_user_id",
			"reply_user.name AS reply_user_name",
			"reply_user.phone AS reply_user_phone",
			"reply_user.avatar AS reply_user_avatar",
			"reply_message.content AS reply_message_content",
			"reply_message.type AS reply_message_type",
			"reply_message.room_id AS reply_message_room_id",
			"reply_message.created_at AS reply_message_created_at",
			"reply_message.updated_at AS reply_message_updated_at",
		).
		From("room_message").
		InnerJoin("public.\"user\" ON room_message.sender_id = public.\"user\".id").
		LeftJoin("public.\"user\" AS forwarded_user ON room_message.forwarded_message_original_sender = forwarded_user.id").
		LeftJoin("room_message AS reply_message ON room_message.replied_message_id = reply_message.id").
		LeftJoin("public.\"user\" AS reply_user ON reply_message.sender_id = reply_user.id").
		Where(sq.Eq{"room_message.id": messageId}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		var message chatv1.MessageData

		replyIdNull := sql.NullString{}
		replySenderIdNull := sql.NullInt32{}
		replySenderNameNull := sql.NullString{}
		replySenderPhoneNull := sql.NullString{}
		replySenderAvatarNull := sql.NullString{}
		replyContentNull := sql.NullString{}
		replyTypeNull := sql.NullString{}
		replyMessageRoomIdNull := sql.NullString{}
		replyMessageCreatedAtNull := sql.NullString{}
		replyMessageUpdatedAtNull := sql.NullString{}

		err = rows.Scan(
			&message.Id, &message.RoomId, &message.SenderId, &message.SenderName, &message.SenderPhone, &message.SenderAvatar, &message.Content, &message.Status,
			&message.CreatedAt, &message.UpdatedAt, &message.Type, &message.Lifetime, &message.LocationName, &message.LocationLatitude,
			&message.LocationLongitude, &message.Origin, &message.ContactId, &message.ContactName, &message.ContactPhone, &message.File,
			&message.Edited, &message.IsDeleted, &message.Event, &message.SenderMessageId,

			&message.ForwardedMessageId, &message.ForwardedMessageSenderId, &message.ForwardedMessageSenderName, &message.ForwardedMessageSenderPhone, &message.ForwardedMessageSenderAvatar,

			&replyIdNull, &replySenderIdNull, &replySenderNameNull, &replySenderPhoneNull, &replySenderAvatarNull, &replyContentNull, &replyTypeNull,
			&replyMessageRoomIdNull, &replyMessageCreatedAtNull, &replyMessageUpdatedAtNull)

		if err != nil {
			fmt.Println("error scanning message", err)
			return nil, err
		}

		if replyIdNull.Valid {
			reply := chatv1.MessageData{}
			message.Reply = &reply

			message.Reply.Id = replyIdNull.String
			if replySenderIdNull.Valid {
				message.Reply.SenderId = replySenderIdNull.Int32
			}
			if replySenderNameNull.Valid {
				message.Reply.SenderName = replySenderNameNull.String
			}
			if replySenderPhoneNull.Valid {
				message.Reply.SenderPhone = replySenderPhoneNull.String
			}
			if replySenderAvatarNull.Valid {
				message.Reply.SenderAvatar = replySenderAvatarNull.String
			}
			if replyContentNull.Valid {
				message.Reply.Content = replyContentNull.String
			}
			if replyMessageRoomIdNull.Valid {
				message.Reply.RoomId = replyMessageRoomIdNull.String
			}
			if replyMessageCreatedAtNull.Valid {
				message.Reply.CreatedAt = replyMessageCreatedAtNull.String
			}
			if replyMessageUpdatedAtNull.Valid {
				message.Reply.UpdatedAt = replyMessageUpdatedAtNull.String
			}
		}

		//tags
		queryTags := dbpq.QueryBuilder().
			Select("room_message_tag.user_id", "public.\"user\".name", "public.\"user\".phone", "room_message_tag.tag", "room_message_tag.message_id").
			From("room_message_tag").
			InnerJoin("public.\"user\" ON room_message_tag.user_id = public.\"user\".id").
			Where(sq.Eq{"room_message_tag.message_id": messageId}).
			Where(sq.Eq{"room_message_tag.deleted_at": nil})

		queryString, args, err := queryTags.ToSql()
		if err != nil {
			return nil, err
		}

		rowsTags, err := r.db.QueryContext(ctx, queryString, args...)
		if err != nil {
			return nil, err
		}

		for rowsTags.Next() {
			var tag chatv1.Mention
			err = rowsTags.Scan(&tag.Id, &tag.Name, &tag.Phone, &tag.Tag, &tag.MessageId)
			if err != nil {
				return nil, err
			}
			message.Mentions = append(message.Mentions, &tag)
		}

		rowsTags.Close()

		//reactions
		queryReactions := dbpq.QueryBuilder().
			Select("room_message_reaction.\"reactedById\"", "room_message_reaction.reaction", "room_message_reaction.\"messageId\"").
			From("room_message_reaction").
			Where(sq.Eq{"room_message_reaction.\"messageId\"": messageId}).
			Where(sq.Eq{"room_message_reaction.deleted_at": nil})

		queryString, args, err = queryReactions.ToSql()
		if err != nil {
			return nil, err
		}

		rowsReactions, err := r.db.QueryContext(ctx, queryString, args...)
		if err != nil {
			return nil, err
		}

		for rowsReactions.Next() {
			var reaction chatv1.Reaction
			err = rowsReactions.Scan(&reaction.ReactedById, &reaction.Reaction, &reaction.MessageId)
			if err != nil {
				return nil, err
			}
			message.Reactions = append(message.Reactions, &reaction)
		}

		rowsReactions.Close()

		return &message, nil
	}

	rows.Close()

	return nil, nil
}

func (r *SQLRoomRepository) UpdateMessage(ctx context.Context, userId int, messageId string, content string) error {

	query := dbpq.QueryBuilder().
		Update("room_message").
		Set("content", content).
		Set("updated_at", time.Now()).
		Set("edited", true).
		Where(sq.Eq{"id": messageId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *SQLRoomRepository) DeleteMessage(ctx context.Context, userId int, messageId []string) error {

	query := dbpq.QueryBuilder().
		Update("room_message").
		Set("deleted_at", time.Now()).
		Set("updated_at", time.Now()).
		Set("\"isDeleted\"", true).
		Where(sq.Eq{"id": messageId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *SQLRoomRepository) GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error) {

	// Establecer un límite predeterminado si no se proporciona uno
	/*var effectiveLimit uint32 = 50
	if limit != nil && *limit > 0 {
		effectiveLimit = *limit
	}*/

	var beforeCreatedAt = ""
	var afterCreatedAt = ""
	if req != nil {
		if req.BeforeMessageId != nil {
			query := dbpq.QueryBuilder().Select("created_at").From("room_message").Where(sq.Eq{"id": *req.BeforeMessageId})
			queryString, args, err := query.ToSql()
			if err != nil {
				return nil, nil, err
			}
			rows, err := r.db.QueryContext(ctx, queryString, args...)
			if err != nil {
				return nil, nil, err
			}
			if rows.Next() {
				rows.Scan(&beforeCreatedAt)
			}
		}
		if req.AfterMessageId != nil {
			query := dbpq.QueryBuilder().Select("created_at").From("room_message").Where(sq.Eq{"id": *req.AfterMessageId})
			queryString, args, err := query.ToSql()
			if err != nil {
				return nil, nil, err
			}
			rows, err := r.db.QueryContext(ctx, queryString, args...)
			if err != nil {
				return nil, nil, err
			}
			if rows.Next() {
				rows.Scan(&afterCreatedAt)
			}
		}
	}

	rowNumber := "1"
	if req != nil {
		if req.MessagesPerRoom > 0 {
			rowNumber = "ROW_NUMBER() OVER (PARTITION BY msg.room_id ORDER BY msg.created_at DESC) as rn"
		}
	}

	// Consulta base para obtener los mensajes
	query := dbpq.QueryBuilder().
		Select(
			"msg.id", "msg.room_id", "msg.sender_id", "sender.name", "sender.phone", "sender.avatar",
			"msg.content", "msg.status", "msg.created_at", "msg.updated_at", "msg.type",
			"msg.lifetime", "msg.location_name", "msg.location_latitude", "msg.location_longitude",
			"msg.origin", "msg.contact_id", "msg.contact_name", "msg.contact_phone", "msg.file", "msg.edited", "msg.\"isDeleted\"",
			"msg.event",
			"msg.forwarded_message_id", "fwd_sender.id", "fwd_sender.name", "fwd_sender.phone", "fwd_sender.avatar",
			"msg.replied_message_id", "reply.sender_id", "reply_sender.name", "reply_sender.phone", "reply_sender.avatar", "reply.content", "reply.type",
			"reply.room_id", "reply.created_at", "reply.updated_at", "meta.read_at",
			rowNumber,
		).
		From("room_message AS msg").
		InnerJoin("public.\"user\" AS sender ON msg.sender_id = sender.id").
		InnerJoin("room_member AS member ON member.user_id = ? AND member.room_id = msg.room_id").
		LeftJoin("room_message_meta AS meta ON msg.id = meta.message_id AND meta.user_id = ? AND meta.\"isDeleted\" = false").
		LeftJoin("public.\"user\" AS fwd_sender ON msg.forwarded_message_original_sender = fwd_sender.id").
		LeftJoin("room_message AS reply ON msg.replied_message_id = reply.id").
		LeftJoin("public.\"user\" AS reply_sender ON reply.sender_id = reply_sender.id").
		Where("(meta.\"isSenderBlocked\" IS NULL OR meta.\"isSenderBlocked\" = false)").
		Where(sq.Eq{"msg.deleted_at": nil}).
		Where(sq.Eq{"member.removed_at": nil})

	if req != nil {
		if req.Id != "" {
			query = query.Where(sq.Eq{"msg.room_id": req.Id})
		}
		if req.BeforeDate != nil {
			if *req.BeforeDate != "" {
				query = query.Where(sq.Lt{"msg.updated_at": *req.BeforeDate})
			}
		}
		if req.BeforeMessageId != nil {
			if *req.BeforeMessageId != "" && beforeCreatedAt != "" {
				query = query.Where(sq.Lt{"msg.created_at": beforeCreatedAt})
			}
		}
		if req.AfterDate != nil {
			if *req.AfterDate != "" {
				query = query.Where(sq.Gt{"msg.updated_at": *req.AfterDate})
			}
		}
		if req.AfterMessageId != nil {
			if *req.AfterMessageId != "" && afterCreatedAt != "" {
				query = query.Where(sq.Gt{"msg.created_at": afterCreatedAt})
			}
		}
		if req.Page > 0 && req.Limit > 0 && req.MessagesPerRoom == 0 {
			query = query.Offset(uint64((req.Page - 1) * req.Limit)).
				Limit(uint64(req.Limit))
		}
	}

	query = query.OrderBy("msg.created_at DESC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, nil, err
	}

	if req.MessagesPerRoom > 0 {
		queryMessagesPerRoom := dbpq.QueryBuilder().
			Select("*").
			From("(" + queryString + ") ranked_messages").
			Where("rn <= " + fmt.Sprintf("%d", req.MessagesPerRoom))

		queryString2, _, err := queryMessagesPerRoom.ToSql()
		if err != nil {
			return nil, nil, err
		}

		queryString = queryString2
	}

	args = append([]any{userId, userId}, args...)

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	data := []*chatv1.MessageData{}
	for rows.Next() {
		var message chatv1.MessageData
		replyIdNull := sql.NullString{}
		replySenderIdNull := sql.NullInt32{}
		replySenderNameNull := sql.NullString{}
		replySenderPhoneNull := sql.NullString{}
		replySenderAvatarNull := sql.NullString{}
		replyContentNull := sql.NullString{}
		replyTypeNull := sql.NullString{}
		replyMessageRoomIdNull := sql.NullString{}
		replyMessageCreatedAtNull := sql.NullString{}
		replyMessageUpdatedAtNull := sql.NullString{}
		rowNumberNull := sql.NullString{}

		senderIdNull := sql.NullInt32{}
		senderNameNull := sql.NullString{}
		senderPhoneNull := sql.NullString{}
		senderAvatarNull := sql.NullString{}

		readAtNull := sql.NullString{}

		err = rows.Scan(
			&message.Id, &message.RoomId, &senderIdNull, &senderNameNull, &senderPhoneNull, &senderAvatarNull,
			&message.Content, &message.Status, &message.CreatedAt, &message.UpdatedAt, &message.Type,
			&message.Lifetime, &message.LocationName, &message.LocationLatitude, &message.LocationLongitude,
			&message.Origin, &message.ContactId, &message.ContactName, &message.ContactPhone, &message.File, &message.Edited, &message.IsDeleted,
			&message.Event,
			&message.ForwardedMessageId, &message.ForwardedMessageSenderId, &message.ForwardedMessageSenderName, &message.ForwardedMessageSenderPhone, &message.ForwardedMessageSenderAvatar,
			&replyIdNull, &replySenderIdNull, &replySenderNameNull, &replySenderPhoneNull, &replySenderAvatarNull, &replyContentNull, &replyTypeNull,
			&replyMessageRoomIdNull, &replyMessageCreatedAtNull, &replyMessageUpdatedAtNull, &readAtNull,
			&rowNumberNull,
		)
		if err != nil {
			return nil, nil, err
		}

		if senderIdNull.Valid {
			message.SenderId = senderIdNull.Int32
		}
		if senderNameNull.Valid {
			message.SenderName = senderNameNull.String
		}
		if senderPhoneNull.Valid {
			message.SenderPhone = senderPhoneNull.String
		}
		if senderAvatarNull.Valid {
			message.SenderAvatar = senderAvatarNull.String
		}

		if replyIdNull.Valid {
			message.Reply = &chatv1.MessageData{
				Id:           replyIdNull.String,
				SenderId:     replySenderIdNull.Int32,
				SenderName:   replySenderNameNull.String,
				SenderPhone:  replySenderPhoneNull.String,
				SenderAvatar: replySenderAvatarNull.String,
				Content:      replyContentNull.String,
				Type:         replyTypeNull.String,
				RoomId:       replyMessageRoomIdNull.String,
				CreatedAt:    replyMessageCreatedAtNull.String,
				UpdatedAt:    replyMessageUpdatedAtNull.String,
			}
		}

		if message.SenderId != int32(userId) {
			if readAtNull.Valid && readAtNull.String != "" {
				message.Status = chatv1.MessageStatus_MESSAGE_STATUS_READ
			} else {
				message.Status = chatv1.MessageStatus_MESSAGE_STATUS_SENT
			}
		}

		data = append(data, &message)

	}

	if len(data) > 0 {

		var allMessageIds []string
		for _, message := range data {
			allMessageIds = append(allMessageIds, message.Id)
		}

		var allTags []*chatv1.Mention
		var allReactions []*chatv1.Reaction

		//tags
		queryTags := dbpq.QueryBuilder().
			Select("room_message_tag.user_id", "public.\"user\".name", "public.\"user\".phone", "room_message_tag.tag", "room_message_tag.message_id").
			From("room_message_tag").
			InnerJoin("public.\"user\" ON room_message_tag.user_id = public.\"user\".id").
			Where(sq.Eq{"room_message_tag.message_id": allMessageIds}).
			Where(sq.Eq{"room_message_tag.deleted_at": nil})

		queryString, args, err := queryTags.ToSql()
		if err != nil {
			return nil, nil, err
		}

		rowsTags, err := r.db.QueryContext(ctx, queryString, args...)
		if err != nil {
			return nil, nil, err
		}

		for rowsTags.Next() {
			var tag chatv1.Mention
			err = rowsTags.Scan(&tag.Id, &tag.Name, &tag.Phone, &tag.Tag, &tag.MessageId)
			if err != nil {
				return nil, nil, err
			}

			allTags = append(allTags, &tag)
		}

		rowsTags.Close()

		//reactions
		queryReactions := dbpq.QueryBuilder().
			Select("room_message_reaction.\"reactedById\"", "room_message_reaction.reaction", "room_message_reaction.\"messageId\"").
			From("room_message_reaction").
			Where(sq.Eq{"room_message_reaction.\"messageId\"": allMessageIds}).
			Where(sq.Eq{"room_message_reaction.deleted_at": nil})

		queryString, args, err = queryReactions.ToSql()
		if err != nil {
			return nil, nil, err
		}

		rowsReactions, err := r.db.QueryContext(ctx, queryString, args...)
		if err != nil {
			return nil, nil, err
		}

		for rowsReactions.Next() {
			var reaction chatv1.Reaction
			err = rowsReactions.Scan(&reaction.ReactedById, &reaction.Reaction, &reaction.MessageId)
			if err != nil {
				return nil, nil, err
			}
			allReactions = append(allReactions, &reaction)
		}

		rowsReactions.Close()

		for i, message := range data {
			for _, tag := range allTags {
				if tag.MessageId == message.Id {
					data[i].Mentions = append(data[i].Mentions, tag)
				}
			}
			for _, reaction := range allReactions {
				if reaction.MessageId == message.Id {
					data[i].Reactions = append(data[i].Reactions, reaction)
				}
			}
		}
	}

	if req.MessagesPerRoom == 0 {

		// Consulta para obtener el total de items
		queryTotal := dbpq.QueryBuilder().
			Select("COUNT(*)").
			From("room_message AS msg").
			InnerJoin("room_member AS member ON member.user_id = ? AND member.room_id = msg.room_id").
			InnerJoin("room_message_meta AS meta ON msg.id = meta.message_id AND meta.user_id = ? AND meta.\"isDeleted\" = false").
			Where("(meta.\"isSenderBlocked\" IS NULL OR meta.\"isSenderBlocked\" = false)").
			Where(sq.Eq{"msg.deleted_at": nil}).
			Where(sq.Eq{"member.removed_at": nil})

		if req != nil {
			if req.Id != "" {
				queryTotal = queryTotal.Where(sq.Eq{"msg.room_id": req.Id})
			}

			if req.BeforeDate != nil {
				if *req.BeforeDate != "" {
					queryTotal = queryTotal.Where(sq.Lt{"msg.updated_at": *req.BeforeDate})
				}
			}

			if req.BeforeMessageId != nil {
				if *req.BeforeMessageId != "" && beforeCreatedAt != "" {
					queryTotal = queryTotal.Where(sq.Lt{"msg.created_at": beforeCreatedAt})
				}
			}

			if req.AfterDate != nil {
				if *req.AfterDate != "" {
					queryTotal = queryTotal.Where(sq.Gt{"msg.updated_at": *req.AfterDate})
				}
			}

			if req.AfterMessageId != nil {
				if *req.AfterMessageId != "" && afterCreatedAt != "" {
					queryTotal = queryTotal.Where(sq.Gt{"msg.created_at": afterCreatedAt})
				}
			}
		}

		queryTotalString, argsTotal, err := queryTotal.ToSql()
		if err != nil {
			return nil, nil, err
		}

		argsTotal = append([]any{userId, userId}, argsTotal...)

		var totalItemsCount int64
		err = r.db.QueryRowContext(ctx, queryTotalString, argsTotal...).Scan(&totalItemsCount)
		if err != nil {
			return nil, nil, err
		}

		meta := chatv1.PaginationMeta{
			TotalItems:   uint32(totalItemsCount),
			ItemCount:    uint32(len(data)),
			ItemsPerPage: req.Limit,
			TotalPages:   uint32(math.Ceil(float64(totalItemsCount) / float64(req.Limit))),
			CurrentPage:  req.Page,
		}

		return data, &meta, nil

	} else {
		meta := chatv1.PaginationMeta{
			TotalItems:   uint32(0),
			ItemCount:    uint32(len(data)),
			ItemsPerPage: req.Limit,
			TotalPages:   1,
			CurrentPage:  req.Page,
		}

		return data, &meta, nil
	}
}

func (r *SQLRoomRepository) ReactToMessage(ctx context.Context, userId int, messageId string, reaction string) error {

	// Iniciar transacción
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Verificar si ya existe una reacción para este mensaje y usuario
	query := dbpq.QueryBuilder().
		Select("id", "reaction").
		From("room_message_reaction").
		Where(sq.Eq{"\"messageId\"": messageId}).
		Where(sq.Eq{"\"reactedById\"": userId}).
		Where(sq.Eq{"deleted_at": nil}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx, queryString, args...)
	if err != nil {
		return err
	}

	var existingReactionId sql.NullString
	var existingReaction sql.NullString
	var reactionExists bool

	if rows.Next() {
		err = rows.Scan(&existingReactionId, &existingReaction)
		if err != nil {
			return err
		}
		reactionExists = true
	}

	rows.Close()

	// 2. Si existe una reacción
	if reactionExists {
		// Si es la misma reacción, la eliminamos (soft delete)
		if reaction == "" {
			queryDelete := dbpq.QueryBuilder().
				Update("room_message_reaction").
				Set("deleted_at", sq.Expr("NOW()")).
				Where(sq.Eq{"id": existingReactionId.String})

			queryString, args, err := queryDelete.ToSql()
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx, queryString, args...)
			if err != nil {
				return err
			}
		} else {
			// Si es diferente, la actualizamos
			queryUpdate := dbpq.QueryBuilder().
				Update("room_message_reaction").
				Set("reaction", reaction).
				Set("updated_at", sq.Expr("NOW()")).
				Where(sq.Eq{"id": existingReactionId.String})

			queryString, args, err := queryUpdate.ToSql()
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx, queryString, args...)
			if err != nil {
				return err
			}
		}
	} else {
		// 3. Si no existe, se guarda una nueva reacción
		queryInsert := dbpq.QueryBuilder().
			Insert("room_message_reaction").
			SetMap(sq.Eq{
				"messageId":   messageId,
				"reactedById": userId,
				"reaction":    reaction,
				"created_at":  sq.Expr("NOW()"),
				"updated_at":  sq.Expr("NOW()"),
			})

		queryString, args, err := queryInsert.ToSql()
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, queryString, args...)
		if err != nil {
			return err
		}
	}

	// Actualizar el updated_at del mensaje
	queryUpdate := dbpq.QueryBuilder().
		Update("room_message").
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": messageId})

	queryStringUpdate, argsUpdate, err := queryUpdate.ToSql()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, queryStringUpdate, argsUpdate...)
	if err != nil {
		return err
	}

	// Commit de la transacción
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *SQLRoomRepository) MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error) {
	// Si no hay IDs de mensajes, no hay nada que hacer.
	if len(messageIds) == 0 {
		return 0, nil
	}

	// Iniciar transacción
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Since
	if since != "" {
		query := dbpq.QueryBuilder().
			Select("room_message.id").
			From("room_message").
			LeftJoin("room_message_meta ON room_message.id = room_message_meta.message_id AND room_message_meta.user_id = ?").
			Where(sq.Lt{"room_message.created_at": since}).
			Where(sq.Eq{"room_message.room_id": roomId}).
			Where(sq.Expr("room_message_meta.read_at IS NULL"))

		queryString, args, err := query.ToSql()
		if err != nil {
			return 0, fmt.Errorf("error building sql for selecting existing records: %w", err)
		}

		args = append([]any{userId}, args...)

		rows, err := tx.QueryContext(ctx, queryString, args...)
		if err != nil {
			return 0, fmt.Errorf("error executing select query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var messageId string
			err = rows.Scan(&messageId)
			if err != nil {
				return 0, fmt.Errorf("error scanning existing record: %w", err)
			}
			messageIds = append(messageIds, messageId)
		}
	}

	// 1. Obtener los registros existentes de room_message_meta
	query := dbpq.QueryBuilder().
		Select("message_id", "read_at").
		From("public.room_message_meta").
		Where(sq.Eq{"user_id": userId}).
		Where(sq.Eq{"message_id": messageIds})

	queryString, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("error building sql for selecting existing records: %w", err)
	}

	rows, err := tx.QueryContext(ctx, queryString, args...)
	if err != nil {
		return 0, fmt.Errorf("error executing select query: %w", err)
	}
	defer rows.Close()

	// Mapa para rastrear mensajes que ya tienen metadatos
	existingMessages := make(map[string]bool)
	messagesToUpdate := make([]string, 0)
	messagesToCreate := make([]string, 0)

	// Procesar registros existentes
	for rows.Next() {
		var messageId string
		var readAt sql.NullString

		err = rows.Scan(&messageId, &readAt)
		if err != nil {
			return 0, fmt.Errorf("error scanning existing record: %w", err)
		}

		existingMessages[messageId] = true

		// Si read_at es NULL, agregar a la lista de actualizaciones
		if !readAt.Valid {
			messagesToUpdate = append(messagesToUpdate, messageId)
		}
	}

	// Identificar mensajes que necesitan nuevos registros
	for _, messageId := range messageIds {
		if !existingMessages[messageId] {
			messagesToCreate = append(messagesToCreate, messageId)
		}
	}

	// 2. Actualizar registros existentes con read_at NULL
	if len(messagesToUpdate) > 0 {

		updateQuery := dbpq.QueryBuilder().
			Update("public.room_message_meta").
			Set("read_at", sq.Expr("NOW()")).
			Where(sq.Eq{"user_id": userId}).
			Where(sq.Eq{"message_id": messagesToUpdate}).
			Where(sq.Eq{"read_at": nil})

		updateQueryString, updateArgs, err := updateQuery.ToSql()
		if err != nil {
			return 0, fmt.Errorf("error building sql for updating records: %w", err)
		}

		_, err = tx.ExecContext(ctx, updateQueryString, updateArgs...)
		if err != nil {
			return 0, fmt.Errorf("error executing update query: %w", err)
		}
	}

	// 3. Crear nuevos registros para mensajes que no existen
	if len(messagesToCreate) > 0 {
		// Crear un INSERT con múltiples VALUES para todos los mensajes
		insertQuery := dbpq.QueryBuilder().
			Insert("public.room_message_meta").
			Columns("message_id", "user_id", "read_at", "\"isDeleted\"", "\"isSenderBlocked\"")

		// Agregar VALUES para cada mensaje
		for _, messageId := range messagesToCreate {
			insertQuery = insertQuery.Values(messageId, userId, sq.Expr("NOW()"), false, false)
		}

		insertQueryString, insertArgs, err := insertQuery.ToSql()
		if err != nil {
			return 0, fmt.Errorf("error building sql for inserting records: %w", err)
		}

		_, err = tx.ExecContext(ctx, insertQueryString, insertArgs...)
		if err != nil {
			return 0, fmt.Errorf("error executing insert query: %w", err)
		}
	}

	queryUpdateMessages := dbpq.QueryBuilder().
		Update("room_message").
		Set("status", chatv1.MessageStatus_MESSAGE_STATUS_READ).
		Where(sq.Eq{"id": messageIds})

	queryStringMessages, argsMessages, err := queryUpdateMessages.ToSql()
	if err != nil {
		return 0, fmt.Errorf("error building sql for updating records: %w", err)
	}

	_, err = tx.ExecContext(ctx, queryStringMessages, argsMessages...)
	if err != nil {
		return 0, fmt.Errorf("error executing update query: %w", err)
	}

	// Commit de la transacción
	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("error committing transaction: %w", err)
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	return int32(len(messagesToCreate) + len(messagesToUpdate)), nil

}

// Función auxiliar para obtener los últimos mensajes de múltiples salas
// Útil cuando se necesita una alternativa a LATERAL JOIN para listas grandes
/*func (r *SQLRoomRepository) getLastMessagesForRooms(ctx context.Context, roomIds []string) (map[string]*chatv1.MessageData, error) {
	if len(roomIds) == 0 {
		return make(map[string]*chatv1.MessageData), nil
	}

	query := dbpq.QueryBuilder().
		Select("DISTINCT ON (msg.room_id) msg.room_id, msg.id, msg.content, msg.type, msg.created_at, msg.sender_id, sender.name, sender.phone").
		From("room_message AS msg").
		InnerJoin("public.\"user\" AS sender ON msg.sender_id = sender.id").
		Where(sq.Eq{"msg.room_id": roomIds}).
		Where(sq.Eq{"msg.deleted_at": nil}).
		Where(sq.Eq{"meta.\"isDeleted\"": false}).
		OrderBy("msg.room_id, msg.created_at DESC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*chatv1.MessageData)
	for rows.Next() {
		var roomId string
		var message chatv1.MessageData

		err = rows.Scan(&roomId, &message.Id, &message.Content, &message.Type,
			&message.CreatedAt, &message.SenderId, &message.SenderName, &message.SenderPhone)
		if err != nil {
			return nil, err
		}

		result[roomId] = &message
	}

	return result, nil
}*/

func (r *SQLRoomRepository) GetMessageRead(ctx context.Context, req *chatv1.GetMessageReadRequest) ([]*chatv1.MessageUserRead, *chatv1.PaginationMeta, error) {

	query := dbpq.QueryBuilder().
		Select("uu.id", "uu.name", "uu.avatar", "uu.phone", "room_message_meta.read_at").
		From("room_message_meta").
		InnerJoin(`public."user" AS uu ON room_message_meta.user_id = uu.id`).
		Where(sq.Eq{"room_message_meta.message_id": req.Id}).
		Where(sq.Eq{"room_message_meta.deleted_at": nil}).
		Where(sq.Expr("room_message_meta.read_at IS NOT NULL"))

	if req.Page > 0 && req.Limit > 0 {
		query = query.Offset(uint64((req.Page - 1) * req.Limit)).Limit(uint64(req.Limit))
	}

	query = query.OrderBy("room_message_meta.read_at ASC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, nil, err
	}

	items := make([]*chatv1.MessageUserRead, 0)
	for rows.Next() {
		var item chatv1.MessageUserRead
		err = rows.Scan(&item.UserId, &item.UserName, &item.UserAvatar, &item.UserPhone, &item.ReadAt)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, &item)
	}

	queryTotal := dbpq.QueryBuilder().
		Select("COUNT(*)").
		From("room_message_meta").
		Where(sq.Eq{"room_message_meta.message_id": req.Id}).
		Where(sq.Eq{"room_message_meta.deleted_at": nil}).
		Where(sq.Expr("room_message_meta.read_at IS NOT NULL"))

	queryTotalString, argsTotal, err := queryTotal.ToSql()
	if err != nil {
		return nil, nil, err
	}

	var totalItemsCount int64
	err = r.db.QueryRowContext(ctx, queryTotalString, argsTotal...).Scan(&totalItemsCount)
	if err != nil {
		return nil, nil, err
	}

	meta := chatv1.PaginationMeta{
		TotalItems:   uint32(totalItemsCount),
		ItemCount:    uint32(len(items)),
		ItemsPerPage: req.Limit,
		TotalPages:   uint32(math.Ceil(float64(totalItemsCount) / float64(req.GetLimit()))),
		CurrentPage:  req.GetPage(),
	}

	return items, &meta, nil
}

func (r *SQLRoomRepository) GetMessageReactions(ctx context.Context, req *chatv1.GetMessageReactionsRequest) ([]*chatv1.Reaction, *chatv1.PaginationMeta, error) {

	query := dbpq.QueryBuilder().
		Select("reaction", "uu.id", "uu.name", "uu.avatar", "uu.phone", "room_message_reaction.\"messageId\"").
		From("room_message_reaction").
		InnerJoin(`public."user" AS uu ON room_message_reaction."reactedById" = uu.id`).
		Where(sq.Eq{`room_message_reaction."messageId"`: req.Id}).
		Where(sq.Eq{"room_message_reaction.deleted_at": nil})

	if req.Page > 0 && req.Limit > 0 {
		query = query.Offset(uint64((req.Page - 1) * req.Limit)).Limit(uint64(req.Limit))
	}

	query = query.OrderBy("room_message_reaction.created_at DESC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, nil, err
	}

	items := make([]*chatv1.Reaction, 0)
	for rows.Next() {
		var item chatv1.Reaction
		err = rows.Scan(&item.Reaction, &item.ReactedById, &item.ReactedByName, &item.ReactedByAvatar, &item.ReactedByPhone, &item.MessageId)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, &item)
	}

	queryTotal := dbpq.QueryBuilder().
		Select("COUNT(*)").
		From("room_message_reaction").
		Where(sq.Eq{"room_message_reaction.\"messageId\"": req.Id}).
		Where(sq.Eq{"room_message_reaction.deleted_at": nil})

	queryTotalString, argsTotal, err := queryTotal.ToSql()
	if err != nil {
		return nil, nil, err
	}

	var totalItemsCount int64
	err = r.db.QueryRowContext(ctx, queryTotalString, argsTotal...).Scan(&totalItemsCount)
	if err != nil {
		return nil, nil, err
	}

	meta := chatv1.PaginationMeta{
		TotalItems:   uint32(totalItemsCount),
		ItemCount:    uint32(len(items)),
		ItemsPerPage: req.Limit,
		TotalPages:   uint32(math.Ceil(float64(totalItemsCount) / float64(req.Limit))),
		CurrentPage:  req.Page,
	}

	return items, &meta, nil
}

func (r *SQLRoomRepository) GetUserByID(ctx context.Context, id int) (*User, error) {

	query := dbpq.QueryBuilder().
		Select(
			`public.user."id"`,
			`public.user."name"`,
			`public.user."phone"`,
			`public.user."email"`,
			`public.user."avatar"`,
			`public.user."created_at"`,
			`public.user."dni"`,
		).
		From("public.user").
		Where(sq.Eq{"public.user.\"id\"": id}).
		Where(sq.Eq{"public.user.\"deleted_at\"": nil}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		var user User
		err = rows.Scan(
			&user.ID,
			&user.Name,
			&user.Phone,
			&user.Email,
			&user.Avatar,
			&user.CreatedAt,
			&user.Dni,
		)
		if err != nil {
			return nil, err
		}
		return &user, nil
	}

	return nil, nil
}

func (r *SQLRoomRepository) GetUsersByID(ctx context.Context, ids []int) ([]User, error) {

	query := dbpq.QueryBuilder().
		Select(`public.user."id"`, `public.user."name"`, `public.user."phone"`, `public.user."email"`, `public.user."avatar"`, `public.user."created_at"`, `public.user."dni"`).
		From("public.user").
		Where(sq.Eq{"public.user.\"id\"": ids}).
		Where(sq.Eq{"public.user.\"deleted_at\"": nil}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0)

	for rows.Next() {
		var user User
		err = rows.Scan(&user.ID, &user.Name, &user.Phone, &user.Email, &user.Avatar, &user.CreatedAt, &user.Dni)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *SQLRoomRepository) GetAllUserIDs(ctx context.Context) ([]int, error) {
	query := dbpq.QueryBuilder().
		Select(`public.user."id"`).
		From("public.user").
		Where(sq.Eq{"public.user.\"deleted_at\"": nil}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	users := make([]int, 0)

	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		users = append(users, id)
	}

	return users, nil
}

func (r *SQLRoomRepository) GetMessageSender(ctx context.Context, userId int, senderMessageId string) (*chatv1.MessageData, error) {

	query := dbpq.QueryBuilder().
		Select(
			"room_message.id",
			"room_message.created_at",
			"room_message.room_id",
			"room_message.sender_id",
			"room_message.audio_transcription",
			"room_message.file",
			"room_message.type",
			"room_message.sender_message_id",
			"room_message.status",
		).
		From("room_message").
		Where(sq.Eq{"room_message.sender_message_id": senderMessageId}).
		Where(sq.Eq{"room_message.sender_id": userId}).
		Where(sq.Eq{"room_message.deleted_at": nil}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		var message chatv1.MessageData

		err = rows.Scan(
			&message.Id,
			&message.CreatedAt,
			&message.RoomId,
			&message.SenderId,
			&message.AudioTranscription,
			&message.File,
			&message.Type,
			&message.SenderMessageId,
			&message.Status,
		)

		if err != nil {
			fmt.Println("error scanning message", err)
			return nil, err
		}

		return &message, nil
	}

	rows.Close()

	return nil, nil
}

func (r *SQLRoomRepository) CreateMessageMetaForParticipants(ctx context.Context, roomID string, messageID string, senderID int) error {
	// 1. Get all participants for the room
	participants, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: roomID})
	if err != nil {
		return fmt.Errorf("failed to get room participants for fan-out: %w", err)
	}

	if len(participants) == 0 {
		return nil
	}

	// 2. Prepare values for batch insert, excluding the sender
	type metaValue struct {
		messageID string
		userID    int32
		isBlocked bool
	}
	var valuesToInsert []metaValue

	for _, p := range participants {
		if int(p.Id) == senderID {
			continue
		}
		// Simplified isSenderBlocked logic for now
		valuesToInsert = append(valuesToInsert, metaValue{messageID: messageID, userID: p.Id, isBlocked: false})
	}

	if len(valuesToInsert) == 0 {
		return nil // No one else to notify
	}

	// 3. Execute in batches

	batchSize := 100
	for i := 0; i < len(valuesToInsert); i += batchSize {
		end := min(i+batchSize, len(valuesToInsert))
		batch := valuesToInsert[i:end]

		insertQuery := dbpq.QueryBuilder().
			Insert("public.room_message_meta").
			Columns("message_id", "user_id", "read_at", "\"isDeleted\"", "\"isSenderBlocked\"")

		for _, v := range batch {
			insertQuery = insertQuery.Values(v.messageID, v.userID, nil, false, v.isBlocked)
		}

		queryString, args, err := insertQuery.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build message meta insert query for batch: %w", err)
		}

		_, err = r.db.ExecContext(ctx, queryString, args...)
		if err != nil {
			// Log the error and decide if we should continue or fail the whole process
			fmt.Printf("Error inserting message meta batch: %v. Continuing with next batches.\n", err)
		}
	}

	return nil
}

func (r *SQLRoomRepository) IsPartnerMuted(ctx context.Context, userId int, roomId string) (bool, error) {
	query := dbpq.QueryBuilder().
		Select("room_member.is_muted").
		From("room_member").
		Where(sq.Eq{"room_member.room_id": roomId}).
		Where(sq.Eq{"room_member.user_id": userId}).
		Where(sq.Eq{"room_member.deleted_at": nil}).
		Limit(1)

	queryString, args, err := query.ToSql()
	if err != nil {
		return false, err
	}

	rows, err := r.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return false, err
	}

	if rows.Next() {
		var isMuted sql.NullBool
		err = rows.Scan(&isMuted)
		if err != nil {
			return false, err
		}
		return isMuted.Bool, nil
	}

	return false, nil
}
