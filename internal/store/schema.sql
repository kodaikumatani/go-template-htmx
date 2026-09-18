-- 起動時に毎回実行する。IF NOT EXISTS なので冪等。
-- 本格的なマイグレーションツールはこの規模では要らない。

CREATE TABLE IF NOT EXISTS issues (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT    NOT NULL,
    body       TEXT    NOT NULL DEFAULT '',
    -- status は CHECK 制約で 2 値に縛る。domain.Status と対応。
    status     TEXT    NOT NULL CHECK (status IN ('open', 'closed')),
    -- 時刻は RFC3339 の UTC 文字列。TEXT なので sqlite3 CLI でそのまま読めて、
    -- UTC 固定なので文字列のまま時系列ソートできる。
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL
);

-- 一覧のデフォルト並び順（新しい順）と、ステータス絞り込みのための索引。
CREATE INDEX IF NOT EXISTS idx_issues_created_at ON issues (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues (status);
