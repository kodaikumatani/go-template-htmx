package web

import (
	"net/http"
	"time"

	"github.com/kodaikumatani/go-template-htmx/internal/web/view"
)

type issueHandler struct {
	view *view.Renderer
	// Step 1 で *service.IssueService を持たせ、dummyIssues を置き換える。
}

// index handles GET /issues and returns the whole page.
func (h *issueHandler) index(w http.ResponseWriter, r *http.Request) {
	h.view.Page(w, http.StatusOK, "issues/index", view.IssuesIndex{
		Title:  "Issues",
		Issues: dummyIssues(),
	})
}

// dummyIssues is placeholder data for Step 0. Step 1 で SQLite に差し替える。
func dummyIssues() []view.Issue {
	created := time.Date(2026, 9, 19, 10, 0, 0, 0, time.Local)
	return []view.Issue{
		{ID: 1, Title: "検索が遅い", StatusLabel: "Open", IsOpen: true, CreatedAt: created.Format("2006-01-02 15:04")},
		{ID: 2, Title: "ログイン後にリダイレクトされない", StatusLabel: "Open", IsOpen: true, CreatedAt: created.Format("2006-01-02 15:04")},
		{ID: 3, Title: "READMEのセットアップ手順を更新", StatusLabel: "Closed", IsOpen: false, CreatedAt: created.Format("2006-01-02 15:04")},
	}
}
