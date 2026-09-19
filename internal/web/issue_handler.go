package web

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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

// create handles POST /issues.
//
// 成功時に返すのは「作った 1 行 + 件数 + 空のフォーム」だけで、一覧全体は返さない。
// 失敗時に返すのはエラー入りのフォームだけ。
// どちらの場合も、返した HTML の行き先は htmx 側の属性が決めている。
func (h *issueHandler) create(w http.ResponseWriter, r *http.Request) {
	q := h.listQuery(r) // ParseForm もここで済む
	title := strings.TrimSpace(r.Form.Get("title"))

	issue, err := h.issues.Create(r.Context(), title)
	if err != nil {
		if msg, ok := validationMessage(err); ok {
			// htmx 4 は 4xx も swap するので、エラー入りのフォームをそのまま返せる。
			// 行き先はフォーム側の hx-status:422 が決める。
			h.view.Partial(w, http.StatusUnprocessableEntity, "issues/new-form",
				view.NewIssueForm{Title: title, Error: msg})
			return
		}
		h.serverError(w, r, err)
		return
	}

	total, err := h.issues.Count(r.Context(), q)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	h.view.Partial(w, http.StatusOK, "issues/created", view.IssueCreated{
		Row:  view.NewIssue(issue),
		Meta: view.Meta{Total: total, OOB: true},
		Form: view.NewIssueForm{OOB: true},
	})
}

// validationMessage maps an input error to a message for the user.
func validationMessage(err error) (string, bool) {
	switch {
	case errors.Is(err, domain.ErrTitleRequired):
		return "タイトルを入力してください", true
	case errors.Is(err, domain.ErrTitleTooLong):
		return fmt.Sprintf("タイトルは %d 文字以内にしてください", domain.MaxTitleLen), true
	default:
		return "", false
	}
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
	// ParseForm を通すと r.Form に「URL のクエリ + POST のボディ」が入る。
	// POST /issues も hx-include="#filters" で絞り込み条件を送ってくるので、
	// GET と POST で同じ読み取り方ができる。
	_ = r.ParseForm()

	q := service.ListQuery{
		Q:       r.Form.Get("q"),
		PerPage: h.perPage,
	}
	if status, ok := domain.ParseStatus(r.Form.Get("status")); ok {
		q.Status = status
	}
	// 不正なページ番号は service 側で 1 に正規化されるので、ここでは弾かない。
	q.Page, _ = strconv.Atoi(r.Form.Get("page"))
	return q
}

func (h *issueHandler) serverError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error("request failed", "method", r.Method, "path", r.URL.RequestURI(), "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
