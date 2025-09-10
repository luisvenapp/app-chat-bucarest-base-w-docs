package roomsrepository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
	"github.com/scylladb-solutions/gocql/v2"
	"google.golang.org/protobuf/proto"
)

type ScyllaRoomRepository struct {
	session     *gocql.Session
	userFetcher UserFetcher
}

func NewScyllaRoomRepository(session *gocql.Session, userFetcher UserFetcher) RoomsRepository {
	return &ScyllaRoomRepository{
		session:     session,
		userFetcher: userFetcher,
	}
}

func (r *ScyllaRoomRepository) CreateRoom(ctx context.Context, userId int, req *chatv1.CreateRoomRequest) (*chatv1.Room, error) {
	// Lógica anti-duplicados para salas P2P
	if req.Type == "p2p" && len(req.Participants) > 0 {
		user1, user2 := sortUserIDs(userId, int(req.Participants[0]))
		var existingRoomID gocql.UUID
		err := r.session.Query(`SELECT room_id FROM p2p_room_by_users WHERE user1_id = ? AND user2_id = ?`, user1, user2).WithContext(ctx).Scan(&existingRoomID)
		if err == nil {
			return r.GetRoom(ctx, userId, existingRoomID.String(), true, false)
		}
		if err != gocql.ErrNotFound {
			return nil, fmt.Errorf("error al verificar si la sala p2p existe: %w", err)
		}
	}

	roomID := gocql.MustRandomUUID()
	now := time.Now()
	participantsSet := make(map[int32]bool)
	// Los participantes iniciales vienen en el request
	for _, p := range req.Participants {
		participantsSet[p] = true
	}
	// El creador siempre es un participante
	participantsSet[int32(userId)] = true

	sendMessage := req.SendMessage
	addMember := req.AddMember
	editGroup := req.EditGroup
	if req.Type == "p2p" {
		sendMessage = proto.Bool(true)
		addMember = proto.Bool(false)
		editGroup = proto.Bool(false)
	}
	joinAllUser := false

	encryptionData, err := utils.GenerateKeyEncript()
	if err != nil {
		fmt.Println("error", err)
		return nil, err
	}

	// --- PASO 1: Batch para operaciones que no son de contador ---
	batch := r.session.Batch(gocql.LoggedBatch)
	batch.Query(`INSERT INTO room_details (room_id, name, description, image, type, encryption_data, created_at, updated_at, join_all_user, send_message, add_member, edit_group) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		roomID, req.Name, req.Description, req.PhotoUrl, req.Type, encryptionData, now, now, joinAllUser, sendMessage, addMember, editGroup)

	for participantID := range participantsSet {
		role := "MEMBER"
		// El creador siempre es OWNER
		if int(participantID) == userId {
			role = "OWNER"
		}
		// En canales, el creador es OWNER
		if req.Type == "channel" && int(participantID) == userId {
			role = "OWNER"
		}
		batch.Query(`INSERT INTO participants_by_room (room_id, user_id, role, joined_at, is_muted) VALUES (?, ?, ?, ?, ?)`,
			roomID, participantID, role, now, false)
		batch.Query(`INSERT INTO rooms_by_user (user_id, is_pinned, last_message_at, room_id, room_name, room_image, room_type, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			participantID, false, now, roomID, req.Name, req.PhotoUrl, req.Type, role)
		batch.Query(`INSERT INTO room_membership_lookup (user_id, room_id, is_pinned, last_message_at) VALUES (?, ?, ?, ?)`,
			participantID, roomID, false, now)
	}

	if req.Type == "p2p" {
		user1, user2 := sortUserIDs(userId, int(req.Participants[0]))
		batch.Query(`INSERT INTO p2p_room_by_users (user1_id, user2_id, room_id) VALUES (?, ?, ?)`, user1, user2, roomID)
	}

	if err := r.session.ExecuteBatch(batch); err != nil {
		return nil, fmt.Errorf("error en batch de creación de sala: %w", err)
	}

	// --- PASO 2: Operaciones de contador por separado ---
	for participantID := range participantsSet {
		err := r.session.Query(`UPDATE room_counters_by_user SET unread_count = unread_count + 0 WHERE user_id = ? AND room_id = ?`, participantID, roomID).WithContext(ctx).Exec()
		if err != nil {
			fmt.Printf("Advertencia: no se pudo inicializar el contador para el usuario %d en la sala %s: %v\n", participantID, roomID, err)
		}
	}

	// --- PASO 3: Si es un canal `join_all_user`, iniciar la adición masiva en segundo plano ---
	if joinAllUser && req.Type == "channel" {
		go r.addAllSystemUsersToChannel(context.Background(), roomID)
	}

	return r.GetRoom(ctx, userId, roomID.String(), true, false)
}

// addAllSystemUsersToChannel se ejecuta en segundo plano para añadir todos los usuarios a un canal.
func (r *ScyllaRoomRepository) addAllSystemUsersToChannel(ctx context.Context, roomID gocql.UUID) {
	fmt.Printf("Iniciando adición masiva de usuarios al canal %s\n", roomID.String())

	allUserIDs, err := r.userFetcher.GetAllUserIDs(ctx)
	if err != nil {
		fmt.Printf("ERROR CRÍTICO: no se pudieron obtener todos los usuarios para el canal %s: %v\n", roomID.String(), err)
		return
	}

	batchSize := 100
	for i := 0; i < len(allUserIDs); i += batchSize {
		end := min(i+batchSize, len(allUserIDs))
		batchIDs := allUserIDs[i:end]

		_, err := r.AddParticipantToRoom(ctx, 0, roomID.String(), batchIDs) // userId 0 indica que es una operación de sistema
		if err != nil {
			fmt.Printf("Error al añadir lote de usuarios al canal %s: %v\n", roomID.String(), err)
		} else {
			fmt.Printf("Añadido lote de %d usuarios al canal %s\n", len(batchIDs), roomID.String())
		}
	}
	fmt.Printf("Finalizada la adición masiva de usuarios al canal %s\n", roomID.String())
}

