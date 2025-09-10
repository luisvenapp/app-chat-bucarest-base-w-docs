package tokensrepository

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	tokensv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/tokens/v1"
	dbpq "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/postgres"
)

type SQLTokensRepository struct {
	db *sql.DB
}

func NewSQLTokensRepository(db *sql.DB) TokensRepository {
	return &SQLTokensRepository{
		db: db,
	}
}

func (r *SQLTokensRepository) SaveToken(ctx context.Context, userId int, room *tokensv1.SaveTokenRequest) error {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query := dbpq.QueryBuilder().
		Insert("public.messaging_token").
		SetMap(sq.Eq{
			"token":            room.Token,
			"platform":         room.Platform,
			"platform_version": room.PlatformVersion,
			"device":           room.Device,
			"lang":             room.Lang,
			"is_voip":          room.IsVoip,
			"debug":            room.Debug,
			"user_id":          userId,
			"created_at":       sq.Expr("NOW()"),
		}).
		Suffix("RETURNING id")

	queryString, args, err := query.ToSql()
	if err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx, queryString, args...)
	if err != nil {
		return err
	}
	rows.Close()

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
