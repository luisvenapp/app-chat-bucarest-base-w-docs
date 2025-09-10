package roomsrepository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/cache"
)

type CachedRoomResponse struct {
	Data *chatv1.Room `json:"data"`
}

type CachedMessageResponse struct {
	Data *chatv1.MessageData `json:"data"`
}

var (
	roomCacheLocks   = make(map[string]*sync.Mutex)
	roomCacheLocksMu sync.Mutex
)

func getRoomLock(roomID string) *sync.Mutex {
	roomCacheLocksMu.Lock()
	defer roomCacheLocksMu.Unlock()

	lock, exists := roomCacheLocks[roomID]
	if !exists {
		lock = &sync.Mutex{}
		roomCacheLocks[roomID] = lock
	}
	return lock
}

func GetCachedRoom(ctx context.Context, cacheKey string) (*chatv1.Room, bool) {
	cacheValue, err := cache.Get(ctx, cacheKey)
	if err != nil || cacheValue == "" {
		return nil, false
	}

	var cachedResponse CachedRoomResponse
	if err := json.Unmarshal([]byte(cacheValue), &cachedResponse); err != nil {
		return nil, false
	}

	return cachedResponse.Data, true
}

func SetCachedRoom(ctx context.Context, roomId string, cacheKey string, data *chatv1.Room) {
	cachedResponse := CachedRoomResponse{
		Data: data,
	}

	cacheData, err := json.Marshal(cachedResponse)
	if err == nil {
		cache.Set(ctx, cacheKey, string(cacheData), 1*time.Hour)
		setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", roomId)
		cache.SAdd(ctx, setKey, cacheKey)
	}
}

// UpdateRoomCacheWithNewMessage atomically updates the LastMessage for all cached versions of a room.
func UpdateRoomCacheWithNewMessage(ctx context.Context, message *chatv1.MessageData) {
	if message == nil || message.RoomId == "" {
		return
	}

	roomLock := getRoomLock(message.RoomId)
	roomLock.Lock()
	defer roomLock.Unlock()

	setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", message.RoomId)
	cacheKeys, err := cache.SMembers(ctx, setKey)
	if err != nil {
		fmt.Println("error getting cache members for update", err)
		return
	}

	for _, key := range cacheKeys {
		cachedRoom, exists := GetCachedRoom(ctx, key)
		if exists {
			cachedRoom.LastMessage = message
			cachedRoom.LastMessageAt = message.CreatedAt
			// SetCachedRoom will re-add the key to the set, which is fine.
			SetCachedRoom(ctx, message.RoomId, key, cachedRoom)
		}
	}
}

func GetCachedMessageSimple(ctx context.Context, cacheKey string) (*chatv1.MessageData, bool) {
	cacheValue, err := cache.Get(ctx, cacheKey)
	if err != nil || cacheValue == "" {
		return nil, false
	}

	var cachedResponse CachedMessageResponse
	if err := json.Unmarshal([]byte(cacheValue), &cachedResponse); err != nil {
		return nil, false
	}

	return cachedResponse.Data, true
}

func SetCachedMessageSimple(ctx context.Context, cacheKey string, data *chatv1.MessageData) {
	cachedResponse := CachedMessageResponse{
		Data: data,
	}

	cacheData, err := json.Marshal(cachedResponse)
	if err == nil {
		cache.Set(ctx, cacheKey, string(cacheData), 1*time.Hour)
	}
}

func DeleteRoomCacheByRoomID(ctx context.Context, roomId string) {
	setKey := fmt.Sprintf("endpoint:chat:room:{%s}:members", roomId)
	keys, err := cache.SMembers(ctx, setKey)
	if err != nil {
		fmt.Println("error getting cache members", err)
		return
	}

	if len(keys) > 0 {
		err := cache.Del(ctx, keys...)
		if err != nil {
			fmt.Println("error deleting cache", err)
		}
	}

	err = cache.Del(ctx, setKey)
	if err != nil {
		fmt.Println("error deleting cache set", err)
	}
}

func DeleteCache(ctx context.Context, key string) {
	err := cache.Del(ctx, key)
	if err != nil {
		fmt.Println("error deleting cache", err)
	}
}