func (r *ScyllaRoomRepository) GetRoom(ctx context.Context, userId int, roomId string, allData bool, useCache bool) (*chatv1.Room, error) {
	cacheKey := fmt.Sprintf("endpoint:chat:room:{%s}:shim:user:%d", roomId, userId)
	if allData {
		cacheKey = fmt.Sprintf("endpoint:chat:room:{%s}:user:%d", roomId, userId)
	}
	if useCache {
		if dataCached, existsCached := GetCachedRoom(ctx, cacheKey); existsCached {
			return dataCached, nil
		}
	}

	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return nil, fmt.Errorf("ID de sala inválido: %w", err)
	}

	room := &chatv1.Room{Id: roomId}
	var createdAt, updatedAt time.Time
	err = r.session.Query(`SELECT name, description, image, type, encryption_data, created_at, updated_at, join_all_user, send_message, add_member, edit_group FROM room_details WHERE room_id = ? LIMIT 1`, roomUUID).
		WithContext(ctx).Scan(&room.Name, &room.Description, &room.PhotoUrl, &room.Type, &room.EncryptionData, &createdAt, &updatedAt, &room.JoinAllUser, &room.SendMessage, &room.AddMember, &room.EditGroup)
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("error al obtener detalles de la sala: %w", err)
	}
	room.CreatedAt = createdAt.Format(time.RFC3339)
	room.UpdatedAt = updatedAt.Format(time.RFC3339)

	var isPinned bool
	var lastMessageAt time.Time
	err = r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)
	if err != nil && err != gocql.ErrNotFound {
		return nil, fmt.Errorf("error al buscar la membresía: %w", err)
	}

	if err == nil {
		var lastMessage chatv1.MessageData
		var lastMessageID gocql.UUID
		var lastMessageUpdatedAt time.Time
		err = r.session.Query(`SELECT role, is_muted, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
			userId, isPinned, lastMessageAt, roomUUID).WithContext(ctx).Scan(&room.Role, &room.IsMuted, &lastMessageID, &lastMessage.Content, &lastMessage.Type, &lastMessage.SenderId, &lastMessage.SenderName, &lastMessage.SenderPhone, &lastMessage.Status, &lastMessageUpdatedAt)
		if err != nil && err != gocql.ErrNotFound {
			return nil, fmt.Errorf("error al obtener datos de la sala del usuario: %w", err)
		}

		if lastMessageID.String() != (gocql.UUID{}).String() {
			lastMessage.Id = lastMessageID.String()
			lastMessage.UpdatedAt = lastMessageUpdatedAt.Format(time.RFC3339)
			room.LastMessage = &lastMessage
		} else {
			room.LastMessage = nil
		}
		room.IsPinned = isPinned
	}

	var unreadCount int64
	err = r.session.Query(`SELECT unread_count FROM room_counters_by_user WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&unreadCount)
	if err != nil && err != gocql.ErrNotFound {
		// No es un error fatal, el contador puede no existir
	}
	room.UnreadCount = int32(unreadCount)

	if allData {
		participants, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: roomId})
		if err != nil {
			return nil, err
		}
		if room.Type == "p2p" {
			for _, p := range participants {
				if int(p.Id) != userId {
					room.Partner = p
					var partnerMuted bool
					r.session.Query(`SELECT is_muted FROM participants_by_room WHERE room_id = ? AND user_id = ?`, roomUUID, p.Id).WithContext(ctx).Scan(&partnerMuted)
					room.Partner.IsPartnerMuted = partnerMuted
					break
				}
			}
		} else {
			room.Participants = participants
		}
	}

	if useCache {
		SetCachedRoom(ctx, roomId, cacheKey, room)
	}
	return room, nil
}

func (r *ScyllaRoomRepository) GetRoomList(ctx context.Context, userId int, pagination *chatv1.GetRoomsRequest) ([]*chatv1.Room, *chatv1.PaginationMeta, error) {
	baseQuery := `SELECT room_id, room_name, room_image, room_type, last_message_at, is_muted, is_pinned, role, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at FROM rooms_by_user WHERE user_id = ?`
	args := []any{userId}

	iter := r.session.Query(baseQuery, args...).WithContext(ctx).Iter()
	defer iter.Close()

	var allRooms []*chatv1.Room
	roomMap := make(map[string]*chatv1.Room)
	var roomIDs []gocql.UUID

	scanner := iter.Scanner()
	for scanner.Next() {
		var roomID gocql.UUID
		var roomName, roomImage, roomType, role string
		var lastMessageAt, lastMessageUpdatedAt time.Time
		var isMuted, isPinned bool
		var lastMessage chatv1.MessageData
		var lastMessageID gocql.UUID

		err := scanner.Scan(&roomID, &roomName, &roomImage, &roomType, &lastMessageAt, &isMuted, &isPinned, &role, &lastMessageID, &lastMessage.Content, &lastMessage.Type, &lastMessage.SenderId, &lastMessage.SenderName, &lastMessage.SenderPhone, &lastMessage.Status, &lastMessageUpdatedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("error al escanear fila de sala: %w", err)
		}

		room := &chatv1.Room{
			Id:            roomID.String(),
			Name:          roomName,
			PhotoUrl:      roomImage,
			Type:          roomType,
			LastMessageAt: lastMessageAt.Format(time.RFC3339),
			IsMuted:       isMuted,
			IsPinned:      isPinned,
			Role:          role,
		}

		if lastMessageID.String() != (gocql.UUID{}).String() {
			lastMessage.Id = lastMessageID.String()
			lastMessage.UpdatedAt = lastMessageUpdatedAt.Format(time.RFC3339)
			room.LastMessage = &lastMessage
		} else {
			room.LastMessage = nil
		}

		allRooms = append(allRooms, room)
		roomIDs = append(roomIDs, roomID)
		roomMap[roomID.String()] = room
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error del scanner al leer lista de salas: %w", err)
	}

	// Enriquecer con contadores de no leídos
	if len(roomIDs) > 0 {
		counterIter := r.session.Query(`SELECT room_id, unread_count FROM room_counters_by_user WHERE user_id = ? AND room_id IN ?`, userId, roomIDs).WithContext(ctx).Iter()
		var roomID gocql.UUID
		var unreadCount int64
		for counterIter.Scan(&roomID, &unreadCount) {
			if room, ok := roomMap[roomID.String()]; ok {
				room.UnreadCount = int32(unreadCount)
			}
		}
		counterIter.Close()
	}

	// Enriquecer con participantes para salas de grupo
	var groupRoomIDs []string
	for _, room := range allRooms {
		if room.Type == "group" {
			groupRoomIDs = append(groupRoomIDs, room.Id)
		}
	}

	if len(groupRoomIDs) > 0 {
		var wg sync.WaitGroup
		participantChan := make(chan struct {
			RoomID string
			Parts  []*chatv1.RoomParticipant
		}, len(groupRoomIDs))

		for _, roomID := range groupRoomIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				parts, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: id, Limit: 5})
				if err == nil {
					participantChan <- struct {
						RoomID string
						Parts  []*chatv1.RoomParticipant
					}{RoomID: id, Parts: parts}
				}
			}(roomID)
		}
		wg.Wait()
		close(participantChan)

		for pData := range participantChan {
			if room, ok := roomMap[pData.RoomID]; ok {
				room.Participants = pData.Parts
			}
		}
	}

	// Filtrado en la aplicación para la búsqueda
	var filteredRooms []*chatv1.Room
	if pagination != nil && pagination.Search != "" {
		searchTerm, err := removeAccents(strings.ToLower(pagination.Search))
		if err != nil {
			return nil, nil, fmt.Errorf("error al normalizar el término de búsqueda: %w", err)
		}

		for _, room := range allRooms {
			roomName, _ := removeAccents(strings.ToLower(room.Name))
			if strings.Contains(roomName, searchTerm) {
				filteredRooms = append(filteredRooms, room)
			}
		}
	} else {
		filteredRooms = allRooms
	}

	// Paginación en la aplicación
	if pagination != nil && pagination.Page > 0 && pagination.Limit > 0 {
		start := (pagination.Page - 1) * pagination.Limit
		end := start + pagination.Limit
		if start > uint32(len(filteredRooms)) {
			filteredRooms = []*chatv1.Room{}
		} else {
			if end > uint32(len(filteredRooms)) {
				end = uint32(len(filteredRooms))
			}
			filteredRooms = filteredRooms[start:end]
		}
	}

	meta := &chatv1.PaginationMeta{TotalItems: uint32(len(allRooms)), ItemCount: uint32(len(filteredRooms))}
	return filteredRooms, meta, nil
}

