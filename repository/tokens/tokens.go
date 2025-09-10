package tokensrepository

import (
	"context"

	tokensv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1"
)

type TokensRepository interface {
	SaveToken(ctx context.Context, userId int, room *tokensv1.SaveTokenRequest) error
}
