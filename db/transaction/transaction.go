package transaction

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type txContextKey struct{}

func WithTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func FromContext(ctx context.Context) (*sqlx.Tx, bool) {
	value, ok := ctx.Value(txContextKey{}).(*sqlx.Tx)
	return value, ok
}