func (r *ScyllaRoomRepository) GetRoomParticipants(ctx context.Context, pagination *chatv1.GetRoomParticipantsRequest) ([]*chatv1.RoomParticipant, *chatv1.PaginationMeta, error) {
	roomUUID, err := gocql.ParseUUID(pagination.Id)
	if err != nil {
		return nil, nil, fmt.Errorf("ID de sala inválido: %w", err)
	}

	baseQuery := `SELECT user_id, role FROM participants_by_room WHERE room_id = ?`
	args := []any{roomUUID}

	if pagination != nil && pagination.Limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, int(pagination.Limit))
	}

	iter := r.session.Query(baseQuery, args...).WithContext(ctx).Iter()
	defer iter.Close()

	var userIDs []int
	participantMap := make(map[int]*chatv1.RoomParticipant)

	var userID int
	var role string
	for iter.Scan(&userID, &role) {
		userIDs = append(userIDs, userID)
		participantMap[userID] = &chatv1.RoomParticipant{Id: int32(userID), Role: role}
	}
	if err := iter.Close(); err != nil {
		return nil, nil, fmt.Errorf("error al leer participantes: %w", err)
	}

	users, err := r.userFetcher.GetUsersByID(ctx, userIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("error al obtener detalles de usuarios: %w", err)
	}

	var finalParticipants []*chatv1.RoomParticipant
	userMapFromService := make(map[int]User)
	for _, u := range users {
		userMapFromService[u.ID] = u
	}

	for _, id := range userIDs {
		if p, ok := participantMap[id]; ok {
			if user, userOK := userMapFromService[id]; userOK {
				p.Name = user.Name
				p.Phone = user.Phone
				if user.Avatar != nil {
					p.Avatar = *user.Avatar
				}
			}
			finalParticipants = append(finalParticipants, p)
		}
	}

	meta := &chatv1.PaginationMeta{TotalItems: uint32(len(finalParticipants)), ItemCount: uint32(len(finalParticipants))}
	return finalParticipants, meta, nil
}

