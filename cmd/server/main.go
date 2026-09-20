package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Issue は課題 1 件。DB はまだ使わず、メモリ上のスライスに持つ。
type Issue struct {
	ID     int
	Title  string
	Status string // "open" または "closed"
}

// issues はメモリ上のデータ。複数のリクエストが同時に触るので mutex で守る。
var (
	mu     sync.Mutex
	nextID = 4
)

var issues = []Issue{
	{ID: 1, Title: "検索が遅い", Status: "open"},
	{ID: 2, Title: "ログイン後にリダイレクトされない", Status: "open"},
	{ID: 3, Title: "READMEのセットアップ手順を更新", Status: "closed"},
}

// indexData はテンプレートに渡すデータ。テンプレートの中の . がこれになる。
type indexData struct {
	Title  string
	Issues []Issue
}

// filterIssues は status と検索語 q で絞り込む。どちらも空なら全件。
func filterIssues(status, q string) []Issue {
	out := []Issue{}
	for _, i := range issues {
		if status != "" && i.Status != status {
			continue
		}
		// 大文字小文字を区別せずタイトルの部分一致を見る
		if q != "" && !strings.Contains(strings.ToLower(i.Title), strings.ToLower(q)) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// pathID は URL の {id} を数値で取り出す。取れなければ 0。
func pathID(r *http.Request) int {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return 0
	}
	return id
}

// render はテンプレートを 1 つ描く。エラーはログに出すだけ。
func render(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Print(err)
	}
}

// findIssue は id の Issue を探す。
func findIssue(id int) (Issue, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, i := range issues {
		if i.ID == id {
			return i, true
		}
	}
	return Issue{}, false
}

// updateIssue は id の Issue のタイトルを書き換えて、更新後の値を返す。
func updateIssue(id int, title string) (Issue, bool) {
	mu.Lock()
	defer mu.Unlock()
	for n, i := range issues {
		if i.ID == id {
			issues[n].Title = title
			return issues[n], true
		}
	}
	return Issue{}, false
}

func main() {
	mux := http.NewServeMux()

	// ファイルが 2 つになった。index.html の中から list.html を呼べる。
	tmpl := template.Must(template.ParseFiles(
		"internal/web/templates/index.html",
		"internal/web/templates/list.html",
		"internal/web/templates/row.html",
		"internal/web/templates/new-form.html",
		"internal/web/templates/edit-row.html",
	))

	// htmx.min.js を配る。/static/htmx.min.js で参照できる。
	mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("internal/web/static"))))

	// ページ全体を返す
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data := indexData{Title: "Issues", Issues: filterIssues("", "")}
		if err := tmpl.Execute(w, data); err != nil {
			log.Print(err)
		}
	})

	// 一覧の部分だけを返す。htmx が呼ぶのはこちら。
	// ExecuteTemplate で list.html だけを描くので、<html> や <head> は付かない。
	mux.HandleFunc("GET /issues/list", func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		q := r.URL.Query().Get("q")
		data := indexData{Title: "Issues", Issues: filterIssues(status, q)}

		log.Printf("fragment: q=%q status=%q -> %d 件", q, status, len(data.Issues))

		if err := tmpl.ExecuteTemplate(w, "list.html", data); err != nil {
			log.Print(err)
		}
	})

	// 新しい Issue を作る。返すのは「作った 1 行」だけ。
	mux.HandleFunc("POST /issues", func(w http.ResponseWriter, r *http.Request) {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			// 204 No Content を返すと htmx は swap しない（= 何も起きない）。
			// 入力が空のときはこれで十分。
			log.Print("create: 空のタイトルなので無視")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		mu.Lock()
		issue := Issue{ID: nextID, Title: title, Status: "open"}
		nextID++
		issues = append([]Issue{issue}, issues...) // 先頭に足す
		mu.Unlock()

		log.Printf("create: %+v", issue)

		// レスポンスに 2 つ入れる。
		//   1. 作った行           → hx-target/hx-swap の指定どおり一覧の先頭へ
		//   2. 空の作成フォーム    → hx-swap-oob="true" が付いているので id で置き換わる
		if err := tmpl.ExecuteTemplate(w, "row.html", issue); err != nil {
			log.Print(err)
		}
		if err := tmpl.ExecuteTemplate(w, "new-form.html", true); err != nil {
			log.Print(err)
		}
	})

	// 表示状態の行を返す。編集のキャンセルで使う。
	mux.HandleFunc("GET /issues/{id}", func(w http.ResponseWriter, r *http.Request) {
		issue, ok := findIssue(pathID(r))
		if !ok {
			http.NotFound(w, r)
			return
		}
		render(w, tmpl, "row.html", issue)
	})

	// 編集フォームを返す。行と同じ id を持つので、行の位置に置き換わる。
	mux.HandleFunc("GET /issues/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		issue, ok := findIssue(pathID(r))
		if !ok {
			http.NotFound(w, r)
			return
		}
		log.Printf("edit form: id=%d", issue.ID)
		render(w, tmpl, "edit-row.html", issue)
	})

	// 更新して、更新後の行を返す。
	mux.HandleFunc("PATCH /issues/{id}", func(w http.ResponseWriter, r *http.Request) {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			// 空なら変更せず、元の行をそのまま返して編集を終える
			issue, ok := findIssue(pathID(r))
			if !ok {
				http.NotFound(w, r)
				return
			}
			render(w, tmpl, "row.html", issue)
			return
		}

		issue, ok := updateIssue(pathID(r), title)
		if !ok {
			http.NotFound(w, r)
			return
		}
		log.Printf("update: %+v", issue)
		render(w, tmpl, "row.html", issue)
	})

	log.Print("listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
