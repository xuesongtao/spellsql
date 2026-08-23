package sqldb

import (
	"context"
	"database/sql"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/dialect"
)

var _ DBer = (*DB)(nil)

type DB struct {
	*sql.DB
	DbType dialect.DbType
}

func Open(dbType dialect.DbType, dsn string) (*DB, error) {
	db, err := sql.Open(dbType.String(), dsn)
	if err != nil {
		return nil, err
	}
	return &DB{DB: db, DbType: dbType}, nil
}

// ExecContext implements [DBer].
func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.DB.ExecContext(ctx, query, args...)
}

// QueryContext implements [DBer].
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.DB.QueryContext(ctx, query, args...)
}

// QueryRowContext implements [DBer].
func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.DB.QueryRowContext(ctx, query, args...)
}

func (d *DB) SelectBuilder() *builder.Select {
	return builder.NewSelect(d.DbType)
}

func (d *DB) InsertBuilder() *builder.Insert {
	return builder.NewInsert(d.DbType)
}

func (d *DB) UpdateBuilder() *builder.Update {
	return builder.NewUpdate(d.DbType)
}

func (d *DB) DeleteBuilder() *builder.Delete {
	return builder.NewDelete(d.DbType)
}
