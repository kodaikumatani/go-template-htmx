package web

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kodaikumatani/go-template-htmx/internal/domain"
	"github.com/kodaikumatani/go-template-htmx/internal/service"
	"github.com/kodaikumatani/go-template-htmx/internal/web/view"
)

type issueHandler struct {
	view    *view.Renderer
	issues  *service.IssueService
	logger  *slog.Logger
	perPage int
}

// index handles GET /issues and returns the whole page.
func (h *issueHandler) index(w http.ResponseWriter, r *http.Request) {
	q := h.listQuery(r)
	res, err := h.issues.List(r.Context(), q)
	if err != nil {
		h.serverError(w, r, err)
		return
	}
	h.view.Page(w, http.StatusOK, "issues/index", view.NewIssuesIndex(q, res))
}

// list handles GET /issues/list and returns only the #issue-list fragment.
//
// index との違いは「レイアウトを通すかどうか」だけで、
// 一覧そのものは同じ issues/list template を描いている。
// 同じ HTML を 2 箇所に書かないことが、この構成の一番のルール。
func (h *issueHandler) list(w http.ResponseWriter, r *http.Request) {
	res, err := h.issues.List(r.Context(), h.listQuery(r))
	if err != nil {
		h.serverError(w, r, err)
		return
	}
	h.view.Partial(w, http.StatusOK, "issues/list", view.NewIssueList(res))
}

// listQuery reads the list parameters out of the URL.
//
// クエリパラメータを読むのは「HTTP を知っている層」の仕事なので handler に置く。
// 絞り込み UI は Step 2、ページリンクは Step 6 で付けるが、
// URL を唯一の状態の置き場所にするため入口はここに固定しておく。
func (h *issueHandler) listQuery(r *http.Request) service.ListQuery {
	q := service.ListQuery{
		Q:       r.URL.Query().Get("q"),
		PerPage: h.perPage,
	}
	if status, ok := domain.ParseStatus(r.URL.Query().Get("status")); ok {
		q.Status = status
	}
	// 不正なページ番号は service 側で 1 に正規化されるので、ここでは弾かない。
	q.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	return q
}

func (h *issueHandler) serverError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error("request failed", "method", r.Method, "path", r.URL.RequestURI(), "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