func (r *ScyllaRoomRepository) LeaveRoom(ctx context.Context, userId int, roomId string, participants []int32, leaveAll bool) ([]User, error) {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return nil, fmt.Errorf("ID de sala inválido: %w", err)
	}

	pInt := make([]int, len(participants))
	for i, p := range participants {
		pInt[i] = int(p)
	}
	users, err := r.userFetcher.GetUsersByID(ctx, pInt)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los usuarios a eliminar: %w", err)
	}

	now := time.Now()
	for _, pID := range participants {
		var isPinned bool
		var lastMessageAt time.Time
		err := r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, pID, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)

		batch := r.session.Batch(gocql.LoggedBatch)
		if err == nil {
			batch.Query(`DELETE FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`, pID, isPinned, lastMessageAt, roomUUID)
		}
		batch.Query(`DELETE FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, pID, roomUUID)
		batch.Query(`INSERT INTO deleted_rooms_by_user (user_id, deleted_at, room_id, reason) VALUES (?, ?, ?, ?)`, pID, now, roomUUID, "removed")
		if err := r.session.ExecuteBatch(batch); err != nil {
			fmt.Printf("Error al procesar la salida del usuario %d de la sala %s: %v\n", pID, roomId, err)
		}
		r.session.Query(`DELETE FROM room_counters_by_user WHERE user_id = ? AND room_id = ?`, pID, roomUUID).WithContext(ctx).Exec()
	}

	batchParticipants := r.session.Batch(gocql.LoggedBatch)
	for _, pID := range participants {
		batchParticipants.Query(`DELETE FROM participants_by_room WHERE room_id = ? AND user_id = ?`, roomUUID, pID)
	}
	if err := r.session.ExecuteBatch(batchParticipants); err != nil {
		return nil, fmt.Errorf("error al eliminar de la tabla de participantes: %w", err)
	}

	DeleteRoomCacheByRoomID(ctx, roomId)
	return users, nil
}

func (r *ScyllaRoomRepository) UpdateRoom(ctx context.Context, userId int, roomId string, req *chatv1.UpdateRoomRequest) error {
	roomUUID, err := gocql.ParseUUID(req.Id)
	if err != nil {
		return err
	}
	err = r.session.Query(`UPDATE room_details SET name = ?, description = ?, image = ?, send_message = ?, add_member = ?, edit_group = ?, updated_at = ? WHERE room_id = ?`,
		req.Name, req.Description, req.PhotoUrl, req.SendMessage, req.AddMember, req.EditGroup, time.Now(), roomUUID).WithContext(ctx).Exec()
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)
	return nil
}

func (r *ScyllaRoomRepository) AddParticipantToRoom(ctx context.Context, userId int, roomId string, participants []int) ([]User, error) {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return nil, fmt.Errorf("ID de sala inválido: %w", err)
	}

	var roomName, roomImage, roomType string
	err = r.session.Query(`SELECT name, image, type FROM room_details WHERE room_id = ?`, roomUUID).WithContext(ctx).Scan(&roomName, &roomImage, &roomType)
	if err != nil {
		return nil, fmt.Errorf("no se pudo encontrar la sala para añadir participantes: %w", err)
	}

	users, err := r.userFetcher.GetUsersByID(ctx, participants)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los detalles de los usuarios a añadir: %w", err)
	}

	batch := r.session.Batch(gocql.LoggedBatch)
	now := time.Now()
	for _, user := range users {
		batch.Query(`INSERT INTO participants_by_room (room_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
			roomUUID, user.ID, "MEMBER", now)
		batch.Query(`INSERT INTO rooms_by_user (user_id, is_pinned, last_message_at, room_id, room_name, room_image, room_type, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			user.ID, false, now, roomUUID, roomName, roomImage, roomType, "MEMBER")
		batch.Query(`INSERT INTO room_membership_lookup (user_id, room_id, is_pinned, last_message_at) VALUES (?, ?, ?, ?)`,
			user.ID, roomUUID, false, now)
	}

	if err := r.session.ExecuteBatch(batch); err != nil {
		return nil, fmt.Errorf("error al añadir participantes en batch: %w", err)
	}

	for _, user := range users {
		r.session.Query(`UPDATE room_counters_by_user SET unread_count = unread_count + 0 WHERE user_id = ? AND room_id = ?`, user.ID, roomUUID).WithContext(ctx).Exec()
	}

	DeleteRoomCacheByRoomID(ctx, roomId)

	return users, nil
}

func (r *ScyllaRoomRepository) GetRoomListDeleted(ctx context.Context, userId int, since string) ([]string, error) {
	query := `SELECT room_id FROM deleted_rooms_by_user WHERE user_id = ?`
	args := []any{userId}

	if since != "" {
		const layout = "2006-01-02T15:04:05.000"
		sinceTime, err := time.Parse(layout, since)
		if err != nil {
			sinceTime, err = time.Parse(time.RFC3339, since)
		}
		if err != nil {
			return nil, fmt.Errorf("formato de fecha 'since' inválido (%s): %w", since, err)
		}
		query += " AND deleted_at > ?"
		args = append(args, sinceTime)
	}

	iter := r.session.Query(query, args...).WithContext(ctx).Iter()
	defer iter.Close()

	var deletedRoomIDs []string
	var roomID gocql.UUID

	for iter.Scan(&roomID) {
		deletedRoomIDs = append(deletedRoomIDs, roomID.String())
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("error al leer la lista de salas eliminadas: %w", err)
	}

	return deletedRoomIDs, nil
}

func (r *ScyllaRoomRepository) SaveMessage(ctx context.Context, userId int, req *chatv1.SendMessageRequest, room *chatv1.Room, contentDecrypted *string) (*chatv1.MessageData, error) {
	roomUUID, err := gocql.ParseUUID(req.RoomId)
	if err != nil {
		return nil, fmt.Errorf("ID de sala inválido: %w", err)
	}

	messageID := gocql.TimeUUID()
	now := time.Now()
	batch := r.session.Batch(gocql.LoggedBatch)

	batch.Query(`INSERT INTO messages_by_room (room_id, message_id, sender_id, content, content_decrypted, type, created_at, sender_message_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		roomUUID, messageID, userId, req.Content, contentDecrypted, req.Type, now, req.SenderMessageId)

	if req.SenderMessageId != nil && *req.SenderMessageId != "" {
		batch.Query(`INSERT INTO message_by_sender_message_id (sender_message_id, room_id, message_id) VALUES (?, ?, ?)`,
			*req.SenderMessageId, roomUUID, messageID)
	}

	batch.Query(`INSERT INTO room_by_message (message_id, room_id) VALUES (?, ?)`, messageID, roomUUID)

	if err := r.session.ExecuteBatch(batch); err != nil {
		return nil, fmt.Errorf("error al ejecutar el batch de guardado de mensaje: %w", err)
	}

	participants, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: req.RoomId})
	if err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los participantes para el fan-out: %w", err)
	}

	sender, err := r.userFetcher.GetUserByID(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener la información del remitente: %w", err)
	}

	// Fan-out para actualizar la lista de salas de cada participante
	for _, p := range participants {
		// Este es el patrón correcto: leer-eliminar-insertar para cada participante
		r.updateRoomForUser(ctx, int(p.Id), roomUUID, now, messageID, req, sender)
	}

	// Actualizar contadores y estados por separado
	for _, p := range participants {
		if p.Id != int32(userId) {
			r.session.Query(`UPDATE room_counters_by_user SET unread_count = unread_count + 1 WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID).WithContext(ctx).Exec()
			r.session.Query(`INSERT INTO message_status_by_user (user_id, room_id, message_id, status) VALUES (?, ?, ?, ?)`, p.Id, roomUUID, messageID, chatv1.MessageStatus_MESSAGE_STATUS_DELIVERED).WithContext(ctx).Exec()
		} else {
			r.session.Query(`INSERT INTO message_status_by_user (user_id, room_id, message_id, status) VALUES (?, ?, ?, ?)`, p.Id, roomUUID, messageID, chatv1.MessageStatus_MESSAGE_STATUS_SENT).WithContext(ctx).Exec()
		}
	}

	msg := &chatv1.MessageData{
		Id:         messageID.String(),
		RoomId:     req.RoomId,
		SenderId:   int32(userId),
		SenderName: sender.Name,
		Content:    req.Content,
		Status:     chatv1.MessageStatus_MESSAGE_STATUS_SENT,
		CreatedAt:  now.Format(time.RFC3339),
	}

	UpdateRoomCacheWithNewMessage(ctx, msg)
	return msg, nil
}

// updateRoomForUser es una función helper para el complejo fan-out de SaveMessage
func (r *ScyllaRoomRepository) updateRoomForUser(ctx context.Context, userId int, roomUUID gocql.UUID, newTime time.Time, newMsgId gocql.UUID, req *chatv1.SendMessageRequest, sender *User) {
	var isPinned bool
	var lastMessageAt time.Time
	err := r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)
	if err != nil {
		fmt.Printf("Error al buscar membresía para fan-out para usuario %d: %v\n", userId, err)
		return
	}

	var roomName, roomImage, roomType, role string
	var isMuted bool
	err = r.session.Query(`SELECT room_name, room_image, room_type, is_muted, role FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
		userId, isPinned, lastMessageAt, roomUUID).WithContext(ctx).Scan(&roomName, &roomImage, &roomType, &isMuted, &role)
	if err != nil {
		fmt.Printf("Error al leer datos de rooms_by_user para fan-out para usuario %d: %v\n", userId, err)
		return
	}

	batch := r.session.Batch(gocql.LoggedBatch)
	batch.Query(`DELETE FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
		userId, isPinned, lastMessageAt, roomUUID)
	batch.Query(`INSERT INTO rooms_by_user (user_id, is_pinned, last_message_at, room_id, room_name, room_image, room_type, is_muted, role, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userId, isPinned, newTime, roomUUID, roomName, roomImage, roomType, isMuted, role, newMsgId, req.Content, req.Type, sender.ID, sender.Name, sender.Phone, int(chatv1.MessageStatus_MESSAGE_STATUS_DELIVERED), newTime)
	batch.Query(`UPDATE room_membership_lookup SET last_message_at = ? WHERE user_id = ? AND room_id = ?`, newTime, userId, roomUUID)

	if err := r.session.ExecuteBatch(batch); err != nil {
		fmt.Printf("Error en batch de fan-out para usuario %d: %v\n", userId, err)
	}
}

func (r *ScyllaRoomRepository) GetMessagesFromRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error) {
	if req != nil && req.Id != "" {
		return r.getMessagesFromSingleRoom(ctx, userId, req)
	}
	return r.getMessagesFromAllRooms(ctx, userId, req)
}

func (r *ScyllaRoomRepository) getMessagesFromSingleRoom(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error) {
	roomUUID, err := gocql.ParseUUID(req.Id)
	if err != nil {
		return nil, nil, fmt.Errorf("ID de sala inválido: %w", err)
	}

	baseQuery := `SELECT message_id, sender_id, content, type, created_at, edited FROM messages_by_room WHERE room_id = ?`
	args := []any{roomUUID}

	if req.BeforeMessageId != nil && *req.BeforeMessageId != "" {
		beforeUUID, err := gocql.ParseUUID(*req.BeforeMessageId)
		if err != nil {
			return nil, nil, fmt.Errorf("before_message_id inválido: %w", err)
		}
		baseQuery += " AND message_id < ?"
		args = append(args, beforeUUID)
	}
	if req.AfterMessageId != nil && *req.AfterMessageId != "" {
		afterUUID, err := gocql.ParseUUID(*req.AfterMessageId)
		if err != nil {
			return nil, nil, fmt.Errorf("after_message_id inválido: %w", err)
		}
		baseQuery += " AND message_id > ?"
		args = append(args, afterUUID)
	}

	if req.Limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, int(req.Limit))
	}

	iter := r.session.Query(baseQuery, args...).WithContext(ctx).Iter()
	defer iter.Close()

	messages, userIDs, err := r.scanMessagesAndCollectUserIDs(iter)
	if err != nil {
		return nil, nil, err
	}

	err = r.enrichMessagesWithUserDetails(ctx, messages, userIDs)
	if err != nil {
		return nil, nil, err
	}

	err = r.enrichMessagesWithStatus(ctx, messages, userId, roomUUID)
	if err != nil {
		return nil, nil, err
	}

	meta := &chatv1.PaginationMeta{ItemCount: uint32(len(messages))}
	return messages, meta, nil
}

