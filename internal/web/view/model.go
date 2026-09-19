package view

import (
	"strings"

	"github.com/kodaikumatani/go-template-htmx/internal/domain"
	"github.com/kodaikumatani/go-template-htmx/internal/service"
)

// ViewModel は template に渡す専用の型。domain の型をそのまま渡さないことで、
// 「整形と分岐は Go 側、template は組み立てだけ」という境界を保つ。

// IssuesIndex is the ViewModel for GET /issues (ページ全体).
type IssuesIndex struct {
	Title  string
	Filter Filter
	Form   NewIssueForm
	List   IssueList
}

// IssueList is the ViewModel for the #issue-list fragment.
// ページ全体でも GET /issues/list でも、同じ型で同じ template を描く。
type IssueList struct {
	Meta   Meta
	Issues []Issue
}

// Meta is the count line above the rows.
//
// OOB が true のとき hx-swap-oob="true" が付き、
// 「レスポンスの主役ではないが、ついでに差し替えたい要素」になる。
type Meta struct {
	Total int
	OOB   bool
}

// NewIssueForm is the state of the create form.
type NewIssueForm struct {
	Title string
	Error string
	OOB   bool
}

// IssueCreated is the response of POST /issues.
// 1 レスポンスで「行の追加」「件数の更新」「フォームのクリア」を同時に行う。
type IssueCreated struct {
	Row  Issue
	Meta Meta
	Form NewIssueForm
}

// Filter is the state of the search form. 値を持たせておかないと、
// 部分更新後にリロードしたときに入力内容が消える。
type Filter struct {
	Q       string
	Options []StatusOption
}

// StatusOption is one <option> of the status filter.
type StatusOption struct {
	Value    string
	Label    string
	Selected bool
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

// NewIssueList builds the fragment ViewModel.
func NewIssueList(res service.ListResult) IssueList {
	return IssueList{
		Meta:   Meta{Total: res.Total},
		Issues: NewIssues(res.Issues),
	}
}

// NewIssuesIndex builds the page ViewModel.
func NewIssuesIndex(q service.ListQuery, res service.ListResult) IssuesIndex {
	return IssuesIndex{
		Title:  "Issues",
		Filter: NewFilter(q),
		List:   NewIssueList(res),
	}
}

// NewFilter builds the search form state from the current query.
func NewFilter(q service.ListQuery) Filter {
	return Filter{
		Q: q.Q,
		Options: []StatusOption{
			{Value: "", Label: "すべて", Selected: q.Status == ""},
			{Value: string(domain.StatusOpen), Label: "Open", Selected: q.Status == domain.StatusOpen},
			{Value: string(domain.StatusClosed), Label: "Closed", Selected: q.Status == domain.StatusClosed},
		},
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
