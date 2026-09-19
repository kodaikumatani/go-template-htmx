package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/kodaikumatani/go-template-htmx/internal/domain"
)

// IssueStore reads and writes issues.
type IssueStore struct {
	db *sql.DB
}

func NewIssueStore(db *sql.DB) *IssueStore {
	return &IssueStore{db: db}
}

const issueColumns = "id, title, body, status, created_at, updated_at"

// List returns one page of issues and the total number of matches.
// 2 つ目の戻り値（総数）はページングの表示に必要なので、LIMIT を無視して数える。
func (s *IssueStore) List(ctx context.Context, q domain.IssueQuery) ([]domain.Issue, int, error) {
	where, args := issueWhere(q)

	var total int
	countSQL := "SELECT COUNT(*) FROM issues WHERE " + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count issues: %w", err)
	}

	listSQL := "SELECT " + issueColumns + " FROM issues WHERE " + where +
		" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	rows, err := s.db.QueryContext(ctx, listSQL, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("select issues: %w", err)
	}
	defer rows.Close()

	var issues []domain.Issue
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, 0, err
		}
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate issues: %w", err)
	}
	return issues, total, nil
}

// Count returns the number of issues matching the query.
func (s *IssueStore) Count(ctx context.Context, q domain.IssueQuery) (int, error) {
	where, args := issueWhere(q)
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM issues WHERE "+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count issues: %w", err)
	}
	return total, nil
}

// Create inserts an issue and returns it with the generated id.
func (s *IssueStore) Create(ctx context.Context, issue domain.Issue) (domain.Issue, error) {
	at := formatTime(issue.CreatedAt)
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO issues (title, body, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		issue.Title, issue.Body, string(issue.Status), at, at)
	if err != nil {
		return domain.Issue{}, fmt.Errorf("insert issue: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Issue{}, fmt.Errorf("last insert id: %w", err)
	}
	issue.ID = id
	return issue, nil
}

// issueWhere builds the shared WHERE clause. 条件が増えても
// COUNT 側と SELECT 側でズレないよう、1 箇所で組み立てる。
func issueWhere(q domain.IssueQuery) (string, []any) {
	conds := []string{"1 = 1"}
	var args []any

	if q.Q != "" {
		// LIKE のメタ文字を打ち消すため ESCAPE を明示する。
		// これを忘れると "100%" の検索で全件が返る。
		conds = append(conds, `(title LIKE ? ESCAPE '\' OR body LIKE ? ESCAPE '\')`)
		pattern := "%" + escapeLike(q.Q) + "%"
		args = append(args, pattern, pattern)
	}
	if q.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, string(q.Status))
	}
	return strings.Join(conds, " AND "), args
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// rowScanner covers both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanIssue(sc rowScanner) (domain.Issue, error) {
	var (
		issue                domain.Issue
		status               string
		createdAt, updatedAt string
	)
	if err := sc.Scan(&issue.ID, &issue.Title, &issue.Body, &status, &createdAt, &updatedAt); err != nil {
		return domain.Issue{}, fmt.Errorf("scan issue: %w", err)
	}
	issue.Status = domain.Status(status)

	var err error
	if issue.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Issue{}, err
	}
	if issue.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Issue{}, err
	}
	return issue, nil
}

// SeedIfEmpty inserts sample rows when the table is empty.
// 開発用。Step 3 で作成フォームができたら消してよい。
func (s *IssueStore) SeedIfEmpty(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM issues").Scan(&count); err != nil {
		return 0, fmt.Errorf("count issues: %w", err)
	}
	if count > 0 {
		return 0, nil
	}

	samples := []struct {
		title  string
		body   string
		status domain.Status
	}{
		{"検索が遅い", "1000 件を超えると一覧の表示に 3 秒かかる。", domain.StatusOpen},
		{"ログイン後にリダイレクトされない", "遷移前の URL を保持していない。", domain.StatusOpen},
		{"READMEのセットアップ手順を更新", "make run の記述が古い。", domain.StatusClosed},
		{"タイトルが長いと一覧が崩れる", "省略表示にしたい。", domain.StatusOpen},
		{"削除に確認ダイアログを出す", "誤操作で消える。", domain.StatusClosed},
		{"ページングを追加する", "1 ページ 20 件にする。", domain.StatusOpen},
		{"コメント通知を送る", "担当者にメールしたい。", domain.StatusOpen},
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin seed tx: %w", err)
	}
	defer tx.Rollback() // Commit 済みなら no-op

	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO issues (title, body, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return 0, fmt.Errorf("prepare seed insert: %w", err)
	}
	defer stmt.Close()

	// created_at をずらして入れておく。降順ソートが効いていることが分かるように。
	now := time.Now()
	for i, sample := range samples {
		at := formatTime(now.Add(-time.Duration(len(samples)-i) * time.Hour))
		if _, err := stmt.ExecContext(ctx, sample.title, sample.body, string(sample.status), at, at); err != nil {
			return 0, fmt.Errorf("insert seed %q: %w", sample.title, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit seed: %w", err)
	}
	return len(samples), nil
}