func (r *ScyllaRoomRepository) getMessagesFromAllRooms(ctx context.Context, userId int, req *chatv1.GetMessageHistoryRequest) ([]*chatv1.MessageData, *chatv1.PaginationMeta, error) {
	userRooms, _, err := r.GetRoomList(ctx, userId, &chatv1.GetRoomsRequest{})
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudieron obtener las salas del usuario: %w", err)
	}

	var wg sync.WaitGroup
	msgChan := make(chan []*chatv1.MessageData, len(userRooms))
	errChan := make(chan error, len(userRooms))

	for _, room := range userRooms {
		wg.Add(1)
		go func(roomId string) {
			defer wg.Done()

			limit := uint32(10)
			if req != nil && req.MessagesPerRoom > 0 {
				limit = uint32(req.MessagesPerRoom)
			}

			singleRoomReq := &chatv1.GetMessageHistoryRequest{
				Id:    roomId,
				Limit: limit,
			}

			messages, _, err := r.getMessagesFromSingleRoom(ctx, userId, singleRoomReq)
			if err != nil {
				errChan <- fmt.Errorf("error obteniendo mensajes de la sala %s: %w", roomId, err)
				return
			}
			msgChan <- messages
		}(room.Id)
	}

	wg.Wait()
	close(msgChan)
	close(errChan)

	if len(errChan) > 0 {
		return nil, nil, <-errChan
	}

	var allMessages []*chatv1.MessageData
	for messages := range msgChan {
		allMessages = append(allMessages, messages...)
	}

	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].CreatedAt > allMessages[j].CreatedAt
	})

	if req != nil && req.Limit > 0 && len(allMessages) > int(req.Limit) {
		allMessages = allMessages[:req.Limit]
	}

	meta := &chatv1.PaginationMeta{ItemCount: uint32(len(allMessages))}
	return allMessages, meta, nil
}

