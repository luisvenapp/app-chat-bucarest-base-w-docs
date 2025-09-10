package tokensv1handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/database"
	tokensv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1/tokensv1connect"
	tokensrepository "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/repository/tokens"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
)

var handler tokensv1connect.TokensServiceHandler = &handlerImpl{}

type handlerImpl struct{}

func (h *handlerImpl) SaveToken(ctx context.Context, req *connect.Request[tokensv1.SaveTokenRequest]) (*connect.Response[tokensv1.SaveTokenResponse], error) {
	//validate auth token
	userID, err := utils.ValidateAuthToken(req)
	if err != nil {
		return nil, err
	}

	roomRepository := tokensrepository.NewSQLTokensRepository(database.DB())

	err = roomRepository.SaveToken(ctx, userID, req.Msg)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tokensv1.SaveTokenResponse{Success: true}), nil
}
