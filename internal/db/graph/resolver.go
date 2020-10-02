package graph

import (
	"context"
	"database/sql"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver resolves GraphQL queries. It requires TransactionExtension to be added to the server.
type Resolver struct{}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
	graphql.ResponseInterceptor
} = (*TransactionExtension)(nil)

// TransactionExtension manages SQL transactions.
type TransactionExtension struct {
	db *sql.DB
}

// NewTransacter creates an extension that will provide SQL transactions to the server.
func NewTransacter(db *sql.DB) *TransactionExtension {
	return &TransactionExtension{
		db: db,
	}
}

func (*TransactionExtension) ExtensionName() string {
	return "TransactionExtension"
}

func (*TransactionExtension) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (t *TransactionExtension) MutateOperationContext(ctx context.Context, oc *graphql.OperationContext) *gqlerror.Error {
	previous := oc.ResolverMiddleware
	var mu sync.Mutex
	oc.ResolverMiddleware = func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
		mu.Lock()
		defer mu.Unlock()
		return previous(ctx, next)
	}
	return nil
}

func (t *TransactionExtension) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) (resp *graphql.Response) {
	err := dbx.Transact(ctx, t.db, func(ctx context.Context, tx *sql.Tx) error {
		ctx = withTx(ctx, tx)
		resp = next(ctx)

		if len(resp.Errors) > 0 {
			// Return a non-nil error to trigger a rollback.
			return resp.Errors
		}

		return nil
	})

	// "Normal" error during GraphQL query.
	if resp != nil && len(resp.Errors) > 0 {
		return &graphql.Response{
			Errors: resp.Errors,
		}
	}

	// Internal error.
	if err != nil {
		ctxlog.Error(ctx, "error during graphql transaction", zap.Error(err))
		return graphql.ErrorResponse(ctx, "transaction error") // TODO: Don't show internal details.
	}

	return resp
}

type contextKey int

const (
	transactionKey contextKey = iota
)

func withTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, transactionKey, tx)
}

func txFromContext(ctx context.Context) *sql.Tx {
	tx, ok := ctx.Value(transactionKey).(*sql.Tx)
	if !ok {
		panic("no transaction")
	}
	return tx
}
