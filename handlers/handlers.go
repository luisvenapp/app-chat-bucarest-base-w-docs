package handlers

import (
	chatv1handler "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/handlers/chat/v1"
	tokensv1handler "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/handlers/tokens/v1"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/server"
)

var RegisterServicesFns = []server.RegisterServiceFn{
	chatv1handler.RegisterServiceHandler,
	tokensv1handler.RegisterServiceHandler,
}
