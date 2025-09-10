package chatv1handler

import (
	"connectrpc.com/vanguard"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1/chatv1connect"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/server"
)

var options = server.ServiceHandlerOptions()

func RegisterServiceHandler() *vanguard.Service {
	return vanguard.NewService(chatv1connect.NewChatServiceHandler(NewHandler(), options...))
}
