package view

import (
	"strings"

	"github.com/kodaikumatani/go-template-htmx/internal/domain"
	"github.com/kodaikumatani/go-template-htmx/internal/service"
)

// ViewModel は template に渡す専用の型。domain の型をそのまま渡さないことで、
// 「整形と分岐は Go 側、template は組み立てだけ」という境界を保つ。

// IssuesIndex is the ViewModel for GET /issues.
type IssuesIndex struct {
	Title  string
	Total  int
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

const timeLayout = "2006-01-02 15:04"

// NewIssue converts a domain issue into its ViewModel.
func NewIssue(i domain.Issue) Issue {
	return Issue{
		ID:          i.ID,
		Title:       i.Title,
		StatusLabel: statusLabel(i.Status),
		IsOpen:      i.Status == domain.StatusOpen,
		// DB には UTC で入っているので、表示はローカルタイムに戻す。
		CreatedAt: i.CreatedAt.Local().Format(timeLayout),
	}
}

// NewIssues converts a slice of domain issues.
func NewIssues(issues []domain.Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, i := range issues {
		out = append(out, NewIssue(i))
	}
	return out
}

// NewIssuesIndex builds the page ViewModel from a use case result.
func NewIssuesIndex(res service.ListResult) IssuesIndex {
	return IssuesIndex{
		Title:  "Issues",
		Total:  res.Total,
		Issues: NewIssues(res.Issues),
	}
}

func statusLabel(s domain.Status) string {
	switch s {
	case domain.StatusOpen:
		return "Open"
	case domain.StatusClosed:
		return "Closed"
	default:
		// 想定外の値でも空欄にはせず、そのまま見せて気付けるようにする。
		return strings.ToUpper(string(s))
	}
}
