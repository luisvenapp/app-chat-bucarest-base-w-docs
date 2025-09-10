package chatv1handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamChatEventsName                = "CHAT_EVENTS"
	StreamChatDirectEventsSubjectPrefix = "CHAT_DIRECT_EVENTS"
)

var requiredStreams = []jetstream.StreamConfig{
	{
		Name: StreamChatEventsName,
		Subjects: []string{
			chatRoomEventSubject("*"),
			StreamChatDirectEventsSubjectPrefix + ".*",
		},
		Storage:           jetstream.FileStorage,
		Retention:         jetstream.LimitsPolicy,
		MaxMsgsPerSubject: 1000,
		Compression:       jetstream.S2Compression,
		MaxAge:            7 * 24 * time.Hour, // 7 días
	},
}

func chatRoomEventSubject(roomId string) string {
	return strings.Join([]string{StreamChatEventsName, roomId}, ".")
}

func chatDirectEventSubject(userId int) string {
	return strings.Join([]string{StreamChatDirectEventsSubjectPrefix, strconv.Itoa(userId)}, ".")
}