func (r *ScyllaRoomRepository) scanMessagesAndCollectUserIDs(iter *gocql.Iter) ([]*chatv1.MessageData, []int, error) {
	var messages []*chatv1.MessageData
	userIDsMap := make(map[int]bool)
	var userIDs []int

	scanner := iter.Scanner()
	for scanner.Next() {
		msg := &chatv1.MessageData{}
		var msgID gocql.UUID
		var createdAt time.Time
		err := scanner.Scan(&msgID, &msg.SenderId, &msg.Content, &msg.Type, &createdAt, &msg.Edited)
		if err != nil {
			return nil, nil, err
		}
		msg.Id = msgID.String()
		msg.CreatedAt = createdAt.Format(time.RFC3339)
		messages = append(messages, msg)

		if !userIDsMap[int(msg.SenderId)] {
			userIDs = append(userIDs, int(msg.SenderId))
			userIDsMap[int(msg.SenderId)] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return messages, userIDs, nil
}

func (r *ScyllaRoomRepository) enrichMessagesWithUserDetails(ctx context.Context, messages []*chatv1.MessageData, userIDs []int) error {
	if len(userIDs) == 0 {
		return nil
	}
	users, err := r.userFetcher.GetUsersByID(ctx, userIDs)
	if err != nil {
		return err
	}
	userMap := make(map[int32]User)
	for _, u := range users {
		userMap[int32(u.ID)] = u
	}

	for _, msg := range messages {
		if user, ok := userMap[msg.SenderId]; ok {
			msg.SenderName = user.Name
			if user.Avatar != nil {
				msg.SenderAvatar = *user.Avatar
			}
		}
	}
	return nil
}

func (r *ScyllaRoomRepository) enrichMessagesWithStatus(ctx context.Context, messages []*chatv1.MessageData, userId int, roomUUID gocql.UUID) error {
	if len(messages) == 0 {
		return nil
	}
	messageIDs := make([]gocql.UUID, len(messages))
	messageMap := make(map[string]*chatv1.MessageData)
	for i, msg := range messages {
		msgUUID, _ := gocql.ParseUUID(msg.Id)
		messageIDs[i] = msgUUID
		messageMap[msg.Id] = msg
	}

	iter := r.session.Query(`SELECT message_id, status FROM message_status_by_user WHERE user_id = ? AND room_id = ? AND message_id IN ?`,
		userId, roomUUID, messageIDs).WithContext(ctx).Iter()

	var msgID gocql.UUID
	var status int
	for iter.Scan(&msgID, &status) {
		if msg, ok := messageMap[msgID.String()]; ok {
			msg.Status = chatv1.MessageStatus(status)
		}
	}
	return iter.Close()
}

func (r *ScyllaRoomRepository) MarkMessagesAsRead(ctx context.Context, userId int, roomId string, messageIds []string, since string) (int32, error) {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return 0, err
	}

	// Lógica para 'since': obtener todos los mensajes no leídos antes de una fecha
	if since != "" {
		sinceTime, err := time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return 0, fmt.Errorf("formato de fecha 'since' inválido: %w", err)
		}
		// Generar un timeuuid a partir del timestamp para la comparación
		sinceUUID := gocql.MaxTimeUUID(sinceTime)

		// 1. Obtener todos los IDs de mensajes en la sala antes de la fecha `since`
		iter := r.session.Query(`SELECT message_id FROM messages_by_room WHERE room_id = ? AND message_id < ?`, roomUUID, sinceUUID).WithContext(ctx).Iter()
		var msgID gocql.UUID
		for iter.Scan(&msgID) {
			messageIds = append(messageIds, msgID.String())
		}
		if err := iter.Close(); err != nil {
			return 0, fmt.Errorf("error al obtener mensajes por 'since': %w", err)
		}
	}

	if len(messageIds) == 0 {
		return 0, nil
	}

	// Eliminar duplicados en caso de que se hayan añadido
	uniqueIDs := make(map[string]bool)
	var finalMessageIds []string
	for _, id := range messageIds {
		if !uniqueIDs[id] {
			uniqueIDs[id] = true
			finalMessageIds = append(finalMessageIds, id)
		}
	}

	if len(finalMessageIds) > 0 {
		batch := r.session.Batch(gocql.LoggedBatch)
		now := time.Now()
		for _, msgIdStr := range finalMessageIds {
			msgUUID, err := gocql.ParseUUID(msgIdStr)
			if err != nil {
				continue
			}
			batch.Query(`INSERT INTO read_receipts_by_message (message_id, user_id, read_at) VALUES (?, ?, ?)`, msgUUID, userId, now)
			// Actualizar el estado general a LEÍDO
			batch.Query(`INSERT INTO message_status_by_user (user_id, room_id, message_id, status) VALUES (?, ?, ?, ?)`, userId, roomUUID, msgUUID, chatv1.MessageStatus_MESSAGE_STATUS_READ)
		}
		if err := r.session.ExecuteBatch(batch); err != nil {
			return 0, fmt.Errorf("error al marcar mensajes como leídos: %w", err)
		}
	}

	err = r.session.Query(`UPDATE room_counters_by_user SET unread_count = 0 WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Exec()
	if err != nil {
		return 0, fmt.Errorf("error al resetear contador de no leídos: %w", err)
	}

	DeleteRoomCacheByRoomID(ctx, roomId)
	return int32(len(finalMessageIds)), nil
}

func (r *ScyllaRoomRepository) ReactToMessage(ctx context.Context, userId int, messageId string, reaction string) error {
	messageUUID, err := gocql.ParseUUID(messageId)
	if err != nil {
		return err
	}
	now := time.Now()

	if reaction == "" {
		err = r.session.Query(`DELETE FROM reactions_by_message WHERE message_id = ? AND user_id = ?`, messageUUID, userId).WithContext(ctx).Exec()
	} else {
		err = r.session.Query(`INSERT INTO reactions_by_message (message_id, user_id, reaction, created_at) VALUES (?, ?, ?, ?)`, messageUUID, userId, reaction, now).WithContext(ctx).Exec()
	}

	return err
}

func (r *ScyllaRoomRepository) GetMessage(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error) {
	messageUUID, err := gocql.ParseUUID(messageId)
	if err != nil {
		return nil, err
	}

	var roomUUID gocql.UUID
	err = r.session.Query(`SELECT room_id FROM room_by_message WHERE message_id = ?`, messageUUID).WithContext(ctx).Scan(&roomUUID)
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	msg := &chatv1.MessageData{}
	var createdAt time.Time
	err = r.session.Query(`SELECT sender_id, content, type, created_at, edited FROM messages_by_room WHERE room_id = ? AND message_id = ?`, roomUUID, messageUUID).
		WithContext(ctx).Scan(&msg.SenderId, &msg.Content, &msg.Type, &createdAt, &msg.Edited)
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	msg.Id = messageId
	msg.RoomId = roomUUID.String()
	msg.CreatedAt = createdAt.Format(time.RFC3339)

	user, err := r.userFetcher.GetUserByID(ctx, int(msg.SenderId))
	if err != nil {
		return nil, err
	}
	msg.SenderName = user.Name
	if user.Avatar != nil {
		msg.SenderAvatar = *user.Avatar
	}

	// Obtener el estado para el usuario actual
	var status int
	err = r.session.Query(`SELECT status FROM message_status_by_user WHERE user_id = ? AND room_id = ? AND message_id = ?`, userId, roomUUID, messageUUID).WithContext(ctx).Scan(&status)
	if err != nil && err != gocql.ErrNotFound {
		return nil, err
	}
	msg.Status = chatv1.MessageStatus(status)

	return msg, nil
}

func (r *ScyllaRoomRepository) UpdateMessage(ctx context.Context, userId int, messageId string, content string) error {
	messageUUID, err := gocql.ParseUUID(messageId)
	if err != nil {
		return err
	}

	var roomUUID gocql.UUID
	err = r.session.Query(`SELECT room_id FROM room_by_message WHERE message_id = ?`, messageUUID).WithContext(ctx).Scan(&roomUUID)
	if err != nil {
		return err
	}

	return r.session.Query(`UPDATE messages_by_room SET content = ?, edited = true WHERE room_id = ? AND message_id = ?`, content, roomUUID, messageUUID).WithContext(ctx).Exec()
}

func (r *ScyllaRoomRepository) DeleteMessage(ctx context.Context, userId int, messageIds []string) error {
	batch := r.session.Batch(gocql.LoggedBatch)
	for _, msgIdStr := range messageIds {
		messageUUID, err := gocql.ParseUUID(msgIdStr)
		if err != nil {
			continue
		}

		var roomUUID gocql.UUID
		err = r.session.Query(`SELECT room_id FROM room_by_message WHERE message_id = ?`, messageUUID).WithContext(ctx).Scan(&roomUUID)
		if err != nil {
			continue
		}

		batch.Query(`UPDATE messages_by_room SET is_deleted = true WHERE room_id = ? AND message_id = ?`, roomUUID, messageUUID)
	}

	return r.session.ExecuteBatch(batch)
}

func (r *ScyllaRoomRepository) GetMessageRead(ctx context.Context, req *chatv1.GetMessageReadRequest) ([]*chatv1.MessageUserRead, *chatv1.PaginationMeta, error) {
	messageUUID, err := gocql.ParseUUID(req.Id)
	if err != nil {
		return nil, nil, err
	}

	baseQuery := `SELECT user_id, read_at FROM read_receipts_by_message WHERE message_id = ?`
	args := []any{messageUUID}

	if req != nil && req.Limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, int(req.Limit))
	}

	iter := r.session.Query(baseQuery, args...).WithContext(ctx).Iter()
	defer iter.Close()

	var userReads []*chatv1.MessageUserRead
	var userIDs []int
	for {
		var userID int
		var readAt time.Time
		if !iter.Scan(&userID, &readAt) {
			break
		}
		userIDs = append(userIDs, userID)
		userReads = append(userReads, &chatv1.MessageUserRead{UserId: int32(userID), ReadAt: readAt.Format(time.RFC3339)})
	}
	if err := iter.Close(); err != nil {
		return nil, nil, err
	}

	users, err := r.userFetcher.GetUsersByID(ctx, userIDs)
	if err != nil {
		return nil, nil, err
	}
	userMap := make(map[int32]User)
	for _, u := range users {
		userMap[int32(u.ID)] = u
	}

	for _, ur := range userReads {
		if user, ok := userMap[ur.UserId]; ok {
			ur.UserName = user.Name
			if user.Avatar != nil {
				ur.UserAvatar = *user.Avatar
			}
		}
	}

	meta := &chatv1.PaginationMeta{ItemCount: uint32(len(userReads))}
	return userReads, meta, nil
}

func (r *ScyllaRoomRepository) GetMessageReactions(ctx context.Context, req *chatv1.GetMessageReactionsRequest) ([]*chatv1.Reaction, *chatv1.PaginationMeta, error) {
	messageUUID, err := gocql.ParseUUID(req.Id)
	if err != nil {
		return nil, nil, err
	}

	baseQuery := `SELECT user_id, reaction FROM reactions_by_message WHERE message_id = ?`
	args := []any{messageUUID}

	if req != nil && req.Limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, int(req.Limit))
	}

	iter := r.session.Query(baseQuery, args...).WithContext(ctx).Iter()
	defer iter.Close()

	var reactions []*chatv1.Reaction
	var userIDs []int
	for {
		var userID int
		var reactionStr string
		if !iter.Scan(&userID, &reactionStr) {
			break
		}
		userIDs = append(userIDs, userID)
		reactions = append(reactions, &chatv1.Reaction{MessageId: req.Id, ReactedById: fmt.Sprint(userID), Reaction: reactionStr})
	}
	if err := iter.Close(); err != nil {
		return nil, nil, err
	}

	users, err := r.userFetcher.GetUsersByID(ctx, userIDs)
	if err != nil {
		return nil, nil, err
	}
	userMap := make(map[string]User)
	for _, u := range users {
		userMap[fmt.Sprint(u.ID)] = u
	}

	for _, reaction := range reactions {
		if user, ok := userMap[reaction.ReactedById]; ok {
			reaction.ReactedByName = user.Name
			if user.Avatar != nil {
				reaction.ReactedByAvatar = *user.Avatar
			}
		}
	}

	meta := &chatv1.PaginationMeta{ItemCount: uint32(len(reactions))}
	return reactions, meta, nil
}

func (r *ScyllaRoomRepository) GetMessageSender(ctx context.Context, userId int, senderMessageId string) (*chatv1.MessageData, error) {
	var roomUUID, messageUUID gocql.UUID
	err := r.session.Query(`SELECT room_id, message_id FROM message_by_sender_message_id WHERE sender_message_id = ?`, senderMessageId).WithContext(ctx).Scan(&roomUUID, &messageUUID)
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.GetMessage(ctx, userId, messageUUID.String())
}

func (r *ScyllaRoomRepository) CreateMessageMetaForParticipants(ctx context.Context, roomID string, messageID string, senderID int) error {
	return nil
}

func (r *ScyllaRoomRepository) GetUserByID(ctx context.Context, id int) (*User, error) {
	return r.userFetcher.GetUserByID(ctx, id)
}

func (r *ScyllaRoomRepository) GetUsersByID(ctx context.Context, ids []int) ([]User, error) {
	return r.userFetcher.GetUsersByID(ctx, ids)
}

func (r *ScyllaRoomRepository) GetAllUserIDs(ctx context.Context) ([]int, error) {
	return r.userFetcher.GetAllUserIDs(ctx)
}

func (r *ScyllaRoomRepository) GetMessageSimple(ctx context.Context, userId int, messageId string) (*chatv1.MessageData, error) {
	return r.GetMessage(ctx, userId, messageId)
}

func (r *ScyllaRoomRepository) DeleteRoom(ctx context.Context, userId int, roomId string, partner *int) error {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return err
	}

	participants, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: roomId})
	if err != nil {
		return fmt.Errorf("no se pudieron obtener los participantes para borrar la sala: %w", err)
	}

	now := time.Now()
	// Se ejecuta secuencialmente para cada usuario para evitar batches multi-partición.
	for _, p := range participants {
		var isPinned bool
		var lastMessageAt time.Time
		err := r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)

		userBatch := r.session.Batch(gocql.LoggedBatch)
		if err == nil {
			userBatch.Query(`DELETE FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`, p.Id, isPinned, lastMessageAt, roomUUID)
		}
		userBatch.Query(`DELETE FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID)
		userBatch.Query(`INSERT INTO deleted_rooms_by_user (user_id, deleted_at, room_id, reason) VALUES (?, ?, ?, ?)`, p.Id, now, roomUUID, "deleted")
		if err := r.session.ExecuteBatch(userBatch); err != nil {
			fmt.Printf("Error al procesar la eliminación para el usuario %d: %v\n", p.Id, err)
		}
		r.session.Query(`DELETE FROM room_counters_by_user WHERE user_id = ? AND room_id = ?`, p.Id, roomUUID).WithContext(ctx).Exec()
	}

	// Eliminar datos de la sala (partición única)
	batch := r.session.Batch(gocql.LoggedBatch)
	batch.Query(`DELETE FROM participants_by_room WHERE room_id = ?`, roomUUID)
	batch.Query(`DELETE FROM room_details WHERE room_id = ?`, roomUUID)
	batch.Query(`DELETE FROM messages_by_room WHERE room_id = ?`, roomUUID)

	if err := r.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("error en el batch de eliminación de datos de la sala: %w", err)
	}

	DeleteRoomCacheByRoomID(ctx, roomId)
	return nil
}

