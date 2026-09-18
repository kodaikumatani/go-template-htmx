// Package web wires HTTP routes, templates and static assets together.
package web

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/kodaikumatani/go-template-htmx/internal/service"
	"github.com/kodaikumatani/go-template-htmx/internal/web/view"
)

// templates と static はバイナリに埋め込む。実行ディレクトリに依存しない。
//
//go:embed templates static
var assets embed.FS

// NewHandler builds the router. テンプレートの Parse もここで行うので、
// 壊れたテンプレートは起動時に error になる。
func NewHandler(logger *slog.Logger, issueService *service.IssueService) (http.Handler, error) {
	templatesFS, err := fs.Sub(assets, "templates")
	if err != nil {
		return nil, fmt.Errorf("sub templates: %w", err)
	}
	renderer, err := view.New(templatesFS)
	if err != nil {
		return nil, fmt.Errorf("new renderer: %w", err)
	}
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, fmt.Errorf("sub static: %w", err)
	}

	issues := &issueHandler{
		view:    renderer,
		issues:  issueService,
		logger:  logger,
		perPage: service.DefaultPerPage,
	}

	mux := http.NewServeMux()
	// Go 1.22+ の ServeMux は "METHOD /path/{param}" を解釈できるので
	// chi などのルータを入れずに済む。
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/issues", http.StatusFound)
	})
	mux.HandleFunc("GET /issues", issues.index)

	return requestLogger(logger, mux), nil
}

// requestLogger logs one line per request. HTMX の部分更新が実際に
// どの URL を叩いているのかを追いやすくするために入れている。
func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.RequestURI(),
			"status", rec.status,
			"htmx", r.Header.Get("HX-Request") == "true",
			"duration", time.Since(start).Round(100*time.Microsecond),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
