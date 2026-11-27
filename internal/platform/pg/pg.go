// Package pg connects to PostgreSQL: the connection pool, versioned
// migrations applied on start (NFR-MNT-02) and the transaction manager the
// repositories share.
package pg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/voloshmiak/tamagochi/internal/platform/pg/migrations"
	"github.com/voloshmiak/tamagochi/internal/platform/pg/pgdb"
)

// Open connects to the database and checks that it answers.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return pool, nil
}

// Migrate brings the schema up to the latest migration. An advisory lock
// keeps several API instances that start at once from migrating twice.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	results, err := p.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	for _, r := range results {
		slog.InfoContext(ctx, "migration applied", "version", r.Source.Version, "file", r.Source.Path, "took", r.Duration)
	}
	version, err := p.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "database schema is up to date", "version", version)
	return nil
}

type txKey struct{}

// TxManager implements tx.Manager on top of a pgx pool. The transaction is
// carried in the context, so repositories called inside WithinTx use it
// without knowing about it.
type TxManager struct {
	Pool *pgxpool.Pool
}

func (m TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx) // already inside a transaction: join it
	}
	tx, err := m.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed { // fn failed or panicked
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	committed = true
	return nil
}

// Queries returns the sqlc queries bound to the transaction in ctx, or to the
// pool when there is none.
func Queries(ctx context.Context, pool *pgxpool.Pool) *pgdb.Queries {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return pgdb.New(tx)
	}
	return pgdb.New(pool)
}

// IsNoRows reports whether err means that a query returned no rows.
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// UniqueViolation returns the name of the violated unique constraint or
// index, or "" if err is not a unique violation.
func UniqueViolation(err error) string {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return pgErr.ConstraintName
	}
	return ""
}
