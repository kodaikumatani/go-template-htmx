// Package store implements persistence on top of SQLite.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	// modernc.org/sqlite は cgo なしの Pure Go 実装。
	// ドライバ名は "sqlite"（mattn/go-sqlite3 の "sqlite3" ではない）。
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// DefaultDSN is the connection string used when DB_DSN is not set.
//
//	journal_mode(WAL)  ... 読み取りが書き込みでブロックされない
//	busy_timeout(5000) ... ロック待ちで即エラーにせず 5 秒待つ
//	foreign_keys(on)   ... SQLite は既定で外部キーを無視するので明示的に有効化
const DefaultDSN = "file:issues.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"

// Open connects to SQLite and applies the schema.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite の書き込みは 1 本しか通らない。接続数を絞って
	// "database is locked" を database/sql 側のキューに変換しておく。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return db, nil
}

// 時刻は RFC3339 の UTC 文字列で保存する。
// ドライバや SQLite の日付関数の挙動に依存させたくないので、
// 変換は必ずこの 2 つの関数を通す。
const timeLayout = time.RFC3339

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return t, nil
}
