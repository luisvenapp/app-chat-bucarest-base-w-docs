package utils

import (
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api/auth"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/config"
)

var publictoken = config.GetString("publictoken")

func ValidatePublicToken(header http.Header) (bool, error) {
	token, err := auth.GetTokenFromHeader(header)
	if err != nil {
		return false, err
	}

	if token != publictoken {
		return false, errors.New("invalid token")
	}

	return true, nil
}

func ValidateAuthToken[T any](req *connect.Request[T]) (int, error) {
	session, err := api.CheckSessionFromConnectRequest(req)
	if err != nil {
		return 0, connect.NewError(connect.CodeUnauthenticated, errors.New(ERRORS.INVALID_TOKEN))

	}
	return session.UserID, nil
}
