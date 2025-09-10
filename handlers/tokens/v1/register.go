package tokensv1handler

import (
	"connectrpc.com/vanguard"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1/tokensv1connect"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/server"
)

var options = server.ServiceHandlerOptions()

func RegisterServiceHandler() *vanguard.Service {
	return vanguard.NewService(tokensv1connect.NewTokensServiceHandler(handler, options...))
}