func (r *ScyllaRoomRepository) PinRoom(ctx context.Context, userId int, roomId string, pin bool) error {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return err
	}

	var roomName, roomImage, roomType, role string
	var lastMessageAt, lastMessageUpdatedAt time.Time
	var isMuted, isPinnedOld bool
	var lastMessage chatv1.MessageData
	var lastMessageID gocql.UUID

	// 1. Leer la clave de clúster actual desde la tabla de búsqueda
	err = r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&isPinnedOld, &lastMessageAt)
	if err != nil {
		return fmt.Errorf("no se encontró la membresía para el usuario %d en la sala %s: %w", userId, roomId, err)
	}

	// 2. Leer el resto de los datos de la fila que se va a modificar
	err = r.session.Query(`SELECT room_name, room_image, room_type, is_muted, role, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
		userId, isPinnedOld, lastMessageAt, roomUUID).
		WithContext(ctx).Scan(&roomName, &roomImage, &roomType, &isMuted, &role, &lastMessageID, &lastMessage.Content, &lastMessage.Type, &lastMessage.SenderId, &lastMessage.SenderName, &lastMessage.SenderPhone, &lastMessage.Status, &lastMessageUpdatedAt)
	if err != nil {
		return fmt.Errorf("no se encontró la sala para el usuario %d: %w", userId, err)
	}

	// 3. Ejecutar la eliminación y la inserción en un batch para atomicidad dentro de la misma partición
	batch := r.session.Batch(gocql.LoggedBatch)
	batch.Query(`DELETE FROM rooms_by_user WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`, userId, isPinnedOld, lastMessageAt, roomUUID)
	batch.Query(`INSERT INTO rooms_by_user (user_id, is_pinned, last_message_at, room_id, room_name, room_image, room_type, last_message_id, last_message_preview, last_message_type, last_message_sender_id, last_message_sender_name, last_message_sender_phone, last_message_status, last_message_updated_at, is_muted, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userId, pin, lastMessageAt, roomUUID, roomName, roomImage, roomType, lastMessageID, lastMessage.Content, lastMessage.Type, lastMessage.SenderId, lastMessage.SenderName, lastMessage.SenderPhone, lastMessage.Status, lastMessageUpdatedAt, isMuted, role)
	batch.Query(`UPDATE room_membership_lookup SET is_pinned = ? WHERE user_id = ? AND room_id = ?`, pin, userId, roomUUID)

	if err := r.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("error en el batch de pin/unpin: %w", err)
	}

	DeleteRoomCacheByRoomID(ctx, roomId)
	return nil
}

