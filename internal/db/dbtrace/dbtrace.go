package dbtrace

import (
	"context"
	"database/sql"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/volatiletech/sqlboiler/boil"
)

type d struct {
	*sql.DB
}

type ContextBeginExecutor interface {
	boil.Beginner
	boil.ContextBeginner
	boil.ContextExecutor
}

func DB(db *sql.DB) ContextBeginExecutor {
	return d{DB: db}
}

func (d d) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ExecContext")
	defer span.Finish()
	ext.DBStatement.Set(span, query)
	return d.DB.ExecContext(ctx, query, args...)
}

func (d d) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "QueryContext")
	defer span.Finish()
	ext.DBStatement.Set(span, query)
	return d.DB.QueryContext(ctx, query, args...)
}

func (d d) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	span, ctx := opentracing.StartSpanFromContext(ctx, "QueryRowContext")
	defer span.Finish()
	ext.DBStatement.Set(span, query)
	return d.DB.QueryRowContext(ctx, query, args...)
}

type t struct {
	*sql.Tx
}

func Tx(tx *sql.Tx) boil.ContextTransactor {
	return t{Tx: tx}
}

func (t t) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ExecContext")
	defer span.Finish()
	ext.DBStatement.Set(span, query)
	return t.Tx.ExecContext(ctx, query, args...)
}

func (t t) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "QueryContext")
	defer span.Finish()
	ext.DBStatement.Set(span, query)
	return t.Tx.QueryContext(ctx, query, args...)
}

func (t t) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	span, ctx := opentracing.StartSpanFromContext(ctx, "QueryRowContext")
	defer span.Finish()
	ext.DBStatement.Set(span, query)
	return t.Tx.QueryRowContext(ctx, query, args...)
}
