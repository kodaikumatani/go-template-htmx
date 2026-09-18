// Package view turns ViewModel values into HTML.
//
// テンプレートは 3 種類に分かれる。
//
//	layout.html       ... 全ページ共通の外枠。{{ template "content" . }} を持つ
//	pages/**/*.html   ... 1 ページ = 1 ファイル。{{ define "content" }} を持つ
//	partials/**/*.html ... fragment。{{ define "issues/row" }} のように一意な名前を持つ
//
// pages は layout + 全 partials を Clone したセットに 1 ファイルだけ足して作る。
// こうすると "content" という同じ名前をページごとに使い回せて、
// partials は名前指定で単独 Execute できる（= HTMX に返す fragment になる）。
package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

const (
	layoutFile  = "layout.html"
	layoutName  = "layout"
	contentName = "content"
	pagesDir    = "pages"
	partialsDir = "partials"
)

// Renderer holds the parsed template sets.
type Renderer struct {
	partials *template.Template            // layout + partials
	pages    map[string]*template.Template // "issues/index" -> layout + partials + そのページ
}

// New parses every template under fsys. テンプレートの構文エラーは
// リクエスト時ではなく起動時に落ちてほしいので、ここで全部 Parse しておく。
func New(fsys fs.FS) (*Renderer, error) {
	base, err := template.New(layoutFile).Funcs(funcMap()).ParseFS(fsys, layoutFile)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", layoutFile, err)
	}

	partialPaths, err := htmlPaths(fsys, partialsDir)
	if err != nil {
		return nil, err
	}
	if len(partialPaths) > 0 {
		if base, err = base.ParseFS(fsys, partialPaths...); err != nil {
			return nil, fmt.Errorf("parse partials: %w", err)
		}
	}

	pagePaths, err := htmlPaths(fsys, pagesDir)
	if err != nil {
		return nil, err
	}
	if len(pagePaths) == 0 {
		return nil, fmt.Errorf("no page templates under %s/", pagesDir)
	}

	pages := make(map[string]*template.Template, len(pagePaths))
	for _, p := range pagePaths {
		set, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone base for %s: %w", p, err)
		}
		if _, err := set.ParseFS(fsys, p); err != nil {
			return nil, fmt.Errorf("parse page %s: %w", p, err)
		}
		if set.Lookup(contentName) == nil {
			return nil, fmt.Errorf("page %s does not define %q", p, contentName)
		}
		pages[pageName(p)] = set
	}

	return &Renderer{partials: base, pages: pages}, nil
}

// Page renders a whole page through the layout. name は "issues/index" のような
// pages/ 以下の拡張子なしパス。
func (r *Renderer) Page(w http.ResponseWriter, status int, name string, data any) {
	set, ok := r.pages[name]
	if !ok {
		r.fail(w, fmt.Errorf("unknown page %q", name))
		return
	}
	r.execute(w, status, set, layoutName, data)
}

// Partial renders a single fragment. HTMX に部分更新を返すときはこれを使う。
func (r *Renderer) Partial(w http.ResponseWriter, status int, name string, data any) {
	if r.partials.Lookup(name) == nil {
		r.fail(w, fmt.Errorf("unknown partial %q", name))
		return
	}
	r.execute(w, status, r.partials, name, data)
}

// execute renders into a buffer first. 途中でエラーになったときに
// 壊れた HTML を書き出してしまわないようにするため。
func (r *Renderer) execute(w http.ResponseWriter, status int, set *template.Template, name string, data any) {
	var buf bytes.Buffer
	if err := set.ExecuteTemplate(&buf, name, data); err != nil {
		r.fail(w, fmt.Errorf("execute %q: %w", name, err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func (r *Renderer) fail(w http.ResponseWriter, err error) {
	slog.Error("render failed", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// htmlPaths lists *.html under dir recursively. dir がまだ無いときは空を返す。
func htmlPaths(fsys fs.FS, dir string) ([]string, error) {
	if _, err := fs.Stat(fsys, dir); err != nil {
		return nil, nil
	}
	var paths []string
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), !strings.HasSuffix(p, ".html"):
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", dir, err)
	}
	return paths, nil
}

func pageName(p string) string {
	rel := strings.TrimPrefix(p, pagesDir+"/")
	return strings.TrimSuffix(rel, path.Ext(rel))
}

// funcMap holds template helpers. 整形は基本 ViewModel 側でやるので、
// ここに増やすのは「テンプレートでしか決まらないもの」だけにする。
func funcMap() template.FuncMap {
	return template.FuncMap{}
}