func (r *ScyllaRoomRepository) MuteRoom(ctx context.Context, userId int, roomId string, mute bool) error {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return err
	}

	var isPinned bool
	var lastMessageAt time.Time
	err = r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, userId, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)
	if err != nil {
		return fmt.Errorf("no se encontró la sala para mutear para el usuario %d: %w", userId, err)
	}

	// Necesitamos también actualizar `participants_by_room`
	batch := r.session.Batch(gocql.LoggedBatch)
	batch.Query(`UPDATE rooms_by_user SET is_muted = ? WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
		mute, userId, isPinned, lastMessageAt, roomUUID)
	batch.Query(`UPDATE participants_by_room SET is_muted = ? WHERE room_id = ? AND user_id = ?`, mute, roomUUID, userId)

	if err := r.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("error en batch de mute/unmute: %w", err)
	}

	DeleteRoomCacheByRoomID(ctx, roomId)
	return nil
}

func (r *ScyllaRoomRepository) BlockUser(ctx context.Context, userId int, roomId string, block bool, partner *int) error {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return err
	}

	err = r.session.Query(`UPDATE participants_by_room SET is_partner_blocked = ? WHERE room_id = ? AND user_id = ?`,
		block, roomUUID, userId).WithContext(ctx).Exec()
	if err != nil {
		return err
	}

	DeleteRoomCacheByRoomID(ctx, roomId)
	return nil
}

func (r *ScyllaRoomRepository) UpdateParticipantRoom(ctx context.Context, userId int, req *chatv1.UpdateParticipantRoomRequest) error {
	roomUUID, err := gocql.ParseUUID(req.Id)
	if err != nil {
		return err
	}

	participantID := int(req.Participant)

	// 1. Leer la clave de clúster actual desde la tabla de búsqueda
	var isPinned bool
	var lastMessageAt time.Time
	err = r.session.Query(`SELECT is_pinned, last_message_at FROM room_membership_lookup WHERE user_id = ? AND room_id = ?`, participantID, roomUUID).WithContext(ctx).Scan(&isPinned, &lastMessageAt)
	if err != nil {
		return fmt.Errorf("no se encontró la membresía para el participante %d: %w", participantID, err)
	}

	// 2. Ejecutar las actualizaciones en un batch para la misma partición
	batch := r.session.Batch(gocql.LoggedBatch)
	batch.Query(`UPDATE participants_by_room SET role = ? WHERE room_id = ? AND user_id = ?`, req.Role, roomUUID, participantID)
	batch.Query(`UPDATE rooms_by_user SET role = ? WHERE user_id = ? AND is_pinned = ? AND last_message_at = ? AND room_id = ?`,
		req.Role, participantID, isPinned, lastMessageAt, roomUUID)

	if err := r.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("error en el batch de actualización de participante: %w", err)
	}

	DeleteRoomCacheByRoomID(ctx, req.Id)
	return nil
}

func (r *ScyllaRoomRepository) IsPartnerMuted(ctx context.Context, userId int, roomId string) (bool, error) {
	roomUUID, err := gocql.ParseUUID(roomId)
	if err != nil {
		return false, err
	}

	// 1. Obtener todos los participantes de la sala
	participants, _, err := r.GetRoomParticipants(ctx, &chatv1.GetRoomParticipantsRequest{Id: roomId})
	if err != nil {
		return false, fmt.Errorf("error al obtener participantes para IsPartnerMuted: %w", err)
	}

	// 2. Encontrar el ID del compañero (el que no es el `userId` actual)
	var partnerID int32
	for _, p := range participants {
		if p.Id != int32(userId) {
			partnerID = p.Id
			break
		}
	}

	if partnerID == 0 {
		return false, errors.New("no se encontró un compañero en esta sala P2P")
	}

	// 3. Consultar directamente el estado `is_muted` del compañero
	var isMuted bool
	err = r.session.Query(`SELECT is_muted FROM participants_by_room WHERE room_id = ? AND user_id = ?`, roomUUID, partnerID).WithContext(ctx).Scan(&isMuted)
	if err != nil {
		if err == gocql.ErrNotFound {
			return false, nil // El participante no tiene registro, no está muteado
		}
		return false, fmt.Errorf("error al consultar el estado de mute del compañero: %w", err)
	}

	return isMuted, nil
}
