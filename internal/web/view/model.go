package view

// ViewModel は template に渡す専用の型。domain の型をそのまま渡さないことで、
// 「整形は Go 側、template は組み立てだけ」という境界を保つ。

// IssuesIndex is the ViewModel for GET /issues.
type IssuesIndex struct {
	Title  string
	Issues []Issue
}

// Issue is one row in the issue list.
type Issue struct {
	ID          int64
	Title       string
	StatusLabel string // "Open" / "Closed"
	IsOpen      bool
	CreatedAt   string // 表示用に整形済みの文字列
}
